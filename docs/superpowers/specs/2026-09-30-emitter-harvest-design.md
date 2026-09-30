# Emitter harvest: events from contracts we do not call — design

Date: 2026-09-30.
Status: awaiting review (owner).
Parent: `docs/superpowers/specs/2026-09-30-eventtool-design.md` (rev 3,
implemented; R14 in `docs/verification.md`).
Scope: add a log-driven harvest that finds **emitting** contracts — including
ones never directly called (proxy implementations, internal-call emitters) —
and feed the existing `capture` pipeline. No change to the corpus schema, the
store, the generator, or the decode paths.

## 1. Why: measured gap

The existing pipeline captures each contract's **declared** events from its
on-chain ABI. That part is complete and was verified live: for the 100
snapshotted contracts, **349 distinct declared signatures, 0 missing** from the
902-entry corpus.

Emissions are a different set. A 250-block probe (window
86696448..86696698) measured:

| Metric | Value |
|---|---|
| Log entries | 31,141 (94.1% emitted by the called contract, **5.9% by another address**) |
| Distinct emitting contracts | 123 — **112 not in the top-100** |
| Distinct signatures observed | 99 — **56 not in the corpus** |
| Snapshot contracts declaring zero events | **23/100** |
| …of those, observed actively emitting | **2** |

Two mechanisms account for the missing events:

- **Proxy delegatecall.** `TU2MJ5Veik…` (`MarketProxy`, rank 60) declares no
  events, yet emitted 148 logs carrying **6 signatures absent from the corpus**.
  In a delegatecall the logic runs in the proxy's storage context, so the logs
  are emitted under the **proxy's** address while the code (and its ABI) belongs
  to a **implementation contract that is never a call target**. Its chain is
  `MarketProxy → MarketG1 → T9yD14Nj…`, and every hop's **on-chain ABI is empty**.
- **Indirect emitters** reached through internal calls (routers → pools,
  factories → pairs). Of 99 observed signatures, 56 are unknown to us; roughly
  half of the sampled indirect emitters did have a usable on-chain ABI.

Constraint discovered: **there is no programmatic ABI fallback.** TronScan's
`/api/contract` returns metadata only (`is_proxy`, `proxy_implementation`), its
`/api/contract/abi` is empty, and the ABI lives only in the Cloudflare-protected
web UI. A `topic0` alone carries no parameter types and no indexed flags, so an
observed-but-untyped signature can never become a decoder. It must be
**reported**, not ingested.

Also measured: TronScan's public endpoint rate-limits aggressively (HTTP 429
returned even at ~3 requests/s in this session), so any per-contract metadata
scan needs pacing, retry, and graceful degradation.

## 2. Decisions (owner-approved)

- D1 New subcommand `eventtool emitters` walks `--hours` of blocks
  (**default 1**) back from the tip through the node and tallies emitters.
- D2 The harvest output `internal/eventdata/top_emitters.json` **reuses the
  existing `Snapshot` schema**, so `capture --in internal/eventdata/top_emitters.json`
  works with no new capture path.
- D3 **Recursive proxy follow is included**: `proxy_implementation` chains are
  resolved at snapshot time through TronScan (paced, retried, bounded depth),
  and the resolved implementations are added to the snapshot as extra entries,
  so `capture` still never talks to TronScan.
- D4 **Untypeable signatures are report-only**: observed sighashes that no
  obtainable ABI explains are written to
  `internal/eventdata/unresolved_signatures.json` and never enter the corpus.
- D5 The 7 tracked seed ABIs under `internal/eventdata/abi/` are
  **reference-only**; the runbook says so (they duplicate signatures already in
  the corpus and are read by no workflow).
- D6 Determinism holds: only `contracts`/`emitters` touch TronScan; `capture`
  reads snapshots plus the node.

## 3. `eventtool emitters`

```
eventtool emitters
  --node=grpc://127.0.0.1:50051
  --hours=1                  # window length, chain time
  --top=200                  # snapshot size, by log volume
  --concurrency=12           # block-walk workers
  --out=internal/eventdata/top_emitters.json
  --observed-out=internal/eventdata/observed_signatures.json
  --corpus=internal/eventdata/events_registry.json
```

1. `Dial`, `ChainTip` → tip block; `T` = tip block's timestamp. Walk heights
   downward until `blockTimeStamp < T - hours` (chain-time cutoff, not a block
   count — the same rule the parent spec adopted for capture).
