# Event tooling v2: 32-byte selective capture — design

Date: 2026-09-30.
Status: rev 3 — awaiting re-review (owner). Rev 3 replaces popularity
discovery (the rolling-24h caller walk) with a **snapshotted TronScan
top-100 by `trxCount`**; the block walk, caller/owner decoding, txid join,
worker pool and `--min-callers/--workers/--sleep-ms` are deleted. Rev 2
findings (tracked corpus, shared signature derivation, CI package split)
are retained.
Scope: clean up the last v1 4-byte event concepts and rebuild the
capture/insert/migrate/generate loop on native 32-byte keys.
**No decode-*behavior* change** (`event/event.go` untouched;
`event/registry.go` gains only exported pure helpers, §4).

## 1. Background

v2 decodes logs keyed by the full 32-byte `keccak256("Name(type,...)")`
(`event/registry.go:sigKeyOf`). Two remnants still speak v1's 4-byte dialect:

- `tmp/events_registry.json`: 747 entries in `{selector, signature, name,
  inputs}` schema (`selector` = first 4 hash bytes as 8-hex). **This file is
  currently untracked and gitignored (`tmp/`), and nothing in the tree reads
  or writes it** — see §2 D2/§3.
- `event/builtin_gen.go`: `builtinSig4 map[[4]byte]*Definition`, re-keyed to
  32 bytes at `init()`; header still says "vendored for v2, regenerate from
  the v1 generator".

The three v1 commands (`event_abi_generator`, `event_abi_loader`,
`generate_event_builtins`) are already gone from the v2 tree. Nothing else
references them. `TestBuiltinTableCountAndKeys` proves
`keccak(signature)[:4] == selector` for all 747 entries, so migration is
lossless.

The v1 corpus was curated by walking blocks and counting distinct callers.
That discovery step is the part worth replacing: it costs ~57k RPCs per 24h
window, depends on wall-clock and node health, and yields a different set
every run. A ranked, snapshotted source is smaller and reproducible.

## 2. Decisions (owner-approved; D4 replaced in rev 3)

- D1 One binary `cmd/eventtool` with `contracts` / `capture` / `insert` /
  `migrate` / `generate` subcommands (precedent: `cmd/docgen`). Reusable
  logic lives in `internal/eventtool/`; `cmd/eventtool/main.go` is flag
  parsing and wiring only (precedent: `internal/format`,
  `internal/compilecheck`). See §6 for why the split is load-bearing.
- D2 The corpus is **relocated to a tracked path, then migrated in place**:
  `internal/eventdata/events_registry.json`, new 32-byte schema, no 4-byte
  field retained. The 7 seed ABIs move from `tmp/abi/` to
  `internal/eventdata/abi/`. Nothing in this design may depend on a
  gitignored path.
- D3 `event/builtin_gen.go` regenerated to `map[[32]byte]*Definition` now.
- D4 **Contract selection = TronScan's contract ranking by call volume.**
  `eventtool contracts` fetches the top 100 from
  `https://apilist.tronscanapi.com/api/contracts?sort=-trxCount` and writes
  the ranked list to the tracked `internal/eventdata/top_contracts.json`.
  `capture` never calls TronScan — it consumes only the snapshot plus
  on-chain ABIs, so the corpus is a pure function of (snapshot, chain) and
  the same commit regenerates the same corpus. Refreshing the list is a
  deliberate, diff-reviewable command.
  - The endpoint caps `limit` at **50** (100 → HTTP 400), so top-100 is two
    requests (`start=0` and `start=50`).
  - **No verification filter.** Every deployed TRON contract carries an
    on-chain ABI whether or not its source is verified, so all 100 are
    fetched; `verify_status` is recorded in the snapshot for provenance
    only. TronScan's `open-source-only`/`verified-only` params are ignored
    by this endpoint anyway (verified against the live API).
- D5 Gate/payload: selection only **gates** which contracts are captured;
  once selected, `capture` ingests the contract's **entire ABI** (all named
  event entries), not just events that appear in one call path.
- D6 Store stays additive across runs (upsert-if-absent, first-wins).
  First-wins is permanent (pruning is a non-goal); the correction path — hand
  edit the corpus, regenerate — is stated in the runbook.
- D7 **Signature derivation is shared, not re-implemented.** `event` exports
  two pure helpers, `CanonicalSignature(name string, inputTypes []string)
  string` and `SignatureKey(name string, inputTypes []string) [32]byte`;
  `Definition.signature()` and `sigKeyOf` delegate to them. `cmd/eventtool`
  and its tests use the exported helpers, so tool-derived hashes agree with
  the registry **by construction**, not by convention. This is additive and
  behavior-preserving; it is the only change to `event/registry.go` and
  supersedes the "registry.go untouched" wording of the earlier scope note.

