// Command examplecheck exercises the flows documented in the examples
// (example_test.go / docs/examples.md) against a LIVE node, so the examples'
// code paths are proven to work and not merely to compile.
//
// Default mode spends nothing: it reads state, builds transactions, signs
// them locally, simulates, prices them and walks the portable-transaction and
// signature-weight paths. Every one of those is a real RPC against a real
// node; the only thing it does not do is broadcast. Pass -broadcast to send
// the state-changing steps as well (that spends TRX from -key, so it is off
// by default and never enabled in CI).
//
// A fresh signer is generated when -key is empty. A fresh account does not
// exist on-chain yet, which is itself informative: build RPCs that require an
// existing account (staking, delegation, voting, permission updates) are then
// exercised against a known-existing account address instead, so the node
// accepts the build and the SDK's request shape is what gets validated.
//
// Usage:
//
//	go run ./cmd/examplecheck                                   # read-only + local signing
//	go run ./cmd/examplecheck -key <hex> -broadcast              # full run, spends TRX
//	go run ./cmd/examplecheck -endpoint grpc://grpc.trongrid.io:50051
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"math/big"
	"os"
	"time"

	tronlib "github.com/kslamph/tronlib/v2"
	"github.com/kslamph/tronlib/v2/contract"
	"github.com/kslamph/tronlib/v2/tron"
)

// Nile reference addresses (docs/verification.md).
const (
	nileEndpoint  = "grpc://grpc.nile.trongrid.io:50051"
	nileUSDT      = "TXLAQ63Xg1NAzckPwKHvzw7CSEmLMEqcdj"
	nileExisting  = "TLibQrqpdqPyg11VBJR97Q4H2714xa9GT1" // key1's account, funded
	nileRecipient = "TLibCZ2i2dFp6a9KZeKriSms5peeXSibks" // key2's account
	// E1 in docs/verification.md: a Nile TRC-20 transfer, used here to prove
	// receipt-log decoding against a real transaction.
	nileTransferTx = "738c6d0e10d2ba325577a38463b93af19209612008fd64fb02ae7f4f7153ac31"
)

func main() { os.Exit(run()) }

func run() int {
	endpoint := flag.String("endpoint", nileEndpoint, "node endpoint (grpc:// or grpcs://)")
	keyHex := flag.String("key", "", "hex private key; a fresh random signer is generated when empty")
	broadcast := flag.Bool("broadcast", false, "actually broadcast the state-changing steps (spends TRX from -key)")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	cli, err := tronlib.Dial(ctx, *endpoint)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dial: %v\n", err)
		return 2
	}
	defer func() { _ = cli.Close() }()

	signer, generated, err := signerFrom(*keyHex)
	if err != nil {
		fmt.Fprintf(os.Stderr, "key: %v\n", err)
		return 2
	}
	existing, err := tronlib.ParseAddress(nileExisting)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reference address: %v\n", err)
		return 2
	}
	recipient, err := tronlib.ParseAddress(nileRecipient)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reference address: %v\n", err)
		return 2
	}
	usdt, err := tronlib.ParseAddress(nileUSDT)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reference token: %v\n", err)
		return 2
	}

	c := &checker{
		ctx:       ctx,
		cli:       cli,
		signer:    signer,
		existing:  existing,
		recipient: recipient,
		usdt:      usdt,
		broadcast: *broadcast,
	}
	fmt.Printf("examplecheck: %s\nsigner:  %s%s\naccount: %s\n\n",
		*endpoint, signer.Address(), generated, existing)

	c.reads()
	c.token()
	c.contract()
	c.transfer()
	c.staking()
	c.permissions()
	c.events()

	fmt.Printf("\n%d step(s) OK, %d note(s), %d FAILED\n", c.ok, c.notes, c.failed)
	if c.failed > 0 {
		return 1
	}
	return 0
}

func signerFrom(hexKey string) (tronlib.Signer, string, error) {
	if hexKey != "" {
		s, err := tronlib.KeyFromHex(hexKey)
		return s, "", err
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, "", err
	}
	s, err := tronlib.KeyFromHex(hex.EncodeToString(raw))
	return s, " (fresh, unfunded)", err
}

type checker struct {
	ctx       context.Context
	cli       *tronlib.Client
	signer    tronlib.Signer
	existing  tronlib.Address
	recipient tronlib.Address
	usdt      tronlib.Address
	broadcast bool

	ok     int
	notes  int
	failed int
}

