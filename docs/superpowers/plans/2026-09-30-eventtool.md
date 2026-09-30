# Event tooling v2 (32-byte selective capture) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace tronlib's v1 4-byte event remnants with a native 32-byte corpus and a small `cmd/eventtool` (`contracts`/`capture`/`insert`/`migrate`/`generate`) that rebuilds `event/builtin_gen.go` from a snapshotted TronScan top-100 contract list.

**Architecture:** `event` gains two exported pure helpers (`CanonicalSignature`, `SignatureKey`) so tool and registry derive hashes by construction; `tmp/events_registry.json` + 7 ABIs move to tracked `internal/eventdata/`; `internal/eventtool` holds all logic (hermetically tested), `cmd/eventtool/main.go` is flags + wiring only.

**Tech Stack:** Go 1.27.1, stdlib `net/http`/`go/format`/`encoding/json`, `golang.org/x/crypto/sha3` (already a dep), tronlib root facade + `rpc.GetContract`, `pb/core`.

**Spec:** `docs/superpowers/specs/2026-09-30-eventtool-design.md` (rev 3)

## Global Constraints

- `go 1.27.1`; **no new module dependencies** (stdlib + existing deps only).
- No decode-*behavior* change: `event/event.go` untouched; `event/registry.go` gains only the D7 helpers.
- The corpus and snapshot are **tracked** files under `internal/eventdata/`; nothing may depend on the gitignored `tmp/`.
- Coverage floor 80% (CI); `cmd/eventtool` joins the coverage exclusion list, logic lives in `internal/eventtool`.
- `sigKeyOf` / `Definition.signature()` must delegate to the new helpers (single hashing path) — drift is impossible by construction.
- Signature hash is **`sha3.NewLegacyKeccak256()`**, never `sha3.Sum256`.
- Default node endpoint is the local Envoy gRPC proxy: `grpc://127.0.0.1:50051` (Envoy config `~/envoy/envoy.yaml`, listener 50051 → 19 mainnet full nodes, no rate limits).

## Review Focus

- An **old-schema** corpus reaching `store.Load` — must fail with an explicit "run `eventtool migrate`" error, never silently reinterpret `selector`.
- TronScan **paging boundary**: `limit` caps at 50, so 100 rows = exactly two pages; a short/final page must stop cleanly, not loop.
- An ABI entry that is `Anonymous`, unnamed, or a non-event (function/constructor) — must be skipped without error.
- **Duplicate sighash** across two contracts — first-wins, deterministic, no error (D6).
- **Absent/malformed snapshot** reaching `capture` — must refuse, not fall back to a live TronScan fetch.

---

### Task 1: Shared signature derivation in `event`

**Files:**
- Create: `event/signature.go`
- Modify: `event/registry.go` (`Definition.signature`, `sigKeyOf`)
- Test: `event/signature_test.go`

**Interfaces:**
- Produces: `event.CanonicalSignature(name string, inputTypes []string) string`; `event.SignatureKey(name string, inputTypes []string) [32]byte`; private `hashSignature(signature string) [32]byte`.
- `Definition.signature()` returns `CanonicalSignature(d.Name, types)`; `sigKeyOf(s)` returns `sigKey(hashSignature(s))`.

- [ ] **Step 1: Write the failing test**

```go
func TestCanonicalSignature(t *testing.T) {
	if got := CanonicalSignature("Transfer", []string{"address", "address", "uint256"});
		got != "Transfer(address,address,uint256)" {
		t.Fatalf("got %q", got)
	}
	if got := CanonicalSignature("Ping", nil); got != "Ping()" {
		t.Fatalf("empty types: got %q", got)
	}
}

func TestSignatureKeyMatchesKeccak(t *testing.T) {
	types := []string{"uint256", "address", "uint256", "bytes"}
	k := SignatureKey("SubmitTransaction", types)
	want := crypto.Keccak256([]byte("SubmitTransaction(uint256,address,uint256,bytes)"))
	if !bytes.Equal(k[:], want) {
		t.Fatalf("SignatureKey = %x, want %x", k, want)
	}
	if sigKey(k) != sigKeyOf("SubmitTransaction(uint256,address,uint256,bytes)") {
		t.Fatal("SignatureKey and sigKeyOf disagree")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./event/ -run 'TestCanonicalSignature|TestSignatureKey' -v`
