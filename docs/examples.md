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

transfer, err := cli.TransferTRX(ctx, signer.Address(), to, tronlib.TRX(1))
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
bal, err := cli.TronBalance(ctx, from)
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