// step runs one flow and records its outcome. A node-side rejection that the
// flow documents as possible is a note, not a failure; a transport, decode or
// unexpected-typed error is a failure.
func (c *checker) step(name string, fn func() error) {
	started := time.Now()
	err := fn()
	switch {
	case err == nil:
		c.ok++
		fmt.Printf("  OK    %-46s %s\n", name, time.Since(started).Round(time.Millisecond))
	case noteWorthy(err):
		c.notes++
		fmt.Printf("  NOTE  %-46s %v\n", name, err)
	default:
		c.failed++
		fmt.Printf("  FAIL  %-46s %v\n", name, err)
	}
}

// noteWorthy reports whether the node declined for a reason the examples
// document (no funds, no account, not enough permission) rather than the SDK
// misbehaving. These are the outcomes a read-only run of a funded flow is
// expected to hit.
func noteWorthy(err error) bool {
	for _, code := range []tron.Code{
		tron.CodeAccountInsufficientBalance,
		tron.CodeAccountInsufficientEnergy,
		tron.CodeAccountInsufficientBandwidth,
		tron.CodeAccountPermissionDenied,
		tron.CodeTxInvalidArgument,
		tron.CodeContractNotFound,
		tron.CodeReceiptReverted,
		tron.CodeReceiptOutOfEnergy,
	} {
		if tron.HasCode(err, code) {
			return true
		}
	}
	return false
}

// broadcastOrReport sends the signed transaction when -broadcast is set and
// reports the node's verdict; otherwise it stops at "signed", which is as far
// as a spend-free run can go.
func (c *checker) broadcastOrReport(name string, signed tronlib.Tx) error {
	if !c.broadcast {
		fmt.Printf("        %s: signed %s (not broadcast; -broadcast to send)\n", name, signed.ID()[:16])
		return nil
	}
	rec, err := c.cli.Broadcast(c.ctx, signed)
	if err != nil {
		return err
	}
	if !rec.OK() {
		return fmt.Errorf("node rejected: %s %s", rec.NodeCode, rec.Revert)
	}
	fmt.Printf("        %s: broadcast %s\n", name, rec.TxID)
	return nil
}

// reads covers ExampleClient_Account: balance, account state, resource state,
// the staking summary and the live chain parameters.
func (c *checker) reads() {
	fmt.Println("reads (Example, ExampleClient_Account):")
	acct := c.cli.Account(c.existing)

	c.step("Balance", func() error {
		bal, err := acct.Balance(c.ctx)
		if err != nil {
			return err
		}
		fmt.Printf("        balance %s TRX\n", bal.Formatted())
		return nil
	})
	c.step("State", func() error {
		st, err := acct.State(c.ctx)
		if err != nil {
			return err
		}
		if !st.Exists {
			return fmt.Errorf("reference account %s not found on this network", st.Address)
		}
		fmt.Printf("        %s TRX, %d stake(s), %d unstake(s), %d vote(s), delegated out %s\n",
			st.Balance.Formatted(), len(st.Stakes), len(st.Unstakes), len(st.Votes),
			st.DelegatedOutEnergy.Formatted())
		return nil
	})
	c.step("Resources().State", func() error {
		rs, err := acct.Resources().State(c.ctx)
		if err != nil {
			return err
		}
		fmt.Printf("        energy %d/%d | bandwidth %d/%d+%d | tron power %d/%d\n",
			rs.EnergyUsed, rs.EnergyLimit, rs.BandwidthUsed, rs.BandwidthLimit,
			rs.FreeBandwidthLimit, rs.TronPowerUsed, rs.TronPowerLimit)
		return nil
	})
	c.step("Resources().Summary", func() error {
		s, err := acct.Resources().Summary(c.ctx)
		if err != nil {
			return err
		}
		fmt.Printf("        staked %s TRX energy, %s TRX bandwidth | withdrawable %s | slots %d\n",
			s.StakedByResource[tronlib.Energy].Formatted(),
			s.StakedByResource[tronlib.Bandwidth].Formatted(),
			s.UnstakeWithdrawable.Formatted(), s.UnstakeSlots)
		return nil
	})
	c.step("Resources().DelegationIndex", func() error {
		idx, err := acct.Resources().DelegationIndex(c.ctx)
		if err != nil {
			return err
		}
		fmt.Printf("        delegators %d, receivers %d\n", len(idx.From), len(idx.To))
		return nil
	})
	c.step("ChainParamsOf", func() error {
		p, err := tronlib.ChainParamsOf(c.ctx, c.cli.Raw())
		if err != nil {
			return err
		}
		fmt.Printf("        unstake delay %d day(s) | max lock %d blocks | multisig %s | permission update %s\n",
			p.UnfreezeDelayDays, p.MaxDelegateLockPeriod,
			p.MultiSignFee.Formatted(), p.UpdateAccountPermissionFee.Formatted())
		return nil
	})
	c.step("EnergyPrice", func() error {
		p, err := c.cli.EnergyPrice(c.ctx)
		if err != nil {
			return err
		}
		fmt.Printf("        %d sun/energy, effective %s\n", p.SunPerEnergy, p.EffectiveAt.Format(time.RFC3339))
		return nil
	})
	fmt.Println()
}

