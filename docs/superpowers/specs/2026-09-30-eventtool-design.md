# Event tooling v2: 32-byte selective capture — design

Date: 2026-09-30.
Status: rev 2 — awaiting re-review (owner). Rev 2 resolves the review
findings: the corpus moves out of the gitignored `tmp/`, signature
derivation is shared with the registry instead of re-implemented, the
caller/attribution rules are pinned, and the CI coverage/package split is
stated.
Scope: clean up the last v1 4-byte event concepts and rebuild the
capture/insert/generate loop on native 32-byte keys, with popularity-gated
selective capture. **No decode-*behavior* change** (`event/event.go`
untouched; `event/registry.go` gains only exported pure helpers, §4).

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

## 2. Decisions (owner-approved; D2/D7 amended in rev 2)

- D1 Single binary `cmd/eventtool` with `capture` / `insert` / `migrate` /
  `generate` subcommands (precedent: `cmd/docgen`). Reusable logic lives in
  `internal/eventtool/`; `cmd/eventtool/main.go` is flag parsing and wiring
  only (precedent: `internal/format`, `internal/compilecheck`). See §5/§6
  for why the split is load-bearing.
- D2 The corpus is **relocated to a tracked path, then migrated in place**:
  `internal/eventdata/events_registry.json`, new 32-byte schema, no 4-byte
  field retained. The 7 ABIs move from `tmp/abi/` to
  `internal/eventdata/abi/`. Nothing in this design may depend on a
  gitignored path.
- D3 `event/builtin_gen.go` regenerated to `map[[32]byte]*Definition` now.
- D4 Caller = **transaction sender** (tx owner joined by txid), threshold
  **≥10 distinct senders**, window **exactly 24h of chain time ending at the
  tip block's timestamp**, sub-threshold contracts **skipped entirely**.
  Owner/attribution rules are pinned in §5.1.
- D5 Gate/payload split: the caller count only **gates** a contract; once
  gated, `capture` ingests the contract's **entire ABI** (all named event
  entries), not just caller-triggered events.
- D6 Store stays additive across periodic runs (upsert-if-absent, first-wins).
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
- Migration is a pure function exposed as `eventtool migrate` (§5.3):
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
`internal/eventtool/{store,migrate,capture,abi,generate}.go`.

Dependencies: stdlib (`go/format`) + `tronlib` root (`Dial`) +
`rpc.GetBlockByNum2` / `rpc.GetTransactionInfoByBlockNum` / `rpc.GetContract`
+ `event` (pure derivation helpers) + `pb/core` + `google.golang.org/protobuf`
(`proto`, `types/known/anypb`, `reflect/protoreflect`). Importing `event`
runs its builtin `init()` in-process; the tool never *mutates* the registry
and never decodes logs.

Shared store: `map[sighash-hex]SavedEvent` with `Load` (new schema only),
atomic `Save` (tmp + rename), `Upsert` (insert-if-absent, returns bool).

### 5.1 `capture [--node=grpc://127.0.0.1:50051 --out=internal/eventdata/events_registry.json --min-callers=10 --sleep-ms=250 --workers=8]`

1. `Dial`, `ChainTip` → tip `N`; fetch the tip block, take `T` = its
   `blockTimeStamp`, so the window is exactly 24h of chain time ending at the
   tip.
2. Walk heights `N, N-1, ...`, fetching block AND infos per height (worker
   pool over heights), until the block's `blockTimeStamp < T-24h`. `--sleep-ms`
   is a **global** inter-dispatch throttle shared by all `--workers`, not a
   per-worker delay. A fetch that fails after a small bounded retry (3
   attempts, backoff) **aborts the run** with a non-zero exit and writes
   nothing: qualification is a property of the whole window, so a partial
   window would under-count. (Cross-run resumability is a non-goal; the
   documented fallback for a window too slow to walk is sampling — out of
   scope.)
3. Join by txid. The block carries `core.Transaction`s but no txid, so
   compute `txid = sha256(serialized raw_data)`, matching `TransactionInfo.id`.
   Caller = the tx's **owner**: decode the contract `parameter` `Any`
   (`anypb.UnmarshalNew`) and read its `owner_address` field generically via
   `protoreflect` (every TRON contract message that has a sender names it
   `owner_address`); txs whose parameter does not decode, or has no
   `owner_address`, are skipped.
   Attribution is to the **directly-called contract only** —
   `info.ContractAddress` — with `log[].address` used only as a fallback when
   `ContractAddress` is empty. Internal-call emitters do not earn a caller.
   Tally `contract → distinct-owner set` in memory.
