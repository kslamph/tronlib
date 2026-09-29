package tronlib_test

// Compile-only examples for the root facade: common tasks end to end.
//
// These examples deliberately carry NO // Output: comment. go test compiles
// them but never executes them, so each one builds against the real surface —
// including Dial, Sign and Broadcast — without contacting a node or needing a
// funded key (CODING_STANDARDS.md §6.4). cmd/docgen extracts them into
// docs/examples.md through the go:example markers, and CI fails if the two
// diverge, so an edit here must be followed by:
//
//	go run ./cmd/docgen sync-docs -pkg ./tron -example-pkg . -docs ./docs/errors.md -docs ./docs/examples.md
//
// The set is intentionally small: each example covers a whole task and
// combines the capabilities that task needs (a contract call is simulated,
// costed and its logs decoded in one place) instead of one example per method.
// Private keys appear in both forms the SDK accepts — hex and BIP-39 mnemonic
// — and both single-signature and multi-signature flows are shown.

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"time"

	tronlib "github.com/kslamph/tronlib/v2"
	"github.com/kslamph/tronlib/v2/contract"
)

// Example is the quickstart: dial, read the balance, transfer TRX, price the
// transaction before signing it, broadcast, and wait for solidification.
//
// The stages are explicit on purpose — build, sign, broadcast are separate
// calls, so a mistake (wrong recipient, unexpected cost) is caught before
// anything is irreversibly sent. `Sign` returns a copy: the unsigned
// transaction stays valid for a second attempt.
func Example() {
	// One deadline for the whole send: an unbounded broadcast call is how a
	// program hangs on a node that stopped answering.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cli, err := tronlib.Dial(ctx, "grpc://grpc.nile.trongrid.io:50051")
	if err != nil {
		fmt.Println("dial:", err)
		return
	}
	defer cli.Close()

	// A hex private key: 64 hex characters, with or without 0x.
	signer, err := tronlib.KeyFromHex(os.Getenv("TRON_PRIVATE_KEY"))
	if err != nil {
		fmt.Println("key:", err)
		return
	}
	to, err := tronlib.ParseAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
	if err != nil {
		fmt.Println("address:", err)
		return
	}

	acct := cli.Account(signer.Address())
	bal, err := acct.Balance(ctx)
	if err != nil {
		fmt.Println("balance:", err)
		return
	}
	fmt.Println("balance:", bal.Formatted(), "TRX")

	transfer, err := acct.TransferTRX(ctx, to, tronlib.TRX(1))
	if err != nil {
		fmt.Println("build:", err)
		return
	}
	signed, err := transfer.Sign(signer)
	if err != nil {
		fmt.Println("sign:", err)
		return
	}

	// Price the exact bytes that will be broadcast: energy, bandwidth,
	// account-creation and any governance fee. Call it after Sign — the
	// bandwidth half is measured on the signed transaction.
	cost, err := acct.TotalCost(ctx, signed)
	if err != nil {
		fmt.Println("cost:", err)
		return
	}
	fmt.Println(cost.String())

	rec, err := cli.Broadcast(ctx, signed)
	if err != nil {
		fmt.Println("broadcast:", err)
		return
	}
	if !rec.OK() {
		fmt.Println("node rejected:", rec.NodeCode)
		return
	}
	// Inclusion is not finality; custody and deposit-crediting wait for
	// solidification (confirmed by the super representatives).
	if _, err := cli.WaitForSolid(ctx, rec.TxID); err != nil {
		fmt.Println("wait:", err)
	}
}

