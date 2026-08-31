# tronlib v2 — Compiled Examples

<!-- Regenerate: go -C v2 run ./cmd/docgen sync-docs -pkg ./tron -docs ./docs/examples.md -->

Every snippet below is extracted from a compiled `Example` function in the
package's `_test.go` files — it is exactly what CI builds. To add one, write
an `ExampleXxx` function; it appears here on the next sync.

<!-- go:example ExampleHasCode -->
inner := &tron.Error{Code: tron.CodeAmountTooManyDecimals, Op: "example"}
wrapped := fmt.Errorf("building transfer: %w", inner)

if tron.HasCode(wrapped, tron.CodeAmountTooManyDecimals) {
	fmt.Println("round to 6 decimal places")
}<!-- /go:example -->

<!-- go:example ExampleParseAddress -->
a, err := tron.ParseAddress("TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb")
if err != nil {
	fmt.Println("err:", err)
	return
}
fmt.Println(a.String())<!-- /go:example -->

<!-- go:example ExampleParseTRX -->
v, err := tron.ParseTRX("1.6")
if err != nil {
	fmt.Println("err:", err)
	return
}
fmt.Println(v.String())<!-- /go:example -->

<!-- go:example ExampleTRX -->
fmt.Println(tron.TRX(1).Int64())<!-- /go:example -->
