package tronlib_test

// Compile-only examples for the root facade (architecture §10's program shape).
//
// These examples deliberately carry NO // Output: comment: go test compiles
// them but never executes them, so the happy path is proven to build
// against the real surface without dialing a real node. docgen sync-docs
// DOES extract them: the CI drift gate passes this directory as
// -example-pkg ., docs/examples.md carries the tronlib.* markers, and every
// Example here must have one — so an edit to a body must be followed by a
// docgen sync of that file or the gate fails.

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kslamph/tronlib/v2"
)

// Example is the architecture §10 happy path: one import, dial, sign, broadcast.
func Example() {
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

	transfer, err := cli.Account(signer.Address()).TransferTRX(ctx, to, tronlib.TRX(1))
	if err != nil {
		fmt.Println("build:", err)
		return
	}
	signed, err := transfer.Sign(signer)
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
		fmt.Println("node rejected:", rec.NodeCode)
		return
	}
	// Inclusion is not finality; custody waits for solidification.
	if _, err := cli.WaitForSolid(ctx, rec.TxID); err != nil {
		fmt.Println("wait:", err)
	}
}

// ExampleClient_trx reads chain state and the account's balance the way a
// transfer building step does: tip, balance and energy price. The account is
// bound once with Client.Account and owns every owner-specific operation.
func ExampleClient_trx() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cli, err := tronlib.Dial(ctx, "grpc://grpc.nile.trongrid.io:50051",
		tronlib.WithTimeout(5*time.Second), tronlib.WithPool(1, 2))
	if err != nil {
		fmt.Println("dial:", err)
		return
	}
	defer cli.Close()

	from, err := tronlib.ParseAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
	if err != nil {
		fmt.Println("address:", err)
		return
	}
	acct := cli.Account(from)
	bal, err := acct.Balance(ctx)
	if err != nil {
		fmt.Println("balance:", err)
		return
	}
	if _, err := cli.ChainTip(ctx); err != nil {
		fmt.Println("tip:", err)
		return
	}
	price, err := cli.EnergyPrice(ctx)
	if err != nil {
		fmt.Println("price:", err)
		return
	}
	fmt.Println(bal.Formatted(), "TRX at", price.SunPerEnergy, "sun/energy")
}

// ExampleClient_Account is the account-scoped shape: one handle for state, one
// for staking, and the same build → sign → broadcast pipeline for both a
// transfer and a stake. The account being operated on (owner) is separate from
// the signer, which is what lets a multi-signature signer authorize someone
// else's account.
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
	to, err := tronlib.ParseAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
	if err != nil {
		fmt.Println("address:", err)
		return
	}

	acct := cli.Account(signer.Address())
	state, err := acct.State(ctx)
	if err != nil {
		fmt.Println("state:", err)
		return
	}
	fmt.Println(state.Balance.Formatted(), "TRX,", len(state.Stakes), "stake(s)")

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
	if _, err := cli.Broadcast(ctx, signed); err != nil {
		fmt.Println("broadcast:", err)
		return
	}

	// Staking is the same pipeline: build an unsigned transaction, sign it,
	// broadcast it. The TRX leaves the spendable balance immediately and the
	// resource share appears once the transaction is included.
	stake, err := acct.Resources().Stake(ctx, tronlib.Energy, tronlib.TRX(100))
	if err != nil {
		fmt.Println("stake:", err)
		return
	}
	signedStake, err := stake.Sign(signer)
	if err != nil {
		fmt.Println("sign stake:", err)
		return
	}
	if _, err := cli.Broadcast(ctx, signedStake); err != nil {
		fmt.Println("broadcast stake:", err)
	}
}

// ExampleResources_Stake shows the stake lifecycle as three distinct
// operations, because the TRX is in a different state after each: staked,
// in cooldown (not spendable), then spendable. Unstaking never returns TRX
// directly — the chain's delay (14 days on Mainnet, 1 on Nile) has to elapse
// first, and the delay is a chain parameter, not a constant.
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
	res := cli.Account(signer.Address()).Resources()

	// 1. Stake 100 TRX for Energy.
	stake, err := res.Stake(ctx, tronlib.Energy, tronlib.TRX(100))
	if err != nil {
		fmt.Println("stake:", err)
		return
	}
	if signed, err := stake.Sign(signer); err == nil {
		_, _ = cli.Broadcast(ctx, signed)
	}

	// 2. Start the cooldown for 50 TRX of it; the TRX is not spendable yet.
	//    Read the chain's delay rather than assuming 14 days.
	params, err := tronlib.ChainParamsOf(ctx, cli.Raw())
	if err != nil {
		fmt.Println("params:", err)
		return
	}
	fmt.Println("unstake delay:", params.UnfreezeDelayDays, "days")
	unstake, err := res.Unstake(ctx, tronlib.Energy, tronlib.TRX(50))
	if err != nil {
		fmt.Println("unstake:", err)
		return
	}
	if signed, err := unstake.Sign(signer); err == nil {
		_, _ = cli.Broadcast(ctx, signed)
	}

	// 3. Once the cooldown elapses, claim the matured TRX. Withdrawable
	//    answers "how much can I claim right now"; the unstake slots answer
	//    "how many more unstakes may I start" (32 at most, network-wide).
	if amount, err := res.Withdrawable(ctx); err == nil {
		fmt.Println("withdrawable:", amount.Formatted(), "TRX")
	}
	if slots, err := res.UnstakeSlots(ctx); err == nil {
		fmt.Println("unstakes remaining:", slots)
	}
	withdraw, err := res.WithdrawUnstaked(ctx)
	if err != nil {
		fmt.Println("withdraw:", err)
		return
	}
	if signed, err := withdraw.Sign(signer); err == nil {
		_, _ = cli.Broadcast(ctx, signed)
	}
}

