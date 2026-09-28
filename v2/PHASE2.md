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

- **P11: pb dependency story — RESOLVED 2026-09-01.** The v1 module now carries the
  regenerated pb (GreatVoyage-v4.8.2, adds `GetPaginatedNowWitnessList`) as
  v1.3.0 (tagged, pushed to GitHub). `v2/go.mod` requires `github.com/kslamph/tronlib
  v1.3.0` with no local replace: local builds resolve the published v1.3.0 from the
  module proxy, exactly like downstream consumers. The same pb for both v1 and v2.
  Verified: v1 builds and passes all tests against the regenerated pb; v2 builds,
  tests, vet, and docgen gates all green against the proxy-resolved v1.3.0.
- **Network()/VerifyNetwork() — IMPLEMENTED 2026-09-28.** `Network` is explicit
  configuration (`WithNetwork`); `Client.Network()` reports it with no I/O, and
  `Client.VerifyNetwork(ctx)` compares block 0's id against a table of
  live-fetched genesis hashes (Mainnet/Nile/Shasta). The undeclared zero value
  and `Private` skip the read; a declared network absent from the table fails
  closed as `chain.network_mismatch`. `tronlib.DialOption` is now an opaque
  facade struct (built by `WithTimeout`/`WithPool`/`WithNetwork`), not an alias
  to `rpc.DialOption`. §10's "Dial auto-verifies" is NOT implemented: `Dial`
  stays always-lazy, so verification is an explicit call.
- **docgen multi-package support — IMPLEMENTED 2026-09-28.** `sync-docs` takes a
  repeatable `-example-pkg`; example markers are namespaced `<package>.<Example>`
  (`tron.ExampleTRX`, `tronlib.ExampleClient_token`). The facade Examples are now
  documented, not compile-only.
- **Migration guide — IMPLEMENTED 2026-09-28.** `cmd/migrate` generates
  `v2/docs/migration.md` by AST diff (426 v1 symbols: 137 moved, 4 renamed,
  25 removed, 68 candidates, 192 unmapped); CI runs `migrate -check` as a drift gate.
- **§7.5 item 6 (TIP-491 penalty) — VERIFIED 2026-09-28 (mainnet, read-only).**
  The mechanism was established from the java-tron source (clone of
  tronprotocol/java-tron, GreatVoyage-era `develop`): `VM.play`
  (`actuator/.../vm/VM.java`) hoists `factor = energyFactor + 10_000`
  (`DYNAMIC_ENERGY_FACTOR_DECIMAL`) per execution and charges each opcode
  `floor(base*factor/10_000) - base` into `energyPenaltyTotal`
  (`ProgramResult.spendEnergyWithPenalty`); `triggerConstantContract`
  runs the same VM path and reports it as `TransactionExtention.energy_penalty`
  (Wallet.java `builder.setEnergyPenalty(...)`); `estimateEnergy` is a
  binary search for the smallest succeeding fee cap over repeated constant
  calls (`ceil(high/energyFee)`) — conservative by construction, which
  explains the live 1.5× observation and confirms §7.5 item 1 (the penalty
  is included in every iteration). The stored per-contract state is
  readable via `GetContractInfo` → `SmartContractDataWrapper.contract_state
  {energy_usage, energy_factor, update_cycle}`, caught up to the current
  cycle by the node at read time — no new RPC needed. New v2 surface:
  `tx.DynamicEnergy` + `tx.DynamicEnergyOf` + `(*DynamicEnergy).PredictPenalty`
  (+ `HasPenalty`), `(*Estimate).EffectiveFactor`,
  `contract.Instance.DynamicEnergy`; `FactorDecimal = 10_000`.
  Live run (no key, no spend, fixed owner; public gateways rate-limit, so
  the probe retries read-only steps): mainnet USDT
  `TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t`, `balanceOf` constant call —
  `Simulate` returned energy=63999 penalty=49415 base=14584 (reproduced
  exactly twice) and `GetContractInfo` returned factor=34000
  (= getDynamicEnergyMaxFactor, pinned at the ceiling), usage=1154798690,
  cycle=10097. Cross-checks: predicted penalty
  floor(14584*44000/10000)-14584 = 49585 vs actual 49415 (gap 170,
  per-opcode flooring); derived factor floor(10000*63999/14584)-10000 =
  33883 vs stored 34000 (gap 0.34%). Probe exit 0. Nile control run
  (official USDT, incl. a `-random-owner` variant): factor 0, both
  independent reads agree at zero, exit 1 inconclusive — the zero path
  verified end-to-end. Items 2–5 and 7 remain verified by the 2026-09-01
  Nile run (delta 0 SUN).
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

