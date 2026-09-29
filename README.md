# TronLib

A typed Go SDK for the TRON blockchain. One import for the happy path,
explicit subpackages when you need the full surface.

- **Module:** `github.com/kslamph/tronlib/v2`
- **Go:** 1.27.1 or newer
- **Transport:** gRPC to a TRON node (`grpc://` plaintext, `grpcs://` TLS)

## Install

v2 is pre-release — no `v2.x` tag exists yet. Go requires a `/v2` module path
to be served by a `v2.0.0` or later tag, so an un-pinned `go get` cannot
resolve until release; pin a commit in the meantime.

```bash
go get github.com/kslamph/tronlib/v2@<commit-sha>  # interim: replace with a full commit SHA
go get github.com/kslamph/tronlib/v2@v2.0.0        # valid once v2.0.0 is tagged
```

## Quickstart

```go
package main

import (
	"context"
	"fmt"

	tronlib "github.com/kslamph/tronlib/v2"
)

func main() {
	ctx := context.Background()

	cli, err := tronlib.Dial(ctx, "grpc://grpc.nile.trongrid.io:50051")
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	signer, err := tronlib.KeyFromHex("<private-key-hex>")
	if err != nil {
		panic(err)
	}
	to := tronlib.MustAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")

	transfer, err := cli.TransferTRX(ctx, signer.Address(), to, tronlib.TRX(1))
	if err != nil {
		panic(err)
	}
	signed, err := transfer.Sign(signer)
	if err != nil {
		panic(err)
	}

	rec, err := cli.Broadcast(ctx, signed)
	if err != nil {
		panic(err)
	}
	if !rec.OK() {
		panic("node rejected: " + rec.NodeCode)
	}
	// Inclusion is not finality; WaitForSolid waits for solidification.
	if _, err := cli.WaitForSolid(ctx, rec.TxID); err != nil {
		panic(err)
	}
	fmt.Println("confirmed", rec.TxID)
}
```

`Dial` is lazy — it performs no network I/O, so reachability is proven by
the first call. If you declared a network with `WithNetwork`, call
`Client.VerifyNetwork` before broadcasting.

## Packages

| Package | Purpose |
| --- | --- |
| `github.com/kslamph/tronlib/v2` | Facade: dial, happy-path client, type aliases. |
| `.../v2/key` | Signers (private key, mnemonic) and message signing. |
| `.../v2/rpc` | Full 1:1 gRPC wrapper surface. |
| `.../v2/tx` | Transaction builders, signing, broadcast, receipts, cost preview. |
| `.../v2/contract` | ABI-driven contract calls and deploys. |
| `.../v2/token` | TRC-20 token handle with decimal-aware amounts (TRC-10 transfers go through `Client.TransferToken`). |
| `.../v2/event` | Log and event decoding. |
| `.../v2/tron` | Core types: addresses, amounts (SUN), errors. |

## Notes

- Amounts are integer **SUN**. Use `tronlib.TRX` only for literals and
  constants; dynamic decimal input must go through `tronlib.ParseTRX`.
- `Client.Broadcast` returns a `Receipt` for node-level rejections —
  `rec.OK()` reports them; they are not Go errors.
- `Client.Wait` reports inclusion; use `WaitForSolid` for custody or
  deposit-crediting semantics.

## Documentation

- [Error reference](docs/errors.md)
- [Examples](docs/examples.md)
- [Live verification ledger](docs/verification.md)