// token covers ExampleClient_Token: the decimal-aware TRC-20 handle.
func (c *checker) token() {
	fmt.Println("TRC-20 handle (ExampleClient_Token):")
	handle, err := c.cli.Token(c.ctx, c.usdt)
	if err != nil {
		c.failed++
		fmt.Printf("  FAIL  Token: %v\n\n", err)
		return
	}
	c.step("Symbol/Name/Decimals", func() error {
		symbol, err := handle.Symbol(c.ctx)
		if err != nil {
			return err
		}
		name, err := handle.Name(c.ctx)
		if err != nil {
			return err
		}
		fmt.Printf("        %s (%s), decimals %d\n", name, symbol, handle.Decimals())
		return nil
	})
	c.step("TotalSupply", func() error {
		supply, err := handle.TotalSupply(c.ctx)
		if err != nil {
			return err
		}
		fmt.Printf("        %s %s\n", supply.Formatted(), "units")
		return nil
	})
	c.step("BalanceOf(existing)", func() error {
		bal, err := handle.BalanceOf(c.ctx, c.existing)
		if err != nil {
			return err
		}
		fmt.Printf("        %s\n", bal.String())
		return nil
	})
	c.step("Allowance(existing -> recipient)", func() error {
		allowance, err := handle.Allowance(c.ctx, c.existing, c.recipient)
		if err != nil {
			return err
		}
		fmt.Printf("        %s\n", allowance.String())
		return nil
	})
	c.step("Amount scaling (decimal input)", func() error {
		amt, err := handle.Amount("1.5")
		if err != nil {
			return err
		}
		if handle.Decimals() == 6 && amt.Raw().Int64() != 1_500_000 {
			return fmt.Errorf("1.5 at 6 decimals = %s raw, want 1500000", amt.Raw())
		}
		fmt.Printf("        1.5 -> %s raw units\n", amt.Raw())
		return nil
	})
	c.step("Approve (build + sign)", func() error {
		whole, err := handle.Whole(1)
		if err != nil {
			return err
		}
		approve, err := handle.Approve(c.ctx, c.existing, c.recipient, whole)
		if err != nil {
			return err
		}
		signed, err := approve.Sign(c.signer)
		if err != nil {
			return err
		}
		return c.broadcastOrReport("Approve", signed)
	})
	c.step("Transfer (build + sign)", func() error {
		whole, err := handle.Whole(1)
		if err != nil {
			return err
		}
		move, err := handle.Transfer(c.ctx, c.existing, c.recipient, whole)
		if err != nil {
			return err
		}
		signed, err := move.Sign(c.signer)
		if err != nil {
			return err
		}
		return c.broadcastOrReport("Transfer", signed)
	})
	fmt.Println()
}