// ExampleClient_Account reads everything a decision needs before spending:
// the account state, its staked resources (Energy and Bandwidth), the
// stake/unstake/delegation summary, and the all-in cost of a transaction.
// Nothing is broadcast here — this is the "should I send this?" pass.
//
// Resource amounts are always TRX in SUN; how much Energy a stake actually
// buys depends on the network-wide staked total, so it is read, never
// calculated from a fixed rate.
func ExampleClient_Account() {
	ctx := context.Background()

	cli, err := tronlib.Dial(ctx, "grpc://grpc.nile.trongrid.io:50051")
	if err != nil {
		fmt.Println("dial:", err)
		return
	}
	defer cli.Close()

	signer, err := tronlib.KeyFromHex(os.Getenv("TRON_PRIVATE_KEY"))
	if err != nil {
		fmt.Println("key:", err)
		return
	}
	acct := cli.Account(signer.Address())

	// Balance, stake positions still in their cooldown, current votes and the
	// TRX lent to / borrowed from other accounts.
	state, err := acct.State(ctx)
	if err != nil {
		fmt.Println("state:", err)
		return
	}
	fmt.Println(state.Balance.Formatted(), "TRX,", len(state.Stakes), "stake(s),",
		len(state.Unstakes), "unstake(s),", len(state.Votes), "vote(s)")

	// Energy and Bandwidth limits, current usage, and the TRON Power that
	// voting consumes.
	resources, err := acct.Resources().State(ctx)
	if err != nil {
		fmt.Println("resources:", err)
		return
	}
	fmt.Println("energy", resources.EnergyAvailable(), "of", resources.EnergyLimit,
		"| bandwidth", resources.BandwidthAvailable(), "of",
		resources.BandwidthLimit+resources.FreeBandwidthLimit,
		"| tron power", resources.TronPowerAvailable())

	// One call that folds the account read and the resource read into the
	// numbers a staking UI shows.
	summary, err := acct.Resources().Summary(ctx)
	if err != nil {
		fmt.Println("summary:", err)
		return
	}
	fmt.Println("staked", summary.StakedByResource[tronlib.Energy].Formatted(), "TRX for energy,",
		summary.UnstakeWithdrawable.Formatted(), "TRX withdrawable,",
		summary.UnstakeSlots, "unstake slots left")

	to, err := tronlib.ParseAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
	if err != nil {
		fmt.Println("address:", err)
		return
	}
	transfer, err := acct.TransferTRX(ctx, to, tronlib.TRX(1))
	if err != nil {
		fmt.Println("build:", err)
		return
	}
	signed, err := transfer.Sign(signer)
	if err != nil {
		fmt.Println("sign:", err)
		return
	}
	cost, err := acct.TotalCost(ctx, signed)
	if err != nil {
		fmt.Println("cost:", err)
		return
	}
	fmt.Println("this transfer costs:", cost.Total.Formatted(), "TRX —", cost.String())
}

