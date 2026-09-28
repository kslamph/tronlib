# tronlib v2 — Release Hardening Design

**Status:** approved for planning (brainstorming complete 2026-09-28)
**Supersedes:** nothing. Amends §7.3, §7.5, §10 and §14 of
`2026-08-31-tronlib-v2-design.md` where stated.

This spec covers the work that stands between the shipped Phase 2 core API and
the `v2.0.0` tag: the two §14 tag-cycle gates that are implementable now (**B**),
plus the Phase 2.1 spec deltas recorded in `v2/PHASE2.md` (**C**). It does not
re-open the Phase 2 design; every decision below is either a deferred item from
the original spec or a finding made while closing it out.

---

## 1. Classification and scope

The work was classified **architectural**: B5 introduces network identity where
no flow exists to modify, and B3 introduces a new generator command. C is a set
of bounded changes to existing code and is bundled here only so planning sees
the whole picture.

| Item | Origin | Decision |
|---|---|---|
| **B2** §7.5 item 6 (TIP-491 penalty) | spec §7.5 | harness only — see §3 |
| **B3** v1→v2 migration guide | spec §14 step 12 | new `v2/cmd/migrate`, generated doc, CI gate |
| **B4** tag `v2.0.0` | spec §14 step 13 | **out of scope** — user-gated, follows B2 |
| **B5** `Network()` / `VerifyNetwork()` | spec §10 + R7 | option A: explicit verification, lazy `Dial` kept |
| **C1** §7.3 energy-price cache | spec §7.3 / G5 | facade-owned TTL cache |
| **C2** §7.3 `CostPreview.BandwidthNote` | spec §7.3 / P2.6 | explicit field |
| **C3** §7.2 `Estimate.HasResult()` | spec §7.2 | one-line method |
| **C4** §5.4 token `Amount.Formatted()` | spec §5.4 | display method |
| **C5** docgen multi-package support | `PHASE2.md` D3 | generalize the `package tron` gate |

**Explicitly not in scope** (chosen away during brainstorming, not forgotten):
the §11 surface-budget ratchet (`api/v2.txt`), §12 filling of `go:errors` in
`docs/API_REFERENCE.md`, and everything in spec §13 (shielded, TRC-10 issuance,
CLI, `schema.json`).

---

## 2. Evidence gathered before design

All facts below were fetched live on 2026-09-28 and are reproducible; none are
inferred.

### 2.1 Genesis fingerprints (unblocks B5)

`POST /wallet/getblockbynum {"num":0}` on each public endpoint:

| Network | Endpoint | Genesis blockID |
|---|---|---|
| Mainnet | `api.trongrid.io` | `00000000000000001ebf88508a03865c71d452e25f4d51194196a1d22b6653dc` |
| Nile | `nile.trongrid.io` | `0000000000000000d698d4192c56cb6be724a558448e2684802de4d6cd8690dc` |
| Shasta | `api.shasta.trongrid.io` | `0000000000000000de1aa88295e1fcf982742f773e0419c5a9c134c994a9059e` |

`PHASE2.md` deferred B5 on the grounds that no verifiable fingerprints existed
offline. They are verifiable online, so the deferral reason is discharged. These
three constants are the table; they are recorded, not derived.

### 2.2 §7.5 item 6 is unreachable on Nile (B2 outcome)

`POST /wallet/getchainparameters` on Nile:

```
getAllowDynamicEnergy            1
getDynamicEnergyThreshold        5000000000   (5e9 energy)
getDynamicEnergyIncreaseFactor   2000
getDynamicEnergyMaxFactor        34000        (max 3.4x)
getEnergyFee                     100
```

