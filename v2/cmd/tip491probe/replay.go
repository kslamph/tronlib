package main

// Exact-replay verification for spec §7.5 item 2 on penalized contracts.
//
// The library's exactness contract is: Simulate.Energy and Simulate.Penalty
// equal the broadcast receipt's EnergyUsageTotal and EnergyPenaltyTotal
// EXACTLY — no tolerances — provided the broadcast lands in the same
// maintenance cycle as the simulation (the factor is hoisted once per
// execution from the cycle-caught-up state, so it cannot move mid-cycle).
// The 2026-09-01 Nile run proved this for a penalty-free call (delta 0);
// this mode proves it for a penalized call without spending anything, by
// re-simulating the identical calldata of an already-broadcast mainnet
// transaction and comparing against its receipt.
//
// A mismatch is a verdict (exit 1), never a tolerance debate: either the
// replayed state moved (different cycle, state-dependent branching) or the
// exactness claim is wrong, and the printed numbers say which.

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	tronlib "github.com/kslamph/tronlib/v2"
	"github.com/kslamph/tronlib/v2/rpc"
	"github.com/kslamph/tronlib/v2/tron"
	"github.com/kslamph/tronlib/v2/tx"
	"google.golang.org/protobuf/proto"
)

// extractTrigger pulls the TriggerSmartContract message out of a broadcast
// transaction. The contract-type check before the unmarshal is the F1 rule:
// unmarshalling an unchecked contract parameter as TriggerSmartContract is
// the defect spec §6 exists to eliminate.
func extractTrigger(btx *core.Transaction) (*core.TriggerSmartContract, error) {
	const op = "tip491probe.extractTrigger"
	contracts := btx.GetRawData().GetContract()
	if len(contracts) != 1 {
		return nil, fmt.Errorf("%s: transaction holds %d contracts, want exactly 1", op, len(contracts))
	}
	c := contracts[0]
	if c.GetType() != core.Transaction_Contract_TriggerSmartContract {
		return nil, fmt.Errorf("%s: contract type %s is not TriggerSmartContract", op, c.GetType())
	}
	var req core.TriggerSmartContract
	if err := proto.Unmarshal(c.GetParameter().GetValue(), &req); err != nil {
		return nil, fmt.Errorf("%s: parameter does not decode as TriggerSmartContract: %w", op, err)
	}
	return &req, nil
}

// replayVerdict compares a re-simulation against its broadcast receipt.
// Exact equality on both fields is the only pass: energy and penalty are
// integers produced by the same VM path, so any difference is information,
// not noise.
func replayVerdict(est *tx.Estimate, receipt *core.ResourceReceipt) (detail string, pass bool) {
	wantEnergy, wantPenalty := receipt.GetEnergyUsageTotal(), receipt.GetEnergyPenaltyTotal()
	detail = fmt.Sprintf("replay energy=%d penalty=%d base=%d vs receipt energy=%d penalty=%d base=%d",
		est.Energy, est.Penalty, est.Energy-est.Penalty,
		wantEnergy, wantPenalty, wantEnergy-wantPenalty)
	return detail, est.Energy == wantEnergy && est.Penalty == wantPenalty
}

// runReplay fetches a broadcast transaction and its receipt, re-simulates
// the identical calldata, and asserts exact equality. Exit 0 on exact
// match, 1 on mismatch, 2 when the inputs cannot be assembled (bad txid,
// receipt not yet available, non-contract transaction).
func runReplay(ctx context.Context, cli *tronlib.Client, txidHex string) int {
	txid, err := hex.DecodeString(strings.TrimPrefix(txidHex, "0x"))
	if err != nil || len(txid) != 32 {
		fmt.Fprintf(os.Stderr, "replay: txid must be 32 hex bytes: %q\n", txidHex)
		return 2
	}
	cp := cli.Raw()

	btx, err := withRetry("fetch transaction", func() (*core.Transaction, error) {
		return rpc.GetTransactionById(cp, ctx, &api.BytesMessage{Value: txid})
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "replay: %v\n", err)
		return 2
	}
	info, err := withRetry("fetch receipt", func() (*core.TransactionInfo, error) {
		return rpc.GetTransactionInfoById(cp, ctx, &api.BytesMessage{Value: txid})
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "replay: %v\n", err)
		return 2
	}
	if len(info.GetId()) == 0 || info.GetReceipt() == nil {
		fmt.Fprintln(os.Stderr, "replay: receipt not available (transaction unknown or not yet solidified)")
		return 2
	}
	rec := info.GetReceipt()
	fmt.Printf("receipt: block=%d energy=%d penalty=%d base=%d origin=%d\n",
		info.GetBlockNumber(), rec.GetEnergyUsageTotal(), rec.GetEnergyPenaltyTotal(),
		rec.GetEnergyUsageTotal()-rec.GetEnergyPenaltyTotal(), rec.GetOriginEnergyUsage())

	req, err := extractTrigger(btx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "replay: %v\n", err)
		return 2
	}
	owner, err := tron.AddressFromBytes(req.GetOwnerAddress())
	if err != nil {
		fmt.Fprintf(os.Stderr, "replay: owner address: %v\n", err)
		return 2
	}
	contract, err := tron.AddressFromBytes(req.GetContractAddress())
	if err != nil {
		fmt.Fprintf(os.Stderr, "replay: contract address: %v\n", err)
		return 2
	}

	ct, err := withRetry("build", func() (*tx.ContractTx, error) {
		return tx.BuildTriggerSmartContract(cp, ctx, owner, contract, req.GetData(), tron.SUN(req.GetCallValue()))
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "replay build: %v\n", err)
		return 2
	}
	est, err := withRetry("simulate", func() (*tx.Estimate, error) {
		return ct.Simulate(ctx)
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "replay simulate: %v\n", err)
		return 2
	}
	if est.Code != "" || est.Revert != "" {
		fmt.Printf("replay energy=%d penalty=%d base=%d code=%s revert=%q\n",
			est.Energy, est.Penalty, est.Energy-est.Penalty, est.Code, est.Revert)
		fmt.Fprintln(os.Stderr, "replay did not execute (revert or node rejection): chain state moved since the broadcast — the comparison is void, not a mismatch")
		return 1
	}

	detail, pass := replayVerdict(est, rec)
	fmt.Println(detail)
	if pass {
		fmt.Println("PASS: replay matches the broadcast receipt exactly (spec §7.5 item 2, penalized)")
		return 0
	}
	fmt.Fprintln(os.Stderr, "MISMATCH (not verified): replay and receipt disagree — same-cycle replay should be exact")
	return 1
}