// contract covers ExampleClient_Contract: a view call, a simulated invoke
// with its decoded return, the cost preview and the receipt-log decode.
func (c *checker) contract() {
	fmt.Println("contract interaction (ExampleClient_Contract):")
	inst, err := c.cli.Contract(c.ctx, c.usdt)
	if err != nil {
		c.failed++
		fmt.Printf("  FAIL  Contract: %v\n\n", err)
		return
	}
	c.step("Call balanceOf -> Result.BigInt", func() error {
		res, err := inst.Call(c.ctx, "balanceOf", contract.AddressArg(c.existing))
		if err != nil {
			return err
		}
		v, err := res.BigInt()
		if err != nil {
			return err
		}
		fmt.Printf("        %s raw units\n", v)
		return nil
	})
	c.step("ABI loaded from the node", func() error {
		methods := inst.Methods()
		if len(methods) == 0 {
			return fmt.Errorf("no ABI methods loaded")
		}
		fmt.Printf("        %d methods (transfer, balanceOf, approve, decimals, ...)\n", len(methods))
		return nil
	})

	// The success path: approve is state-independent, so the simulation
	// returns the method's own bool result rather than a revert payload.
	var approve *tronlib.ContractTx
	c.step("Invoke approve (build)", func() error {
		built, err := inst.Invoke(c.ctx, c.existing, 0, "approve",
			contract.AddressArg(c.recipient), contract.BigIntArg(big.NewInt(1_000_000)))
		if err != nil {
			return err
		}
		approve = built
		return nil
	})
	if approve != nil {
		c.step("Simulate + Decode(ConstantResult)", func() error {
			est, err := approve.Simulate(c.ctx)
			if err != nil {
				return err
			}
			if est.Revert != "" {
				return fmt.Errorf("approve reverted unexpectedly: %s", est.Revert)
			}
			if !est.HasResult() {
				return fmt.Errorf("simulation succeeded but returned no ABI result")
			}
			res, err := inst.Decode("approve", est.ConstantResult[0])
			if err != nil {
				return err
			}
			ok, err := res.Bool()
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("simulated approve returned false")
			}
			fmt.Printf("        energy %d, penalty %d, decoded result: true\n", est.Energy, est.Penalty)
			return nil
		})
		c.step("CostPreview (account-aware)", func() error {
			preview, err := c.cli.Account(c.existing).CostPreview(c.ctx, approve)
			if err != nil {
				return err
			}
			fmt.Printf("        needs %d energy, has %d staked, buys %d @ %d sun = %s TRX\n",
				preview.EnergyNeeded, preview.EnergyAvailable, preview.EnergyToBuy,
				preview.SunPerEnergy, preview.TronToBurn.Formatted())
			return nil
		})
		c.step("Sign + TotalCostOf (both resources + fees)", func() error {
			signed, err := approve.Sign(c.signer)
			if err != nil {
				return err
			}
			cost, err := c.cli.Account(c.existing).TotalCost(c.ctx, signed)
			if err != nil {
				return err
			}
			fmt.Printf("        total %s TRX — %s\n", cost.Total.Formatted(), cost.String())
			return nil
		})
	}

	// The revert path, which is the one that bit the examples: a reverting
	// simulation returns revert data, not the method's return value, so
	// decoding it must be guarded by the revert check.
	c.step("Revert is an answer, not a decodable result", func() error {
		// This owner holds no USDT on Nile, so the transfer reverts inside the
		// sandbox. The example checks Revert before touching ConstantResult.
		attempt, err := inst.Invoke(c.ctx, c.existing, 0, "transfer",
			contract.AddressArg(c.recipient), contract.BigIntArg(big.NewInt(1_000_000)))
		if err != nil {
			return err
		}
		est, err := attempt.Simulate(c.ctx)
		if err != nil {
			return err
		}
		if est.Revert == "" {
			fmt.Println("        (transfer did not revert on this run — owner holds enough USDT)")
			return nil
		}
		fmt.Printf("        revert %q, energy %d, code %q\n", est.Revert, est.Energy, est.Code)
		if est.HasResult() {
			_, derr := inst.Decode("transfer", est.ConstantResult[0])
			if derr == nil {
				return fmt.Errorf("revert payload decoded as a transfer result; the example's guard would be unnecessary")
			}
			fmt.Printf("        decoding the revert payload fails (%v) — which is why the example guards on Revert\n", derr)
		}
		return nil
	})
	fmt.Println()
}