## 3. Corpus migration (`internal/eventdata/events_registry.json`)

New schema per entry: `{sighash, signature, name, inputs}`.

- `sighash` = lowercase hex of `keccak256(signature)`, 64 chars, no `0x`,
  produced by `event.SignatureKey` (D7).
- `signature` / `name` / `inputs` unchanged (`inputs`: `{type, indexed,
  name}`).
- Migration is a pure function exposed as `eventtool migrate` (§5.4):
  recompute from `signature`, assert the first 4 bytes equal the old
  `selector`, fail loud on any mismatch, write atomically. The tuple /
  `trcToken` entries hash their literal stored type strings — the same input
  the v1 generator used, so derivation is exact. The command is idempotent
  (an already-new file is verified and rewritten unchanged).
- `generate` and `insert` verify `signature == CanonicalSignature(name,
  types)` and `sighash == hex(SignatureKey(...))` on load and reject
  mismatches, so the file cannot silently drift from the registry.
- `store.Load` accepts only the new schema; an old `selector` file gets an
  explicit "run `eventtool migrate`" error, never silent reinterpretation.
- The 7 seed ABIs move to `internal/eventdata/abi/` and are tracked with the
  corpus (D2).

## 4. `builtin_gen.go` re-key

- `builtinSig4 map[[4]byte]*Definition` becomes
  `builtinSig map[[32]byte]*Definition`, keys as 32-byte array literals
  (`event.SignatureKey` output), entries sorted by sighash hex for stable
  diffs.
- `init()` keeps insert-if-absent `registerBuiltin` semantics
  (order-independent: 747 distinct signatures).
- Provenance header rewritten: generated by `cmd/eventtool generate` from
  `internal/eventdata/events_registry.json`; the v1-vendored wording is
  removed.
- `event/builtin.go` (explicit TRC-20 pair) untouched — it already derives
  keys from definitions.
- `event/registry.go`: add the D7 helpers; `sigKeyOf` and
  `Definition.signature()` delegate to them; update the `registerBuiltin` doc
  comment (no more "generated entry stored under a selector" wording).
  No decode path changes.

## 5. `cmd/eventtool`

Layout: `cmd/eventtool/main.go` (flags + wiring only) and
`internal/eventtool/{store,contracts,capture,abi,migrate,generate}.go`.

Dependencies: stdlib (`net/http`, `encoding/json`, `go/format`) + `tronlib`
root (`Dial`) + `rpc.GetContract` + `event` (pure derivation helpers).
Importing `event` runs its builtin `init()` in-process; the tool never
*mutates* the registry and never decodes logs.

Shared store: `map[sighash-hex]SavedEvent` with `Load` (new schema only),
atomic `Save` (tmp + rename), `Upsert` (insert-if-absent, returns bool).

### 5.1 `contracts [--limit=100 --out=internal/eventdata/top_contracts.json --api=...api/contracts]`

1. GET the ranking in 50-row pages until `--limit` (100) rows are collected,
   pacing requests to stay under TronScan's 5 req/s (2 requests here).
2. Write the snapshot atomically in rank order:

   ```json
   {
     "source": "https://apilist.tronscanapi.com/api/contracts?sort=-trxCount",
     "rank_by": "trxCount",
     "fetched_at": "2026-09-30T13:30:00Z",
     "limit": 100,
     "contracts": [
       {"rank": 1, "address": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
        "name": "TetherToken", "trxCount": 3644148431, "verify_status": 2}
     ]
   }
   ```

3. Uses the public endpoint with no API key (works anonymously); sends the
   key as a header when `TRONSCAN_API_KEY` is set. Prints
   `N contracts (rank 1..N by trxCount)`.

### 5.2 `capture [--node=grpc://127.0.0.1:50051 --in=internal/eventdata/top_contracts.json --out=internal/eventdata/events_registry.json]`

1. Load the snapshot (§5.1 output). Refuses to run if it is absent or
   malformed — it does not fall back to a live fetch (D4 determinism).
2. `Dial` the node. For each contract in rank order: `rpc.GetContract` once →
   keep ABI entries with `type == "Event"` and a non-empty `name` (anonymous
   events are skipped: they have no topic0 and cannot be signature-decoded) →
   canonical `CanonicalSignature(name, types)` → `Upsert` (D5: whole ABI, D6:
   first-wins). An address whose ABI carries no event entries is counted
   skipped, not fatal (defensive: TRON contracts normally ship an ABI).
3. Sequential by default (100 calls); an optional `--concurrency` (default 1)
   may widen it without changing the result — upsert is order-independent.