TIP-491 raises a contract's instruction cost only when that contract consumes
more than `getDynamicEnergyThreshold` energy in one maintenance period. At 100
SUN/energy, 5e9 energy is ~500,000 TRX burned by a single contract in six
hours. Nile does not carry that traffic: a constant `balanceOf(address)` call
against the busiest known Nile contract, official USDT
`TXLAQ63Xg1NAzckPwKHvzw7CSEmLMEqcdj`, returned `energy_used: 651` and **no
penalty field**.

Conclusion: item 6 is gated by network configuration, not by the §7.5 checklist's
"funded key" precondition. Funding the test keys cannot fix it. It requires a
Mainnet contract known to carry a factor (or a private chain that lowers the
threshold). B2 therefore ships a **harness that can run on either**, plus this
recorded negative result — it does not run against Nile.

### 2.3 Test-key resources (for the record)

| Account | Balance | Staked energy | Note |
|---|---|---|---|
| `TLibQrqpdqPyg11VBJR97Q4H2714xa9GT1` | 55.49 TRX | 163 TRX frozen → 12,006 energy | acquired stake since the 2026-09-01 run |
| `TLibCZ2i2dFp6a9KZeKriSms5peeXSibks` | 96.73 TRX | none | clean account (ideal §7.5 shape) |

Both are sufficient for a cost-accuracy run (already verified delta 0 on
2026-09-01); neither helps item 6. No spend is proposed by this spec.

---

## 3. B2 — §7.5 item 6 verification harness (harness only)

**Deliverable:** `v2/cmd/tip491probe`, a small program that exercises the real
exported API — it does not reimplement anything.

```
go -C v2 run ./cmd/tip491probe -endpoint grpc://... -contract <addr> [-key <hex>]
```

Behavior:

1. Read the endpoint, contract address, and an optional private key from flags
   / environment (never checked in).
2. `Simulate` a call against the target contract and print
   `Estimate.Energy` and `Estimate.Penalty` (`TransactionExtention.EnergyPenalty`).
3. If a key is supplied, also `EstimateEnergy` and print `Base`/`Penalty`/`Energy`.
4. **Assert** `Estimate.Penalty > 0`; on zero, print the §2.2 explanation
   (threshold not reached / contract factor 0) and exit non-zero, so a
   "inconclusive" run cannot be mistaken for a pass.
5. Print `rpc.GetChainParams`-derived dynamic-energy parameters when available,
   so the run is self-documenting.

**Not run by this spec.** No broadcast, no spend, no funded key required. The
recorded Nile result in §2.2 and in `PHASE2.md` is the item-6 outcome for v2.0:
*inconclusive on Nile — network-parameter bound*.

---

## 4. B5 — network identity (`Network`, `WithNetwork`, `VerifyNetwork`)

### 4.1 Decision

**Option A — explicit verification, lazy `Dial` kept.** Phase 2 deliberately
made `Dial` always-lazy and recorded §10's eager round-trip as a documented
inversion. Reversing that would change `Dial`'s behavior for every caller and
re-introduce a network round trip on a constructor. Instead:

- `WithNetwork` declares intent;
- `Client.Network()` reports the declaration (no I/O);
- `Client.VerifyNetwork(ctx)` performs the fingerprint read on demand;
- `Dial` does **not** auto-verify. §10's "Dial calls it automatically" remains
  unimplemented, consistent with the always-lazy deviation already recorded in
  `PHASE2.md`. This is added to the deviation list, not silently dropped.

### 4.2 Surface

```go
package tronlib // v2/network.go

type Network string

const (
    Mainnet Network = "mainnet"
    Shasta  Network = "shasta"
    Nile    Network = "nile"
    Private Network = "private"
)

func WithNetwork(n Network) DialOption
func (c *Client) Network() Network                  // configured value, no I/O
func (c *Client) VerifyNetwork(ctx context.Context) error
```

`tronlib.DialOption` stops being `= rpc.DialOption`. It becomes an opaque struct
in the facade with constructors `WithTimeout`, `WithPool`, `WithNetwork`; `Dial`
unpacks the rpc options and stores the network on `Client`. Spec §10 does not
list `DialOption` as an alias, so nothing in the published surface changes except
that a caller can no longer pass an `rpc.DialOption` where a `tronlib.DialOption`
is expected — which was never a documented affordance.

