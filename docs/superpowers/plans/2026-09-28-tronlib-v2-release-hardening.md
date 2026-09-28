# tronlib v2 — Release Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Land the implementable `v2.0.0` tag-cycle gates — network identity (`Network`/`VerifyNetwork`), the generated v1→v2 migration guide, and the TIP-491 probe harness — then the five deferred Phase 2.1 spec deltas, so the v2 surface is complete against its own design.

**Architecture:** Additive changes inside the existing v2 module and its strict DAG (`tron` ← `key`/`event` ← `rpc` ← `tx` ← `contract` ← `token` ← root facade). No package gains a new dependency: `network.go` lives in the facade and reads through `rpc`; `cmd/migrate` and `cmd/tip491probe` are leaf commands that parse source or call the public facade. The spec deltas are field/method additions to types that already exist.

**Tech Stack:** Go 1.25, `go/ast`/`go/parser` (migration scanner), gRPC bufconn fakes (hermetic tests), protobuf `pb/api` + `pb/core`, testify, the repo's `cmd/docgen` marker/render pattern.

**Spec:** `docs/superpowers/specs/2026-09-28-tronlib-v2-release-hardening-design.md` (this plan implements its sections 3–6). Base design: `docs/superpowers/specs/2026-08-31-tronlib-v2-design.md`.

## Global Constraints

- **NO GIT PUSH.** The user directed all work stays local on branch `v2`. No task, subagent, or command may run `git push` in any form. Tagging (`v2.0.0`) is **out of scope** — it is user-gated on the §7.5 item-6 outcome.
- Every v2 command is `go -C v2 <cmd>` (v2 is a nested module).
- Coverage floor **80%** of statements for `./v2/...`; CI enforces it.
- Error model: every exported function returns `*tron.Error` with a Code from the §8.3 set, a `Hint` carrying remediation, and `Op` naming the function.
- DAG is law: `key`/`event` import only `tron`; `rpc` imports `tron` + pb; `tx` imports `tron, key, rpc`; `contract` imports `tron, rpc, tx`; `token` imports `tron, rpc, tx, contract`; root imports all. **No package may import the root facade.**
- N5 carve-out: `any` is permitted ONLY as a type-parameter constraint, never in an exported param/result type.
- Hermetic tests only: every network-touching test uses the existing bufconn fakes. The only live code is the B2 harness, which is built and never run by this plan.
- docgen discipline: any new `Example*` function needs a marker in a docs file, and both drift gates (`sync-docs -check`, `migrate -check`) must pass.
- Conventional commit per task; no task leaves the tree red.
- Verify command at every task's final step:
  `go -C v2 build ./... && go -C v2 test ./... -count=1 -race && go -C v2 vet ./... && gofmt -l v2/` (the only permitted `gofmt` output is `v2/cmd/docgen/testdata/brokenpkg/broken.go`).

## Review Focus

Failure modes the spec implies but no task's own happy-path test would otherwise catch. Each gets a test in the task that owns the code:

1. **`VerifyNetwork` against an unreachable endpoint** must surface the transport error (`chain.connection`), never `chain.network_mismatch` — a mismatch claim the client cannot substantiate is worse than an honest failure.
2. **A declared network the table does not know** (e.g. `Network("bogus")`) must fail closed as `chain.network_mismatch`, not pass because no table entry matched.
3. **Concurrent `Client.EnergyPrice` calls** must be race-clean and fetch once inside the TTL — the whole point of the cache is a 500-transfer batch, which is concurrent in practice.
4. **`Amount.Formatted()` on the zero value** (`raw == nil`) and on a 255-decimal amount must not panic and must render a parseable-looking-but-display-only string.
5. **A `renames` entry whose v2 target does not exist** must fail the generator loudly (config error), never render a dead mapping into the guide.

---

## File Structure

```
v2/
├── network.go                    # NEW: Network type, constants, genesis table, VerifyNetwork
├── tronlib.go                    # MOD: DialOption struct, Dial wiring, Client fields,
│                                 #      Network/VerifyNetwork, EnergyPrice cache
├── network_test.go               # NEW: hermetic VerifyNetwork tests (bufconn fake)
├── facade_test.go                # MOD: fake GetBlockByNum2 + call counter
├── internal/format/
│   └── format.go (+ format_test.go)   # NEW: Thousands() shared by tron and token
├── tron/sun.go                   # MOD: Formatted delegates to internal/format
├── token/amount.go               # MOD: Amount.Formatted
├── token/amount_test.go          # MOD: formatted cases (zero, 255-decimals)
├── tx/energy.go                  # MOD: MaintenancePeriod const
├── tx/cost.go                    # MOD: CostPreview.BandwidthNote + populate + String
├── tx/estimate.go                # MOD: (*Estimate).HasResult
├── tx/estimate_test.go           # MOD: HasResult + BandwidthNote tests
├── cmd/tip491probe/
│   ├── main.go                   # NEW: live TIP-491 probe (built, not run by this plan)
│   └── probe.go (+ probe_test.go)     # NEW: pure verdict logic
├── cmd/migrate/
│   ├── main.go                   # NEW: flags, run, -check
│   ├── scan.go                   # NEW: AST symbol scan for v1 and v2 trees
│   ├── classify.go               # NEW: Removed/Renamed/Moved/Candidate/Unmapped
│   ├── render.go                 # NEW: Markdown tables + marker fill
│   ├── tables.go                 # NEW: curated removals/renames inputs
│   └── testdata/{v1,v2}/...      # NEW: fixture trees for the golden test
├── cmd/docgen/
│   ├── main.go                   # MOD: -example-pkg (repeatable)
│   └── sync.go                   # MOD: runSync takes example package list, namespaced keys
└── docs/
    ├── migration.md              # NEW: generated guide (go:migration block)
    └── examples.md               # MOD: markers namespaced <pkg>.<Example>, facade examples added

.github/workflows/test-coverage.yml # MOD: migrate -check + updated docgen command
```