// ExampleClient_Contract is one contract interaction from end to end: read a
// view function, build a state-changing call, dry-run it, price it, broadcast
// it, and decode the events it emitted. A wrong ABI argument or an
// under-priced call is caught by the simulation, before signing.
func ExampleClient_Contract() {
	ctx := context.Background()

	cli, err := tronlib.Dial(ctx, "grpc://grpc.nile.trongrid.io:50051")
	if err != nil {
		fmt.Println("dial:", err)
		return
	}
	defer cli.Close()

	signer, err := tronlib.KeyFromHex(os.Getenv("TRON_PRIVATE_KEY"))
	if err != nil {
		fmt.Println("key:", err)
		return
	}
	to, err := tronlib.ParseAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
	if err != nil {
		fmt.Println("address:", err)
		return
	}
	// Nile's official USDT contract (docs/verification.md, address appendix).
	token, err := tronlib.ParseAddress("TXLAQ63Xg1NAzckPwKHvzw7CSEmLMEqcdj")
	if err != nil {
		fmt.Println("address:", err)
		return
	}
	inst, err := cli.Contract(ctx, token)
	if err != nil {
		fmt.Println("contract:", err)
		return
	}

	// 1. Read-only call. Results carry their ABI type; accessors return an
	// error rather than a zero value when the type does not match.
	supply, err := inst.Call(ctx, "balanceOf", contract.AddressArg(signer.Address()))
	if err != nil {
		fmt.Println("call:", err)
		return
	}
	held, err := supply.BigInt()
	if err != nil {
		fmt.Println("decode:", err)
		return
	}
	fmt.Println("token balance:", held.String())

	// 2. Build the state-changing call. The ABI is fetched from the node on
	// first use (Instance.UseABI supplies one offline). USDT has 6 decimals, so
	// one token is 1_000_000 raw units; the generic contract layer takes the
	// raw ABI value, and token.Handle does the scaling for you (see
	// ExampleClient_Token).
	units := new(big.Int).Mul(big.NewInt(1), big.NewInt(1_000_000))
	call, err := inst.Invoke(ctx, signer.Address(), 0, "transfer",
		contract.AddressArg(to), contract.BigIntArg(units))
	if err != nil {
		fmt.Println("invoke:", err)
		return
	}

	// 3. Dry-run: the accurate energy cost, any revert message, and the ABI
	// return value of the call the node simulated.
	est, err := call.Simulate(ctx)
	if err != nil {
		fmt.Println("simulate:", err)
		return
	}
	fmt.Println("energy:", est.Energy, "penalty:", est.Penalty, "revert:", est.Revert)
	if est.HasResult() {
		result, err := inst.Decode("transfer", est.ConstantResult[0])
		if err != nil {
			fmt.Println("decode result:", err)
			return
		}
		if ok, err := result.Bool(); err == nil {
			fmt.Println("transfer would succeed:", ok)
		}
	}

	// 4. Price it against this account's staked energy, then broadcast.
	acct := cli.Account(signer.Address())
	preview, err := acct.CostPreview(ctx, call)
	if err != nil {
		fmt.Println("preview:", err)
		return
	}
	fmt.Println("burn", preview.TronToBurn.Formatted(), "TRX of", preview.SunPerEnergy, "sun/energy",
		"("+preview.BandwidthNote+")")

	signed, err := call.Sign(signer)
	if err != nil {
		fmt.Println("sign:", err)
		return
	}
	rec, err := cli.Broadcast(ctx, signed)
	if err != nil {
		fmt.Println("broadcast:", err)
		return
	}
	if !rec.OK() {
		fmt.Println("node rejected:", rec.NodeCode, rec.Revert)
		return
	}

	// 5. Decode what the transaction emitted. Receipt logs are decoded against
	// every ABI registered for the emitting contract — Instance.UseABI (and
	// the token handle) register theirs — and unknown signatures stay in the
	// receipt with their raw topics instead of being dropped.
	for _, lg := range rec.Logs {
		fmt.Println("event", lg.EventName, "from", lg.Address)
		for _, p := range lg.Parameters {
			fmt.Println("   ", p.Name, "=", p.Value)
		}
	}
	// The same logs are retrievable later, by txid, without a receipt.
	logs, err := cli.Events(ctx, rec.TxID)
	if err != nil {
		fmt.Println("events:", err)
		return
	}
	fmt.Println("stored logs:", len(logs))
}

