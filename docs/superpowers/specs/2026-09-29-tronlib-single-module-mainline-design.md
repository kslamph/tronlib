# TronLib: v2 becomes the mainline as a single root module

- Date: 2026-09-29
- Status: Awaiting review
- Supersedes (layout only): the nested `v2/` module introduced in
  `docs/superpowers/specs/2026-08-31-tronlib-v2-design.md`

## 1. Context

`github.com/kslamph/tronlib` currently ships two Go modules in one repo:

| Location | Module path | Published as |
|---|---|---|
| repo root | `github.com/kslamph/tronlib` | v1.3.0 (legacy) |
| `v2/` | `github.com/kslamph/tronlib/v2` | not yet tagged (`v2/v2.0.0` planned) |

v1 is retired: there will be no further v1 releases. The owner wants the v2
codebase to be the mainline and wants consumers to depend on a path **without**
the `/v2` suffix.

## 2. Decision

Ship the v2 codebase as the **single root module** `github.com/kslamph/tronlib`,
released as **`v1.4.0`**, and delete the v1 code.

Because a `v2.0.0` version requires the module path to end in `/v2`
(Go's import-compatibility rule, verified via
`golang.org/x/mod/module.CheckPathMajor`), a suffix-less path can only carry
`v1.x.y` versions. The project keeps the name "TronLib v2" in prose; the Go
version is `v1.4.0`.

### Alternatives rejected

- **Root module `.../tronlib/v2`, tag `v2.0.0`** — keeps `/v2` in the import
  path. Rejected: owner explicitly wants no `/v2`.
- **Suffix-less path + `v2.0.0`** — impossible with a `go.mod`. Only
  `v2.0.0+incompatible` is accepted at a suffix-less path, and it requires the
  module to have **no** `go.mod` (GOPATH era). Rejected: loses modules,
  checksums, reproducibility.
- **Keep the nested two-module layout** — rejected: v1 is abandoned.

## 3. Target module identity

| Property | Value |
|---|---|
| Module path | `github.com/kslamph/tronlib` |
| Release tag | `v1.4.0` (annotated, no subdir prefix) |
| `go` directive | `1.27.1` (inherited from the v2 module) |
| Import of root facade | `github.com/kslamph/tronlib` |
| Import of packages | `github.com/kslamph/tronlib/{contract,key,rpc,tx,tron,token,event}` |
| Import of protobufs | `github.com/kslamph/tronlib/pb/{api,core}` — **unchanged** |

## 4. Repository layout

```
/                       go.mod  module github.com/kslamph/tronlib (go 1.27.1)
                        README.md (rewritten)  LICENSE  SECURITY.md  CONTRIBUTING.md
                        CODING_STANDARDS.md  .golangci.yml  codecov.yml  context7.json
                        .github/workflows/test-coverage.yml
                        pb/                     (unchanged location; now in-module)
                        protos/                 (git submodule, unchanged)
                        scripts/proto-gen.sh    (unchanged)
                        contract/ event/ internal/ key/ rpc/ token/ tron/ tx/
                        cmd/docgen/  cmd/tip491probe/
                        docs/errors.md docs/examples.md docs/verification.md
                        docs/reviews/ docs/superpowers/
                        doc.go tronlib.go network.go + *_test.go
                        PHASE2.md
```

Removed: `v2/` (its contents move up), v1 `pkg/ example/ internal/ cmd/
integration_test/`, v1 user docs under `docs/*.md`, `v2/cmd/migrate/`,
`v2/docs/migration.md`.

## 5. File operations (history-preserving)

Order matters — delete v1 counterparts before moving v2 paths that collide
(`cmd/`, `internal/`, `docs/`, `go.mod`, `go.sum`).

1. **Delete v1 code** (git history preserves it):
   `pkg/`, `example/`, `internal/`, `cmd/`, `integration_test/`
2. **Delete v1 user docs**: `docs/API_REFERENCE.md`, `docs/account.md`,
   `docs/architecture.md`, `docs/client.md`, `docs/eventdecoder.md`,
   `docs/network.md`, `docs/quickstart.md`, `docs/resources.md`,
   `docs/shielded.md`, `docs/signer.md`, `docs/smartcontract.md`,
   `docs/trc10.md`, `docs/trc20.md`, `docs/types.md`.
   **Keep** `docs/superpowers/**` and `docs/reviews/**`.
3. **Delete remove-tool artifacts**: `v2/cmd/migrate/`, `v2/docs/migration.md`.
4. **Replace module files**: remove root `go.mod`/`go.sum`; move `v2/go.mod`,
   `v2/go.sum` to root.
