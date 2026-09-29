# TronLib: v2 becomes the mainline as a single root module

- Date: 2026-09-29
- Status: Awaiting review (rev 3 — supersedes rev 2; rev 1 assumed a
  suffix-less path + `v1.4.0` tag)
- Supersedes (layout only): the nested `v2/` module introduced in
  `docs/superpowers/specs/2026-08-31-tronlib-v2-design.md`

## 1. Context

`github.com/kslamph/tronlib` currently ships two Go modules in one repo:

| Location | Module path | Published as |
|---|---|---|
| repo root | `github.com/kslamph/tronlib` | v1.3.0 (legacy) |
| `v2/` | `github.com/kslamph/tronlib/v2` | not yet tagged (`v2/v2.0.0` planned) |

v1 is retired. The v2 codebase becomes the mainline, living as the **single
module at the repo root**, and is released later as **`v2.0.0`**.

## 2. Decision

One root module, path **`github.com/kslamph/tronlib/v2`**, released as
**`v2.0.0`**. Move the `v2/` module contents to the repo root, delete the v1
code, and re-home the protobuf package under the v2 module path.

This keeps `/v2` in the import path. That is unavoidable: Go's
import-compatibility rule requires a `v2.0.0` version to be served by a module
whose path ends in `/v2` (verified via
`golang.org/x/mod/module.CheckPathMajor`). The benefit of moving to the repo
root is a **single module** and a plain tag (`v2.0.0`, not `v2/v2.0.0`).

**No tag is created in this work.** Details and documentation are finished
first; the `v2.0.0` release is a later, owner-gated step.

### Alternatives rejected

- **Suffix-less path `.../tronlib`** — cannot carry a `v2.0.0` tag; would force
  `v1.x` versioning. Rejected: the owner wants `v2.0.0`.
- **Keep the nested two-module layout** — rejected: v1 is abandoned and the
  owner wants one module.
- **Delete `go.mod` for `v2.0.0+incompatible`** — rejected: loses modules,
  checksums, reproducibility.

## 3. Target module identity

| Property | Value |
|---|---|
| Module path | `github.com/kslamph/tronlib/v2` (unchanged from today) |
| Release tag | `v2.0.0` (annotated, created later — not in this work) |
| `go` directive | `1.27.1` (unchanged) |
| Import of root facade | `github.com/kslamph/tronlib/v2` |
| Import of packages | `github.com/kslamph/tronlib/v2/{contract,key,rpc,tx,tron,token,event}` (unchanged) |
| Import of protobufs | `github.com/kslamph/tronlib/v2/pb/{api,core}` (**changed**) |

## 4. Repository layout