// ExampleClient_Token uses the TRC-20 handle, which owns the token's scale:
// every Amount it mints carries the token's decimals, so "1.5" means 1.5
// USDT and not 1.5 raw units.
//
// It also shows the two-step approval flow: the owner approves a spender, and
// the spender pulls the funds with transferFrom — a plain `transfer` moves
// only the owner's own tokens.
//
// The key here is a BIP-39 mnemonic, the other form the SDK accepts.
func ExampleClient_Token() {
	ctx := context.Background()

	cli, err := tronlib.Dial(ctx, "grpc://grpc.nile.trongrid.io:50051")
	if err != nil {
		fmt.Println("dial:", err)
		return
	}
	defer cli.Close()

	// A mnemonic plus derivation path (TRON uses coin type 195).
	ownerKey, err := tronlib.KeyFromMnemonic(os.Getenv("TRON_MNEMONIC"), "", "m/44'/195'/0'/0/0")
	if err != nil {
		fmt.Println("mnemonic:", err)
		return
	}
	spenderKey, err := tronlib.KeyFromMnemonic(os.Getenv("TRON_MNEMONIC"), "", "m/44'/195'/0'/0/1")
	if err != nil {
		fmt.Println("mnemonic:", err)
		return
	}
	usdt, err := tronlib.ParseAddress("TXLAQ63Xg1NAzckPwKHvzw7CSEmLMEqcdj")
	if err != nil {
		fmt.Println("address:", err)
		return
	}
	handle, err := cli.Token(ctx, usdt)
	if err != nil {
		fmt.Println("token:", err)
		return
	}

	// Metadata and reads. Decimals are read once, eagerly, by Token.
	symbol, err := handle.Symbol(ctx)
	if err != nil {
		fmt.Println("symbol:", err)
		return
	}
	name, err := handle.Name(ctx)
	if err != nil {
		fmt.Println("name:", err)
		return
	}
	supply, err := handle.TotalSupply(ctx)
	if err != nil {
		fmt.Println("totalSupply:", err)
		return
	}
	fmt.Println(name, "("+symbol+")", "decimals:", handle.Decimals(), "supply:", supply.Formatted())

	owner := ownerKey.Address()
	spender := spenderKey.Address()
	balance, err := handle.BalanceOf(ctx, owner)
	if err != nil {
		fmt.Println("balanceOf:", err)
		return
	}
	allowance, err := handle.Allowance(ctx, owner, spender)
	if err != nil {
		fmt.Println("allowance:", err)
		return
	}
	fmt.Println("owner holds", balance.Formatted(), "| spender may pull", allowance.Formatted())

	// Step 1: the owner approves a limited allowance. Decimal input goes
	// through the handle ("1.5" -> 1500000 raw units at 6 decimals); the
	// integer-only Whole is for whole tokens.
	spend, err := handle.Whole(1)
	if err != nil {
		fmt.Println("amount:", err)
		return
	}
	approve, err := handle.Approve(ctx, owner, spender, spend)
	if err != nil {
		fmt.Println("approve:", err)
		return
	}
	signedApprove, err := approve.Sign(ownerKey)
	if err != nil {
		fmt.Println("sign approve:", err)
		return
	}
	if _, err := cli.Broadcast(ctx, signedApprove); err != nil {
		fmt.Println("broadcast approve:", err)
		return
	}

	// Step 2: the spender pulls the approved amount. transferFrom is a plain
	// ABI method, so it goes through the generic contract instance.
	to, err := tronlib.ParseAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
	if err != nil {
		fmt.Println("address:", err)
		return
	}
	inst, err := cli.Contract(ctx, usdt)
	if err != nil {
		fmt.Println("contract:", err)
		return
	}
	pull, err := inst.Invoke(ctx, spender, 0, "transferFrom",
		contract.AddressArg(owner), contract.AddressArg(to), contract.BigIntArg(spend.Raw()))
	if err != nil {
		fmt.Println("transferFrom:", err)
		return
	}
	signedPull, err := pull.Sign(spenderKey)
	if err != nil {
		fmt.Println("sign transferFrom:", err)
		return
	}
	if _, err := cli.Broadcast(ctx, signedPull); err != nil {
		fmt.Println("broadcast transferFrom:", err)
		return
	}

	// Or move your own tokens directly, no approval needed.
	move, err := handle.Transfer(ctx, owner, to, spend)
	if err != nil {
		fmt.Println("transfer:", err)
		return
	}
	signedMove, err := move.Sign(ownerKey)
	if err != nil {
		fmt.Println("sign transfer:", err)
		return
	}
	if _, err := cli.Broadcast(ctx, signedMove); err != nil {
		fmt.Println("broadcast transfer:", err)
	}
}

