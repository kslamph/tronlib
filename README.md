# TronLib

[![Go Reference](https://pkg.go.dev/badge/github.com/kslamph/tronlib/v2.svg)](https://pkg.go.dev/github.com/kslamph/tronlib/v2)
[![codecov](https://codecov.io/gh/kslamph/tronlib/branch/v2/graph/badge.svg?token=QIN77Y7S2T)](https://codecov.io/gh/kslamph/tronlib/branch/v2)
[![tests](https://github.com/kslamph/tronlib/actions/workflows/tests.yaml/badge.svg?branch=v2)](https://github.com/kslamph/tronlib/actions/workflows/tests.yaml)
[![checks](https://github.com/kslamph/tronlib/actions/workflows/checks.yaml/badge.svg?branch=v2)](https://github.com/kslamph/tronlib/actions/workflows/checks.yaml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)

`tests` = build, `go test -short`, the 80% coverage floor and the Codecov
upload ([`tests.yaml`](.github/workflows/tests.yaml)). `checks` =
`golangci-lint` v2.14.0, the docgen drift check and `govulncheck`
([`checks.yaml`](.github/workflows/checks.yaml)) — CI-only, and the lint
config that defines it is [`.golangci.yml`](.golangci.yml).

A typed Go SDK for the TRON blockchain. One import for the happy path,
explicit subpackages when you need the full surface. Human developers and
AI coding agents are equal first-class users — see
[Two audiences](#two-audiences).

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

	// Amounts are integer SUN: 1 TRX = 1_000_000 SUN. Wrap literals in
	// tronlib.TRX — a bare 1 here would mean 1 SUN, not 1 TRX.
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
| `.../v2/token` | TRC-20 token handle with decimal-aware amounts — `Client.Token(ctx, addr)`. TRC-10 legacy assets go through `Client.Account(owner).TransferTRC10`. |
| `.../v2/event` | Log and event decoding. |
| `.../v2/tron` | Core types: addresses, amounts (SUN), errors. |

## Notes

- `Client` is the chain handle (dial, broadcast, wait, contract and token
  reads). Everything bound to one address lives on
  `cli.Account(owner)`; the handle holds no key and never signs, which is what
  lets a multi-signature signer authorize someone else's account. The token
  handle is the one deliberate exception: it is bound to the token *contract*,
  not an owner, so it hangs off `Client.Token` and its `Transfer`/`Approve`
  take the sending account as an explicit first argument.
- Amounts are integer **SUN** (1 TRX = 1_000_000 SUN). Use `tronlib.TRX`
  for literals and constants; dynamic decimal input must go through
  `tronlib.ParseTRX`. A bare integer where a `SUN` is expected compiles and
  means SUN — `TransferTRX(ctx, to, 10)` sends 0.00001 TRX.
- Staking amounts are TRX in SUN, never Energy or Bandwidth quantities, and
  `Unstake` starts the chain's cooldown rather than returning TRX —
  `WithdrawUnstaked` claims the matured balance. `ClaimRewards` is voting
  rewards, a different balance again.
- `Client.Broadcast` returns a `Receipt` for node-level rejections —
  `rec.OK()` reports them; they are not Go errors. Check `rec.OK()` before
  waiting: a rejected transaction never lands, so `Wait` on it polls until
  the context deadline.
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
  chain parameters. `CostPreview` (pre-sign) prices energy **and** bandwidth
  — `TotalFloor` is the all-in floor on a single-signature estimate of the
  broadcast bytes; sign and call `TotalCost` for the all-in answer.

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
harness behind them, `cmd/examplecheck`, walks every documented flow against
a real node — reads, builds, local signing, simulation, cost pricing, the
multi-signature envelope round trip and the node's signature-weight verdict —
and spends nothing unless asked to:

```bash
go run ./cmd/examplecheck   # every flow, live node, zero spend
```

Recorded runs live in the [verification ledger](docs/verification.md): R9
(spend-free, 34 steps OK), R10 (on-chain, 55 steps OK / 0 failed) with every
txid — along with the flags for a broadcasting run, the run-to-run cost
measurements, the example bug only execution could find (a reverting
simulation returns revert data, not the method's return value), and what is
deliberately not yet proven.

## Two audiences

Human developers and AI coding agents are equal first-class users, and the
documentation serves each natively:

- **Errors are machine-readable remediation.** Every failure is a
  `*tron.Error` with a stable `Code`, a `Hint` that names the fix, and a
  `Next` action (`retry`, `wait`, `fix_call`, `fix_transaction`, `fund`).
  The [error table](docs/errors.md) is generated from the source by
  `cmd/docgen`, so it cannot drift from the code it documents.
- **Examples are compiled and were executed.** Every example in
  [docs/examples.md](docs/examples.md) is a Go example compiled by `go test`,
  and the [verification ledger](docs/verification.md) records them running
  against a live node, with txids.
- **[`llms.txt`](llms.txt)** is the curated reading order — the four
  documents an agent (or a human in a hurry) needs, plus the API's few
  non-negotiable rules restated where they cannot be missed.

## Documentation

- [`llms.txt`](llms.txt) — the curated reading order, for AI agents and fast readers
- [Error reference](docs/errors.md)
- [Examples](docs/examples.md)
- [Architecture](docs/architecture.md) — the design, described and explained
- [Review records & deferred work](docs/review.md) — review dispositions, TODO list
- [Live verification ledger](docs/verification.md)
- [Runbook](docs/runbook.md) — maintainer operations: live verification, docgen, release gates