---

### Task 1: B5 — `Network`, `WithNetwork`, `VerifyNetwork`

**Files:**
- Create: `v2/network.go`
- Create: `v2/network_test.go`
- Modify: `v2/tronlib.go` (the `DialOption`/`Dial`/`Client` block, ~lines 78–115)
- Modify: `v2/facade_test.go` (`fakeFacadeServer` struct + `GetBlockByNum2` method)
- Modify: `v2/doc.go` (network-identity prose)

**Interfaces:**
- Consumes: `rpc.Dial`, `rpc.DialOption`, `rpc.WithTimeout`, `rpc.WithPool`, `rpc.GetBlockByNum2` (returns `*api.BlockExtention`), `tron.Error`, `tron.CodeChainNetworkMismatch`.
- Produces (used by Tasks 5 and 10):

```go
type Network string

const (
    Mainnet Network = "mainnet"
    Shasta  Network = "shasta"
    Nile    Network = "nile"
    Private Network = "private"
)

// DialOption is opaque now: callers build it with WithTimeout/WithPool/WithNetwork.
type DialOption struct { /* rpcOpts []rpc.DialOption; network Network */ }

func WithTimeout(d time.Duration) DialOption
func WithPool(initConnections, maxConnections int) DialOption
func WithNetwork(n Network) DialOption

func (c *Client) Network() Network
func (c *Client) VerifyNetwork(ctx context.Context) error
```

Exact genesis values (from spec §2.1; do not derive them):

```go
var genesisID = map[Network]string{
    Mainnet: "00000000000000001ebf88508a03865c71d452e25f4d51194196a1d22b6653dc",
    Nile:    "0000000000000000d698d4192c56cb6be724a558448e2684802de4d6cd8690dc",
    Shasta:  "0000000000000000de1aa88295e1fcf982742f773e0419c5a9c134c994a9059e",
}
```

- [ ] **Step 1: Extend the facade fake to serve block 0**

In `v2/facade_test.go` add a field and method to `fakeFacadeServer`:

```go
BlockByNum2     func(ctx context.Context, in *api.NumberMessage) (*api.BlockExtention, error)
blockByNumCalls int // counts calls, for the "private/undeclared skips the read" assertion
```