2. Worker pool over heights: `GetTransactionInfoByBlockNum`. For every log whose
   `topics[0]` is exactly 32 bytes, tally the **emitter** — `log.address`
   normalized to 21 bytes by prepending `0x41` (log addresses are 20-byte
   EVM-style; `contract_address` and base58 are 21 — comparing them raw makes
   every log look indirect). Record per emitter: log count, distinct sighashes,
   and the sighash→count map.
3. Rank emitters by log count, then **keep only those emitting at least one
   sighash absent from `--corpus`** (`--top` caps the list). An emitter whose
   signatures are all known cannot add anything, and dropping it keeps the
   snapshot small and the capture cheap. (Proxy entries added in step 4 are not
   subject to this filter — their point is that the node decides whether they
   help.)
4. **Proxy follow (D3):** for each kept emitter, `GET /api/contract?contract=…`
   (paced ≤5 req/s, retried with backoff, 429-aware). If `is_proxy`, append
   `proxy_implementation` as an extra snapshot entry and recurse, guarded by a
   visited set and a depth cap of 4. If metadata is unavailable for an address,
   skip its proxy resolution and count it in the report; never fail the run.
5. Write `top_emitters.json` using the shared `Snapshot` shape
   (`rank_by: "logVolume"`, `source` naming the node and window), with two
   additive per-entry fields: `role` (`"emitter"` | `"proxy"`) and
   `proxy_for` (the address it implements, for `proxy` entries).
6. Write `observed_signatures.json`:
   `{fetched_at, window:{from_height,to_height,from_time,to_time},
   signatures:[{sighash, logs, emitters:[{address, logs}]}]}` sorted by count.
   This is the raw evidence the report step diffs against.
7. Report: `heights / logs / emitters / kept / proxy entries / observed
   signatures / new-to-corpus`. Exit 0.

Cost: `hours × 1200` block readings at ~3s blocks (1h ≈ 1,200 RPCs), plus one
TronScan metadata call per kept emitter (bounded by `--top`). Sampling is
documented as the fallback for large `--hours`; it is not implemented here.

## 4. `capture` changes (minimal)

- The snapshot may now contain `role`/`proxy_for`; `capture` ignores unknown
  JSON fields and fetches every entry's ABI unchanged. Proxy entries are just
  extra addresses.
- New optional flag `--observed=internal/eventdata/observed_signatures.json`:
  after ingesting, compute the sighashes that remain absent from the store and
  write `--unresolved-out` (default
  `internal/eventdata/unresolved_signatures.json`) as
  `{fetched_at, unresolved:[{sighash, logs, emitters:[{address, logs}]}]}`,
  deterministically sorted by sighash. Without `--observed`, behaviour is
  unchanged.
- `unresolved_signatures.json` is **never** read by `generate`; it is a gap
  report for maintainers.

## 5. Tests

- Emitter tally: fixture transaction-info lists → exact per-emitter log/sighash
  counts, correct ranking, and window cutoff by timestamp.
- **Log-address normalization**: a 20-byte log address and its 21-byte
  `contract_address` count as the same emitter (the measured bug class).
- Selection: an emitter whose every signature is already in the corpus is
  dropped; one with a single unknown signature is kept.
- Proxy follow: fixture metadata pages → chain resolved in order; cycle guard
  terminates; depth cap 4 enforced; a 429 then success retries; a permanent
  metadata failure skips the address without failing the run.
- Snapshot compatibility: a `top_emitters.json` (with `role`/`proxy_for`) loads
  through `LoadSnapshot` and through `capture` without schema changes.
- Unresolved report: observed sighashes minus store contents, deterministic
  ordering; a fully resolvable set writes an empty report, not a stale one.
- Gate sequence unchanged: build → vet → `test -race ./...` → lint → coverage →
  `docgen -check`.

## 6. Docs

- `docs/runbook.md`: extend the Event corpus section with the harvest recipe
  (`emitters` → `capture --observed` → `generate` → gate), the 1h default and
  `--hours` cost, the TronScan rate-limit caveat, and the reference-only status
  of `internal/eventdata/abi/`.
- `docs/verification.md`: `R15` recording a live harvest through Envoy with the
  counts and the corpus delta.
- `event/doc.go`: one sentence that the corpus is now fed by declared events
  (top-100) **and** observed emitters.

## 7. Non-goals

- Scraping the TronScan web UI for ABIs (Cloudflare-protected; ToS).
- Reconstructing a decodable definition from a raw `topic0` (impossible without
  types + indexed flags); 4byte naming is out of scope.
- A 24h default window, block sampling, or persisting harvest state across runs.
- Any change to the corpus schema, store, generator, or decode paths.