- §7.3 `EnergyPrice.CostOf(energy)` — IMPLEMENTED 2026-09-01: the pure
  energy-burn calculator (energy × SunPerEnergy with checked overflow),
  added as the fix for the §7.5 energy-accuracy finding (see verification
  results below).
- §7.3 price cache (one-maintenance-period TTL) — IMPLEMENTED 2026-09-28:
  `tx.MaintenancePeriod = 6h`; `Client.EnergyPrice` memoises one read per period
  (mutex-guarded; `CostPreview` keeps its own fresh read with `PricedAt`).
- §7.3 `CostPreview.BandwidthNote` — IMPLEMENTED 2026-09-28: the field is
  populated with `tx.BandwidthNotModelled` and `String()` appends it.
- §7.2 `Estimate.HasResult()` — IMPLEMENTED 2026-09-28 (nil-safe).
- §5.4 token `Amount.Formatted()` — IMPLEMENTED 2026-09-28, sharing
  `internal/format.Thousands` with `tron.SUN.Formatted`.
- §10 Dial eager round-trip — INVERTED: v2 `Dial` is always-lazy (v1
  semantics; reachability proven by first call), no `WithLazyDial`.
  Documented in `rpc` doc.go as a feature, but it is a spec deviation.
  **Consequence (2026-09-28):** Dial does NOT auto-verify the network;
  `Client.VerifyNetwork` is explicit.
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

### §7.5 live-verification results (2026-09-01, Nile, chain tip 70,572,241 — CORRECTED 2026-09-01)

Run on branch `v2` via a throwaway `go run` program using the facade
(`cli.Token`/`cli.CostPreview`/`cli.Broadcast`/`cli.Wait`/`cli.Raw`) — see
`v2/cmd/docgen`-adjacent scratch, reproduced below.

**Setup notes (deviations from the checklist text, user-directed):**
- Token: the self-issued v1 TRC-20 `TWRvzd6FQcsyp7hwCtttjZGpU1kfvVEtNK`
  ("TronLib Test", decimals 18 — standard `transfer(address,uint256)`) was used
  instead of official Nile USDT: the Nile faucet was in its 24h USDT cooldown,
  and the 5000 TRX the user sent for the check landed on the USDT contract
  address (`TXLAQ63Xg1NAzckPwKHvzw7CSEmLMEqcdj`), not the fresh key, so it is
  unrecoverable. The fresh key was funded instead from v1 key1
  (`TLibQrqpdqPyg11VBJR97Q4H2714xa9GT1`, 50 TRX + 1000 TLT). Decimals do not
  affect the energy/cost comparison.
- Item 6 (TIP-491 penalty assertion) is SKIPPED per the user: no heavily-consumed
  contract assumed on Nile. The observed penalty is recorded below for
  completeness.
- **Fix applied between runs**: the initial run used the EstimateEnergy RPC
  (20354, conservative 1.5×) as EnergyNeeded; this was corrected to use
  Simulate.Energy (TriggerConstantContract.EnergyUsed = 13569, matching the
  exact execution cost). See the root-cause analysis below the results table.