and the method `func (f *fakeFacadeServer) GetBlockByNum2(ctx, in) (*api.BlockExtention, error)` delegating to the field (returning a zero `*api.BlockExtention` when nil, matching the struct's other optional fields), incrementing the counter.

- [ ] **Step 2: Write the failing tests in `v2/network_test.go`**

Use `newFacadeTestClient(t, f)`. Table with these exact assertions:

```go
func TestVerifyNetworkMatchesConfigured(t *testing.T)       // Mainnet/Nile/Shasta each return nil for their own hash
func TestVerifyNetworkMismatch(t *testing.T)                // Nile configured, Mainnet hash observed
    // err != nil; tron.HasCode(err, tron.CodeChainNetworkMismatch)
    // err.(*tron.Error).Hint contains both the configured name and the observed hex
func TestVerifyNetworkUnknownDeclarationFailsClosed(t *testing.T) // Network("bogus") + any block0 -> chain.network_mismatch
func TestVerifyNetworkPrivateAndUndeclaredSkipRead(t *testing.T)  // Private and "" -> nil, f.blockByNumCalls == 0
func TestVerifyNetworkTransportErrorPropagates(t *testing.T)      // fake returns chain.connection *tron.Error
    // the returned code is chain.connection, NOT chain.network_mismatch
func TestNetworkRoundTripsWithNetworkOption(t *testing.T)
    // c, err := Dial(context.Background(), "grpc://127.0.0.1:1", WithNetwork(Nile))
    // err == nil (Dial is lazy, no I/O); c.Network() == Nile; c.Close()
```

Helper `mustHex(t, hexStr) []byte` decodes with `encoding/hex` and asserts no error.

- [ ] **Step 3: Run to verify failure**

Run: `go -C v2 test ./ -run 'TestVerifyNetwork|TestNetworkRoundTrips' -v`
Expected: FAIL — `Network`, `WithNetwork`, `VerifyNetwork` undefined; compile error.

- [ ] **Step 4: Implement `v2/network.go` and the `tronlib.go` option refactor**

`v2/network.go`: the `Network` type, constants, `genesisID` table, `WithNetwork`, `Client.Network()`, `Client.VerifyNetwork(ctx)`.

`VerifyNetwork` logic, pinned:
1. `n := c.network`; if `n == ""` or `n == Private` → return nil (no read).
2. Look up `want, ok := genesisID[n]`; if `!ok` → return `&tron.Error{Code: tron.CodeChainNetworkMismatch, Op: "tronlib.Client.VerifyNetwork", Hint: ...}` naming the unknown declaration (fail closed).
3. `blk, err := rpc.GetBlockByNum2(c.inner, ctx, &api.NumberMessage{Num: 0})`; on error return the error unchanged.
4. `got := hex.EncodeToString(blk.GetBlockid())`; if `got != want` → mismatch error whose `Hint` names both `want` and `got`. Else nil.

`v2/tronlib.go`: replace `type DialOption = rpc.DialOption` with the opaque struct; `WithTimeout`/`WithPool` wrap one `rpc.DialOption` each; `WithNetwork` sets the field. `Dial` collects every option's `rpcOpts`, calls `rpc.Dial(ctx, endpoint, rpcOpts...)`, and stores `network` on the `Client` (add field `network Network`). Keep the existing lazy-`Dial` doc comment and add: verification is explicit; `Dial` does not auto-verify.

- [ ] **Step 5: Run tests to verify pass, then the task gate**

Run: `go -C v2 test ./ -run 'TestVerifyNetwork|TestNetworkRoundTrips' -v` → PASS.
Run: `go -C v2 build ./... && go -C v2 test ./... -count=1 -race && go -C v2 vet ./...` → all green.
Update `v2/doc.go`'s network paragraph: identity is explicit configuration, `VerifyNetwork` is a heuristic compared against a recorded genesis table, and `Dial` does not auto-verify.

- [ ] **Step 6: Commit**

```bash
git add v2/network.go v2/network_test.go v2/tronlib.go v2/facade_test.go v2/doc.go
git commit -m "feat(v2): Network identity with explicit VerifyNetwork fingerprint check"
```

---

### Task 2: B2 — `cmd/tip491probe` harness (built, never run)

**Files:**
- Create: `v2/cmd/tip491probe/probe.go`
- Create: `v2/cmd/tip491probe/probe_test.go`
- Create: `v2/cmd/tip491probe/main.go`

**Interfaces:**
- Consumes: `tronlib.Dial`, `tronlib.KeyFromHex`, `tronlib.ParseAddress`, `tx.BuildTriggerSmartContract`, `(*tx.ContractTx).Simulate`, `rpc.GetChainParameters`.
- Produces: `func checkPenalty(penalty int64) error` (pure).

- [ ] **Step 1: Write the failing test in `probe_test.go`**

```go
func TestCheckPenaltyAcceptsPositive(t *testing.T)      // checkPenalty(1) == nil, checkPenalty(9492) == nil
func TestCheckPenaltyInconclusiveZero(t *testing.T)     // err != nil; message contains "inconclusive"
    // and "getDynamicEnergyThreshold" and "5e9" (the recorded Nile bound)
func TestCheckPenaltyNegativeIsError(t *testing.T)      // checkPenalty(-1) != nil (defensive; a negative penalty is not a pass)
```

- [ ] **Step 2: Run to verify failure**

Run: `go -C v2 test ./cmd/tip491probe/ -v`
Expected: FAIL — package/function undefined.

- [ ] **Step 3: Implement `checkPenalty`**

Return nil for `penalty > 0`. For `penalty <= 0`, return a `fmt.Errorf` whose text states that the result is inconclusive (not a pass) and quotes the Nile bound: `getDynamicEnergyThreshold = 5000000000` energy makes item 6 unreachable on Nile; use a Mainnet contract with a non-zero consumption factor or a private chain with a lowered threshold.

- [ ] **Step 4: Implement `main.go`**

Flags, pinned: `-endpoint` (default `grpc://grpc.nile.trongrid.io:50051`), `-contract` (required), `-owner` (default `-key`'s address when `-key` is set), `-key` (hex, optional), `-data` (hex calldata, required).

Behavior: dial (lazy), build a `ContractTx` with `tx.BuildTriggerSmartContract`, call `Simulate`, print `Energy` and `Penalty`, then print `checkPenalty(est.Penalty)` and exit non-zero when it is non-nil. Print `rpc.GetChainParameters`'s dynamic-energy entries when readable (best-effort; a read error is printed, not fatal). It must never broadcast.

- [ ] **Step 5: Run the task gate**

Run: `go -C v2 test ./cmd/tip491probe/ -v` → PASS (the test never dials).
Run: `go -C v2 build ./... && go -C v2 test ./... -count=1 -race && go -C v2 vet ./...` → green.

- [ ] **Step 6: Commit**

```bash
git add v2/cmd/tip491probe
git commit -m "feat(v2): TIP-491 probe harness for the user-gated §7.5 item-6 check"
```

---

### Task 3: B3 — `cmd/migrate` scanner, classifier, renderer

**Files:**
- Create: `v2/cmd/migrate/scan.go`, `v2/cmd/migrate/classify.go`, `v2/cmd/migrate/render.go`, `v2/cmd/migrate/tables.go`, `v2/cmd/migrate/main.go`
- Create: `v2/cmd/migrate/scan_test.go`, `v2/cmd/migrate/classify_test.go`, `v2/cmd/migrate/render_test.go`, `v2/cmd/migrate/main_test.go`
- Create: `v2/cmd/migrate/testdata/v1/...`, `v2/cmd/migrate/testdata/v2/...`

**Interfaces:**
- Produces:

```go
type Symbol struct {
    Pkg     string // short package name, e.g. "account"
    Name    string // "Manager", "Transfer", "TRX"
    Recv    string // receiver type name for methods, "" otherwise
    Kind    string // "func" | "type" | "method" | "const"
    Key     string // "pkg.Type.Method" | "pkg.Type" | "pkg.Func"
    File    string // source path, for path-based removal (e.g. pkg/client/lowlevel/shielded.go)
}

func scanDir(dir string) ([]Symbol, error)   // one Go package, non-recursive, skips _test.go
func scanTree(root string) ([]Symbol, error) // walks root, scanDir per dir with non-test .go files

// Inputs is the curated input set from tables.go.
type Inputs struct {
    RemovedPaths    []string          // path substrings -> Removed (file-scoped C3/C4 exclusions)
    RemovedPrefixes []string          // name prefixes -> Removed (e.g. "AssetIssue")
    Renames         map[string]string // v1 key -> v2 key; every target must exist in v2
}

func classify(v1, v2 []Symbol, in Inputs) (Result, error)
func render(r Result) string
```

- `Result` holds `Moved []Pair`, `Renamed []Rename`, `Removed []Removal`, `Candidate []Pair`, `Unmapped []Symbol`, and `Summary` counts.

- [ ] **Step 1: Write the failing scanner test**

`TestScanTreeExtractsExportedSymbols` parses `testdata/v1` and asserts the exact `Symbol` set (funcs, types, methods with receiver, consts) and that unexported names are absent. Fixture `testdata/v1/account/account.go` (package `account`) contains an exported `type Manager struct{}`, `func (m *Manager) Transfer()`, `func NewManager()`, `const MaxRetries`, and an unexported `func helper()`.

- [ ] **Step 2: Run to verify failure**

Run: `go -C v2 test ./cmd/migrate/ -run TestScanTree -v`
Expected: FAIL — undefined.

- [ ] **Step 3: Implement `scanDir` and `scanTree`**

`scanDir` parses every `*.go` file directly in one directory (skip `_test.go` and subdirectories) and
collects exported top-level `FuncDecl` (a method when `Recv != nil`, using the receiver's type name),
`GenDecl` types and consts, recording each symbol's source path in `Symbol.File`. `scanTree` walks
`root`, calling `scanDir` on every directory that contains a non-test `.go` file. Key format as in the
`Symbol` comment. Deterministic order: sort by `Key`.

- [ ] **Step 4: Write the failing classifier test**

`TestClassify` with two fixture trees (add `testdata/v1/lowlevel/shielded.go`, matched by a `RemovedPaths` entry — v1 has **no** `pkg/shielded` package, the shielded functions live in `pkg/client/lowlevel/shielded.go`; and `testdata/v1/trc10/issuance.go` with an `AssetIssueCreate` function), asserting each of the five states:

```go
// same func name, same shape, different package      -> Moved
// same method name, v1 receiver Manager, v2 free func-> Candidate
// path "lowlevel/shielded.go" declared removed       -> Removed (clause "spec §13 (C3)")
// name prefix "AssetIssue"                           -> Removed (clause "spec §13 (C3) TRC-10 issuance")
// present only in v1, no signal                      -> Unmapped
// explicit Renames entry with a real v2 target       -> Renamed
```

- [ ] **Step 5: Run to verify failure, then implement `classify`**

Run: `go -C v2 test ./cmd/migrate/ -run TestClassify -v` → FAIL.
Implement in the order the spec §5.3 pins: (1) `Inputs.RemovedPaths` substrings, (2) `Inputs.RemovedPrefixes` (the `AssetIssue` issuance rule), (3) explicit `Renames`, (4) Moved (func→func or method→method with identical receiver name), (5) Candidate (name matches, shape changed), (6) Unmapped. A `Renames` target absent from the v2 symbol set is a **config error** returned by `classify` (Review Focus 5).

- [ ] **Step 6: Write the failing renderer + main tests**

`TestRenderGolden` byte-compares `render(r)` for a fixed `Result` against a golden string in `render_test.go`. Pin the section headers exactly:

```
## Moved
| v1 | v2 |
## Renamed
| v1 | v2 | why |
## Removed
| v1 | spec |
## Needs review (mechanical candidate — verify)
| v1 | v2 |
## Unmapped (no mechanical v2 counterpart found)
- `account.Manager.Foo`
```

and the summary line `**N v1 symbols: M moved, R renamed, D removed, C candidates, U unmapped.**` (counts computed, not fixed).

`TestRunCheckDetectsDrift` writes a docs file with a stale `go:migration` block, runs `run(..., check=true)`, expects a non-zero result; after a plain `run` it expects zero. `TestRunCheckPassesOnGenerated` round-trips.

- [ ] **Step 7: Implement `render` and `main`**

`render` emits the section order above, omitting empty sections. `main` flags: `-v1` (default `../pkg` — the v1 public surface is under `pkg/`; `cmd/` is the excluded CLI and `integration_test/` is not API), `-v2-root` (default `.`), `-out` (default `docs/migration.md`), `-check`. `v2Pkgs` in `tables.go` lists the v2 package dirs to scan (`tron`, `key`, `event`, `rpc`, `tx`, `contract`, `token`, and `.` for the facade); `main` calls `scanDir` on each and merges, and calls `scanTree("../pkg")` for v1. Marker constants `<!-- go:migration -->` / `<!-- /go:migration -->`; `run` replaces the block (prose outside is untouched) or, in `-check` mode, byte-compares. Reuse `replaceBetween`-style logic; do not import `cmd/docgen`.

- [ ] **Step 8: Run the task gate**

Run: `go -C v2 test ./cmd/migrate/ -v` → PASS.
Run: `go -C v2 build ./... && go -C v2 test ./... -count=1 -race && go -C v2 vet ./...` → green.

- [ ] **Step 9: Commit**

```bash
git add v2/cmd/migrate
git commit -m "feat(v2): AST-diff migration scanner, classifier, and renderer"
```

---

### Task 4: B3 — generate `v2/docs/migration.md` and gate it in CI

**Files:**
- Create: `v2/docs/migration.md`
- Modify: `v2/cmd/migrate/tables.go` (seed the curated inputs)
- Modify: `.github/workflows/test-coverage.yml`

**Interfaces:**
- Consumes: Task 3's `cmd/migrate`.
- Produces: the committed guide and the CI step.

- [ ] **Step 1: Seed `tables.go`**

Populate `removedPaths` with `pkg/client/lowlevel/shielded.go` (v1 has no `pkg/shielded` package — the shielded functions live in that one lowlevel file); keep `removedPrefixes` for the mechanical `AssetIssue` rule. Populate `Renames` **only** with entries whose target you have verified by `grep` in the v2 tree (start with `utils.*`-dissolution and manager→facade mappings you can point at). Every entry carries a one-line reason comment citing the spec section. Do not guess: an unverified entry either fails the config check or misleads the guide.

- [ ] **Step 2: Generate the guide**

Write `v2/docs/migration.md` with a short prose header (what the guide is, that Unmapped is honest, how to regenerate) and an empty marker block, then run:
`go -C v2 run ./cmd/migrate -v1 ../pkg -v2-root . -out docs/migration.md`

Expected: the block fills; the Unmapped section is non-empty and that is correct (spec risk H2).

- [ ] **Step 3: Verify the real run is a fixed point**

Run: `go -C v2 run ./cmd/migrate -check` → exit 0.
If it fails, the renderer is non-deterministic (map iteration) — sort every collection before printing.

- [ ] **Step 4: Add the CI gate**

In `.github/workflows/test-coverage.yml`, add immediately after the `docgen drift check (v2)` step:

```yaml
    - name: migration guide drift check (v2)
      run: go -C v2 run ./cmd/migrate -check
```

- [ ] **Step 5: Run the task gate and commit**

Run: `go -C v2 test ./cmd/migrate/ -count=1 && go -C v2 run ./cmd/migrate -check` → both green.

```bash
git add v2/docs/migration.md v2/cmd/migrate/tables.go .github/workflows/test-coverage.yml
git commit -m "docs(v2): generated v1-to-v2 migration guide with CI drift gate"
```

---

### Task 5: C1 — §7.3 energy-price cache

**Files:**
- Modify: `v2/tx/energy.go` (add `MaintenancePeriod`)
- Modify: `v2/tronlib.go` (`Client` fields + `EnergyPrice` method)
- Modify: `v2/facade_test.go` (add `energyPricesCalls atomic.Int32` to `fakeFacadeServer` and increment it in `GetEnergyPrices`)
- Create: `v2/energy_cache_test.go`

**Interfaces:**
- Consumes: `tx.EnergyPriceOf`, `tx.EnergyPrice`.
- Produces: `const tx.MaintenancePeriod = 6 * time.Hour`; `func (c *Client) EnergyPrice(ctx context.Context) (*tx.EnergyPrice, error)` now memoising.

- [ ] **Step 1: Write the failing tests in `v2/energy_cache_test.go`**

```go
func TestEnergyPriceCachesWithinTTL(t *testing.T)   // two calls -> f.energyPricesCalls == 1
func TestEnergyPriceRefetchesAfterTTL(t *testing.T) // set c.priceAt back by MaintenancePeriod+1s -> 2 calls
func TestEnergyPriceFailedRefetchDoesNotPoisonCache(t *testing.T)
    // first call errors -> price stays nil; second call with a working fake fetches and returns
func TestEnergyPriceConcurrentSingleFetch(t *testing.T)
    // 16 goroutines call EnergyPrice; -race clean; f.energyPricesCalls == 1
```

Use a mutex-protected counter in the fake (the existing `fakeFacadeServer` fields are plain funcs — add an `atomic.Int32` counter like `broadcastCalls`).

- [ ] **Step 2: Run to verify failure**

Run: `go -C v2 test ./ -run TestEnergyPrice -v`
Expected: FAIL — `TestEnergyPriceCachesWithinTTL` sees 2 calls.

- [ ] **Step 3: Implement**

`v2/tx/energy.go`: `const MaintenancePeriod = 6 * time.Hour` with a doc comment that it is TRON's maintenance interval and that the price changes only by governance proposal.

`v2/tronlib.go`: add `priceMu sync.Mutex`, `price *tx.EnergyPrice`, `priceAt time.Time` to `Client`. `EnergyPrice` locks, returns the memoised value when `c.price != nil && time.Since(c.priceAt) < tx.MaintenancePeriod`, else calls `tx.EnergyPriceOf(c.inner, ctx)` and stores `(price, time.Now())` only on success. A caller wanting a guaranteed-fresh read keeps the `tx.EnergyPriceOf(c.Raw(), ctx)` escape hatch — no `Refresh` method is added.

- [ ] **Step 4: Run to verify pass, then the task gate**

Run: `go -C v2 test ./ -run TestEnergyPrice -race -v` → PASS.
Run the global verify command → green.

- [ ] **Step 5: Commit**

```bash
git add v2/tx/energy.go v2/tronlib.go v2/facade_test.go v2/energy_cache_test.go
git commit -m "feat(v2): cache EnergyPrice for one maintenance period (spec §7.3)"
```

---

### Task 6: C2 — §7.3 `CostPreview.BandwidthNote`

**Files:**
- Modify: `v2/tx/cost.go`
- Modify: `v2/tx/estimate_test.go`

**Interfaces:**
- Produces: `CostPreview.BandwidthNote string`; `const tx.BandwidthNotModelled = "bandwidth (NetFee) and recipient activation are not included"`.

- [ ] **Step 1: Write the failing tests**

Add to `v2/tx/estimate_test.go`:

```go
func TestPreviewCostCarriesBandwidthNote(t *testing.T)
    // after PreviewCost: cp.BandwidthNote == tx.BandwidthNotModelled (the exported const)
    // and cp.String() contains "bandwidth"
```

- [ ] **Step 2: Run to verify failure**

Run: `go -C v2 test ./tx/ -run TestPreviewCostCarriesBandwidthNote -v` → FAIL.

- [ ] **Step 3: Implement**

Add the field and constant to `v2/tx/cost.go`; set `BandwidthNote: BandwidthNotModelled` in `PreviewCost`'s returned struct; append the note in `String()` so a logged preview cannot read as a total. Update the `CostPreview` doc comment: the 2026-09-01 live run measured a 345,000 SUN bandwidth delta the energy-only preview does not cover.

- [ ] **Step 4: Run to verify pass, then the task gate**

Run: `go -C v2 test ./tx/ -run 'TestPreviewCost' -v` → PASS.
Run the global verify command → green.

- [ ] **Step 5: Commit**

```bash
git add v2/tx/cost.go v2/tx/estimate_test.go
git commit -m "feat(v2/tx): CostPreview states its bandwidth limitation (spec §7.3)"
```

---

### Task 7: C3 — §7.2 `Estimate.HasResult()`

**Files:**
- Modify: `v2/tx/estimate.go`
- Modify: `v2/tx/estimate_test.go`

**Interfaces:**
- Produces: `func (e *Estimate) HasResult() bool`.

- [ ] **Step 1: Write the failing test**

```go
func TestEstimateHasResult(t *testing.T)
    // (&Estimate{}).HasResult() == false
    // (&Estimate{ConstantResult: [][]byte{{0x01}}}).HasResult() == true
    // var e *Estimate; e.HasResult() == false  (nil receiver, no panic)
```

- [ ] **Step 2: Run to verify failure**

Run: `go -C v2 test ./tx/ -run TestEstimateHasResult -v` → FAIL.

- [ ] **Step 3: Implement**

```go
func (e *Estimate) HasResult() bool { return e != nil && len(e.ConstantResult) > 0 }
```

Update the type doc: this replaces the `len(ConstantResult) > 0` idiom documented in `PHASE2.md`.

- [ ] **Step 4: Run to verify pass, then the task gate**

Run: `go -C v2 test ./tx/ -run TestEstimateHasResult -v` → PASS.
Run the global verify command → green.

- [ ] **Step 5: Commit**

```bash
git add v2/tx/estimate.go v2/tx/estimate_test.go
git commit -m "feat(v2/tx): Estimate.HasResult (spec §7.2)"
```

---

### Task 8: C4 — §5.4 `token.Amount.Formatted()`

**Files:**
- Create: `v2/internal/format/format.go`, `v2/internal/format/format_test.go`
- Modify: `v2/tron/sun.go` (`Formatted` delegates; remove the private `groupThousands`)
- Modify: `v2/token/amount.go`
- Modify: `v2/token/amount_test.go`

**Interfaces:**
- Produces: `func format.Thousands(digits string) string`; `func (a Amount) Formatted() string`.
- Consumes: `format.Thousands` from both `tron` and `token`.

- [ ] **Step 1: Write the failing tests**

`v2/internal/format/format_test.go`:

```go
func TestThousands(t *testing.T) // "0"->"0", "12"->"12", "123"->"123",
                                 // "1234"->"1,234", "1234567"->"1,234,567"
```

Add to `v2/token/amount_test.go`:

```go
func TestAmountFormatted(t *testing.T)
    // {0,6}.Formatted()      == "0"
    // {1500000,6}.Formatted()== "1.5"          (no separators in the fraction)
    // {1234567891,6}.Formatted() == "1,234.567891"
    // zero-value Amount{}.Formatted() == "0"    (raw nil, no panic — Review Focus 4)
    // a 255-decimal fixture renders without panic and keeps String() round-tripping
```

- [ ] **Step 2: Run to verify failure**

Run: `go -C v2 test ./internal/format/ ./token/ -run 'TestThousands|TestAmountFormatted' -v` → FAIL.

- [ ] **Step 3: Implement**

Move the existing `tron.groupThousands` body to `v2/internal/format.Thousands` (same algorithm; it is an algorithm the test does not determine, so keep it verbatim). `tron.SUN.Formatted` calls `format.Thousands`; delete the private copy. `token.Amount.Formatted` renders `a.String()`, splits integer/fraction, groups the integer part, rejoins — mirroring `SUN.Formatted`; the zero value renders `"0"`. Doc: display only, not parseable; `String()` remains the round-trip form.

- [ ] **Step 4: Run to verify pass, then the task gate**

Run: `go -C v2 test ./internal/format/ ./token/ ./tron/ -run 'TestThousands|TestAmountFormatted|TestSUNFormatted' -v` → PASS.
Run the global verify command → green.

- [ ] **Step 5: Commit**

```bash
git add v2/internal/format v2/tron/sun.go v2/token/amount.go v2/token/amount_test.go
git commit -m "feat(v2/token): Amount.Formatted with shared thousands grouping (spec §5.4)"
```

---

### Task 9: C5 — docgen multi-package support

**Files:**
- Modify: `v2/cmd/docgen/main.go` (flags)
- Modify: `v2/cmd/docgen/sync.go` (`runSync` signature + namespaced merge)
- Modify: `v2/cmd/docgen/sync_test.go`, `v2/cmd/docgen/main_test.go` (fixture markers)
- Modify: `v2/docs/examples.md` (migrate + add facade markers)
- Modify: `.github/workflows/test-coverage.yml` (docgen command)

**Interfaces:**
- Consumes: `extractExamples(dir)`, `exampleNames(md)`, `fillExamples`, `checkExampleCoverage`.
- Produces: `func packageNameOf(dir string) (string, error)`; `runSync(codesPkg string, examplePkgs []string, docsFiles []string, check bool) error`; example keys are `<pkgName>.<ExampleFunc>`.

- [ ] **Step 1: Write the failing test**

`TestRunSyncNamespacesExamplesByPackage` writes two temp packages with `os.MkdirTemp` (each defining `ExampleShared`, so the names collide) plus a docs file whose markers are `pkgA.ExampleShared` and `pkgB.ExampleShared`; asserts both fill with their own bodies (previously this collided in one flat map). Also `TestPackageNameOf` for `testdata/syncpkg` and `testdata/examplepkg`.

- [ ] **Step 2: Run to verify failure**

Run: `go -C v2 test ./cmd/docgen/ -run 'TestRunSyncNamespaces|TestPackageNameOf' -v` → FAIL.

- [ ] **Step 3: Implement**

`sync.go`: `runSync` extracts examples from the primary `codesPkg` plus each `examplePkgs` dir, keys each as `packageNameOf(dir) + "." + funcName`, and errors on a duplicate key (never silently overwrites). `checkExampleCoverage` runs over the merged map unchanged.
`main.go`: keep `-pkg` as the codes package; add repeatable `-example-pkg` (a `flag.Func` appending to a slice); pass it through to `runSync`.

- [ ] **Step 4: Migrate the markers and regenerate**

In `v2/docs/examples.md`, rename the four existing markers to namespaced form (`tron.ExampleHasCode`, `tron.ExampleParseAddress`, `tron.ExampleParseTRX`, `tron.ExampleTRX`), then add empty blocks for the facade examples `tronlib.Example`, `tronlib.ExampleClient_trx`, `tronlib.ExampleClient_token`. Run:

`go -C v2 run ./cmd/docgen sync-docs -pkg ./tron -example-pkg . -docs ./docs/errors.md -docs ./docs/examples.md`

Update the docgen tests' inline markers (`ExampleFoo`→`syncpkg.ExampleFoo`, etc.) to the namespaced keys.

- [ ] **Step 5: Update CI and verify drift is a fixed point**

Change the `docgen drift check (v2)` step's command to add `-example-pkg .`, then run it locally:

`go -C v2 run ./cmd/docgen sync-docs -pkg ./tron -example-pkg . -docs ./docs/errors.md -docs ./docs/examples.md -check` → exit 0.

- [ ] **Step 6: Run the task gate**

Run: `go -C v2 build ./... && go -C v2 test ./... -count=1 -race && go -C v2 vet ./...` → green.
**Upgrade trigger (spec §6.5, risk H3):** if Step 3 required reworking the marker format beyond namespacing or the codes loader beyond accepting multiple package dirs, stop and re-classify this task to architectural before continuing.

- [ ] **Step 7: Commit**

```bash
git add v2/cmd/docgen v2/docs/examples.md .github/workflows/test-coverage.yml
git commit -m "feat(v2/docgen): multi-package examples with namespaced markers"
```

---

### Task 10: Closeout — record spec deltas and run the full gate

**Files:**
- Modify: `v2/PHASE2.md`
- Modify: `v2/docs/errors.md` only if a code was added (none is)

**Interfaces:** none.

- [ ] **Step 1: Update `PHASE2.md`**

Move the following from "Deferred/NOT implemented" to implemented, and add the new deviations:

- C1 `EnergyPrice` cache, C2 `CostPreview.BandwidthNote`, C3 `Estimate.HasResult()`, C4 `Amount.Formatted()` — implemented.
- `Network()`/`VerifyNetwork()` — implemented; **new deviation:** §10's "Dial auto-verifies unless `WithLazyDial`" is not implemented; `Dial` stays always-lazy and `VerifyNetwork` is explicit.
- §7.5 item 6 — **inconclusive, network-parameter bound**: Nile's `getDynamicEnergyThreshold = 5e9`; requires Mainnet or a lowered threshold. Reference the spec §2.2 evidence.
- B3 migration guide — generated at `v2/docs/migration.md`, CI-gated.
- Deviation list: `tronlib.DialOption` is now an opaque struct, not an alias to `rpc.DialOption`.

- [ ] **Step 2: Full verification**

Run, in order, and record the output in the commit message:

```bash
go -C v2 build ./...
go -C v2 test ./... -count=1 -race
go -C v2 vet ./...
gofmt -l v2/                                     # only cmd/docgen/testdata/brokenpkg/broken.go
go -C v2 test ./... -count=1 -coverprofile=/tmp/v2-close.out
go -C v2 tool cover -func=/tmp/v2-close.out | grep '^total:'   # >= 80%
go -C v2 run ./cmd/docgen sync-docs -pkg ./tron -example-pkg . -docs ./docs/errors.md -docs ./docs/examples.md -check
go -C v2 run ./cmd/migrate -check
```

- [ ] **Step 3: Commit**

```bash
git add v2/PHASE2.md
git commit -m "docs(v2): record release-hardening deltas and the §7.5 item-6 outcome"
```

---

## Deferred (explicitly not in this plan)

- **B4 — tag `v2.0.0`.** User-gated on §7.5 item 6, which is currently inconclusive on Nile. No push, no tag.
- **Running the B2 harness.** Built and tested for its pure logic only; the live run needs a Mainnet contract with a non-zero consumption factor and funds the user controls.
- **§11 surface-budget ratchet** (`api/v2.txt` via `go/types` + CI growth gate). Chosen away during brainstorming; the Tier-A overage stays unrecorded until it is picked up.
- **§12 filling of `go:errors` in `docs/API_REFERENCE.md`** and v1 `doc.go` files. Chosen away during brainstorming.
- Spec §13 non-goals: shielded/Sapling, TRC-10 issuance, CLI, `schema.json`.

## Self-Review

**Spec coverage:** spec §3 → Task 2; §4 → Task 1; §5 → Tasks 3–4; §6.1 → Task 5; §6.2 → Task 6; §6.3 → Task 7; §6.4 → Task 8; §6.5 → Task 9; §7 verification → every task's gate plus Task 10; §8 risks → Review Focus + the H3 upgrade trigger in Task 9; §9 spec-delta record → Task 10. The out-of-scope list matches the plan's Deferred section.

**Step scan:** every step names one action and a checkable result; no step carries a function body the signature and test already determine (the genesis table, `checkPenalty`'s message, the `Thousands` algorithm, `HasResult`'s one-line body, and the marker/section strings are exact-copy the spec fixes). No "handle edge cases" lines.

**Type consistency:** `Network`/`DialOption`/`WithNetwork`/`Network()`/`VerifyNetwork` are identical between Task 1's Interfaces and Task 5's `Client` edits; `Symbol`/`Inputs`/`Result`/`scanDir`/`scanTree`/`classify`/`render` match between Tasks 3 and 4; `tx.MaintenancePeriod` (Task 5), `tx.BandwidthNotModelled` (Task 6), `Estimate.HasResult` (Task 7), `format.Thousands` and `Amount.Formatted` (Task 8), and `runSync`'s new signature (Task 9) are each defined once and used only where stated. `parseCodesData` is unchanged.

**Review Focus:** (1) Task 1 Step 2 `TestVerifyNetworkTransportErrorPropagates`; (2) Task 1 Step 2 `TestVerifyNetworkUnknownDeclarationFailsClosed`; (3) Task 5 Step 1 `TestEnergyPriceConcurrentSingleFetch`; (4) Task 8 Step 1 zero-value + 255-decimal cases; (5) Task 3 Step 5 config error for a dead `renames` target.

**Proportion:** the plan states decisions and test assertions; the only verbatim code is the fixed data (genesis hashes, the note string, section headers, marker syntax) and the two algorithms the tests do not determine (classification order, thousands grouping).