// ExampleResources_Stake is the staking lifecycle in one place: stake for
// Energy, start an unstake, harvest what has matured, and lend part of the
// stake to another account.
//
// The three unstake states are distinct and the API names them apart, because
// the TRX is not spendable in the middle one. The cooldown length is a chain
// parameter (14 days on Mainnet, 1 on Nile) — read it, never assume it.
func ExampleResources_Stake() {
	ctx := context.Background()

	cli, err := tronlib.Dial(ctx, "grpc://grpc.nile.trongrid.io:50051")
	if err != nil {
		fmt.Println("dial:", err)
		return
	}
	defer cli.Close()

	signer, err := tronlib.KeyFromHex(os.Getenv("TRON_PRIVATE_KEY"))
	if err != nil {
		fmt.Println("key:", err)
		return
	}
	user, err := tronlib.ParseAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
	if err != nil {
		fmt.Println("address:", err)
		return
	}
	res := cli.Account(signer.Address()).Resources()

	// How long TRX stays in the cooldown on this network, and the longest
	// delegation lock it accepts.
	params, err := tronlib.ChainParamsOf(ctx, cli.Raw())
	if err != nil {
		fmt.Println("chain params:", err)
		return
	}
	fmt.Println("unstake delay:", params.UnfreezeDelayDays, "days | max lock:",
		params.MaxDelegateLockPeriod, "blocks")

	// 1. Stake 100 TRX for Energy. The TRX leaves the spendable balance now;
	// the resulting Energy share arrives with the next block.
	stake, err := res.Stake(ctx, tronlib.Energy, tronlib.TRX(100))
	if err != nil {
		fmt.Println("stake:", err)
		return
	}
	if signed, err := stake.Sign(signer); err == nil {
		_, _ = cli.Broadcast(ctx, signed)
	}

	// 2. Start the cooldown for 50 TRX of it. Unstake does NOT return TRX.
	unstake, err := res.Unstake(ctx, tronlib.Energy, tronlib.TRX(50))
	if err != nil {
		fmt.Println("unstake:", err)
		return
	}
	if signed, err := unstake.Sign(signer); err == nil {
		_, _ = cli.Broadcast(ctx, signed)
	}

	// 3. Ask what has matured, how many unstakes may still be started (the
	// network caps concurrent ones), then claim the matured TRX.
	withdrawable, err := res.Withdrawable(ctx)
	if err != nil {
		fmt.Println("withdrawable:", err)
		return
	}
	slots, err := res.UnstakeSlots(ctx)
	if err != nil {
		fmt.Println("slots:", err)
		return
	}
	fmt.Println("withdrawable:", withdrawable.Formatted(), "TRX |", slots, "unstake slots left")
	withdraw, err := res.WithdrawUnstaked(ctx)
	if err != nil {
		fmt.Println("withdraw:", err)
		return
	}
	if signed, err := withdraw.Sign(signer); err == nil {
		_, _ = cli.Broadcast(ctx, signed)
	}

	// 4. Lend 1000 TRX of the Energy stake to another account for a day.
	// LockBlocks is BLOCKS (~3 s each), not seconds: 28,800 ≈ 24 h. A locked
	// delegation cannot be undelegated until it expires.
	free, err := res.Delegatable(ctx, tronlib.Energy)
	if err != nil {
		fmt.Println("delegatable:", err)
		return
	}
	fmt.Println("delegatable:", free.Formatted(), "TRX of energy stake")
	delegate, err := res.Delegate(ctx, tronlib.Energy, user, tronlib.TRX(1000), tronlib.DelegateParams{
		Lock:       true,
		LockBlocks: 28_800,
	})
	if err != nil {
		fmt.Println("delegate:", err)
		return
	}
	if signed, err := delegate.Sign(signer); err == nil {
		_, _ = cli.Broadcast(ctx, signed)
	}

	// 5. Track the delegation: who lends to this account and to whom does it
	// lend, without one read per counterparty.
	index, err := res.DelegationIndex(ctx)
	if err != nil {
		fmt.Println("index:", err)
		return
	}
	fmt.Println("delegators:", len(index.From), "receivers:", len(index.To))
	granted, err := res.DelegationsGrantedTo(ctx, user)
	if err != nil {
		fmt.Println("granted:", err)
		return
	}
	for _, d := range granted {
		fmt.Println("lent to", d.To, d.Energy.Formatted(), "TRX of energy stake, unlocks", d.EnergyExpiresAt)
	}
}