**Step 2 — CostPreview (§7.3) — CORRECTED:** EnergyNeeded **13569** (EnergyBase
13569, EnergyPenalty 0), EnergyAvailable **0** (fresh key, no staked energy),
EnergyToBuy 13569, TronToBurn **1,356,900 SUN**, SunPerEnergy **100**, PricedAt
2026-09-01T13:20:14+08:00.

**Step 3 — Receipt (§7.4):** txid
`738c6d0e10d2ba325577a38463b93af19209612008fd64fb02ae7f4f7153ac31`, block
70,573,724, Code "", NodeCode SUCCESS. `Receipt.Cost`: EnergyFee **1,356,900
SUN**, NetFee **345,000 SUN** (bandwidth burn, not predicted by energy-only
preview), Total 1,701,900 SUN, Energy 13569 (EnergyUsageTotal), BaseEnergy 0,
Penalty 0, Bandwidth 0 (NetUsage).

**Step 4 — VERIFY TronToBurn vs `Receipt.Cost.EnergyFee`:**
preview **1,356,900 SUN** vs actual **1,356,900 SUN** → delta **0 SUN (0%)**.
Exact match. The estimator is accurate.

**Step 5 — VERIFY TronToBurn vs RAW pb `ResourceReceipt.EnergyFee`:**
raw **1,356,900** — parsed `Cost.EnergyFee` == raw pb exactly (delta 0).
Mapping bug check PASS. Raw EnergyUsageTotal 13569, OriginEnergyUsage 0,
EnergyPenaltyTotal 0, NetFee 345000, NetUsage 0, Result SUCESS.

**Step 7 — bandwidth (preview does not cover it):** NetFee **345,000 SUN**
(345 bandwidth bytes × 1000 SUN/byte, the standard Nile rate — NOT covered by
free bandwidth, unlike the prior run). The energy-only preview predicted
1,356,900 SUN and the actual total (energy + bandwidth) was 1,701,900 SUN; the
345,000 SUN delta is fully attributable to bandwidth — validating the documented
energy-only limitation.

**Step 6 — TIP-491 (SKIPPED per user):** observed preview.EnergyPenalty 0,
receipt.Penalty 0, raw EnergyPenaltyTotal 0 — consistent with a fresh/lightly
used contract (factor 0), so §7.5 item 1 remains unverified; a heavily-consumed
contract (e.g. the official Nile USDT or a long-running DEX) is required.

### §7.5 penalized-contract verification (2026-09-28, mainnet, read-only)

Closes the gap Step 6 left: exactness on a contract carrying a penalty.
The library's exactness contract is Simulate == receipt EXACTLY (no
tolerances) within one maintenance cycle; the client-side PredictPenalty
formula stays a planning upper bound and never second-guesses a
simulation (see `tx.DynamicEnergy.PredictPenalty` docs). New harness:
`cmd/tip491probe -replay <txid>` fetches the broadcast tx + receipt,
re-simulates the identical calldata, and asserts exact equality; a
reverted replay reports "comparison void" (chain state moved) instead
of a numeric mismatch.

- Replay `41808e02d08669098860742b19bba2e16de29da6c065725e6394495d0b3ec2e8`
  (mainnet USDT `transfer`, block 86642154; sender balance re-checked to
  still cover the 900 USDT amount before replaying): receipt
  energy=130285 penalty=100635 base=29650 vs replay energy=130285
  penalty=100635 base=29650 — EXACT match, probe exit 0. §7.5 item 2 now
  verified for penalized contracts: delta 0, the same bar as the
  2026-09-01 penalty-free run.
- Replay `646e6d4494a9d9d072b50f62c5a352bca4a0ef45df89fd35271d57f64eb5df3f`
  (same block; sender had since moved the funds, balance 9000 < costs):
  the replay reverted (`REVERT opcode executed`, energy=8624 penalty=6640)
  and the harness reported the comparison void rather than a mismatch —
  the stale-replay guard verified live.
- Factor-mode cross-checks were tightened to exact one-sided bounds
  (derived > stored and actual > predicted are contradictions, never
  noise); the loose flooring sides are reported, not judged.