### 4.3 Semantics

- **Zero value `Network("")` means undeclared.** `Network()` returns it;
  `VerifyNetwork` treats it like `Private` and returns `nil` — there is no
  declared expectation to contradict.
- `Private` is a declaration that the endpoint has its own genesis. It matches
  no table entry by design (§10, R7); `VerifyNetwork` returns `nil`.
- `Mainnet`/`Shasta`/`Nile`: read block 0 and compare.

```go
genesisID := map[Network]string{
    Mainnet: "00000000000000001ebf88508a03865c71d452e25f4d51194196a1d22b6653dc",
    Nile:    "0000000000000000d698d4192c56cb6be724a558448e2684802de4d6cd8690dc",
    Shasta:  "0000000000000000de1aa88295e1fcf982742f773e0419c5a9c134c994a9059e",
}
```

`VerifyNetwork` calls `rpc.GetBlockByNum2(cp, ctx, &api.NumberMessage{Num: 0})`
and hex-encodes `BlockExtention.Blockid`. On disagreement it returns a
`*tron.Error` with `CodeChainNetworkMismatch` (`chain.network_mismatch`, already
in the code table), `Op: "tronlib.Client.VerifyNetwork"`, and a `Hint` naming
both the configured network and the observed hash. On a read failure it returns
the underlying `*tron.Error` unchanged.

The docs must state plainly that this is **heuristic** (a redeployed testnet
changes its genesis; a private chain matches nothing) and that `WithNetwork` is
the source of truth — never an inference from the `0x41` address prefix, which
is identical on Mainnet, Shasta, and Nile.

### 4.4 Testing

Hermetic only. The existing facade bufconn fake is extended to answer
`GetBlockByNum2`: one test per matching network, one mismatch (returns
`chain.network_mismatch` with both hashes in the hint), `Private` and undeclared
both return `nil` without contacting the node, and `Network()` round-trips the
`WithNetwork` value. No live test.

---

## 5. B3 — generated v1→v2 migration guide

### 5.1 Deliverable

```
v2/cmd/migrate            # the generator
v2/docs/migration.md      # generated output, committed, between markers
```

`v2/docs/migration.md` carries prose plus a generated block:

```
<!-- go:migration -->
...generated tables...
<!-- /go:migration -->
```

`migrate` fills the block from source; `migrate -check` re-renders and byte-
compares, exactly like `docgen sync-docs -check`. Workshop default paths:
`-v1 ..` (the v1 module is the worktree root) and `-v2 .`.

Spec §14 step 12 requires the guide to be generated "by AST diff" with **no
hand-written table**. The generated table satisfies that; the curated inputs in
§5.3 are reviewed Go source, not the rendered guide.

### 5.2 Symbols walked

- **v1:** exported top-level funcs, types, methods, and consts in the root
  module's `pkg/...` (the managers under `pkg/account`, `pkg/resources`,
  `pkg/voting`, `pkg/network`, `pkg/trc20`, `pkg/trc10`, `pkg/eventdecoder`,
  `pkg/signer`, `pkg/utils`, `pkg/types`, `pkg/client/lowlevel`).
- **v2:** the same node kinds in `v2`'s packages (`tron`, `key`, `event`, `rpc`,
  `tx`, `contract`, `token`, and the root facade `tronlib`).

Both are parsed from source with `go/parser` + `go/ast`; no import of either
module is needed, so the DAG is untouched.

### 5.3 Classification algorithm

For every v1 symbol, in this order:

1. **Removed** — present in `removals` (curated): C3/C4 exclusions (all
   shielded functions, TRC-10 issuance, CLI, deprecated shims). Rendered with
   the spec clause that removes them.
