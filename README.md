# TronLib

A typed Go SDK for the TRON blockchain. One import for the happy path,
explicit subpackages when you need the full surface.

- **Module:** `github.com/kslamph/tronlib/v2`
- **Go:** 1.27.1 or newer
- **Transport:** gRPC to a TRON node (`grpc://` plaintext, `grpcs://` TLS)

## Install

```bash
go get github.com/kslamph/tronlib/v2@v2.0.0
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

	transfer, err := cli.Account(signer.Address()).TransferTRX(ctx, to, tronlib.TRX(1))
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
| `github.com/kslamph/tronlib/v2` | Facade: dial, chain client, account handles, type aliases. |
| `.../v2/account` | Account-scoped handle: state, staking and delegation, permissions (multi-sig), voting. |
| `.../v2/key` | Signers (private key, mnemonic) and message signing. |
| `.../v2/rpc` | Full 1:1 gRPC wrapper surface. |
| `.../v2/tx` | Transaction builders, signing, broadcast, receipts, cost preview. |
| `.../v2/contract` | ABI-driven contract calls and deploys. |
| `.../v2/token` | TRC-20 token handle with decimal-aware amounts (TRC-10 transfers go through `Client.TransferToken`). |
| `.../v2/event` | Log and event decoding. |
| `.../v2/tron` | Core types: addresses, amounts (SUN), errors. |

## Notes

- `Client` is the chain handle (dial, broadcast, wait, contract and token
  reads). Everything bound to one address lives on
  `cli.Account(owner)`; the handle holds no key and never signs, which is what
  lets a multi-signature signer authorize someone else's account.
- Amounts are integer **SUN**. Use `tronlib.TRX` only for literals and
  constants; dynamic decimal input must go through `tronlib.ParseTRX`.
- Staking amounts are TRX in SUN, never Energy or Bandwidth quantities, and
  `Unstake` starts the chain's cooldown rather than returning TRX —
  `WithdrawUnstaked` claims the matured balance. `ClaimRewards` is voting
  rewards, a different balance again.
- `Client.Broadcast` returns a `Receipt` for node-level rejections —
  `rec.OK()` reports them; they are not Go errors.
- `Client.Wait` reports inclusion; use `WaitForSolid` for custody or
  deposit-crediting semantics.
- Multi-signature transactions can travel between signers:
  `tronlib.Encode` / `tronlib.Decode` carry a partially signed transaction,
  `tronlib.Sign` adds a signature, and `Account(owner).Permissions().SignWeight`
  asks the node whether the collected weight meets the threshold before
  broadcast.
- `tx.With*` options are part of `raw_data`, so they are set **before**
  signing: every option mutator rejects an already-signed transaction with
  `tx.already_signed` rather than producing a transaction the node rejects
  with `SIGERROR`.
- `TotalCostOf` includes the governance fees a transaction triggers
  (multi-signature surcharge, permission-update fee), read live from the
  chain parameters.

## Documentation

- [Error reference](docs/errors.md)
- [Examples](docs/examples.md)
- [Live verification ledger](docs/verification.md)