### Bandwidth cost model (2026-09-28, docs + java-tron source + live)

Bandwidth is pure size accounting — no simulation. Established from the
 official resource-docs (burn 1,000 sun/byte via `getTransactionFee`, free
 600/day via `getFreeNetLimit`, staked → free → burn order, 1 TRX creation
 fee + 0.1 TRX bandwidth-shortfall creation fee + 25,000 energy for
 contract-internal creation) and confirmed in java-tron
 `BandwidthProcessor.consume`: bytes = serializedSize(tx with ret cleared)
 + 64 (`MAX_RESULT_SIZE_IN_TX`) per non-shielded contract; burn reports
 NetUsage 0 with NetFee = bytes × price; covered reports NetUsage = bytes.
 Chain params verified identical on mainnet and Nile (1000 / 100000 /
 1000000 / rate 1 / free 600). New v2 surface (`v2/tx/bandwidth.go`):
 `ResultSizePerContract`, `BandwidthSize` (signed only — unsigned is
 tx.invalid_argument, not a silent undercount), `BandwidthPrice` +
 `BandwidthPriceOf` + `CostOf`, `BandwidthCost` + `BandwidthCostOf` with
 the creation branch (recipient-existence read, ratio-scaled stake else
 flat 0.1 TRX, 1 TRX on top invisible to the receipt) and a
 `account.insufficient_bandwidth` (ActionFund) balance gate. `CostPreview`
 is untouched (still energy-only with its BandwidthNote).
- Exact live verification, zero spend, via `-replay` (bandwidth needs no
 resource state — burn and covered shapes both reduce to exact byte
 equalities): Nile `738c6d0e` (the 2026-09-01 transfer) replays
 bytes=345, receipt usage=0 fee=345000 — burn matches exactly (and energy
 replays 13569/0 exactly even weeks later); mainnet `41808e02` replays
 bytes=345, receipt usage=345 fee=0 — covered matches exactly. Both 345 =
 ~281 serialized + 64, confirming the overhead empirically.
- LIVE-VERIFIED 2026-09-28 (Nile, user-funded 52.3869 TRX to fresh key
  `TFajiYgytkFiBpustBNQSFEDsiHBoAFrBo`; throwaway runner, deleted after):
  - Transfer 1 — 10 TRX to fresh `TCvU2ENS6sksbhcz2x7jX87yrrCWZLt7Rf`:
    predicted CreatesAccount, Burn=100,000, NewAccountFee=1,000,000
    (need 274); broadcast `4abe41a6…` → receipt NetUsage=0,
    NetFee=100,000 EXACT; balance 52.3869 − 10 − 0.1 − 41.2869 =
    1.000000 TRX drift EXACT — the invisible creation burn proven by
    full accounting.
  - Transfer 2 — 1 TRX to the now-existing address: predicted
    free-covered (need 273); broadcast `ef1660a5…` → receipt
    NetUsage=273, NetFee=0 EXACT; balance drift 0.
  - The insufficient_bandwidth rejection stays hermetic-only (needs
    precise draining; not worth testnet choreography).

### Deploy estimation (2026-09-28, research + live)

`triggerconstantcontract` with init bytecode as `data` and an EMPTY
contract address synthesizes a `CreateSmartContract` server-side
(java-tron Wallet:3113, percent 100) and returns the COMPLETE deploy
energy — init exec + 200/byte deposit — in one call. Verified
differentially on Nile (all free): 0B→15, 1B→221, 32B→6421
(6200/31 = 200.0 exact), 1000B→200209; broadcast receipt 221 ==
constant 221. No 32,000 (internal-CREATE-opcode only). New surface:
`tx.DeployEstimate` + `(*DeployTx).Estimate` (owner/bytecode/callvalue
reused from the built tx; rejection in-band like Simulate), verified
live through the real path (`cli.Deploy` → `Estimate`): 221 and
200209 exact, exit 0. This supersedes spec §6.1's "no simulation
path" premise for the constant-call shape (EstimateEnergy RPC still
has none; Simulate stays ContractTx-only).

