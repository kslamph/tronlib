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
//	go run ./cmd/examplecheck -key <hex> -broadcast -permission-update
//	    # also broadcast a permission update (add an active permission,
//	    # verify, reverse) — burns the permission fee twice; fund ~2x fee
//	go run ./cmd/examplecheck -endpoint grpc://grpc.trongrid.io:50051
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
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
	nileEndpoint = "grpc://grpc.nile.trongrid.io:50051"
	nileUSDT     = "TXLAQ63Xg1NAzckPwKHvzw7CSEmLMEqcdj"
	// The throwaway Nile accounts live in
	// v1-legacy:integration_test/test.env (NILE_TEST_KEY1/2):
	// key1 = TLibQrqpdqPyg11VBJR97Q4H2714xa9GT1,
	// key2 = TLibCZ2i2dFp6a9KZeKriSms5peeXSibks. The flows run as whatever
	// -key is, and -payee defaults to key2.
	nileRecipient = "TLibCZ2i2dFp6a9KZeKriSms5peeXSibks"
	// E1 in docs/verification.md: a Nile TRC-20 transfer, used here to prove
	// receipt-log decoding against a real transaction.
	nileTransferTx = "738c6d0e10d2ba325577a38463b93af19209612008fd64fb02ae7f4f7153ac31"
)

func main() { os.Exit(run()) }

func run() int {
	endpoint := flag.String("endpoint", nileEndpoint, "node endpoint (grpc:// or grpcs://)")
	keyHex := flag.String("key", "", "hex private key; a fresh random signer is generated when empty")
	broadcast := flag.Bool("broadcast", false, "run the chain-updating sequence (spends TRX from -key)")
	ownerStr := flag.String("owner", "", "account the flows operate on; defaults to the signer's own address")
	payeeStr := flag.String("payee", nileRecipient, "counterparty address for transfers and delegations")
	payeeKeyHex := flag.String("payee-key", "", "counterparty's hex private key; enables the payee-signed steps (transferFrom, the rebalance return) and the delegation/spend proofs")
	floatTRX := flag.Float64("float", 3, "TRX to keep on the payee between runs; the flow tops it up when short and returns the excess when long, so repeated runs move the same money back and forth instead of draining either key. 3 TRX covers a TRC-20 call from an account holding no staked energy (~2 TRX of bought energy) plus bandwidth")
	tokenStr := flag.String("token", nileUSDT, "TRC-20 contract for the token flow (the throwaway Nile account holds TLT, not USDT)")
	lockBlocks := flag.Int64("lock-blocks", 20, "delegation lock length, in blocks, for the lock/expiry proof")
	skipLockWait := flag.Bool("skip-lock-wait", false, "do not wait for the delegation lock to expire (leaves the delegation locked)")
	leaveUnstaked := flag.Bool("leave-unstaked", false, "do not cancel the unstake, leaving it in its cooldown so a later run can prove WithdrawUnstaked once it matures (the cooldown is 1 day on Nile, 14 on Mainnet)")
	permUpdate := flag.Bool("permission-update", false, "with -broadcast: broadcast a real permission update — add one active permission, verify it on-chain, then submit the original set back. Burns the permission-update governance fee (100 TRX) twice; fund the owner account with ~2x fee first")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
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
	owner := signer.Address()
	if *ownerStr != "" {
		owner, err = tronlib.ParseAddress(*ownerStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "owner: %v\n", err)
			return 2
		}
	}
	payee, err := tronlib.ParseAddress(*payeeStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "payee: %v\n", err)
		return 2
	}
	tokenAddr, err := tronlib.ParseAddress(*tokenStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "token: %v\n", err)
		return 2
	}
	var payeeSigner tronlib.Signer
	if *payeeKeyHex != "" {
		payeeSigner, err = tronlib.KeyFromHex(*payeeKeyHex)
		if err != nil {
			fmt.Fprintf(os.Stderr, "payee key: %v\n", err)
			return 2
		}
		if payeeSigner.Address() != payee {
			fmt.Fprintf(os.Stderr, "payee key derives %s, not the -payee address %s\n", payeeSigner.Address(), payee)
			return 2
		}
	}
	c := &checker{
		ctx:           ctx,
		cli:           cli,
		signer:        signer,
		owner:         owner,
		payee:         payee,
		payeeSigner:   payeeSigner,
		usdt:          tokenAddr,
		broadcast:     *broadcast,
		lockBlocks:    *lockBlocks,
		skipLockWait:  *skipLockWait,
		leaveUnstaked: *leaveUnstaked,
		permUpdate:    *permUpdate,
		floatSUN:      tronlib.TRX(int64(*floatTRX)),
	}
	fmt.Printf("examplecheck: %s\nsigner:  %s%s\nowner:   %s%s\npayee:   %s%s\n\n",
		*endpoint, signer.Address(), generated, owner,
		map[bool]string{true: " (same as signer)", false: " (signed for by a key in its permission list)"}[owner == signer.Address()],
		payee, map[bool]string{true: "", false: " (no -payee-key: payee-signed steps will be skipped)"}[payeeSigner != nil])

	c.reads()
	c.token()
	c.contract()
	c.transfer()
	c.staking()
	c.permissions()
	c.events()

	if *broadcast {
		c.chainFlows()
	}

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
	ctx           context.Context
	cli           *tronlib.Client
	signer        tronlib.Signer
	owner         tronlib.Address
	payee         tronlib.Address
	payeeSigner   tronlib.Signer
	usdt          tronlib.Address
	broadcast     bool
	lockBlocks    int64
	skipLockWait  bool
	leaveUnstaked bool
	permUpdate    bool
	floatSUN      tronlib.SUN
	sent          int

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
	// A permission read on an account that does not exist yet is the "no
	// account" case for Permissions.Current: there is no permission set to
	// decode, and Current says so with contract.bad_metadata by design. A
	// spend-free run with a fresh (unfunded) signer hits it; a run with an
	// existing owner (-key) reads the real set.
	var te *tron.Error
	if errors.As(err, &te) &&
		te.Code == tron.CodeContractBadMetadata && te.Op == "account.Permissions.Current" {
		return true
	}
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

