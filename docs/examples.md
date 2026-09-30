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
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

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

if _, err := cli.WaitForSolid(ctx, rec.TxID); err != nil {
	fmt.Println("wait:", err)
}<!-- /go:example -->

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
acct := cli.Account(signer.Address())

state, err := acct.State(ctx)
if err != nil {
	fmt.Println("state:", err)
	return
}
fmt.Println(state.Balance.Formatted(), "TRX,", len(state.Stakes), "stake(s),",
	len(state.Unstakes), "unstake(s),", len(state.Votes), "vote(s)")

resources, err := acct.Resources().State(ctx)
if err != nil {
	fmt.Println("resources:", err)
	return
}
fmt.Println("energy", resources.EnergyAvailable(), "of", resources.EnergyLimit,
	"| bandwidth", resources.BandwidthAvailable(), "of",
	resources.BandwidthLimit+resources.FreeBandwidthLimit,
	"| tron power", resources.TronPowerAvailable())

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
fmt.Println("this transfer costs:", cost.Total.Formatted(), "TRX —", cost.String())<!-- /go:example -->

<!-- go:example tronlib.ExampleClient_Contract -->
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

units := new(big.Int).Mul(big.NewInt(1), big.NewInt(1_000_000))
call, err := inst.Invoke(ctx, signer.Address(), 0, "transfer",
	contract.AddressArg(to), contract.BigIntArg(units))
if err != nil {
	fmt.Println("invoke:", err)
	return
}

est, err := call.Simulate(ctx)
if err != nil {
	fmt.Println("simulate:", err)
	return
}
if est.Revert != "" {
	fmt.Println("the call would revert:", est.Revert, "— energy", est.Energy, "code", est.Code)
} else {
	fmt.Println("energy:", est.Energy, "penalty:", est.Penalty)
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
}

acct := cli.Account(signer.Address())
preview, err := acct.CostPreview(ctx, call)
if err != nil {
	fmt.Println("preview:", err)
	return
}
fmt.Println("burn", preview.TronToBurn.Formatted(), "TRX of", preview.SunPerEnergy, "sun/energy;")
fmt.Println(preview.Bandwidth.String(), "; total floor", preview.TotalFloor.Formatted(), "TRX",
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

for _, lg := range rec.Logs {
	fmt.Println("event", lg.EventName, "from", lg.Address)
	for _, p := range lg.Parameters {
		fmt.Println("   ", p.Name, "=", p.Value)
	}
}

logs, err := cli.Events(ctx, rec.TxID)
if err != nil {
	fmt.Println("events:", err)
	return
}
fmt.Println("stored logs:", len(logs))<!-- /go:example -->

<!-- go:example tronlib.ExampleClient_Token -->
ctx := context.Background()

cli, err := tronlib.Dial(ctx, "grpc://grpc.nile.trongrid.io:50051")
if err != nil {
	fmt.Println("dial:", err)
	return
}
defer cli.Close()

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
user, err := tronlib.ParseAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
if err != nil {
	fmt.Println("address:", err)
	return
}
res := cli.Account(signer.Address()).Resources()

params, err := tronlib.ChainParamsOf(ctx, cli.Raw())
if err != nil {
	fmt.Println("chain params:", err)
	return
}
fmt.Println("unstake delay:", params.UnfreezeDelayDays, "days | max lock:",
	params.MaxDelegateLockPeriod, "blocks")

stake, err := res.Stake(ctx, tronlib.Energy, tronlib.TRX(100))
if err != nil {
	fmt.Println("stake:", err)
	return
}
if signed, err := stake.Sign(signer); err == nil {
	_, _ = cli.Broadcast(ctx, signed)
}

unstake, err := res.Unstake(ctx, tronlib.Energy, tronlib.TRX(50))
if err != nil {
	fmt.Println("unstake:", err)
	return
}
if signed, err := unstake.Sign(signer); err == nil {
	_, _ = cli.Broadcast(ctx, signed)
}

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
fmt.Println("owner keys:", len(current.Owner.Keys), "threshold:", current.Owner.Threshold,
	"actives:", len(current.Actives))

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

cost, err := cli.Account(owner).TotalCost(ctx, signed)
if err != nil {
	fmt.Println("cost:", err)
	return
}
fmt.Println("permission update fee:", cost.PermissionUpdateFee.Formatted(), "TRX")
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
signature, err := keyA.Sign(digest)
if err != nil {
	fmt.Println("remote sign:", err)
	return
}
if _, err := tronlib.AttachSignature(fresh, keyA.Address(), signature); err != nil {
	fmt.Println("attach:", err)
}<!-- /go:example -->
