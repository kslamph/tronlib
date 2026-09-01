# Phase 2 closeout

## What shipped

Phase 2 delivers the v2 core API: a gRPC-free surface over the TRON chain built from
mechanically verifiable ports of the v1 client, with the four-kind transaction model
(AccountTx, TransferTx, ContractTx, FreezeTx — each with Sign/SignHex/Broadcast via the
shared `tx` pipeline), a `CostPreview` cost estimator (§7.3: energy balance, staked vs
burned TRX, SunPerEnergy pricing snapshot), full error-code/event decoding with a
generated code table (`tron/codes_gen.go`, fixed-point checked by `cmd/docgen`), the
TRC-10/TRC-20 token helpers, the smart-contract ABI layer (decoders, call builders,
CallAtBlock), and the `root facade` (package `tronlib`) that composes everything with
config-driven dial/timeouts and `ChainTip` as the v2.0 network surface. Code-table and
doc-marker drift are enforced in CI by the `cmd/docgen` gates.

Package list and exported-symbol counts (via `go doc -all`):

| Package | Exported symbols |
|---|---|
| tronlib (root facade) | 101 |
| tron | 78 |
| key | 12 |
| event | 27 |
| rpc | 242 |
| tx | 202 |
| contract | 53 |
| token | 28 |
| **Total** | **743** |

## Verification

All checks run at closeout on branch `v2` (Go toolchain, `-count=1`, `-race`):

- `go -C v2 build ./...` — OK.
- `go -C v2 test ./... -count=1 -race` — all 10 packages `ok`
  (root, cmd/docgen, contract, event, internal/compilecheck, key, rpc, token, tron, tx).
- `go -C v2 vet ./...` — clean.
- `gofmt -l v2/` — only the known `cmd/docgen/testdata/brokenpkg/broken.go` fixture.
- Coverage (`-coverprofile`, `go tool cover -func`): total **86.9%** of statements
  (floor ≥ 80%). Per-package: root 93.3%, docgen 76.3%, contract 89.8%, event 91.2%,
  key 89.4%, rpc 89.1%, token 90.5%, tron 95.7%, tx 84.5%.
- docgen gates: `generate-codes` regeneration is byte-identical to
  `v2/tron/codes_gen.go` (fixed-point OK); `sync-docs -check` over
  `docs/errors.md` + `docs/examples.md` passes (no drift).
- CI shape (`.github/workflows/test-coverage.yml`): both modules built and tested
  (v1 `./pkg/...` coverage, v2 `go -C v2 build/test ./...`), and the
  `docgen drift check (v2)` step runs
  `go -C v2 run ./cmd/docgen sync-docs -pkg ./tron -docs ./docs/errors.md -docs ./docs/examples.md -check`.

## Deferred to Phase 2.1 / tag cycle

- **P11 (tag-time): pb dependency story.** `v2/go.mod` resolves the v1 module via
  `replace => ../`, which will not exist for downstream v2 consumers after tagging.
  Before TAGGING v2.0.0: publish v1.9.1+ with the regenerated pb (adds
  `GetPaginatedNowWitnessList`), OR generate pb into `v2/pb`, OR make pb its own
  module — otherwise the v2 module does not compile for consumers.
- **Network()/VerifyNetwork()** — deferred (no verifiable genesis fingerprints offline;
  needs a known-genesis table plus one live run; fabricating hashes would violate the
  verified-data discipline). `ChainTip` is the v2.0 network surface. (spec §10 D2)
- **docgen multi-package support** — `parseCodesData` is hardwired to package `tron`;
  facade Examples are compile-only without docs markers (D3). Phase 2.1: docgen scans
  multiple packages or markers become package-relative.
- **§7.5 live verification (USER-GATED)** — checklist below; needs a funded Nile key.
- **Quality defers carried from task reviews 1–9** (one line each):
  - compilecheck per-fixture rationale comments (why each fixture exists).
  - `ParseTRX`: negative-band table cases not exhaustively enumerated.
  - token `Mul`: negative-n cases not table-tested.
  - `TRX.String`: MinInt64 doc nuance (sign handling wording).
  - `checkParity`: value-uniqueness across test vectors.
  - `sort.SliceStable` noted where ordering of equal keys is observable.
  - `Result.Byte()` accessor missing — uint8 geth-gap (geth decodes uint8 as byte).
  - `Result.BigInt()` doc accuracy (doc claims all intN widths; geth decodes uint8
    as `byte`, not `*big.Int` — discoverable via `decimals()` dead-end).
  - ABI: `BigIntArg` ≥ 2^256 wraps silently (geth parity; guard candidate); five
    constructors can't express `address[]`/`bytes`/`bytesN`/`int256`; array outputs
    lack 0x41 re-prepend inside collections; `Decode` lazily fetches ABI on
    `context.Background`; lazy-fetch `Op` mislabeled "contract.UseABI".
  - tx/rpc: node-reject path drops `ret.GetMessage()` → `Receipt.Revert` empty
    (1-line candidate); `CallAtBlock` reports "unsupported" as `rpc.method_failed`;
    mid-call transport failure maps `rpc.method_failed` not `chain.connection`
    (documented boundary); panic-on-mangled-extension in post-build helpers
    (v1-consistent, documented).
  - Solidity routing test overstated (compile-error-protected); `containsSub`
    hand-rolled; dead assertion block in `TestWaitForSolidUsesSolidityEndpoint`;
    `With*` post-sign doc line ("invalidates any signature").

## §7.5 live-verification checklist (USER-GATED — needs a funded Nile key)

1. Fund a Nile test key (faucet).
2. Build a ContractTx (e.g. TRC-20 transfer via `cli.Token` + `Transfer`) and run
   `cli.CostPreview` — record EnergyNeeded/EnergyAvailable/TronToBurn/SunPerEnergy/PricedAt.
3. Broadcast and wait for the receipt — record `Receipt.Cost` (ActualCost).
4. VERIFY: `CostPreview.TronToBurn` approximates `Receipt.Cost.EnergyFee` (the TIP-491
   penalty factor can move between maintenance periods — floor, not ceiling; §7.3).
5. VERIFY: `Receipt.Cost.EnergyFee == ResourceReceipt.EnergyFee` (the mapping fidelity
   check — spec §7.5 item 2).
6. VERIFY: `EstimateEnergyMessage.EnergyRequired` includes the TIP-491 penalty —
   compare vs `OriginEnergyUsage`/`EnergyPenaltyTotal` in the receipt (spec §7.5 item 1).
7. Record results here or in the issue tracking the tag.

## Known limitations

- `CallAtBlock`: pb `TriggerSmartContract` has no block anchor — classified refusal
  (contract package).
- facade Examples compile-only without docs markers (docgen multi-package limitation, D3).
- `Receipt.NodeCode` carries the raw `api.Return_*`/VM-result enum name; the numeric
  code is recoverable via `rpc.NodeReturnCode`.