// ExamplePermissions_Current configures multi-signature control. A permission
// update REPLACES the account's whole configuration, so the safe pattern is
// always read → edit → submit the complete set: anything omitted is wiped.
//
// The example gives the owner a 2-of-3 threshold and adds a single-key active
// permission scoped to transfers only, which is how a hot key gets a narrow
// path without holding the account.
func ExamplePermissions_Current() {
	ctx := context.Background()

	cli, err := tronlib.Dial(ctx, "grpc://grpc.nile.trongrid.io:50051")
	if err != nil {
		fmt.Println("dial:", err)
		return
	}
	defer cli.Close()

	owner, err := tronlib.ParseAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
	if err != nil {
		fmt.Println("address:", err)
		return
	}
	perms := cli.Account(owner).Permissions()

	current, err := perms.Current(ctx)
	if err != nil {
		fmt.Println("current:", err)
		return
	}
	fmt.Println("owner keys:", len(current.Owner.Keys), "threshold:", current.Owner.Threshold,
		"actives:", len(current.Actives))

	// An active permission is scoped by a 32-byte operations bitmap. Build it
	// from contract types rather than hand-written hex: a wrong byte order
	// silently grants or denies the wrong operations.
	bitmap, err := tronlib.OperationsBitmap(tronlib.TypeTransfer, tronlib.TypeTriggerSmartContract, tronlib.TypeDelegateResource)
	if err != nil {
		fmt.Println("bitmap:", err)
		return
	}
	allowed, err := tronlib.OperationsList(bitmap)
	if err != nil {
		fmt.Println("operations:", err)
		return
	}
	fmt.Println("the active permission may execute:", allowed)

	keyA, err := tronlib.ParseAddress("TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb")
	if err != nil {
		fmt.Println("address:", err)
		return
	}
	keyB, err := tronlib.ParseAddress("TXLAQ63Xg1NAzckPwKHvzw7CSEmLMEqcdj")
	if err != nil {
		fmt.Println("address:", err)
		return
	}
	current.Owner = tronlib.Permission{
		Name:      "owner",
		Threshold: 2,
		Keys: []tronlib.PermissionKey{
			{Address: owner, Weight: 1},
			{Address: keyA, Weight: 1},
			{Address: keyB, Weight: 1},
		},
	}
	current.Actives = append(current.Actives, tronlib.Permission{
		Name:       "payments",
		Threshold:  1,
		Keys:       []tronlib.PermissionKey{{Address: keyA, Weight: 1}},
		Operations: bitmap,
	})

	update, err := perms.Update(ctx, current)
	if err != nil {
		fmt.Println("update:", err)
		return
	}
	signer, err := tronlib.KeyFromHex(os.Getenv("TRON_PRIVATE_KEY"))
	if err != nil {
		fmt.Println("key:", err)
		return
	}
	signed, err := update.Sign(signer)
	if err != nil {
		fmt.Println("sign:", err)
		return
	}
	// The node charges a fixed permission-update fee; TotalCost includes it.
	cost, err := cli.Account(owner).TotalCost(ctx, signed)
	if err != nil {
		fmt.Println("cost:", err)
		return
	}
	fmt.Println("permission update fee:", cost.PermissionUpdateFee.Formatted(), "TRX")
	if _, err := cli.Broadcast(ctx, signed); err != nil {
		fmt.Println("broadcast:", err)
	}
}

