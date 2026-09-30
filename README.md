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

## Examples

The examples live in [`example_test.go`](example_test.go) and are rendered into
[compiled examples](docs/examples.md). Each one is a whole task rather than a
single API call — a contract call is simulated, priced and its logs decoded in
the same example — and `go test` compiles every one, so they cannot drift from
the code.

| Example | Covers |
| --- | --- |
| `Example` | Quickstart: dial, read the balance, transfer TRX, price it, sign, broadcast, wait for solidification. Hex key, one signer. |
| `ExampleClient_Account` | Account and resource state: balance, Energy/Bandwidth, TRON Power, stake/unstake/delegation summary, and an all-in cost preview before spending. |
| `ExampleClient_Contract` | One contract interaction end to end: view call, `Invoke`, `Simulate`, `CostPreview`, broadcast, and event decoding from the receipt. |
| `ExampleClient_Token` | TRC-20 through the decimal-aware handle: symbol/name/decimals/totalSupply, balance and allowance reads, `Approve`, `transferFrom`, `Transfer`. Mnemonic key. |
| `ExampleResources_Stake` | Staking lifecycle and delegation: stake, unstake into the cooldown, harvest, delegate with a block-based lock, track the delegation index. |
| `ExamplePermissions_Current` | Permission configuration: read the whole set, add an operations-bitmap-scoped active permission, submit the complete replacement and its fee. |
| `ExamplePermissions_SignWeight` | Offline multi-signature: portable envelope between two machines and two key forms, node-verified sign weight, plus the `SignHash`/`AttachSignature` path for a remote signer. |

They carry no `// Output:` comment, so they compile without contacting a node
(CODING_STANDARDS.md §6.4). To run one against Nile, copy its body into a
`main` and set the environment it reads: `TRON_PRIVATE_KEY` (hex, 64
characters), `TRON_MNEMONIC`, `TRON_SIGNER_A_KEY`, `TRON_SIGNER_B_MNEMONIC`.

### Checking them against a live node

Compilation proves the examples type-check, not that they are correct. The
harness walks every flow above against a real node — reads, builds, local
signing, simulation, cost pricing, the portable-envelope round trip and the
node's signature-weight verdict — and spends nothing unless you ask it to:

```bash
go run ./cmd/examplecheck                          # spend-free; generates a fresh signer
go run ./cmd/examplecheck -key <hex> -broadcast    # full run, spends TRX
```

The last recorded run on Nile (34 steps OK, 4 notes, 0 failed) is written up in
the [verification ledger](docs/verification.md#r9--every-documented-example-flow-live-checked-nile-2026-09-30),
including the example bug that only execution could find: a reverting
simulation returns revert data, not the method's return value.

## Documentation

- [Error reference](docs/errors.md)
- [Examples](docs/examples.md)
- [Live verification ledger](docs/verification.md)