```
/                       go.mod  module github.com/kslamph/tronlib/v2 (go 1.27.1)
                        README.md (minimal fix now; full rewrite deferred)
                        LICENSE  SECURITY.md  CONTRIBUTING.md  CODING_STANDARDS.md
                        .golangci.yml  codecov.yml  context7.json
                        .github/workflows/test-coverage.yml
                        pb/                     (unchanged location, re-pointed imports)
                        protos/                 (git submodule, unchanged)
                        scripts/proto-gen.sh    (go_package mapping updated)
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

v1 source is **not lost**. It remains on the `master`/`origin/master` branches
and the `v1.3.0` tag, so removing it from the mainline is reversible via git
(recommended: also pin a lightweight `v1-legacy` branch at the last v1 commit
before starting). v1 Go files cannot simply stay at the root: the root `go.mod`
becomes the v2 module, so any leftover v1 directory would be absorbed as
`github.com/kslamph/tronlib/v2/<dir>`, built by v2 CI, and would drag in its
now-broken `.../tronlib/pb/...` imports.

1. **Delete v1 code** (git history preserves it):
   `pkg/`, `example/`, `internal/`, `cmd/`, `integration_test/`
2. **Delete v1 user docs**: `docs/API_REFERENCE.md`, `docs/account.md`,
   `docs/architecture.md`, `docs/client.md`, `docs/eventdecoder.md`,
   `docs/network.md`, `docs/quickstart.md`, `docs/resources.md`,
   `docs/shielded.md`, `docs/signer.md`, `docs/smartcontract.md`,
   `docs/trc10.md`, `docs/trc20.md`, `docs/types.md`.
   **Keep** `docs/superpowers/**` and `docs/reviews/**`.
3. **Delete removed-tool artifacts**: `v2/cmd/migrate/`, `v2/docs/migration.md`.
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

The module path is unchanged, so the 74 intra-module `.../v2/...` imports need
no edit. Only the protobuf path changes:

- `github.com/kslamph/tronlib/pb` → `github.com/kslamph/tronlib/v2/pb`

After v1 deletion, **67 files** still import the old pb path (64 under the
moved v2 tree + 3 generated files in `pb/api`); the other 69 matching files
are v1 code being deleted. (224 occurrences repo-wide today.)

Also update:

- `scripts/proto-gen.sh` — every `--go_opt` / `--go-grpc_opt` mapping from
  `github.com/kslamph/tronlib/pb/...` to `github.com/kslamph/tronlib/v2/pb/...`.
- `go.mod` module line stays `github.com/kslamph/tronlib/v2`.
- `.github/workflows/test-coverage.yml`, `v2/docs/verification.md`,
  `PHASE2.md`, `CODING_STANDARDS.md` where they use `go -C v2`.

Historical artifacts under `docs/superpowers/specs|plans/` that describe the
old nested layout are left as a record; this spec supersedes them.

## 7. Module files

- Root `go.mod`: `module github.com/kslamph/tronlib/v2`, `go 1.27.1`, drop the
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

- `scripts/proto-gen.sh` — update the `go_package` mappings (see §6).
- `cmd/docgen` — no change to arguments; paths remain relative to the module root.
- `cmd/migrate` — deleted.
- `codecov.yml` — drop the v1-only `pkg/client/lowlevel/**` ignore.
- `.golangci.yml` — `local-prefixes: github.com/kslamph/tronlib` already correct
  (matches `/v2` too); `pb`/`protos` remain excluded.

## 10. Documentation

- `README.md`: **rewritten to be v2-only** — v2 as if v1 never existed.
  Install line `go get github.com/kslamph/tronlib/v2@v2.0.0`, `/v2/...`
  imports, v2 quickstart and package map. No v1 wording and no "migrating from
  v1" content (the migration guide is deleted). The owner may still refine
  wording in the follow-up docs pass.
- All user-facing docs under `docs/` are v2-only. Only the internal process
  record under `docs/superpowers/**` and `docs/reviews/**` may mention the old
  layout; those are not published docs.
- Final release note (later): `v2.0.0` is the first release of the new
  single-module v2 line; v1 (`v1.3.0`) is the frozen legacy line.

## 11. Risks & mitigations

| Risk | Mitigation |
|---|---|
| Consumers currently pin `.../v2` module version — none are published yet | The old nested tag `v2/v2.0.0` was never created, so nothing depends on it. |
| Deleting the nested module changes the git tag shape | New plain tag `v2.0.0` at the repo root; no subdir prefix. |
| Consumers need Go ≥ 1.27.1 | Stated in README; `GOTOOLCHAIN=auto` downloads it transparently. |
| Loss of the v1→v2 migration guide | Accepted (v1 abandoned); tool + doc deleted. |
| Coverage denominator grows (pb now in-module) | pb has no tests and no `-coverpkg`, so it is not in the profile; floor unchanged. |
| A stale `github.com/kslamph/tronlib/pb` import survives the rewrite | Verification greps for it and fails if any remain. |

## 12. Verification plan

- `go build ./...` and `go vet ./...` clean.
- `go test -short ./... -count=1` and `-race` — all packages pass (12 today).
- coverage ≥ 80%.
- `go run ./cmd/docgen sync-docs ... -check` green.
- `govulncheck ./...` → 0 reachable.
- `grep -rn '"github.com/kslamph/tronlib/pb/' --include=*.go .` returns nothing.
- `go list -m` reports `github.com/kslamph/tronlib/v2`.
- Hermetic; the committed live-broadcast evidence in `docs/verification.md`
  (R1–R8, E1–E7) is unchanged.

## 13. Git & release

- Work stays local; **no push, no tag, no merge** until the owner approves.
- Before deleting v1, optionally pin `v1-legacy` at the last v1 commit (v1 is
  already preserved on `master`/`origin/master` and the `v1.3.0` tag).
- The `v2.0.0` tag is created later, at the repo root (plain `v2.0.0`), after
  the owner's details/docs pass.
- Release order when the owner is ready:
  `git push origin <branch>` then `git tag -a v2.0.0 -m "tronlib v2.0.0"`
  then `git push origin v2.0.0`.

## 14. Out of scope

- Any change to TronLib's public API, behaviour, or protobuf generation.
- Creating the `v2.0.0` tag or pushing.
- The owner's follow-up details/docs work.
- Rewriting the historical process docs under `docs/superpowers/`.