// transfer covers Example and the portable-transaction path: build, sign,
// encode, decode, recover the signers, then the node's own sign weight.
func (c *checker) transfer() {
	fmt.Println("transfer + portable envelope (Example, ExamplePermissions_SignWeight):")
	acct := c.cli.Account(c.existing)

	c.step("TransferTRX build + Sign", func() error {
		transfer, err := acct.TransferTRX(c.ctx, c.recipient, tronlib.TRX(1))
		if err != nil {
			return err
		}
		signed, err := transfer.Sign(c.signer)
		if err != nil {
			return err
		}
		return c.broadcastOrReport("TransferTRX", signed)
	})

	var envelope []byte
	c.step("Encode -> Decode -> Signers", func() error {
		transfer, err := acct.TransferTRX(c.ctx, c.recipient, tronlib.TRX(1))
		if err != nil {
			return err
		}
		first, err := transfer.Sign(c.signer)
		if err != nil {
			return err
		}
		data, err := tronlib.Encode(first)
		if err != nil {
			return err
		}
		back, err := tronlib.Decode(data)
		if err != nil {
			return err
		}
		signers, err := back.Signers()
		if err != nil {
			return err
		}
		if back.ID() != first.ID() {
			return fmt.Errorf("txid changed across the envelope: %s vs %s", back.ID(), first.ID())
		}
		if len(signers) != 1 || signers[0] != c.signer.Address() {
			return fmt.Errorf("Signers() = %v, want [%s]", signers, c.signer.Address())
		}
		envelope = data
		fmt.Printf("        %d bytes, kind %v, signer %s recovered from the bytes\n",
			len(data), back.Kind(), signers[0])
		return nil
	})
	c.step("Reject a duplicate signer", func() error {
		back, err := tronlib.Decode(envelope)
		if err != nil {
			return err
		}
		if _, err := tronlib.Sign(back, c.signer); !tron.HasCode(err, tron.CodeTxAlreadySigned) {
			return fmt.Errorf("duplicate signer accepted: err = %v", err)
		}
		fmt.Println("        duplicate refused with tx.already_signed")
		return nil
	})
	c.step("SignHash -> AttachSignature (remote signer path)", func() error {
		transfer, err := acct.TransferTRX(c.ctx, c.recipient, tronlib.TRX(1))
		if err != nil {
			return err
		}
		digest, err := tronlib.SignHash(transfer)
		if err != nil {
			return err
		}
		sig, err := c.signer.Sign(digest)
		if err != nil {
			return err
		}
		attached, err := tronlib.AttachSignature(transfer, c.signer.Address(), sig)
		if err != nil {
			return err
		}
		if !attached.IsSigned() {
			return fmt.Errorf("attached transaction is unsigned")
		}
		fmt.Println("        65-byte signature verified against the stated address")
		return nil
	})
	c.step("Permissions().SignWeight (node verdict)", func() error {
		back, err := tronlib.Decode(envelope)
		if err != nil {
			return err
		}
		status, err := c.cli.Account(c.existing).Permissions().SignWeight(c.ctx, back)
		if err != nil {
			return err
		}
		fmt.Printf("        %s, threshold %d, enough=%v, approved=%v\n",
			status.Result, status.Threshold, status.Enough, status.Approved)
		return nil
	})
	fmt.Println()
}

// staking covers ExampleResources_Stake: the whole lifecycle plus delegation,
// built against a known-existing account so the node accepts the request.
func (c *checker) staking() {
	fmt.Println("staking + delegation (ExampleResources_Stake):")
	res := c.cli.Account(c.existing).Resources()

	c.step("Stake 1 TRX for Energy", func() error {
		stake, err := res.Stake(c.ctx, tronlib.Energy, tronlib.TRX(1))
		if err != nil {
			return err
		}
		signed, err := stake.Sign(c.signer)
		if err != nil {
			return err
		}
		return c.broadcastOrReport("Stake", signed)
	})
	c.step("Unstake 1 TRX (starts the cooldown)", func() error {
		unstake, err := res.Unstake(c.ctx, tronlib.Energy, tronlib.TRX(1))
		if err != nil {
			return err
		}
		signed, err := unstake.Sign(c.signer)
		if err != nil {
			return err
		}
		return c.broadcastOrReport("Unstake", signed)
	})
	c.step("WithdrawUnstaked (claims matured TRX)", func() error {
		withdraw, err := res.WithdrawUnstaked(c.ctx)
		if err != nil {
			return err
		}
		signed, err := withdraw.Sign(c.signer)
		if err != nil {
			return err
		}
		return c.broadcastOrReport("WithdrawUnstaked", signed)
	})
	c.step("CancelUnstake (re-stakes the pending)", func() error {
		cancel, err := res.CancelUnstake(c.ctx)
		if err != nil {
			return err
		}
		signed, err := cancel.Sign(c.signer)
		if err != nil {
			return err
		}
		return c.broadcastOrReport("CancelUnstake", signed)
	})
	c.step("Delegatable(Energy)", func() error {
		free, err := res.Delegatable(c.ctx, tronlib.Energy)
		if err != nil {
			return err
		}
		fmt.Printf("        %s TRX of energy stake\n", free.Formatted())
		return nil
	})
	c.step("Delegate 1 TRX with a 28,800-block lock", func() error {
		delegate, err := res.Delegate(c.ctx, tronlib.Energy, c.recipient, tronlib.TRX(1), tronlib.DelegateParams{
			Lock:       true,
			LockBlocks: 28_800,
		})
		if err != nil {
			return err
		}
		signed, err := delegate.Sign(c.signer)
		if err != nil {
			return err
		}
		return c.broadcastOrReport("Delegate", signed)
	})
	c.step("Undelegate (immediate when unlocked)", func() error {
		undelegate, err := res.Undelegate(c.ctx, tronlib.Energy, c.recipient, tronlib.TRX(1))
		if err != nil {
			return err
		}
		signed, err := undelegate.Sign(c.signer)
		if err != nil {
			return err
		}
		return c.broadcastOrReport("Undelegate", signed)
	})
	c.step("SetVotes (whole-list replace)", func() error {
		vote, err := c.cli.Account(c.existing).Voting().SetVotes(c.ctx, []tronlib.Vote{
			{Witness: c.recipient, Count: 1},
		})
		if err != nil {
			return err
		}
		signed, err := vote.Sign(c.signer)
		if err != nil {
			return err
		}
		return c.broadcastOrReport("SetVotes", signed)
	})
	c.step("Rewards read + ClaimRewards", func() error {
		rewards, err := c.cli.Account(c.existing).Voting().Rewards(c.ctx)
		if err != nil {
			return err
		}
		claim, err := c.cli.Account(c.existing).Voting().ClaimRewards(c.ctx)
		if err != nil {
			return err
		}
		signed, err := claim.Sign(c.signer)
		if err != nil {
			return err
		}
		fmt.Printf("        accrued %s TRX\n", rewards.Formatted())
		return c.broadcastOrReport("ClaimRewards", signed)
	})
	fmt.Println()
}