4. Qualifier iff `len(owners) >= --min-callers` (default 10). Below
   threshold: nothing — no ABI fetch, no write, no cross-run state.
5. Per qualifier, `GetContract` once → keep the ABI entries with
   `type == "Event"` and a non-empty `name` (anonymous events are skipped:
   they have no topic0 and cannot be signature-decoded) → canonical
   `CanonicalSignature(name, types)` → `Upsert` (D5: whole ABI, D6:
   first-wins). Atomic save-on-insert; SIGINT/SIGTERM saves and stops.
6. Report: `heights / contracts seen / qualified / new events`. Exit 0.

Cost disclosure: ~2 RPCs/height (~57k per full 24h window at TRON's ~3s
blocks). `--workers` (default 8) bounds wall-time; the ABI fetch lands only
on qualifiers.

### 5.2 `insert [--in (required) --out=internal/eventdata/events_registry.json]`

v1 loader port. Reads an ABI file (raw array or `{"abi":[...]}`), minimal
local JSON parse (name + inputs only), canonical signature + full-hash
upsert. Prints `N new event(s) added`. Same signature/sighash verification as
§3.

### 5.3 `migrate [--in=internal/eventdata/events_registry.json --out=same]`

Reads a v1 `{selector, ...}` corpus, derives `sighash` from each
`signature`, asserts the 4-byte prefix equals the stored `selector`, writes
the new schema atomically. Idempotent. This is the only sanctioned way the
committed corpus changes shape.

### 5.4 `generate [--in=internal/eventdata/events_registry.json --out=event/builtin_gen.go]`

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
- New `internal/eventtool` tests: store round-trip + first-wins + old-schema
  rejection; `migrate` on a fixture old corpus (prefix assertion fires on a
  corrupt entry, idempotent on a new one); **corpus-vs-registry drift**:
  load the tracked corpus and assert every entry's `signature ==
  CanonicalSignature(...)` and `sighash == hex(SignatureKey(...))`, with
  fixtures `Transfer(address,address,uint256)` and
  `SubmitTransaction(uint256,address,uint256,bytes)`; `generate`
  golden-output check (parses via `go/parser`, spot `go build`).
- New `event` test: `SignatureKey` agrees with `sigKeyOf` on fixtures
  (locks D7).
- Coverage/CI: logic in `internal/eventtool` is instrumented and covered. Add
  `cmd/eventtool` to the package exclusion list in
  `.github/workflows/test-coverage.yml` (alongside `cmd/examplecheck`,
  `cmd/tip491probe`) so the thin, untestable `main.go` stays out of the 80%
  denominator.
- Gate sequence for the change: build → vet → `test -race
  ./event/ ./internal/eventtool/` → full suite → lint → `docgen -check`.

## 7. Docs

- `event/doc.go`: source-of-truth wording for the corpus + tool commands.
- `docs/runbook.md`: the four `eventtool` commands (flags, periodic-run
  recipe, first-wins correction path, fallback note).
- No `architecture.md` renumbering (section numbers are load-bearing for
  code comments); touch only if a section names the v1 generator.

## 8. Cleanup list

- Moved: `tmp/events_registry.json` → `internal/eventdata/events_registry.json`
  (then migrated in place to §3's schema, 747 entries); `tmp/abi/*.json` →
  `internal/eventdata/abi/`.
- Regenerated: `event/builtin_gen.go`.
- Added: `cmd/eventtool/main.go`; `internal/eventtool/{store,migrate,capture,
  abi,generate}_*.go`.
- Touched: `event/registry.go` (D7 helpers + comment), `event/event_test.go`,
  `event/doc.go`, `docs/runbook.md`,
  `.github/workflows/test-coverage.yml` (exclusion).
- Deleted: nothing beyond the `tmp/` move (v1 cmds already absent; the 4-byte
  fields above are the last remnants).
- Non-goals: decode-path changes, per-address scoping in the tool, count
  persistence across runs, sampling, pruning of stale builtin entries.