2. **Renamed / relocated** — present in `renames` (curated): the known
   mapping, e.g. `account.Manager.Transfer` → `tronlib.Client.TransferTRX`,
   `utils.*` dissolution → `tron`/`tx`. Rendered with the target and a one-line
   reason.
3. **Moved** — same symbol shape under a different package: func→func or
   method→method with an identical receiver type name (the DAG relocation of
   `rpc` wrappers, `types`→`tron`). Rendered as v1 package → v2 package.
4. **Port matched (candidate)** — name matches but the shape changed:
   method→free function or free function→method (v1 `*Manager.Transfer` → v2
   `rpc.Transfer`/facade method). Rendered with a "verify" marker. A name that
   matches only across *different receivers* is still a candidate, never a
   matched row.
5. **Unmapped** — everything else, rendered in an explicit section. Unmapped is
   an honest state: it means "no mechanical evidence of a v2 counterpart", not
   "deleted". A non-empty Unmapped section is expected on first run and is the
   input to growing `renames`.

A summary line (`N symbols: M matched, R removed, U unmapped`) is generated too.
The generator must never invent a mapping; §5.4's tests pin that.

### 5.4 Testing

- Golden test: a fixture pair of small v1/v2 trees → exact expected Markdown.
- `-check` test: mutate the committed `v2/docs/migration.md` block, expect exit
  non-zero; restore, expect zero.
- Marker-preservation test: prose outside the markers is untouched.
- A "no fabricated mapping" test: an unmapped symbol lands in Unmapped, never in
  a matched table.

### 5.5 CI

`v2/docs/migration.md` is regenerated and byte-compared in
`.github/workflows/test-coverage.yml` next to the existing docgen drift step:

```
go -C v2 run ./cmd/migrate -check
```

Drift fails the build like `gofmt -l`.

---

## 6. C — Phase 2.1 spec deltas (bounded)

Each is a small change to existing flow, implemented test-first. No plan
document of its own.

### 6.1 C1 — §7.3 energy-price cache

Spec §7.3's rule: TTL-only, one maintenance period, never keyed on head block.
`tx.EnergyPriceOf` stays the fresh read (its callers keep that meaning);
the cache is a facade concern because the facade owns the `Client` lifetime.

```go
// v2/tx/energy.go
const MaintenancePeriod = 6 * time.Hour // TRON maintenance interval

// v2/network.go — the Network type and genesis table live here.
// v2/tronlib.go — the cache fields are added to the existing Client struct.
type Client struct {
    ...
    priceMu sync.Mutex
    price   *tx.EnergyPrice
    priceAt time.Time
}
func (c *Client) EnergyPrice(ctx context.Context) (*tx.EnergyPrice, error)
```

`Client.EnergyPrice` returns the memoised price while
`time.Since(priceAt) < tx.MaintenancePeriod`, else refetches via
`tx.EnergyPriceOf`. A caller wanting a guaranteed-fresh read calls
`tx.EnergyPriceOf(c.Raw(), ctx)` directly — the escape hatch already exists, so
no `Refresh` method is added to the facade surface. `CostPreview` keeps its own
fresh read and its `PricedAt` stamp; it is not routed through the cache.

Tests: two calls inside the TTL hit the fake once; advancing the clock past the
TTL refetches; a failed refetch does not poison the cache.

### 6.2 C2 — §7.3 `CostPreview.BandwidthNote`

Add a field, populated by `PreviewCost`:

```go
type CostPreview struct {
    ...
    // BandwidthNote states what the preview does not cover.
    BandwidthNote string
}
const BandwidthNotModelled = "bandwidth (NetFee) and recipient activation are not included"
```

`String()` appends the note so a logged preview cannot be read as a total. The
live run measured the gap: energy-only preview 1,356,900 SUN vs actual total
1,701,900 SUN, the 345,000 SUN delta being bandwidth. Tests assert the field is
non-empty and that `String()` contains it.