Expected: FAIL — `undefined: CanonicalSignature`.

- [ ] **Step 3: Implement `event/signature.go`**

Move the hashing primitive out of `registry.go`: `hashSignature` is the only keccak call; `CanonicalSignature` joins `name + "(" + strings.Join(types, ",") + ")"`; `SignatureKey` hashes the canonical form. Point `Definition.signature()` and `sigKeyOf` at them.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./event/ -run 'TestCanonicalSignature|TestSignatureKey' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add event/signature.go event/signature_test.go event/registry.go
git commit -m "feat(event): export CanonicalSignature/SignatureKey; one hashing path"
```

---

### Task 2: Tracked corpus + embed package

**Files:**
- Move: `tmp/events_registry.json` → `internal/eventdata/events_registry.json`; `tmp/abi/*.json` → `internal/eventdata/abi/`
- Create: `internal/eventdata/embed.go`
- Test: `internal/eventdata/embed_test.go`

**Interfaces:**
- Produces: `eventdata.RegistryJSON []byte` (`//go:embed events_registry.json`); the `abi/` dir (7 seed ABIs) and `top_contracts.json` (added in Task 7).

- [ ] **Step 1: Relocate the files (no code yet)**

```bash
mkdir -p internal/eventdata/abi
git mv 2>/dev/null; mv tmp/events_registry.json internal/eventdata/events_registry.json
mv tmp/abi/*.json internal/eventdata/abi/
```
(The files are untracked/gitignored, so plain `mv`; confirm `git check-ignore internal/eventdata/events_registry.json` exits non-zero.)

- [ ] **Step 2: Write the failing test**

```go
func TestEmbeddedRegistryIsPresent(t *testing.T) {
	if len(RegistryJSON) == 0 {
		t.Fatal("RegistryJSON is empty")
	}
	var v []map[string]any
	if err := json.Unmarshal(RegistryJSON, &v); err != nil {
		t.Fatalf("registry is not JSON: %v", err)
	}
	if len(v) != 747 {
		t.Fatalf("want 747 entries, got %d", len(v))
	}
}
```
(At this step the file still has the **old** schema — the assertion is only count/JSON, so it passes after Step 3.)

- [ ] **Step 3: Implement `internal/eventdata/embed.go`**

`package eventdata` with `//go:embed events_registry.json` → `RegistryJSON []byte`, plus a package doc comment naming the spec.

- [ ] **Step 4: Run and commit**

Run: `go test ./internal/eventdata/ -v` → PASS.
```bash
git add internal/eventdata
git commit -m "chore(eventdata): track events corpus + ABI seeds out of tmp/"
```

---

### Task 3: Store (`internal/eventtool/store.go`)

**Files:**
- Create: `internal/eventtool/store.go`, `internal/eventtool/store_test.go`, `internal/eventtool/doc.go`

**Interfaces:**
- Produces: `SavedInput{Type string; Indexed bool; Name string}`; `SavedEvent{Sighash, Signature, Name string; Inputs []SavedInput}`; `Store` with `Load(path string) (*Store, error)`, `(*Store).Save() error`, `(*Store).Upsert(SavedEvent) bool`, `(*Store).Len() int`, `(*Store).Events() []SavedEvent`.
- Consumes: `event.CanonicalSignature`, `event.SignatureKey`.

- [ ] **Step 1: Write failing tests** (`store_test.go`, table + round-trip)

```go
func TestStoreRoundTripAndFirstWins(t *testing.T) {
	dir := t.TempDir(); path := filepath.Join(dir, "r.json")
	s := New(path)
	first := SavedEvent{Sighash: hexKey("Ping", []string{"uint256"}), Signature: "Ping(uint256)",
		Name: "Ping", Inputs: []SavedInput{{Type: "uint256", Name: "n"}}}
	if !s.Upsert(first) { t.Fatal("first upsert should insert") }
	if s.Upsert(first) { t.Fatal("second upsert should be a no-op") }
	if err := s.Save(); err != nil { t.Fatal(err) }
	got, err := Load(path); if err != nil { t.Fatal(err) }
	if got.Len() != 1 || got.Events()[0].Signature != "Ping(uint256)" { t.Fatalf("got %+v", got.Events()) }
}

func TestStoreRejectsOldSchema(t *testing.T) {
	dir := t.TempDir(); path := filepath.Join(dir, "old.json")
	os.WriteFile(path, []byte(`[{"selector":"c0819c13","signature":"Ping(uint256)","name":"Ping","inputs":[]}]`), 0o644)
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "migrate") {
		t.Fatalf("want migrate-first error, got %v", err)
	}
}

func TestStoreRejectsSighashMismatch(t *testing.T) {
	// entry whose sighash is not keccak(signature) must be rejected on Load.
}
```

- [ ] **Step 2: Run → FAIL** (`go test ./internal/eventtool/ -run TestStore -v`; undefined symbols).

- [ ] **Step 3: Implement**

`Load` decodes `[]SavedEvent`, rejects a file whose first entry has a `selector` field (decode into a probe struct) or whose `Sighash` is empty; for each entry asserts `Signature == event.CanonicalSignature(Name, types)` and `Sighash == hex(event.SignatureKey(Name, types))`. `Save` writes `json.MarshalIndent` to `path+".tmp"` then `os.Rename` (0644), sorted by `Sighash`.

- [ ] **Step 4: Run → PASS.**

- [ ] **Step 5: Commit**

```bash
git add internal/eventtool
git commit -m "feat(eventtool): 32-byte store with old-schema rejection"
```

---

### Task 4: `migrate`

**Files:**
- Create: `internal/eventtool/migrate.go`, `internal/eventtool/migrate_test.go`

**Interfaces:**
- Produces: `Migrate(data []byte) ([]SavedEvent, error)` — pure; accepts v1 `{selector,...}` or an already-new corpus (verifies and returns it).
- Consumes: `event.CanonicalSignature`, `event.SignatureKey`.

- [ ] **Step 1: Failing tests**

```go
func TestMigrateDerivesSighashAndChecksSelector(t *testing.T) {
	in := []byte(`[{"selector":"c0819c13","signature":"FeesWithdrawn(address,uint256)","name":"FeesWithdrawn","inputs":[{"type":"address","indexed":false,"name":"to"},{"type":"uint256","indexed":false,"name":"amount"}]}]`)
	got, err := Migrate(in); if err != nil { t.Fatal(err) }
	if len(got) != 1 || got[0].Sighash != hexKey("FeesWithdrawn", []string{"address","uint256"}) { t.Fatalf("%+v", got) }
}
func TestMigrateRejectsBadSelector(t *testing.T) {
	// selector "deadbeef" against FeesWithdrawn must error.
}
func TestMigrateIsIdempotent(t *testing.T) {
	// output of the first call fed back in returns identical entries.
}
```

- [ ] **Step 2: Run → FAIL.**
- [ ] **Step 3: Implement** the pure rewrite; prefix check `hex(SignatureKey(...))[:8] == strings.ToLower(selector)`, dedup by sighash first-wins.
- [ ] **Step 4: Run → PASS.**
- [ ] **Step 5: Commit** `feat(eventtool): migrate v1 selector corpus to 32-byte schema`.

---

### Task 5: `insert`

**Files:** Create `internal/eventtool/insert.go`, `insert_test.go`.

**Interfaces:**
- Produces: `InsertABI(data []byte, s *Store) (int, error)` — reads a raw Solidity ABI array or `{"abi":[...]}`, keeps `type=="event" && name!="" && !anonymous`, canonical + upsert; returns inserted count.
- Consumes: `Store.Upsert`, `event.CanonicalSignature`/`SignatureKey`.

- [ ] **Step 1: Failing test** — an ABI with one function, one anonymous event, one named event → exactly one insert; re-running inserts 0.
- [ ] **Step 2: Run → FAIL.**
- [ ] **Step 3: Implement** with a minimal local JSON struct (`type,name,anonymous,inputs[{type,indexed,name}]`); unwrap `{"abi":...}` when the top level is an object.
- [ ] **Step 4: Run → PASS.**
- [ ] **Step 5: Commit** `feat(eventtool): insert (v1 ABI loader port)`.

---

### Task 6: `generate`

**Files:** Create `internal/eventtool/generate.go`, `generate_test.go`, `testdata/`.

**Interfaces:**
- Produces: `Generate(events []SavedEvent) ([]byte, error)` → gofmt'd `event/builtin_gen.go` source: `var builtinSig = map[[32]byte]*Definition{…}`, sorted by sighash, plus the existing insert-if-absent `init()` and the new provenance header.
- Consumes: `event.SignatureKey` (keys are pre-verified in the corpus).

- [ ] **Step 1: Failing test**

```go
func TestGenerateParsesAndKeys(t *testing.T) {
	src, err := Generate([]SavedEvent{{Sighash: hexKey("Ping", []string{"uint256"}), Signature: "Ping(uint256)",
		Name: "Ping", Inputs: []SavedInput{{Type: "uint256", Name: "n"}}}})
	if err != nil { t.Fatal(err) }
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "builtin_gen.go", src, 0)
	if err != nil { t.Fatalf("generated source does not parse: %v", err) }
	if !strings.Contains(string(src), "map[[32]byte]*Definition") { t.Fatal("wrong table type") }
	if !strings.Contains(string(src), "// Code generated by cmd/eventtool generate") { t.Fatal("missing provenance") }
	_ = f
}
```

- [ ] **Step 2: Run → FAIL.**
- [ ] **Step 3: Implement** a `text/template` → `format.Source`; emit `[32]byte{0x..}` keys, `strconv.Quote`d names/types, `Indexed` only when true (mirror the v1 table's shape).
- [ ] **Step 4: Run → PASS.**
- [ ] **Step 5: Commit** `feat(eventtool): generate builtin_gen.go through go/format`.

---

### Task 7: `contracts` (TronScan snapshot)

**Files:** Create `internal/eventtool/contracts.go`, `contracts_test.go`, `testdata/contracts_page1.json`, `testdata/contracts_page2.json`.

**Interfaces:**
- Produces: `ContractEntry{Rank int; Address, Name string; TrxCount uint64; VerifyStatus int}`; `Snapshot{Source, RankBy, FetchedAt string; Limit int; Contracts []ContractEntry}`; `FetchTop(ctx context.Context, hc *http.Client, baseURL string, limit int) (*Snapshot, error)`; `(*Snapshot).WriteTo(path string) error`; `eventdata.TopContractsJSON` embed.
- Snapshot JSON shape is the spec's §5.1 block verbatim.

- [ ] **Step 1: Failing test** — `FetchTop` against an `httptest.Server` serving `testdata/contracts_page{1,2}.json` (recorded real responses) with `limit=100`: asserts two requests (`start=0`,`start=50`), 100 ranked entries, `rank` 1..100, `RankBy=="trxCount"`; and with `limit=50` exactly one request.
- [ ] **Step 2: Run → FAIL.**
- [ ] **Step 3: Implement** paging at `pageSize=50` with a 250 ms pace between pages (`http.NewRequestWithContext`, `Accept: application/json`, optional `TRONSCAN_API_KEY` header); decode `{data:[{address,name,trxCount,verify_status}]}`; `FetchedAt = time.Now().UTC().Format(time.RFC3339)`; atomic `WriteTo`.
- [ ] **Step 4: Run → PASS.**
- [ ] **Step 5: Record real fixtures**

```bash
curl -sS 'https://apilist.tronscanapi.com/api/contracts?sort=-trxCount&start=0&limit=50'  > internal/eventtool/testdata/contracts_page1.json
curl -sS 'https://apilist.tronscanapi.com/api/contracts?sort=-trxCount&start=50&limit=50' > internal/eventtool/testdata/contracts_page2.json
```

- [ ] **Step 6: Add `top_contracts.json` to the embed package** (`//go:embed top_contracts.json`) — the file is committed in Task 10, so add the embed there; here only extend `WriteTo`'s default path.

- [ ] **Step 7: Commit** `feat(eventtool): snapshot TronScan top-N contracts by trxCount`.

---

### Task 8: `capture`

**Files:** Create `internal/eventtool/capture.go`, `capture_test.go`, `internal/eventtool/fetch.go`.

**Interfaces:**
- Produces: `ABIFetcher interface { ABI(ctx context.Context, addr tron.Address) (*core.SmartContract_ABI, error) }`; `CaptureReport{Contracts, WithEvents, NewEvents, Skipped int}`; `Capture(ctx context.Context, snap *Snapshot, f ABIFetcher, s *Store, concurrency int) (CaptureReport, error)`; `RPCFetcher` wrapping `rpc.ConnProvider` (`NewRPCFetcher(cp rpc.ConnProvider) *RPCFetcher`).
- Consumes: `Snapshot`, `Store.Upsert`, `rpc.GetContract`, `tron.ParseAddress`.

- [ ] **Step 1: Failing tests**

```go
func TestCaptureFiltersAndUpserts(t *testing.T) {
	// stub fetcher returns an ABI with: 1 function, 1 anonymous event,
	// 1 named event "Ping(uint256)"; snapshot has one contract.
	// assert report{Contracts:1, WithEvents:1, NewEvents:1, Skipped:0}
	// and store holds exactly Ping.
}
func TestCaptureFirstWinsAcrossContracts(t *testing.T) {
	// two contracts, same Ping signature -> NewEvents 1, second counts skipped-by-dup implicitly (NewEvents only).
}
func TestCaptureSkipsContractWithoutEvents(t *testing.T) {
	// ABI with only functions -> Skipped 1, NewEvents 0.
}
```

- [ ] **Step 2: Run → FAIL.**
- [ ] **Step 3: Implement** `Capture`: sequential when `concurrency<=1`; otherwise a bounded worker pool fetching into `abis[rank]`, then a **rank-ordered** upsert pass (determinism). `abiEvents` skips non-`Event` entries, `Anonymous`, or empty `Name`. `Skipped` counts addresses whose ABI yields no events; a fetch error is returned (no silent skip).
- [ ] **Step 4: Run → PASS.**
- [ ] **Step 5: Commit** `feat(eventtool): capture ABIs for snapshotted contracts`.

---

### Task 9: `cmd/eventtool`

**Files:** Create `cmd/eventtool/main.go`.

**Interfaces:**
- Consumes everything above + `tronlib.Dial`.
- Produces the CLI: `eventtool contracts|capture|insert|migrate|generate`, defaults `--node=grpc://127.0.0.1:50051`, `--out=internal/eventdata/events_registry.json`, `--in=internal/eventdata/top_contracts.json`, `--snapshot=internal/eventdata/top_contracts.json`, `--min` not present. `capture --concurrency` defaults 1.

- [ ] **Step 1: Implement** flag sets per subcommand (mirroring `cmd/docgen`'s `flag.NewFlagSet` shape); `capture` dials, builds `NewRPCFetcher(cli.Raw())`, loads the snapshot (`Load`), runs `Capture`, saves. Fail non-zero with a one-line message on any error; print the report line.
- [ ] **Step 2: Build + help smoke**

Run: `go build ./cmd/eventtool && go run ./cmd/eventtool 2>&1 | head`
Expected: build OK; usage listing the five subcommands.

- [ ] **Step 3: Commit** `feat(cmd): eventtool CLI (contracts/capture/insert/migrate/generate)`.

---

### Task 10: Regenerate the table + update `event` tests

**Files:** Regenerate `event/builtin_gen.go`; modify `event/event_test.go`.

**Interfaces:** Consumes `eventtool generate` against the migrated corpus.

- [ ] **Step 1: Migrate + generate for real**

```bash
go run ./cmd/eventtool migrate --in internal/eventdata/events_registry.json --out internal/eventdata/events_registry.json
go run ./cmd/eventtool generate --in internal/eventdata/events_registry.json --out event/builtin_gen.go
grep -c '0x' event/builtin_gen.go | head -1    # sanity: many keys present
grep -m1 'map\[\[32\]byte\]' event/builtin_gen.go
```

- [ ] **Step 2: Update `TestBuiltinTableCountAndKeys`** — iterate `builtinSig`; assert 747, key `== event.SignatureKey(def.Name, inputTypes(def))`, distinct, and `globalDef(key) == def`.
- [ ] **Step 3: Update `TestDecodeBuiltinSubmitTransaction`** — key by `event.SignatureKey("SubmitTransaction", []string{"uint256","address","uint256","bytes"})`.
- [ ] **Step 4: Run**

Run: `go test ./event/ -count=1`
Expected: PASS (full event suite).

- [ ] **Step 5: Commit** `refactor(event): regenerate builtin table on 32-byte keys`.

---

### Task 11: CI exclusion + docs

**Files:** Modify `.github/workflows/test-coverage.yml`, `event/doc.go`, `docs/runbook.md`.

- [ ] **Step 1: CI** — append `| grep -v 'cmd/eventtool$'` to the coverage package list, with a one-line comment matching the existing exclusion rationale.
- [ ] **Step 2: `event/doc.go`** — state the corpus source (`internal/eventdata/events_registry.json`), the generated file, and `cmd/eventtool` as the regeneration path.
- [ ] **Step 3: `docs/runbook.md`** — a "Event corpus — `cmd/eventtool`" section: the five commands, the refresh→capture→generate recipe, the first-wins correction path (hand-edit corpus → regenerate), and the note that TronScan's list is an external dependency snapshotted into the repo.
- [ ] **Step 4: docgen drift + gates**

Run: `go run ./cmd/docgen sync-docs -pkg ./tron -example-pkg . -docs ./docs/errors.md -docs ./docs/examples.md -check`
Expected: no drift.

- [ ] **Step 5: Commit** `docs(eventtool): runbook, doc.go, CI coverage exclusion`.

---

### Task 12: Live verification against Envoy + ledger

**Files:** Modify `docs/verification.md`, `docs/runbook.md` (counts if needed).

- [ ] **Step 1: Start Envoy** (`~/envoy` gRPC proxy → mainnet full nodes)

```bash
docker compose -f ~/envoy/compose.yaml up -d
sleep 3; (exec 3<>/dev/tcp/127.0.0.1/50051 && echo envoy-up)
```

- [ ] **Step 2: Snapshot + capture live**

```bash
go run ./cmd/eventtool contracts --limit 100 --out internal/eventdata/top_contracts.json
go run ./cmd/eventtool capture --node grpc://127.0.0.1:50051 \
  --in internal/eventdata/top_contracts.json --out internal/eventdata/events_registry.json
go run ./cmd/eventtool generate --in internal/eventdata/events_registry.json --out event/builtin_gen.go
```

- [ ] **Step 3: Record `R14`** in `docs/verification.md` (exact commands, counts: contracts / with events / new events / skipped, corpus total, builtin entries), then re-run the full release gate sequence from `docs/runbook.md` (build, vet, `test -race ./...`, golangci-lint, coverage, docgen `-check`).
- [ ] **Step 4: Commit** `test(eventtool): live capture against Envoy (R14) + regenerated table`.

---

## Self-Review

- **Spec coverage:** §3–§5 map to Tasks 3–9; §4 to Tasks 1/10; §6 to Tasks 3–8 tests + Task 11 CI; §7 to Task 11; §8 to Tasks 2/7/10. Live proof is Task 12.
- **Type consistency:** `SavedEvent`/`SavedInput` defined in Task 3 and consumed unchanged by 4–6, 8–10; `Snapshot`/`ContractEntry` in Task 7 consumed by 8–9; `ABIFetcher`/`CaptureReport` in Task 8 consumed by 9.
- **Review Focus** items each have an owning test: old-schema (Task 3), paging boundary (Task 7), anonymous/unnamed/non-event (Tasks 5, 8), duplicate sighash (Task 8), absent snapshot (Task 9 refuses via `Load`).