// ExamplePermissions_SignWeight is offline multi-signature: the transaction
// travels as a portable envelope, each signer works on its own machine with
// its own key form, and the node — not the client — decides whether the
// collected weight meets the threshold.
//
// Two rules make this safe. The envelope records which kind of transaction it
// holds, so an importer cannot misread it as a different contract type; and
// "at least one signature" is not authorization — Enough is.
func ExamplePermissions_SignWeight() {
	ctx := context.Background()

	cli, err := tronlib.Dial(ctx, "grpc://grpc.nile.trongrid.io:50051")
	if err != nil {
		fmt.Println("dial:", err)
		return
	}
	defer cli.Close()

	// The account is controlled 2-of-3 by keys this process may not all hold.
	corporate, err := tronlib.ParseAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
	if err != nil {
		fmt.Println("address:", err)
		return
	}
	vendor, err := tronlib.ParseAddress("TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb")
	if err != nil {
		fmt.Println("address:", err)
		return
	}
	acct := cli.Account(corporate)

	// 1. Build the transaction, then hand it off as bytes: persist the
	// envelope, mail it, or pass it to another process.
	transfer, err := acct.TransferTRX(ctx, vendor, tronlib.TRX(100))
	if err != nil {
		fmt.Println("build:", err)
		return
	}
	envelope, err := tronlib.Encode(transfer)
	if err != nil {
		fmt.Println("encode:", err)
		return
	}

	// 2. Signer A imports it with a hex key, signs, and exports the partial.
	// Signing never changes raw_data, so the txid is stable across handoffs.
	imported, err := tronlib.Decode(envelope)
	if err != nil {
		fmt.Println("decode:", err)
		return
	}
	keyA, err := tronlib.KeyFromHex(os.Getenv("TRON_SIGNER_A_KEY"))
	if err != nil {
		fmt.Println("key A:", err)
		return
	}
	signedA, err := tronlib.Sign(imported, keyA)
	if err != nil {
		fmt.Println("sign A:", err)
		return
	}
	partial, err := tronlib.Encode(signedA)
	if err != nil {
		fmt.Println("encode partial:", err)
		return
	}

	// 3. Signer B is a different machine with only a mnemonic. Re-signing an
	// already-present signer is rejected, so a duplicate cannot slip in.
	keyB, err := tronlib.KeyFromMnemonic(os.Getenv("TRON_SIGNER_B_MNEMONIC"), "", "m/44'/195'/0'/0/0")
	if err != nil {
		fmt.Println("key B:", err)
		return
	}
	roundTrip, err := tronlib.Decode(partial)
	if err != nil {
		fmt.Println("decode partial:", err)
		return
	}
	signers, err := roundTrip.Signers()
	if err != nil {
		fmt.Println("signers:", err)
		return
	}
	fmt.Println("collected so far:", signers)
	full, err := tronlib.Sign(roundTrip, keyB)
	if err != nil {
		fmt.Println("sign B:", err)
		return
	}

	// 4. Ask the node for the weight under the transaction's permission:
	// 2-of-3 needs both signatures, and the answer, not the signature count,
	// is what authorizes the broadcast.
	status, err := acct.Permissions().SignWeight(ctx, full)
	if err != nil {
		fmt.Println("sign weight:", err)
		return
	}
	fmt.Println(status.String(), "approved:", status.Approved)
	if !status.Enough {
		fmt.Println("not enough weight; keep collecting signatures")
		return
	}
	if _, err := cli.Broadcast(ctx, full); err != nil {
		fmt.Println("broadcast:", err)
	}

	// 5. A hardware or remote signer needs only the hash: nothing but the
	// 32-byte digest leaves this process, and the signature comes back to be
	// attached after it is checked against the stated address.
	fresh, err := acct.TransferTRX(ctx, vendor, tronlib.TRX(1))
	if err != nil {
		fmt.Println("build:", err)
		return
	}
	digest, err := tronlib.SignHash(fresh)
	if err != nil {
		fmt.Println("sign hash:", err)
		return
	}
	signature, err := keyA.Sign(digest) // stand-in for the remote signer's result
	if err != nil {
		fmt.Println("remote sign:", err)
		return
	}
	if _, err := tronlib.AttachSignature(fresh, keyA.Address(), signature); err != nil {
		fmt.Println("attach:", err)
	}
}