### Deploy broadcast (2026-09-28, Nile, user-funded key)

First live DeployTx broadcast (`e40e069a…`, block 71359418): minimal
1-byte-STOP init (`6001600c60003960016000f300`), name tronlib-probe,
percent 100, origin 1M, fee_limit 150 TRX cap. Deployed contract
`TGHsNCnqw9fnwB86NcKU36xdoT9BPQUPqV`, runtimecode `00` (1 byte) via
GetContractInfo. Receipt: code "", NodeCode SUCCESS, energy=221,
penalty=0, bandwidth=312, NetFee=0, EnergyFee=2,210. Bandwidth predicted
312/free-covered exactly. Balance drift 0. Total spend: 2,210 SUN
(~0.002 TRX) against a ~3.25 TRX worst-case estimate.
- The estimate was wrong in the safe direction: top-level deploys do NOT
  pay the 32,000 internal-CREATE opcode price (that is contract-creating-
  contract only) — init exec 221 is the whole energy charge, matching the
  constant-call figure exactly (Simulate==receipt holds for deploys too).
- Open source-reading puzzle (not a verification gap — every billed,
  reported and accounted number is exactly consistent): the 200 code-
  deposit (`saveCodeEnergy = 1 byte × 200`) is nowhere in energy_used,
  fee or drift, yet the code persisted. Likely explanation: in
  VMActuator.execute the deposit is spent on the program's result object
  while billing reads the context's result, with the code persisted by a
  direct store write — i.e. top-level deploys effectively skip the
  deposit charge. 200 SUN either way.

#### Root-cause analysis: the energy-accuracy fix

The initial preview returned EnergyNeeded **20354** (1,356,900 SUN burn, delta
+50% vs actual). The `EstimateEnergy` RPC (`EnergyRequired`) is the node's
conservative fee-limit calculator — it returned 20354 for a call whose actual
VM execution cost was 13569 (a 1.5× safety margin, so a fee limit set from it
never runs out of energy). `PreviewCost` (`v2/tx/cost.go`) was using this
conservative number as the authoritative `EnergyNeeded`, even though
`Simulate.Energy` (from `TriggerConstantContract.EnergyUsed`) was already being
computed and matched the actual cost exactly.

**Fix (three related changes, implemented together 2026-09-01):**

1. `tx/cost.go` (`PreviewCost`): use `sim.Energy` (accurate Simulate/TriggerConstantContract EnergyUsed) as `EnergyNeeded`. Remove the `t.EstimateEnergy(ctx)` RPC call (the conservative EstimateEnergy source).

2. `tx/energy.go` (`EnergyPrice.CostOf`): added the "energy burn calculator" primitive — `energy × SunPerEnergy` with overflow check, independent of any contract or transaction, driven purely by the network's current operating parameters. `PreviewCost` calls it via `EnergyPriceOf` to compute `TronToBurn`.

3. `tx/estimate.go` docs: updated to reflect the live finding — `EstimateEnergy` RPC is a conservative fee-limit calculator (1.5× over the actual execution cost), NOT the accurate cost predictor. `Simulate.Energy` is the accurate estimator, and `CostPreview` uses it.

**Result:** EnergyNeeded is now **13569** (= actual EnergyUsageTotal), and
TronToBurn matches EnergyFee **exactly** (delta 0). The estimator is accurate,
and the energy→SUN burn calculator (`EnergyPrice.CostOf`) is the separate,
network-parameter-driven concern the design intended.

## Known limitations

- `CallAtBlock`: pb `TriggerSmartContract` has no block anchor — classified refusal
  (contract package).
- `Receipt.NodeCode` carries the raw `api.Return_*`/VM-result enum name; the numeric
  code is recoverable via `rpc.NodeReturnCode`.
