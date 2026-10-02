// Command tip491probe is a manual harness: it checks
// whether a node's simulated energy estimate already includes the TIP-491
// dynamic-energy penalty, and cross-checks the node's two independent
// factor reports (GetContractInfo's stored factor vs the factor derived
// from Simulate's energy/penalty pair) against the per-opcode formula the
// library implements (tx.DynamicEnergy.PredictPenalty).
//
// It performs constant calls only — it NEVER signs or broadcasts, and needs no
// funded key. Usage:
//
//	go run ./cmd/tip491probe -contract <addr> -data <hex> [-key <hex> | -owner <addr> | -random-owner] [-endpoint <url>]
//
// Exit codes (factor mode): 0 = factor observed live with all cross-checks
// agreeing; 1 = inconclusive (zero factor) or cross-check mismatch;
// 2 = bad flags or a failed setup.
//
// A second mode replays an already-broadcast transaction exactly:
//
//	go run ./cmd/tip491probe -replay <txid-hex> [-endpoint <url>]
//
// It fetches the transaction and its receipt, re-simulates the identical
// calldata, and asserts EXACT equality of energy and penalty (no
// tolerances). Exit 0 on exact match, 1 on mismatch, 2 on setup failure.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	tronlib "github.com/kslamph/tronlib/v2"
	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/rpc"
	"github.com/kslamph/tronlib/v2/tron"
	"github.com/kslamph/tronlib/v2/tx"
)

func main() { os.Exit(run()) }

func run() int {
	endpoint := flag.String("endpoint", "grpc://grpc.nile.trongrid.io:50051", "node endpoint (grpc:// or grpcs://)")
	replayHex := flag.String("replay", "", "txid hex: exact-replay mode (fetch tx + receipt, re-simulate, assert exact match)")
	contractStr := flag.String("contract", "", "target contract address (required)")
	ownerStr := flag.String("owner", "", "caller address (defaults to -key's address)")
	randomOwnerFlag := flag.Bool("random-owner", false, "generate a fresh random owner address on the fly (constant calls need no key)")
	keyHex := flag.String("key", "", "hex private key, optional — never used to sign")
	dataHex := flag.String("data", "", "hex-encoded calldata (required)")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cli, err := tronlib.Dial(ctx, *endpoint)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dial: %v\n", err)
		return 2
	}
	defer func() { _ = cli.Close() }()

	if *replayHex != "" {
		return runReplay(ctx, cli, *replayHex)
	}

	if *contractStr == "" || *dataHex == "" {
		fmt.Fprintln(os.Stderr, "usage: tip491probe -contract <addr> -data <hex> [-key <hex> | -owner <addr> | -random-owner] [-endpoint <url>]")
		fmt.Fprintln(os.Stderr, "   or: tip491probe -replay <txid-hex> [-endpoint <url>]")
		return 2
	}

	owner, ok, err := resolveOwner(*keyHex, *ownerStr, *randomOwnerFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "owner: %v\n", err)
		return 2
	}
	if !ok {
		fmt.Fprintln(os.Stderr, "provide -key, -owner, or -random-owner")
		return 2
	}

	contract, err := tronlib.ParseAddress(*contractStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "contract: %v\n", err)
		return 2
	}
	data, err := hex.DecodeString(strings.TrimPrefix(*dataHex, "0x"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "data: %v\n", err)
		return 2
	}

	printDynamicParams(cli)

	ct, err := withRetry("build", func() (*tx.ContractTx, error) {
		return tx.BuildTriggerSmartContract(ctx, cli.Raw(), owner, contract, data, 0)
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "build: %v\n", err)
		return 2
	}
	est, err := withRetry("simulate", func() (*tx.Estimate, error) {
		return ct.Simulate(ctx)
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "simulate: %v\n", err)
		return 2
	}
	fmt.Printf("simulate: energy=%d penalty=%d base=%d\n", est.Energy, est.Penalty, est.Energy-est.Penalty)

	dyn, err := withRetry("contract state", func() (*tx.DynamicEnergy, error) {
		return tx.DynamicEnergyOf(ctx, cli.Raw(), contract)
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "contract state: %v\n", err)
		return 2
	}
	fmt.Printf("contract state: factor=%d usage=%d cycle=%d\n", dyn.Factor, dyn.Usage, dyn.UpdateCycle)

	v := verifyFactor(dyn, est)
	fmt.Printf("cross-check: stored=%d derived=%d predicted=%d actual=%d\n",
		v.infoFactor, v.simFactor, v.predicted, v.actual)
	if est.Code != "" || est.Revert != "" {
		fmt.Printf("simulate note: code=%s revert=%q (a non-executing call still verifies the accounting, but not the happy path)\n", est.Code, est.Revert)
	}
	switch {
	case v.pass:
		fmt.Printf("PASS: %s\n", v.detail)
		return 0
	case v.live:
		fmt.Fprintf(os.Stderr, "MISMATCH (not verified): %s\n", v.detail)
		return 1
	default:
		fmt.Fprintf(os.Stderr, "%s\n", v.detail)
		return 1
	}
}

