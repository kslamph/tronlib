# tronlib v2 — Runbook (maintainer operations)

How to verify the library against live nodes, regenerate the generated
docs, and run the release gates. Evidence records for specific runs live in
[verification.md](verification.md); this is the how-to. Design rationale:
[architecture.md](architecture.md); review records and deferred work:
[review.md](review.md).

## Live verification — `cmd/examplecheck`

`examplecheck` walks every flow the documented examples teach against a live
node. Spend-free modes need no funded key; `-broadcast` spends real (testnet)
TRX.

```bash
go run ./cmd/examplecheck                  # fresh signer: reads, builds, local
                                           # signing, simulate, pricing, envelope,
                                           # sign-weight — 21 OK / 17 notes / 0 failed
                                           # (CostPreview notes insufficient_bandwidth:
                                           # a fresh account has no bandwidth, no
                                           # balance — the fiction rule at work)
go run ./cmd/examplecheck -key <hex>       # existing owner: same, plus permission
                                           # reads and state-dependent builds —
                                           # 34 OK / 4 notes / 0 failed (R9/R11/R13)
```

### Broadcasting (`-broadcast`, spends TRX)

The throwaway Nile keys are committed on the legacy branch on purpose:

```bash
git show v1-legacy:integration_test/test.env   # NILE_TEST_KEY1 / NILE_TEST_KEY2
```

```bash
export K1=<hex> K2=<hex>
go run ./cmd/examplecheck -key "$K1" \
  -payee TLibCZ2i2dFp6a9KZeKriSms5peeXSibks -payee-key "$K2" \
  -token TWRvzd6FQcsyp7hwCtttjZGpU1kfvVEtNK -broadcast
```

Every state change is paired with its reversal; the run asserts positions
are restored and only fees burnt. Measured cost ≈ 9–10.4 TRX for ~12
broadcasts (≈0.345 TRX bandwidth each + bought energy; a TRC-20 call from an
un-staked account buys ≈2 TRX of energy).

Flags that matter:

| Flag | Effect |
|---|---|
| `-leave-unstaked` | skips `CancelUnstake`, leaving the unstake in cooldown so a later run proves `WithdrawUnstaked` once it matures (1 day Nile / 14 Mainnet) |
| `-permission-update` | broadcasts a real permission update: add one active permission → verify on-chain (owner permission asserted unchanged) → submit the original set back. Burns the governance fee (100 TRX) **twice**; the price gate refuses below 2×fee + 2 TRX reserve |
| `-float N` | payee top-up band (default 3 TRX — enough for a TRC-20 call from an un-staked account) |
| `-payee` / `-payee-key` / `-token` | counterparty and TRC-20 contract for the token flows |

Notes vs failures: the harness treats node-side state validation (nothing
pending to withdraw/cancel/undelegate, no rewards, account does not exist,
insufficient bandwidth in read-only mode) as **notes**; SDK misbehaviour is
the only FAIL class.

## Energy / TIP-491 penalty replay — `cmd/tip491probe`

```bash
# exact energy+bandwidth assertion for any historical txid (no key, no spend):
go run ./cmd/tip491probe -endpoint grpc://grpc.trongrid.io:50051 -replay <txid>

# factor + penalty cross-check for any contract (R1):
go run ./cmd/tip491probe -endpoint grpc://grpc.trongrid.io:50051 \
  -contract TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t -owner <addr> -data <calldata-hex>
```

A stale replay (sender moved its funds) reports "comparison void", not a
mismatch — that is correct behaviour (E6). The public gateway rate-limits;
the probe retries read-only steps.

## Event corpus — `cmd/eventtool`

`event/builtin_gen.go` (the zero-config built-in table, based on the 747-entry
curated corpus) is generated from the tracked corpus
`internal/eventdata/events_registry.json`. `cmd/eventtool` maintains both,
against the local Envoy gRPC proxy `~/envoy` (listener `grpc://127.0.0.1:50051`,
round-robining 19 mainnet full nodes with no rate limits) or any node:

```bash
# 1. refresh the contract ranking (TronScan top-100 by call volume; snapshotted)
go run ./cmd/eventtool contracts --limit 100 \
  --out internal/eventdata/top_contracts.json

# 2. fetch those contracts' on-chain ABIs into the corpus
go run ./cmd/eventtool capture --node grpc://127.0.0.1:50051 \
  --in internal/eventdata/top_contracts.json \
  --out internal/eventdata/events_registry.json

# 3. re-render the built-in table
go run ./cmd/eventtool generate --in internal/eventdata/events_registry.json \
  --out event/builtin_gen.go
```

| Command | Purpose |
|---|---|
| `contracts` | Snapshot the TronScan top-N ranking (`--limit`, `--api` to override). Two 50-row pages (the API caps `limit` at 50). Every TRON contract carries an on-chain ABI whether or not its source is verified, so there is no verification filter. |
| `capture` | `GetContract` each snapshotted address and upsert its named, non-anonymous events (`--concurrency`). Reads the snapshot file only — never TronScan. |
| `insert` | Add events from one ABI file (`--in`, raw array or `{"abi":[...]}`). |
| `migrate` | Rewrite a v1 `{selector,...}` corpus into the 32-byte schema (asserts `keccak(signature)[:4] == selector`; idempotent). |
| `generate` | Render `event/builtin_gen.go` from the corpus through `go/format`. |

The corpus is **first-wins**: a signature already on file is never overwritten
by a later capture, so a bad entry is corrected by hand-editing
`internal/eventdata/events_registry.json` and re-running `generate`. The
snapshot records TronScan's ranking into the repo, so a commit reproduces the
same corpus; only `contracts` talks to TronScan.

## Generated docs — `cmd/docgen`

`docs/errors.md` and `docs/examples.md` are generated from source; hand
edits between the markers are reverted:

```bash
# regenerate:
go run ./cmd/docgen sync-docs -pkg ./tron -example-pkg . \
  -docs ./docs/errors.md -docs ./docs/examples.md

# CI drift gate (write nothing, byte-compare):
go run ./cmd/docgen sync-docs -pkg ./tron -example-pkg . \
  -docs ./docs/errors.md -docs ./docs/examples.md -check
```

## Release gate sequence

Run in this order; a failure anywhere stops the release:

```bash
go build ./...          && \
go vet ./...            && \
go test -race ./...     && \
golangci-lint run       && \
go test ./... -coverprofile=cover.out   # coverage floor 80% (CI-enforced)  && \
go run ./cmd/docgen sync-docs -pkg ./tron -example-pkg . \
  -docs ./docs/errors.md -docs ./docs/examples.md -check
```

`cmd/examplecheck` and `cmd/tip491probe` are excluded from the coverage
denominator (live-node harnesses cannot be covered hermetically) — their
pure helpers keep hermetic tests.

## Ledger conventions — writing into verification.md

- `E<n>` = broadcast evidence (spend transactions); `R<n>` = read-only
  evidence or a recorded run. Numbering is monotonic; never reuse.
- Every entry carries the exact command or txid that reproduces it, and the
  date. Records are **history, not gates**: nothing in the ledger runs in CI.
- When a later fact contradicts a record, **amend with an audit note**
  (see E2's mis-transcription fix) — do not silently rewrite.
- §3 "Negative records" is the honest list of what is NOT proven; a row is
  closed by striking it through and naming the entry that resolved it.
