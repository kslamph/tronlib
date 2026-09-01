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

## Spec deltas (§ references vs shipped)

Every item below is a spec deviation shipped in Phase 2, recorded so Phase 2.1
planning does not assume they exist.

- §7.3 `EnergyPrice.CostOf(energy)` — the pure batching primitive — NOT
  implemented (`EnergyPriceOf` exists; `CostOf` deferred).
- §7.3 price cache (one-maintenance-period TTL) — NOT implemented;
  `EnergyPriceOf` refetches every call.
- §7.3 `CostPreview.BandwidthNote` — NOT implemented; preview covers energy
  only (documented limitation).
- §7.2 `Estimate.HasResult()` — NOT implemented; use
  `len(ConstantResult) > 0`.
- §5.4 token `Amount.Formatted()` — NOT implemented (`tron.SUN` has both
  `String` and `Formatted`; token has `String` only).
- §10 Dial eager round-trip — INVERTED: v2 `Dial` is always-lazy (v1
  semantics; reachability proven by first call), no `WithLazyDial`.
  Documented in `rpc` doc.go as a feature, but it is a spec deviation.
- §5.4 `Handle.Whole` is `Whole(n int64) (Amount, error)` — the
  generic-method signature requires go 1.27+; concrete int64 preserves the
  compile-time float/SUN rejection (controller ruling D1, Task 8).

## §7.5 live-verification checklist (USER-GATED — needs a funded Nile key)

PRECONDITIONS: Use a funded Nile key with NO staked energy
(EnergyAvailable < EnergyNeeded, so TronToBurn > 0 — otherwise the comparison
is trivially zero). Use a heavily-consumed Nile contract for the TIP-491
penalty check (a fresh contract has factor 0).

1. Fund a Nile test key (faucet).
2. Build a ContractTx (e.g. TRC-20 transfer via `cli.Token` + `Transfer`) and run
   `cli.CostPreview` — record EnergyNeeded/EnergyAvailable/TronToBurn/SunPerEnergy/PricedAt.
3. Broadcast and wait for the receipt — record `Receipt.Cost` (ActualCost).
4. VERIFY: Compare `CostPreview.TronToBurn` against `Receipt.Cost.EnergyFee`
   within a stated tolerance (the TIP-491 penalty factor can move between
   maintenance periods — floor, not ceiling; §7.3). Record both plus the delta.
5. VERIFY: Compare `CostPreview.TronToBurn` against the RAW node answer: after
   broadcast, fetch GetTransactionInfoById yourself (`cli.Raw()` escape hatch)
   and read `ResourceReceipt.EnergyFee` from the pb — NOT the library's parsed
   `Cost.EnergyFee`. This is the check that catches a mapping bug.
6. VERIFY: First assert `Receipt.Cost.Penalty > 0`. If 0, record
   "inconclusive — no TIP-491 factor on this contract/period" (a fresh Nile
   contract will hit this). Only Penalty > 0 verifies §7.5 item 1
   (`EstimateEnergy` includes the penalty): compare
   `EstimateEnergy.EnergyRequired` against `OriginEnergyUsage` +
   `EnergyPenaltyTotal`.
7. Record NetFee and Bandwidth alongside — the preview does not cover
   bandwidth (documented limitation); the delta's bandwidth component must be
   observed to validate that limitation.
8. Record results here or in the issue tracking the tag.

## Known limitations

- `CallAtBlock`: pb `TriggerSmartContract` has no block anchor — classified refusal
  (contract package).
- facade Examples compile-only without docs markers (docgen multi-package limitation, D3).
- `Receipt.NodeCode` carries the raw `api.Return_*`/VM-result enum name; the numeric
  code is recoverable via `rpc.NodeReturnCode`.