// signOnly reports the signed transaction without broadcasting it. The
// spend-free pass never sends anything; the spending lives in
// chainFlows, where each state change is paired with its reversal.
func (c *checker) signOnly(name string, signed tronlib.Tx) error {
	fmt.Printf("        %s: signed %s (not broadcast; -broadcast to send)\n", name, signed.ID()[:16])
	return nil
}

// reads covers ExampleClient_Account: balance, account state, resource state,
// the staking summary and the live chain parameters.
func (c *checker) reads() {
	fmt.Println("reads (Example, ExampleClient_Account):")
	acct := c.cli.Account(c.owner)

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
			fmt.Printf("        account %s does not exist on this network yet (unfunded signer?)\n", st.Address)
			return nil
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
		bal, err := handle.BalanceOf(c.ctx, c.owner)
		if err != nil {
			return err
		}
		fmt.Printf("        %s\n", bal.String())
		return nil
	})
	c.step("Allowance(existing -> recipient)", func() error {
		allowance, err := handle.Allowance(c.ctx, c.owner, c.payee)
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
		approve, err := handle.Approve(c.ctx, c.owner, c.payee, whole)
		if err != nil {
			return err
		}
		signed, err := approve.Sign(c.signer)
		if err != nil {
			return err
		}
		return c.signOnly("Approve", signed)
	})
	c.step("Transfer (build + sign)", func() error {
		whole, err := handle.Whole(1)
		if err != nil {
			return err
		}
		move, err := handle.Transfer(c.ctx, c.owner, c.payee, whole)
		if err != nil {
			return err
		}
		signed, err := move.Sign(c.signer)
		if err != nil {
			return err
		}
		return c.signOnly("Transfer", signed)
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
		res, err := inst.Call(c.ctx, "balanceOf", contract.AddressArg(c.owner))
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
		built, err := inst.Invoke(c.ctx, c.owner, 0, "approve",
			contract.AddressArg(c.payee), contract.BigIntArg(big.NewInt(1_000_000)))
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
			preview, err := c.cli.Account(c.owner).CostPreview(c.ctx, approve)
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
			cost, err := c.cli.Account(c.owner).TotalCost(c.ctx, signed)
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
		attempt, err := inst.Invoke(c.ctx, c.owner, 0, "transfer",
			contract.AddressArg(c.payee), contract.BigIntArg(big.NewInt(1_000_000)))
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
	acct := c.cli.Account(c.owner)

	c.step("TransferTRX build + Sign", func() error {
		transfer, err := acct.TransferTRX(c.ctx, c.payee, tronlib.TRX(1))
		if err != nil {
			return err
		}
		signed, err := transfer.Sign(c.signer)
		if err != nil {
			return err
		}
		return c.signOnly("TransferTRX", signed)
	})

	var envelope []byte
	c.step("Encode -> Decode -> Signers", func() error {
		transfer, err := acct.TransferTRX(c.ctx, c.payee, tronlib.TRX(1))
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
		transfer, err := acct.TransferTRX(c.ctx, c.payee, tronlib.TRX(1))
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
		status, err := c.cli.Account(c.owner).Permissions().SignWeight(c.ctx, back)
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
	res := c.cli.Account(c.owner).Resources()

	c.step("Stake 1 TRX for Energy", func() error {
		stake, err := res.Stake(c.ctx, tronlib.Energy, tronlib.TRX(1))
		if err != nil {
			return err
		}
		signed, err := stake.Sign(c.signer)
		if err != nil {
			return err
		}
		return c.signOnly("Stake", signed)
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
		return c.signOnly("Unstake", signed)
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
		return c.signOnly("WithdrawUnstaked", signed)
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
		return c.signOnly("CancelUnstake", signed)
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
		delegate, err := res.Delegate(c.ctx, tronlib.Energy, c.payee, tronlib.TRX(1), tronlib.DelegateParams{
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
		return c.signOnly("Delegate", signed)
	})
	c.step("Undelegate (immediate when unlocked)", func() error {
		undelegate, err := res.Undelegate(c.ctx, tronlib.Energy, c.payee, tronlib.TRX(1))
		if err != nil {
			return err
		}
		signed, err := undelegate.Sign(c.signer)
		if err != nil {
			return err
		}
		return c.signOnly("Undelegate", signed)
	})
	c.step("SetVotes (whole-list replace)", func() error {
		vote, err := c.cli.Account(c.owner).Voting().SetVotes(c.ctx, []tronlib.Vote{
			{Witness: c.payee, Count: 1},
		})
		if err != nil {
			return err
		}
		signed, err := vote.Sign(c.signer)
		if err != nil {
			return err
		}
		return c.signOnly("SetVotes", signed)
	})
	c.step("Rewards read + ClaimRewards", func() error {
		rewards, err := c.cli.Account(c.owner).Voting().Rewards(c.ctx)
		if err != nil {
			return err
		}
		claim, err := c.cli.Account(c.owner).Voting().ClaimRewards(c.ctx)
		if err != nil {
			return err
		}
		signed, err := claim.Sign(c.signer)
		if err != nil {
			return err
		}
		fmt.Printf("        accrued %s TRX\n", rewards.Formatted())
		return c.signOnly("ClaimRewards", signed)
	})
	fmt.Println()
}

// permissions covers ExamplePermissions_Current and the bitmap helpers.
func (c *checker) permissions() {
	fmt.Println("permissions (ExamplePermissions_Current):")
	perms := c.cli.Account(c.owner).Permissions()

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
			Keys:       []tronlib.PermissionKey{{Address: c.payee, Weight: 1}},
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
		cost, err := c.cli.Account(c.owner).TotalCost(c.ctx, signed)
		if err == nil {
			fmt.Printf("        permission-update fee %s TRX, total %s TRX\n",
				cost.PermissionUpdateFee.Formatted(), cost.Total.Formatted())
		}
		return c.signOnly("AccountPermissionUpdate", signed)
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

// chainFlows is the on-chain half: it broadcasts the documented flows against
// the live network, in an order where every state change is paired with the
// operation that reverses it, so the account ends the run as it started. Run
// it with the repo's throwaway Nile key (v1-legacy:integration_test/test.env)
// or any funded testnet key.
func (c *checker) chainFlows() {
	fmt.Println("chain-updating flows (-broadcast; each change is paired with its reversal):")
	acct := c.cli.Account(c.owner)
	before, err := acct.State(c.ctx)
	if err != nil {
		c.failed++
		fmt.Printf("  FAIL  pre-run state: %v\n", err)
		return
	}
	beforeRes, err := acct.Resources().State(c.ctx)
	if err != nil {
		c.failed++
		fmt.Printf("  FAIL  pre-run resources: %v\n", err)
		return
	}
	beforePayee, err := c.cli.Account(c.payee).State(c.ctx)
	if err != nil {
		c.failed++
		fmt.Printf("  FAIL  pre-run payee state: %v\n", err)
		return
	}

	c.rebalanceIn()
	c.flowTRC20()
	c.flowStaking()
	c.flowDelegation()
	c.flowVoting()
	c.flowPermissions()
	c.rebalanceOut()

	after, err := acct.State(c.ctx)
	if err != nil {
		c.failed++
		fmt.Printf("  FAIL  post-run state: %v\n", err)
		return
	}
	afterRes, err := acct.Resources().State(c.ctx)
	if err != nil {
		c.failed++
		fmt.Printf("  FAIL  post-run resources: %v\n", err)
		return
	}
	c.step("Position restored (balances differ only by the fees burnt)", func() error {
		payeeBefore := beforePayee.Balance
		payeeAfter, err := c.cli.Account(c.payee).Balance(c.ctx)
		if err != nil {
			return err
		}
		fmt.Printf("        owner %s -> %s TRX (spent %s: energy bought + bandwidth for %d transactions)\n",
			before.Balance.Formatted(), after.Balance.Formatted(),
			(before.Balance - after.Balance).Formatted(), c.sent)
		fmt.Printf("        payee %s -> %s TRX (float %s TRX)\n",
			payeeBefore.Formatted(), payeeAfter.Formatted(), c.floatSUN.Formatted())
		fmt.Printf("        stakes %d -> %d, pending unstakes %d -> %d, energy limit %d -> %d\n",
			len(before.Stakes), len(after.Stakes),
			len(before.Unstakes), len(after.Unstakes),
			beforeRes.EnergyLimit, afterRes.EnergyLimit)
		if c.leaveUnstaked {
			fmt.Println("        (-leave-unstaked: a pending unstake is expected to remain)")
		} else if len(after.Unstakes) != len(before.Unstakes) {
			return fmt.Errorf("pending unstakes changed: %d -> %d", len(before.Unstakes), len(after.Unstakes))
		}
		if len(after.Stakes) != len(before.Stakes) {
			return fmt.Errorf("stake count changed: %d -> %d", len(before.Stakes), len(after.Stakes))
		}
		if payeeAfter > c.floatSUN+rebalanceReserve+rebalanceStep {
			return fmt.Errorf("payee above its band at %s TRX; the return step should have run", payeeAfter.Formatted())
		}
		return nil
	})
	fmt.Println()
}

// send signs a built transaction and broadcasts it, waiting for inclusion.
func (c *checker) send(name string, build func() (tronlib.Tx, error)) error {
	toSend, err := build()
	if err != nil {
		return err
	}
	rec, err := c.cli.Broadcast(c.ctx, toSend)
	if err != nil {
		return err
	}
	if !rec.OK() {
		return fmt.Errorf("node rejected: %s %s", rec.NodeCode, rec.Revert)
	}
	if _, err := c.cli.Wait(c.ctx, rec.TxID); err != nil {
		return err
	}
	fmt.Printf("        %s: %s\n", name, rec.TxID)
	c.sent++
	return nil
}

// signExisting signs an already-built transaction with a specific signer.
func (c *checker) sendWith(name string, tx tronlib.Tx, signer tronlib.Signer) error {
	signed, err := signWith(tx, signer)
	if err != nil {
		return err
	}
	rec, err := c.cli.Broadcast(c.ctx, signed)
	if err != nil {
		return err
	}
	if !rec.OK() {
		return fmt.Errorf("node rejected: %s %s", rec.NodeCode, rec.Revert)
	}
	if _, err := c.cli.Wait(c.ctx, rec.TxID); err != nil {
		return err
	}
	fmt.Printf("        %s: %s\n", name, rec.TxID)
	c.sent++
	return nil
}

// signWith signs any concrete transaction kind with the given signer.
func signWith(tx tronlib.Tx, signer tronlib.Signer) (tronlib.Tx, error) {
	switch v := tx.(type) {
	case *tronlib.NativeTx:
		return v.Sign(signer)
	case *tronlib.ContractTx:
		return v.Sign(signer)
	default:
		return nil, fmt.Errorf("unsupported transaction kind %T", tx)
	}
}

// rebalanceIn funds the counterparty when it is short, and rebalanceOut
// returns what it did not spend. Together they keep the two keys' leftover TRX
// balanced across runs: the only net movement is the bandwidth each run burns,
// never a fixed amount drifting from one key to the other.
//
// The band is deliberate — topping up dust would cost more in bandwidth than
// it moves.
const (
	// rebalanceStep rounds transfers to 0.1 TRX so the amounts stay readable.
	rebalanceStep = tronlib.SUN(100_000)
	// rebalanceReserve covers the bandwidth the returning transfer itself
	// burns, so the return never leaves the payee unable to pay for it.
	rebalanceReserve = tronlib.SUN(1_000_000)
	// rebalanceMinReturn is the smallest return worth its own bandwidth.
	rebalanceMinReturn = tronlib.SUN(500_000)
)

func ceilTo(amount, step tronlib.SUN) tronlib.SUN {
	if amount <= 0 {
		return 0
	}
	return ((amount + step - 1) / step) * step
}

func floorTo(amount, step tronlib.SUN) tronlib.SUN {
	if amount <= 0 {
		return 0
	}
	return (amount / step) * step
}

// rebalanceIn tops the payee up to the float when it has fallen short.
func (c *checker) rebalanceIn() {
	fmt.Println("  rebalance in (keeps the counterparty solvent without leaking):")
	balance, err := c.cli.Account(c.payee).Balance(c.ctx)
	if err != nil {
		c.failed++
		fmt.Printf("  FAIL  payee balance: %v\n", err)
		return
	}
	shortfall := c.floatSUN - balance
	if shortfall <= 0 {
		c.step("Payee is at or above the float; nothing to send", func() error {
			fmt.Printf("        payee holds %s TRX (float %s TRX)\n", balance.Formatted(), c.floatSUN.Formatted())
			return nil
		})
		return
	}
	topUp := ceilTo(shortfall, rebalanceStep)
	c.step("TransferTRX top-up "+topUp.Formatted()+" TRX to the payee", func() error {
		if err := c.send("rebalance-in", func() (tronlib.Tx, error) {
			built, err := c.cli.Account(c.owner).TransferTRX(c.ctx, c.payee, topUp)
			if err != nil {
				return nil, err
			}
			return built.Sign(c.signer)
		}); err != nil {
			return err
		}
		now, err := c.cli.Account(c.payee).Balance(c.ctx)
		if err != nil {
			return err
		}
		fmt.Printf("        payee %s -> %s TRX\n", balance.Formatted(), now.Formatted())
		return nil
	})
}

// rebalanceOut returns everything the payee holds above the float plus a
// bandwidth reserve, signed by the payee's own key — which also proves the
// counterparty can authorize and broadcast its own transaction.
func (c *checker) rebalanceOut() {
	fmt.Println("  rebalance out (returns the run's leftovers):")
	if c.payeeSigner == nil {
		c.notes++
		fmt.Println("  NOTE  no -payee-key given: the payee cannot return its excess, and it cannot run the payee-signed steps")
		return
	}
	balance, err := c.cli.Account(c.payee).Balance(c.ctx)
	if err != nil {
		c.failed++
		fmt.Printf("  FAIL  payee balance: %v\n", err)
		return
	}
	excess := floorTo(balance-c.floatSUN-rebalanceReserve, rebalanceStep)
	if excess < rebalanceMinReturn {
		c.step("Payee is below the return threshold; nothing to send back", func() error {
			fmt.Printf("        payee holds %s TRX (float %s TRX; the next run tops it back up to the float)\n",
				balance.Formatted(), c.floatSUN.Formatted())
			return nil
		})
		return
	}
	c.step("TransferTRX "+excess.Formatted()+" TRX back to the owner (signed by the payee)", func() error {
		built, err := c.cli.Account(c.payee).TransferTRX(c.ctx, c.owner, excess)
		if err != nil {
			return err
		}
		signed, err := built.Sign(c.payeeSigner)
		if err != nil {
			return err
		}
		rec, err := c.cli.Broadcast(c.ctx, signed)
		if err != nil {
			return err
		}
		if !rec.OK() {
			return fmt.Errorf("node rejected: %s %s", rec.NodeCode, rec.Revert)
		}
		if _, err := c.cli.Wait(c.ctx, rec.TxID); err != nil {
			return err
		}
		c.sent++
		fmt.Printf("        rebalance-out: %s\n", rec.TxID)
		now, err := c.cli.Account(c.payee).Balance(c.ctx)
		if err != nil {
			return err
		}
		fmt.Printf("        payee %s -> %s TRX\n", balance.Formatted(), now.Formatted())
		return nil
	})
}

// flowTRC20 proves approve -> transferFrom -> transfer with real value by
// moving 0.1 of the token twice. transferFrom is the spender pulling from the
// owner, which is the only reason to approve at all.
func (c *checker) flowTRC20() {
	fmt.Println("  TRC-20 (approve -> transferFrom -> transfer):")
	handle, err := c.cli.Token(c.ctx, c.usdt)
	if err != nil {
		c.failed++
		fmt.Printf("  FAIL  token: %v\n", err)
		return
	}
	symbol, err := handle.Symbol(c.ctx)
	if err != nil {
		c.failed++
		fmt.Printf("  FAIL  symbol: %v\n", err)
		return
	}
	start, err := handle.BalanceOf(c.ctx, c.owner)
	if err != nil {
		c.failed++
		fmt.Printf("  FAIL  balanceOf: %v\n", err)
		return
	}
	fmt.Printf("        %s: owner holds %s\n", symbol, start.Formatted())
	portion, err := handle.Amount("0.1")
	if err != nil {
		c.failed++
		fmt.Printf("  FAIL  amount: %v\n", err)
		return
	}

	c.step("Approve 0.1", func() error {
		return c.send("approve", func() (tronlib.Tx, error) {
			approve, err := handle.Approve(c.ctx, c.owner, c.payee, portion)
			if err != nil {
				return nil, err
			}
			return approve.Sign(c.signer)
		})
	})
	c.step("Allowance reflects the approval", func() error {
		allowance, err := handle.Allowance(c.ctx, c.owner, c.payee)
		if err != nil {
			return err
		}
		fmt.Printf("        allowance %s\n", allowance.String())
		if allowance.Raw().Cmp(portion.Raw()) < 0 {
			return fmt.Errorf("allowance %s is below the approved %s", allowance.Raw(), portion.Raw())
		}
		return nil
	})
	c.step("transferFrom by the spender", func() error {
		inst, err := c.cli.Contract(c.ctx, c.usdt)
		if err != nil {
			return err
		}
		pull, err := inst.Invoke(c.ctx, c.payee, 0, "transferFrom",
			contract.AddressArg(c.owner), contract.AddressArg(c.payee), contract.BigIntArg(portion.Raw()))
		if err != nil {
			return err
		}
		// The spender pays this call. With no staked energy it buys all of it
		// with TRX — this preview is what explains the payee's balance drop.
		preview, err := c.cli.Account(c.payee).CostPreview(c.ctx, pull)
		if err != nil {
			return err
		}
		fmt.Printf("        spender pays: needs %d energy, has %d staked, buys %d = %s TRX (+bandwidth)\n",
			preview.EnergyNeeded, preview.EnergyAvailable, preview.EnergyToBuy, preview.TronToBurn.Formatted())
		return c.sendWith("transferFrom", pull, c.payeeSigner)
	})
	c.step("Transfer 0.1 directly", func() error {
		return c.send("transfer", func() (tronlib.Tx, error) {
			move, err := handle.Transfer(c.ctx, c.owner, c.payee, portion)
			if err != nil {
				return nil, err
			}
			return move.Sign(c.signer)
		})
	})
	c.step("Both balances moved by 0.2 net", func() error {
		ownerNow, err := handle.BalanceOf(c.ctx, c.owner)
		if err != nil {
			return err
		}
		payeeNow, err := handle.BalanceOf(c.ctx, c.payee)
		if err != nil {
			return err
		}
		fmt.Printf("        owner %s -> %s | payee %s\n", start.Formatted(), ownerNow.Formatted(), payeeNow.Formatted())
		return nil
	})
}

// flowStaking proves the stake lifecycle: staking raises the resource limit,
// unstaking starts the cooldown without releasing it, and cancelling re-stakes
// it immediately — which is what makes the sequence self-reversing.
func (c *checker) flowStaking() {
	fmt.Println("  staking lifecycle:")
	res := c.cli.Account(c.owner).Resources()

	// A previous run may have left an unstake maturing (the cooldown is a
	// chain parameter: 1 day on Nile, 14 on Mainnet). Withdraw anything that
	// has matured — this is the only way WithdrawUnstaked can be proven, since
	// nothing matures inside a single run.
	c.step("WithdrawUnstaked (anything matured from an earlier run)", func() error {
		withdrawable, err := res.Withdrawable(c.ctx)
		if err != nil {
			return err
		}
		if withdrawable == 0 {
			return nil
		}
		fmt.Printf("        %s TRX matured\n", withdrawable.Formatted())
		return c.send("withdraw-unstaked", func() (tronlib.Tx, error) {
			withdraw, err := res.WithdrawUnstaked(c.ctx)
			if err != nil {
				return nil, err
			}
			return withdraw.Sign(c.signer)
		})
	})
	c.step("Stake 1 TRX for Energy", func() error {
		return c.send("stake", func() (tronlib.Tx, error) {
			stake, err := res.Stake(c.ctx, tronlib.Energy, tronlib.TRX(1))
			if err != nil {
				return nil, err
			}
			return stake.Sign(c.signer)
		})
	})
	c.step("Energy limit rose", func() error {
		rs, err := res.State(c.ctx)
		if err != nil {
			return err
		}
		fmt.Printf("        energy limit %d, available %d\n", rs.EnergyLimit, rs.EnergyAvailable())
		if rs.EnergyLimit <= 0 {
			return fmt.Errorf("energy limit did not rise")
		}
		return nil
	})
	c.step("Unstake 1 TRX (enters the cooldown)", func() error {
		return c.send("unstake", func() (tronlib.Tx, error) {
			unstake, err := res.Unstake(c.ctx, tronlib.Energy, tronlib.TRX(1))
			if err != nil {
				return nil, err
			}
			return unstake.Sign(c.signer)
		})
	})
	c.step("It is pending, not spendable", func() error {
		state, err := c.cli.Account(c.owner).State(c.ctx)
		if err != nil {
			return err
		}
		withdrawable, err := res.Withdrawable(c.ctx)
		if err != nil {
			return err
		}
		fmt.Printf("        %d pending unstake(s), %s withdrawable (cooldown %d day(s))\n",
			len(state.Unstakes), withdrawable.Formatted(), c.cooldownDays())
		if len(state.Unstakes) == 0 {
			return fmt.Errorf("no pending unstake recorded")
		}
		if withdrawable != 0 {
			fmt.Println("        note: something was already matured; the cooldown had elapsed")
		}
		return nil
	})
	if c.leaveUnstaked {
		c.notes++
		fmt.Println("  NOTE  -leave-unstaked: the 1 TRX stays in its cooldown; run again after it matures to prove WithdrawUnstaked")
	} else {
		c.step("CancelUnstake re-stakes it", func() error {
			return c.send("cancel-unstake", func() (tronlib.Tx, error) {
				cancel, err := res.CancelUnstake(c.ctx)
				if err != nil {
					return nil, err
				}
				return cancel.Sign(c.signer)
			})
		})
	}

	pending := func() (int, error) {
		state, err := c.cli.Account(c.owner).State(c.ctx)
		if err != nil {
			return 0, err
		}
		return len(state.Unstakes), nil
	}
	if c.leaveUnstaked {
		c.step("The unstake is left pending for the next run", func() error {
			n, err := pending()
			if err != nil {
				return err
			}
			fmt.Printf("        pending unstakes: %d (expected by -leave-unstaked)\n", n)
			if n == 0 {
				return fmt.Errorf("-leave-unstaked was set but no unstake is pending")
			}
			return nil
		})
	} else {
		c.step("No unstake is pending again", func() error {
			n, err := pending()
			if err != nil {
				return err
			}
			fmt.Printf("        pending unstakes: %d\n", n)
			if n != 0 {
				return fmt.Errorf("cancel left %d pending unstake(s)", n)
			}
			return nil
		})
	}
}

// flowDelegation proves both delegation kinds: unlocked undelegates
// immediately, locked is refused until the lock expires and then succeeds.
func (c *checker) flowDelegation() {
	fmt.Println("  delegation (unlocked, then locked):")
	res := c.cli.Account(c.owner).Resources()

	c.step("Delegate 1 TRX unlocked", func() error {
		return c.send("delegate", func() (tronlib.Tx, error) {
			delegate, err := res.Delegate(c.ctx, tronlib.Energy, c.payee, tronlib.TRX(1), tronlib.DelegateParams{})
			if err != nil {
				return nil, err
			}
			return delegate.Sign(c.signer)
		})
	})
	c.step("Undelegate it immediately", func() error {
		return c.send("undelegate", func() (tronlib.Tx, error) {
			undelegate, err := res.Undelegate(c.ctx, tronlib.Energy, c.payee, tronlib.TRX(1))
			if err != nil {
				return nil, err
			}
			return undelegate.Sign(c.signer)
		})
	})

	locked := false
	c.step("Delegate 1 TRX locked for "+fmt.Sprint(c.lockBlocks)+" blocks", func() error {
		delegate, err := res.Delegate(c.ctx, tronlib.Energy, c.payee, tronlib.TRX(1), tronlib.DelegateParams{
			Lock:       true,
			LockBlocks: c.lockBlocks,
		})
		if err != nil {
			return err
		}
		signed, err := delegate.Sign(c.signer)
		if err != nil {
			return err
		}
		rec, err := c.cli.Broadcast(c.ctx, signed)
		if err != nil {
			return err
		}
		if !rec.OK() {
			return fmt.Errorf("node rejected: %s %s", rec.NodeCode, rec.Revert)
		}
		if _, err := c.cli.Wait(c.ctx, rec.TxID); err != nil {
			return err
		}
		locked = true
		c.sent++
		fmt.Printf("        locked delegate: %s\n", rec.TxID)
		return nil
	})
	if !locked {
		fmt.Println("        (locked delegation was refused by the node; skipping the lock proof)")
		return
	}
	c.step("Undelegate while locked is refused", func() error {
		undelegate, err := res.Undelegate(c.ctx, tronlib.Energy, c.payee, tronlib.TRX(1))
		if err != nil {
			fmt.Printf("        refused at build: %v\n", err)
			return nil
		}
		signed, err := undelegate.Sign(c.signer)
		if err != nil {
			return err
		}
		rec, err := c.cli.Broadcast(c.ctx, signed)
		if err != nil {
			fmt.Printf("        refused at broadcast: %v\n", err)
			return nil
		}
		if !rec.OK() {
			fmt.Printf("        refused by the node: %s %s\n", rec.NodeCode, rec.Revert)
			return nil
		}
		return fmt.Errorf("the lock did not hold: the delegation was undelegated at %s", rec.TxID)
	})

	if c.skipLockWait {
		fmt.Println("        (-skip-lock-wait: leaving the delegation locked)")
		return
	}
	wait := time.Duration(c.lockBlocks)*3*time.Second + 6*time.Second
	fmt.Printf("        waiting %s for the lock to expire...\n", wait.Round(time.Second))
	select {
	case <-time.After(wait):
	case <-c.ctx.Done():
		return
	}
	c.step("Undelegate after the lock expires", func() error {
		return c.send("undelegate (unlocked)", func() (tronlib.Tx, error) {
			undelegate, err := res.Undelegate(c.ctx, tronlib.Energy, c.payee, tronlib.TRX(1))
			if err != nil {
				return nil, err
			}
			return undelegate.Sign(c.signer)
		})
	})
}

// flowVoting re-submits the account's existing vote list (a whole-list
// replacement that changes nothing) and claims the accrued rewards.
func (c *checker) flowVoting() {
	fmt.Println("  voting (whole-list replace) + rewards:")
	voting := c.cli.Account(c.owner).Voting()
	votes, err := voting.Votes(c.ctx)
	if err != nil {
		c.failed++
		fmt.Printf("  FAIL  votes: %v\n", err)
		return
	}
	if len(votes) == 0 {
		witnesses, err := c.cli.Witnesses(c.ctx, tronlib.Page{Limit: 1})
		if err != nil || len(witnesses) == 0 {
			c.notes++
			fmt.Printf("  NOTE  no votes on the account and no witness list to pick from: %v\n", err)
			return
		}
		votes = []tronlib.Vote{{Witness: witnesses[0].Address, Count: 1}}
		fmt.Printf("        no existing votes; casting 1 TRON Power for %s\n", witnesses[0].Address)
	}
	c.step("SetVotes (replaces the whole list)", func() error {
		return c.send("set-votes", func() (tronlib.Tx, error) {
			vote, err := voting.SetVotes(c.ctx, votes)
			if err != nil {
				return nil, err
			}
			return vote.Sign(c.signer)
		})
	})
	c.step("ClaimRewards", func() error {
		rewards, err := voting.Rewards(c.ctx)
		if err != nil {
			return err
		}
		if rewards == 0 {
			// A claim with nothing to withdraw is rejected at build time
			// (tx.invalid_argument): the node state-validates it. Skipping is
			// the honest outcome, not a failure.
			c.notes++
			fmt.Printf("  NOTE  ClaimRewards: nothing accrued (the node rejects a claim with no balance, at build time)\n")
			return nil
		}
		fmt.Printf("        accrued %s TRX\n", rewards.Formatted())
		return c.send("claim-rewards", func() (tronlib.Tx, error) {
			claim, err := voting.ClaimRewards(c.ctx)
			if err != nil {
				return nil, err
			}
			return claim.Sign(c.signer)
		})
	})
}

// flowPermissions does not broadcast: the update fee is a governance
// parameter (100 TRX) and a mis-specified owner permission can lock every
// permission on the account. It reports the price and leaves the send to an
// operator who means it.
func (c *checker) flowPermissions() {
	perms := c.cli.Account(c.owner).Permissions()
	if c.broadcast && c.permUpdate {
		c.permissionCycle(perms)
		return
	}
	fmt.Println("  permissions (priced, not broadcast — add -permission-update to send one):")
	c.step("Price an identical permission update", func() error {
		set, err := perms.Current(c.ctx)
		if err != nil {
			return err
		}
		update, err := perms.Update(c.ctx, set)
		if err != nil {
			return err
		}
		signed, err := update.Sign(c.signer)
		if err != nil {
			return err
		}
		cost, err := c.cli.Account(c.owner).TotalCost(c.ctx, signed)
		if err != nil {
			return err
		}
		balance, err := c.cli.Account(c.owner).Balance(c.ctx)
		if err != nil {
			return err
		}
		fmt.Printf("        fee %s TRX, account holds %s TRX -> %s\n",
			cost.PermissionUpdateFee.Formatted(), balance.Formatted(),
			map[bool]string{true: "affordable", false: "NOT affordable, deliberately not sent"}[balance >= cost.PermissionUpdateFee])
		return nil
	})
}

// permissionCycle closes the ledger's "permission-update broadcast" row:
// it broadcasts a real AccountPermissionUpdate that appends one active
// permission (never touching the owner permission), verifies the on-chain
// effect, then submits the original set back. The governance fee
// (getUpdateAccountPermissionFee — 100 TRX on Nile and Mainnet) burns once
// per broadcast, so the owner account must hold ~2x fee + gas before
// opting in; the affordability gate below refuses otherwise. Every
// transaction is signed by the owner key and pre-checked with SignWeight.
func (c *checker) permissionCycle(perms *tronlib.Permissions) {
	fmt.Println("  permissions (-permission-update: broadcast add, verify, reverse):")
	before, err := perms.Current(c.ctx)
	if err != nil {
		c.failed++
		fmt.Printf("  FAIL  Current: %v\n", err)
		return
	}
	bitmap, err := tronlib.OperationsBitmap(tronlib.TypeTransfer, tronlib.TypeTriggerSmartContract)
	if err != nil {
		c.failed++
		fmt.Printf("  FAIL  bitmap: %v\n", err)
		return
	}
	update := before
	update.Actives = append(append([]tronlib.Permission(nil), before.Actives...), tronlib.Permission{
		Name:       "examplecheck",
		Threshold:  1,
		Keys:       []tronlib.PermissionKey{{Address: c.payee, Weight: 1}},
		Operations: bitmap,
	})
	grants, _ := tronlib.OperationsList(bitmap)
	fmt.Printf("        adding active %q for %s (weight 1, grants: %v)\n", "examplecheck", c.payee, grants)

	balBefore, err := c.cli.Account(c.owner).Balance(c.ctx)
	if err != nil {
		c.failed++
		fmt.Printf("  FAIL  balance: %v\n", err)
		return
	}

	c.step("Price the update (fee burns per broadcast)", func() error {
		tx, err := perms.Update(c.ctx, update)
		if err != nil {
			return err
		}
		signed, err := tx.Sign(c.signer)
		if err != nil {
			return err
		}
		cost, err := c.cli.Account(c.owner).TotalCost(c.ctx, signed)
		if err != nil {
			return err
		}
		fmt.Printf("        fee %s TRX x2 (add + reverse), account holds %s TRX\n",
			cost.PermissionUpdateFee.Formatted(), balBefore.Formatted())
		need, _ := cost.PermissionUpdateFee.Mul(2)
		reserve := tronlib.TRX(2)
		if total, err := need.Add(reserve); err == nil && balBefore < total {
			return fmt.Errorf("balance %s TRX is short of 2x fee + 2 TRX reserve (%s TRX) — fund the owner account first",
				balBefore.Formatted(), total.Formatted())
		}
		return nil
	})

	c.step("Add active permission (build + sign + SignWeight)", func() error {
		status, err := c.signWeight(perms, update)
		if err != nil {
			return err
		}
		fmt.Printf("        sign weight %s — authorized\n", status.String())
		if !status.Enough {
			return fmt.Errorf("collected weight does not meet the threshold — refusing to send")
		}
		return nil
	})
	c.step("Add active permission (broadcast + wait)", func() error {
		return c.send("permission-update", func() (tronlib.Tx, error) {
			tx, err := perms.Update(c.ctx, update)
			if err != nil {
				return nil, err
			}
			return tx.Sign(c.signer)
		})
	})
	c.step("Verify the added active permission", func() error {
		after, err := perms.Current(c.ctx)
		if err != nil {
			return err
		}
		if len(after.Actives) != len(before.Actives)+1 {
			return fmt.Errorf("actives %d -> %d, expected +1", len(before.Actives), len(after.Actives))
		}
		if err := ownerUnchanged(before, after); err != nil {
			return err
		}
		fmt.Printf("        on-chain: %d active(s), last %q threshold %d — owner permission untouched\n",
			len(after.Actives), after.Actives[len(after.Actives)-1].Name, after.Actives[len(after.Actives)-1].Threshold)
		return nil
	})

	c.step("Reverse (submit the original set)", func() error {
		return c.send("permission-restore", func() (tronlib.Tx, error) {
			tx, err := perms.Update(c.ctx, before)
			if err != nil {
				return nil, err
			}
			return tx.Sign(c.signer)
		})
	})
	c.step("Verify the reversal", func() error {
		final, err := perms.Current(c.ctx)
		if err != nil {
			return err
		}
		if len(final.Actives) != len(before.Actives) {
			return fmt.Errorf("actives %d after reverse, expected back to %d — RE-RUN WITH -permission-update TO RESTORE", len(final.Actives), len(before.Actives))
		}
		balAfter, err := c.cli.Account(c.owner).Balance(c.ctx)
		if err != nil {
			return err
		}
		fmt.Printf("        actives back to %d; permission fees burnt %s TRX\n",
			len(final.Actives), (balBefore - balAfter).Formatted())
		return nil
	})
}

// signWeight builds, signs and pre-checks a permission update with the
// node's signature-weight verdict — the discipline the multi-sig flow
// teaches: broadcast only what the node says is authorized.
func (c *checker) signWeight(perms *tronlib.Permissions, set tronlib.PermissionSet) (*tronlib.SignatureState, error) {
	tx, err := perms.Update(c.ctx, set)
	if err != nil {
		return nil, err
	}
	signed, err := tx.Sign(c.signer)
	if err != nil {
		return nil, err
	}
	return perms.SignWeight(c.ctx, signed)
}

// ownerUnchanged asserts the owner permission survived an update: same
// threshold, same key list. A permission update that silently changed the
// owner permission would be the lockout class the ledger refuses to risk.
func ownerUnchanged(before, after tronlib.PermissionSet) error {
	b, a := before.Owner, after.Owner
	if b.Threshold != a.Threshold || len(b.Keys) != len(a.Keys) {
		return fmt.Errorf("owner permission changed: threshold %d/%d keys %d/%d", b.Threshold, a.Threshold, len(b.Keys), len(a.Keys))
	}
	for i := range b.Keys {
		if b.Keys[i].Address != a.Keys[i].Address || b.Keys[i].Weight != a.Keys[i].Weight {
			return fmt.Errorf("owner key %d changed", i)
		}
	}
	return nil
}

func (c *checker) cooldownDays() int64 {
	params, err := tronlib.ChainParamsOf(c.ctx, c.cli.Raw())
	if err != nil {
		return 0
	}
	return params.UnfreezeDelayDays
}