### 6.3 C3 — §7.2 `Estimate.HasResult()`

```go
func (e *Estimate) HasResult() bool { return e != nil && len(e.ConstantResult) > 0 }
```

Nil-safe. Tests: empty, populated, nil receiver.

### 6.4 C4 — §5.4 `Amount.Formatted()`

Display form of a token amount, mirroring `tron.SUN.Formatted` (thousands
separators on the integer part, exact decimals — token amounts are atomic, so no
rounding). The zero value (`raw == nil`) renders `"0"`. `String()` remains the
canonical round-trip form; `Formatted()` is display-only and is documented as
not parseable, matching `SUN`'s wording. Tests cover zero, sub-unit, >18
decimals, and a 255-decimal fixture (already required by §5.4).

### 6.5 C5 — docgen multi-package support

`parseCodesData` currently hardwires `package tron`, which is why facade
`Example`s have no `go:example` markers. Generalize the loader to accept N
`-pkg` roots and namespace generated example markers by package. Result: the
facade's happy-path `Example` gets a real marker, and
`v2/docs/examples.md` is filled from it instead of staying compile-only.

**Path risk (one-way ratchet).** This touches the generator's scanning logic and
may grow past a bounded change. If, on implementation, it requires redesigning
the marker format or the codes loader beyond accepting multiple packages, stop
and re-classify to architectural before proceeding.

---

## 7. Testing and verification strategy

- Every B5, C1–C4 change is hermetic (bufconn fakes, `t.Fatal` on any live dial).
- B3 is tested against fixture trees, not the live repo alone.
- `go -C v2 build ./...`, `go -C v2 test ./... -count=1 -race`, `go -C v2 vet
  ./...`, `gofmt -l v2/` (known `brokenpkg` fixture excepted), coverage floor
  80%, and both drift gates (`docgen sync-docs -check`, `migrate -check`) must
  pass at completion.
- The §2.1 genesis constants are verified once by the harness author (fetched
  above) and thereafter only compared; `VerifyNetwork` never writes them.

## 8. Residual risks

| ID | Risk | Mitigation |
|---|---|---|
| **H1** | The genesis table goes stale when a testnet is redeployed. | `WithNetwork` is the source of truth; mismatch is a *detection*, and the Hint names both hashes so the new genesis is paste-ready. Documented as heuristic. |
| **H2** | The migration guide's curated `renames` under-covers v1, leaving a large Unmapped section. | Unmapped is explicit and honest; the guide's value is the mechanical match plus an actionable Unmapped list, not a claim of completeness. |
| **H3** | docgen multi-package exceeds a bounded change. | Stated upgrade trigger in §6.5: stop and re-classify. |
| **H4** | §7.5 item 6 stays unverified at the tag. | Recorded as an explicit "inconclusive — network-parameter bound" outcome with the parameter evidence, not as a pass. The API shape is unaffected either way (§15 R1). |
| **H5** | Changing `tronlib.DialOption` from an alias to a struct breaks a caller passing `rpc.DialOption`. | Never a documented affordance; noted in the changelog. `tronlib.WithTimeout`/`WithPool` keep working unchanged. |

## 9. Spec deltas this document creates

- §7.5 item 6 outcome: **inconclusive on Nile — `getDynamicEnergyThreshold` =
  5e9 energy; requires Mainnet or a private chain.** Item 1 remains verified by
  the 2026-09-01 run only for penalty-free contracts.
- §10 `Dial` auto-`VerifyNetwork` — not implemented (consistent with the
  always-lazy `Dial` deviation already recorded). `VerifyNetwork` is explicit.
- §7.3 `CostPreview.BandwidthNote` and the energy-price cache move from "NOT
  implemented" to implemented.
- §7.2 `Estimate.HasResult()`, §5.4 `Amount.Formatted()` move from "NOT
  implemented" to implemented.
