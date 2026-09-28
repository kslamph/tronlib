// Command tip491probe is a manual harness for spec §7.5 item 6: it checks
// whether a node's simulated energy estimate already includes the TIP-491
// dynamic-energy penalty.
//
// It performs constant calls only — it NEVER signs or broadcasts, and needs no
// funded key. Usage:
//
//	go -C v2 run ./cmd/tip491probe -contract <addr> -data <hex> [-key <hex> | -owner <addr>] [-endpoint <url>]
//
// Exit codes: 0 = penalty observed (item 6 verified); 1 = inconclusive (zero
// or negative penalty — see checkPenalty); 2 = bad flags or a failed setup.
package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kslamph/tronlib/pb/api"
	tronlib "github.com/kslamph/tronlib/v2"
	"github.com/kslamph/tronlib/v2/rpc"
	"github.com/kslamph/tronlib/v2/tx"
)

func main() { os.Exit(run()) }

func run() int {
	endpoint := flag.String("endpoint", "grpc://grpc.nile.trongrid.io:50051", "node endpoint (grpc:// or grpcs://)")
	contractStr := flag.String("contract", "", "target contract address (required)")
	ownerStr := flag.String("owner", "", "caller address (defaults to -key's address)")
	keyHex := flag.String("key", "", "hex private key, optional — never used to sign")
	dataHex := flag.String("data", "", "hex-encoded calldata (required)")
	flag.Parse()

	if *contractStr == "" || *dataHex == "" {
		fmt.Fprintln(os.Stderr, "usage: tip491probe -contract <addr> -data <hex> [-key <hex> | -owner <addr>] [-endpoint <url>]")
		return 2
	}

	owner, ok, err := resolveOwner(*keyHex, *ownerStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "owner: %v\n", err)
		return 2
	}
	if !ok {
		fmt.Fprintln(os.Stderr, "provide -key or -owner")
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cli, err := tronlib.Dial(ctx, *endpoint)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dial: %v\n", err)
		return 2
	}
	defer func() { _ = cli.Close() }()

	printDynamicParams(cli)

	ct, err := tx.BuildTriggerSmartContract(cli.Raw(), ctx, owner, contract, data, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build: %v\n", err)
		return 2
	}
	est, err := ct.Simulate(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "simulate: %v\n", err)
		return 2
	}
	fmt.Printf("simulate: energy=%d penalty=%d base=%d\n", est.Energy, est.Penalty, est.Energy-est.Penalty)

	if err := checkPenalty(est.Penalty); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	fmt.Println("PASS: the simulated estimate carries a TIP-491 penalty (spec §7.5 item 6)")
	return 0
}

// resolveOwner returns the -key address when a key is given, else the parsed
// -owner. ok is false when neither is supplied; no signing ever happens.
func resolveOwner(keyHex, ownerStr string) (owner tronlib.Address, ok bool, err error) {
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