4. Atomic save-on-insert; SIGINT/SIGTERM saves and stops.
5. Report: `contracts / with events / new events / skipped`. Exit 0.

Cost: 2 HTTP requests per `contracts` refresh; 100 `GetContract` calls per
`capture`. No block walk, no caller tallies, no time window.

### 5.3 `insert [--in (required) --out=internal/eventdata/events_registry.json]`

v1 loader port. Reads an ABI file (raw array or `{"abi":[...]}`), minimal
local JSON parse (name + inputs only), canonical signature + full-hash
upsert. Prints `N new event(s) added`. Same signature/sighash verification as
§3.

### 5.4 `migrate [--in=internal/eventdata/events_registry.json --out=same]`

Reads a v1 `{selector, ...}` corpus, derives `sighash` from each
`signature`, asserts the 4-byte prefix equals the stored `selector`, writes
the new schema atomically. Idempotent. This is the only sanctioned way the
committed corpus changes shape.

### 5.5 `generate [--in=internal/eventdata/events_registry.json --out=event/builtin_gen.go]`

Reads the corpus (sighash-verified), dedups by sighash first-wins, emits
`event/builtin_gen.go` through `go/format` (replaces v1's hand-rolled
`bufio` + string escaping). Sorted output. Prints counts.

## 6. Tests

- Update `TestBuiltinTableCountAndKeys` to the new table shape: 747 entries,
  each key `== event.SignatureKey(name, types)`, keys distinct, each
  reachable in the global registry. The 4-byte prefix assertions are removed.
- Update `TestDecodeBuiltinSubmitTransaction`
  (`event/event_test.go:274-282`), which also indexes the table by a 4-byte
  key, to the 32-byte key.
- New `internal/eventtool` tests:
  - `contracts`: parse a recorded TronScan response fixture
    (`internal/eventtool/testdata/contracts_page{1,2}.json`), assert 2-page
    paging at the 50-row cap and the snapshot schema/rank order.
  - `capture`: against a stub `GetContract` fetcher returning a
    `core.SmartContract_ABI` (functions + events + one anonymous event),
    assert only named events are upserted and first-wins holds; a second
    address with no event entries counts as skipped.
  - store round-trip + first-wins + old-schema rejection;
  - `migrate` on a fixture old corpus (prefix assertion fires on a corrupt
    entry, idempotent on a new one);
  - **corpus-vs-registry drift**: load the tracked corpus and assert every
    entry's `signature == CanonicalSignature(...)` and `sighash ==
    hex(SignatureKey(...))`, with fixtures `Transfer(address,address,uint256)`
    and `SubmitTransaction(uint256,address,uint256,bytes)`;
  - `generate` golden-output check (parses via `go/parser`, spot `go build`).
- New `event` test: `SignatureKey` agrees with `sigKeyOf` on fixtures (locks
  D7).
- Coverage/CI: logic in `internal/eventtool` is instrumented and covered. Add
  `cmd/eventtool` to the package exclusion list in
  `.github/workflows/test-coverage.yml` (alongside `cmd/examplecheck`,
  `cmd/tip491probe`) so the thin, untestable `main.go` stays out of the 80%
  denominator.
- Gate sequence for the change: build → vet → `test -race
  ./event/ ./internal/eventtool/` → full suite → lint → `docgen -check`.

## 7. Docs

- `event/doc.go`: source-of-truth wording for the corpus + tool commands.
- `docs/runbook.md`: the five `eventtool` commands (flags, the
  refresh-diff-capture-generate recipe, first-wins correction path, note that
  TronScan's contract list is an external dependency snapshotted into the
  repo).
- No `architecture.md` renumbering (section numbers are load-bearing for
  code comments); touch only if a section names the v1 generator.

## 8. Cleanup list

- Moved: `tmp/events_registry.json` → `internal/eventdata/events_registry.json`
  (then migrated in place to §3's schema, 747 entries); `tmp/abi/*.json` →
  `internal/eventdata/abi/`.
- Added (tracked data): `internal/eventdata/top_contracts.json`.
- Regenerated: `event/builtin_gen.go`.
- Added (code): `cmd/eventtool/main.go`; `internal/eventtool/{store,contracts,
  capture,abi,migrate,generate}_*.go`; `internal/eventtool/testdata/*`.
- Touched: `event/registry.go` (D7 helpers + comment), `event/event_test.go`,
  `event/doc.go`, `docs/runbook.md`,
  `.github/workflows/test-coverage.yml` (exclusion).
- Deleted: nothing beyond the `tmp/` move (v1 cmds already absent; the 4-byte
  fields above are the last remnants).
- Non-goals: decode-path changes, per-address scoping in the tool, pruning of
  stale builtin entries, live TronScan fetching inside `capture`, block-walk
  discovery.