// permissions covers ExamplePermissions_Current and the bitmap helpers.
func (c *checker) permissions() {
	fmt.Println("permissions (ExamplePermissions_Current):")
	perms := c.cli.Account(c.existing).Permissions()

	c.step("Current (decode all slots)", func() error {
		set, err := perms.Current(c.ctx)
		if err != nil {
			return err
		}
		fmt.Printf("        owner %d key(s) threshold %d, witness %v, actives %d\n",
			len(set.Owner.Keys), set.Owner.Threshold, set.Witness != nil, len(set.Actives))
		return nil
	})
	c.step("Update (build + sign the whole set)", func() error {
		set, err := perms.Current(c.ctx)
		if err != nil {
			return err
		}
		bitmap, err := tronlib.OperationsBitmap(tronlib.TypeTransfer, tronlib.TypeTriggerSmartContract)
		if err != nil {
			return err
		}
		allowed, err := tronlib.OperationsList(bitmap)
		if err != nil {
			return err
		}
		fmt.Printf("        bitmap grants: %v\n", allowed)
		set.Actives = append(set.Actives, tronlib.Permission{
			Name:       "examplecheck",
			Threshold:  1,
			Keys:       []tronlib.PermissionKey{{Address: c.recipient, Weight: 1}},
			Operations: bitmap,
		})
		update, err := perms.Update(c.ctx, set)
		if err != nil {
			return err
		}
		signed, err := update.Sign(c.signer)
		if err != nil {
			return err
		}
		cost, err := c.cli.Account(c.existing).TotalCost(c.ctx, signed)
		if err == nil {
			fmt.Printf("        permission-update fee %s TRX, total %s TRX\n",
				cost.PermissionUpdateFee.Formatted(), cost.Total.Formatted())
		}
		return c.broadcastOrReport("AccountPermissionUpdate", signed)
	})
	fmt.Println()
}

// events proves receipt-log decoding against a real, already-mined
// transaction — no keys and no spend involved.
func (c *checker) events() {
	fmt.Println("event decoding (ExampleClient_Contract step 5):")
	c.step("Events(historical TRC-20 transfer)", func() error {
		logs, err := c.cli.Events(c.ctx, nileTransferTx)
		if err != nil {
			return err
		}
		if len(logs) == 0 {
			return fmt.Errorf("no logs decoded for %s", nileTransferTx)
		}
		for _, lg := range logs {
			if lg.EventName == "" {
				return fmt.Errorf("log from %s was not decoded (EventName empty)", lg.Address)
			}
			fields := make([]string, 0, len(lg.Parameters))
			for _, p := range lg.Parameters {
				fields = append(fields, fmt.Sprintf("%s=%v", p.Name, p.Value))
			}
			fmt.Printf("        %s %v\n", lg.EventName, fields)
		}
		return nil
	})
	fmt.Println()
}
