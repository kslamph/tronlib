# tronlib v2 — Compiled Examples

<!-- Regenerate: go run ./cmd/docgen sync-docs -pkg ./tron -example-pkg . -docs ./docs/examples.md -->

Every snippet below is extracted from a compiled `Example` function in the
package's `_test.go` files — it is exactly what CI builds. To add one, write
an `ExampleXxx` function; it appears here on the next sync.

<!-- go:example tron.ExampleHasCode -->
inner := &tron.Error{Code: tron.CodeAmountTooManyDecimals, Op: "example"}
wrapped := fmt.Errorf("building transfer: %w", inner)

if tron.HasCode(wrapped, tron.CodeAmountTooManyDecimals) {
	fmt.Println("round to 6 decimal places")
}<!-- /go:example -->

<!-- go:example tron.ExampleParseAddress -->
a, err := tron.ParseAddress("TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb")
if err != nil {
	fmt.Println("err:", err)
	return
}
fmt.Println(a.String())<!-- /go:example -->

<!-- go:example tron.ExampleParseTRX -->
v, err := tron.ParseTRX("1.6")
if err != nil {
	fmt.Println("err:", err)
	return
}
fmt.Println(v.String())<!-- /go:example -->

<!-- go:example tron.ExampleTRX -->
fmt.Println(tron.TRX(1).Int64())<!-- /go:example -->

## Root facade (package tronlib)

The one-import happy path, documented under `tronlib.*`.

<!-- go:example tronlib.Example -->
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

if _, err := cli.WaitForSolid(ctx, rec.TxID); err != nil {
	fmt.Println("wait:", err)
}<!-- /go:example -->

<!-- go:example tronlib.ExampleClient_trx -->
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
fmt.Println(bal.Formatted(), "TRX at", price.SunPerEnergy, "sun/energy")<!-- /go:example -->

<!-- go:example tronlib.ExampleClient_Account -->
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
}<!-- /go:example -->

<!-- go:example tronlib.ExampleResources_Stake -->
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

stake, err := res.Stake(ctx, tronlib.Energy, tronlib.TRX(100))
if err != nil {
	fmt.Println("stake:", err)
	return
}
if signed, err := stake.Sign(signer); err == nil {
	_, _ = cli.Broadcast(ctx, signed)
}

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
}<!-- /go:example -->

<!-- go:example tronlib.ExampleResources_Delegate -->
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

free, err := res.Delegatable(ctx, tronlib.Energy)
if err != nil {
	fmt.Println("delegatable:", err)
	return
}
fmt.Println(free.Formatted(), "TRX of Energy stake is delegatable")

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

if idx, err := res.DelegationIndex(ctx); err == nil {
	fmt.Println("delegators:", len(idx.From), "receivers:", len(idx.To))
}<!-- /go:example -->

<!-- go:example tronlib.ExamplePermissions_Current -->
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
}<!-- /go:example -->

<!-- go:example tronlib.ExamplePermissions_SignWeight -->
ctx := context.Background()

cli, err := tronlib.Dial(ctx, "grpc://grpc.nile.trongrid.io:50051")
if err != nil {
	fmt.Println("dial:", err)
	return
}
defer cli.Close()

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

if status, err := acct.Permissions().SignWeight(ctx, signedA); err == nil {
	fmt.Println(status.String(), "approved:", len(status.Approved))
	if !status.Enough {
		fmt.Println("collecting more signatures; partial envelope bytes:", len(partial))
		return
	}
}
if _, err := cli.Broadcast(ctx, signedA); err != nil {
	fmt.Println("broadcast:", err)
}<!-- /go:example -->

<!-- go:example tronlib.ExampleClient_token -->
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
fmt.Println(bal.String())<!-- /go:example -->