// ExampleResources_Delegate shows lending staked Energy to another account.
// The amount is the TRX (SUN) of stake whose resource share is lent — not an
// Energy quantity — and the TRX stays staked under the delegator. A locked
// delegation cannot be undone until its lock expires, and the lock length is
// in BLOCKS (~3 s each), not seconds.
func ExampleResources_Delegate() {
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

	// How much of the Energy stake is free to delegate right now (usage in the
	// 24-hour recovery window reduces it).
	free, err := res.Delegatable(ctx, tronlib.Energy)
	if err != nil {
		fmt.Println("delegatable:", err)
		return
	}
	fmt.Println(free.Formatted(), "TRX of Energy stake is delegatable")

	// Lock the delegation for one day: 28,800 blocks at ~3 s per block.
	delegation, err := res.Delegate(ctx, tronlib.Energy, user, tronlib.TRX(1000), tronlib.DelegateParams{
		Lock:       true,
		LockBlocks: 28_800,
	})
	if err != nil {
		fmt.Println("delegate:", err)
		return
	}
	if signed, err := delegation.Sign(signer); err == nil {
		_, _ = cli.Broadcast(ctx, signed)
	}

	// Who has delegated to this account, and to whom does it delegate? The
	// index answers both without one read per counterparty.
	if idx, err := res.DelegationIndex(ctx); err == nil {
		fmt.Println("delegators:", len(idx.From), "receivers:", len(idx.To))
	}
}

// ExamplePermissions_Current reads an account's complete permission
// configuration and submits a modified copy. The shape matters: an
// AccountPermissionUpdate REPLACES all three slots at once, so the safe
// pattern is always read → edit → submit the whole set. An active permission
// is scoped by a 32-byte operations bitmap, which OperationsBitmap builds from
// contract types rather than hand-written hex.
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
	fmt.Println("owner keys:", len(current.Owner.Keys), "threshold:", current.Owner.Threshold)

	// Give a hot key a single-signature path for transfers only: TRX transfers
	// and TRC-10/TRC-20 moves are TransferContract and TriggerSmartContract;
	// everything else, including changing permissions, stays owner-only.
	bitmap, err := tronlib.OperationsBitmap(tronlib.TypeTransfer, tronlib.TypeTriggerSmartContract)
	if err != nil {
		fmt.Println("bitmap:", err)
		return
	}
	hot, err := tronlib.ParseAddress("TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb")
	if err != nil {
		fmt.Println("hot key:", err)
		return
	}
	current.Actives = append(current.Actives, tronlib.Permission{
		Name:       "payments",
		Threshold:  1,
		Keys:       []tronlib.PermissionKey{{Address: hot, Weight: 1}},
		Operations: bitmap,
	})

	// The update must be signed under the current owner permission (or an
	// active one that enables TypeAccountPermissionUpdate), and the node
	// charges a fixed fee read live from the chain parameters.
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
	if cost, err := cli.Account(owner).TotalCost(ctx, signed); err == nil {
		fmt.Println("permission update fee:", cost.PermissionUpdateFee.Formatted(), "TRX")
	}
	if _, err := cli.Broadcast(ctx, signed); err != nil {
		fmt.Println("broadcast:", err)
	}
}

// ExamplePermissions_SignWeight is the offline multi-signature flow: the
// unsigned transaction travels as a portable envelope, each signer works on
// its own machine, and the node — not the client — decides whether the
// collected weight meets the permission's threshold before broadcast.
// "At least one signature" is not authorization; Enough is.
func ExamplePermissions_SignWeight() {
	ctx := context.Background()

	cli, err := tronlib.Dial(ctx, "grpc://grpc.nile.trongrid.io:50051")
	if err != nil {
		fmt.Println("dial:", err)
		return
	}
	defer cli.Close()

	// The account is controlled 2-of-3; this process may hold only one key.
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

	// 1. Build, then hand the unsigned transaction off as bytes.
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

	// 2. Signer A imports, signs, re-exports. Signatures accumulate across
	//    handoffs and the txid never changes: a txid is a function of
	//    raw_data, and signing does not touch raw_data.
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

	// 3. Ask the node how much weight the partial signature list carries
	//    under the transaction's permission. Broadcast only once Enough.
	if status, err := acct.Permissions().SignWeight(ctx, signedA); err == nil {
		fmt.Println(status.String(), "approved:", len(status.Approved))
		if !status.Enough {
			fmt.Println("collecting more signatures; partial envelope bytes:", len(partial))
			return
		}
	}
	if _, err := cli.Broadcast(ctx, signedA); err != nil {
		fmt.Println("broadcast:", err)
	}
}

// ExampleClient_token reads a TRC-20 balance through the facade's Token
// handle; amounts minted by the Handle carry the token's decimals. The token
// address must be a contract: Client.Token builds a token.Handle, which
// calls decimals() eagerly, so an EOA address fails in that call instead of
// returning a handle. `owner` is the account whose balance is read.
func ExampleClient_token() {
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
	// Nile's official USDT contract (docs/verification.md, address appendix).
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
	bal, err := handle.BalanceOf(ctx, owner)
	if err != nil {
		fmt.Println("balanceOf:", err)
		return
	}
	fmt.Println(bal.String())
}