5. **Move v2 packages to root**: `contract event internal key rpc token tron tx cmd`.
6. **Move v2 docs into `docs/`**: `errors.md`, `examples.md`, `verification.md`.
7. **Move v2 root files to root**: `doc.go tronlib.go network.go
   energy_cache_test.go example_test.go facade_test.go network_test.go`,
   `PHASE2.md`.
8. Remove the now-empty `v2/`.

`pb/` and `protos/` do not move.

## 6. Import rewrite

Replace the exact string `github.com/kslamph/tronlib/v2` with
`github.com/kslamph/tronlib` across **74 `.go` files** (the proto imports
`github.com/kslamph/tronlib/pb/...` do not contain `/v2` and are untouched).

Also update `go.mod` module line, `.github/workflows/test-coverage.yml`,
`v2/docs/verification.md`, `PHASE2.md`, and `CODING_STANDARDS.md` where they
name the module or use `go -C v2`.

Historical artifacts under `docs/superpowers/specs|plans/` that describe the
old nested layout are left as a record; this spec supersedes them.

## 7. Module files

- Root `go.mod`: `module github.com/kslamph/tronlib`, `go 1.27.1`, drop the
  `require github.com/kslamph/tronlib v1.3.0` self-dependency and its explanatory
  comment (pb is now in-module).
- `go mod tidy` to reconcile `go.sum`.

## 8. CI (`.github/workflows/test-coverage.yml`)

Single module — remove all `go -C v2` usage and the v1 steps:

- keep: setup-go 1.27.1, `go mod download`, `go build ./...`,
  `go test -short -coverprofile=coverage.txt -covermode=atomic ./...`,
  coverage floor 80%, docgen drift check, govulncheck, codecov upload;
- change: docgen paths to `go run ./cmd/docgen sync-docs -pkg ./tron
  -example-pkg . -docs ./docs/errors.md -docs ./docs/examples.md -check`;
- remove: root-v1-only steps (`./example/...`, `./pkg/...`), the migrate drift
  gate, and the `working-directory: v2` on govulncheck.

## 9. Tooling

- `scripts/proto-gen.sh` — **no change**: it maps `go_package` to
  `github.com/kslamph/tronlib/pb/...`, which is still correct.
- `cmd/docgen` — no change to arguments; paths remain relative to the module root.
- `cmd/migrate` — deleted.
- `codecov.yml` — drop the v1-only `pkg/client/lowlevel/**` ignore.
- `.golangci.yml` — `local-prefixes: github.com/kslamph/tronlib` already correct.

## 10. Documentation

- Rewrite `README.md` for the v2 API (install line
  `go get github.com/kslamph/tronlib@v1.4.0`, quickstart, package map).
- Note prominently that `v1.4.0` is a **breaking** change shipped as a `v1`
  minor bump (Go cannot signal "major" without a `/v2` path).

## 11. Risks & mitigations

| Risk | Mitigation |
|---|---|
| `v1.4.0` silently breaks users running `go get -u` from v1.3.0 | Document loudly in README + release notes; consumers pinned to v1.3.0 are unaffected. |
| Consumers need Go ≥ 1.27.1 | Stated in README; `GOTOOLCHAIN=auto` downloads it transparently. |
| Loss of the v1→v2 migration guide | Accepted (v1 abandoned); tool + doc deleted. |
| Historic spec/plan docs now describe a stale layout | Left as record; this spec supersedes. |
| Coverage denominator grows (pb now in-module) | pb has no tests and no `-coverpkg`, so it is not in the profile; floor unchanged. |

## 12. Verification plan

- `go build ./...` and `go vet ./...` clean.
- `go test -short ./... -count=1` and `-race` — all packages pass (12 today).
- coverage ≥ 80%.
- `go run ./cmd/docgen sync-docs ... -check` green.
- `govulncheck ./...` → 0 reachable.
- `grep -rn 'tronlib/v2' --include=*.go .` returns nothing.
- `go list -m` reports `github.com/kslamph/tronlib`.
- Hermetic; the committed live-broadcast evidence in
  `docs/verification.md` (R1–R8, E1–E7) is unchanged.

## 13. Git & release (user-gated)

- Work stays local; **no push** until the owner approves.
- Recommended: merge the `v2` branch into `master`, then tag `v1.4.0` on
  `master`. Alternative: tag `v1.4.0` directly on the restructured `v2` branch.
- Release: `git push origin master && git push origin v1.4.0`.

## 14. Out of scope

- Any change to TronLib's public API, behaviour, or protobuf generation.
- Publishing/tagging (owner-gated).
- Rewriting the historical process docs under `docs/superpowers/`.