// withRetry runs a read-only probe step up to 4 times with linear backoff.
// The public gateways rate-limit aggressively (HTTP 429 on the reflection
// endpoint was observed); every step here is a constant call, so retrying
// is safe — nothing is built, signed or broadcast. A *tron.Error carrying
// a non-transport code (a node verdict, not a transport failure) is
// returned immediately without retry.
func withRetry[T any](what string, fn func() (T, error)) (T, error) {
	var zero T
	backoff := []time.Duration{0, 2 * time.Second, 5 * time.Second, 10 * time.Second}
	for i, wait := range backoff {
		if wait > 0 {
			fmt.Printf("%s: attempt %d after %s\n", what, i+1, wait)
			time.Sleep(wait)
		}
		out, err := fn()
		if err == nil {
			return out, nil
		}
		if !isRetryable(err) || i == len(backoff)-1 {
			return zero, err
		}
		fmt.Printf("%s: transient failure (%v), retrying\n", what, err)
	}
	return zero, errors.New("unreachable")
}

// isRetryable reports whether a failed read-only step is worth retrying.
// Fix-call codes (bad address, no contract, malformed input) are node
// verdicts — retrying is pointless. Everything else (transport failures,
// rate limits, node timeouts) may clear on retry; all probe steps are
// constant calls, so a retry can never double-spend or mutate state.
func isRetryable(err error) bool {
	for _, c := range []tron.Code{
		tron.CodeContractNotFound,
		tron.CodeAddressInvalid,
		tron.CodeAmountInvalid,
		tron.CodeAmountTooManyDecimals,
		tron.CodeTxInvalidArgument,
	} {
		if tron.HasCode(err, c) {
			return false
		}
	}
	return true
}

// generated random address with -random-owner, else the parsed -owner.
// ok is false when no owner source is supplied; no signing ever happens —
// constant calls only need a syntactically valid caller address.
func resolveOwner(keyHex, ownerStr string, randomOwnerFlag bool) (owner tronlib.Address, ok bool, err error) {
	if randomOwnerFlag {
		a, err := randomOwner()
		if err != nil {
			return tronlib.Address{}, false, err
		}
		return a, true, nil
	}
	if keyHex != "" {
		s, err := tronlib.KeyFromHex(keyHex)
		if err != nil {
			return tronlib.Address{}, false, err
		}
		return s.Address(), true, nil
	}
	if ownerStr != "" {
		a, err := tronlib.ParseAddress(ownerStr)
		if err != nil {
			return tronlib.Address{}, false, err
		}
		return a, true, nil
	}
	return tronlib.Address{}, false, nil
}

// randomOwner generates a fresh secp256k1 key from crypto/rand and returns
// its address. The key is never used to sign — constant calls accept any
// valid caller — so this is just an on-the-fly source of a well-formed
// owner address with no key management involved.
func randomOwner() (tronlib.Address, error) {
	for i := 0; i < 3; i++ {
		var raw [32]byte
		if _, err := rand.Read(raw[:]); err != nil {
			return tronlib.Address{}, err
		}
		s, err := tronlib.KeyFromHex(hex.EncodeToString(raw[:]))
		if err == nil {
			return s.Address(), nil
		}
	}
	return tronlib.Address{}, fmt.Errorf("random key generation failed the curve-range check 3 times in a row")
}

// printDynamicParams prints the node's dynamic-energy governance parameters,
// best-effort: a read failure is reported, never fatal, because the probe's
// verdict rests on the simulated penalty alone.
func printDynamicParams(cli *tronlib.Client) {
	cp, err := rpc.GetChainParameters(cli.Raw(), context.Background(), &api.EmptyMessage{})
	if err != nil {
		fmt.Printf("chain params: %v\n", err)
		return
	}
	for _, p := range cp.GetChainParameter() {
		if strings.Contains(p.GetKey(), "Dynamic") {
			fmt.Printf("%s = %d\n", p.GetKey(), p.GetValue())
		}
	}
}
