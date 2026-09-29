# TronLib Single-Module Mainline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the v2 codebase the single Go module at the repo root (`github.com/kslamph/tronlib/v2`), remove v1 from the mainline, re-point the protobuf package, and make all docs v2-only — with no tag or push.

**Architecture:** The existing `v2/` module is lifted to the repo root; its module path is unchanged, so intra-module imports stay valid. Only the `pb/` import path changes (`.../tronlib/pb` → `.../tronlib/v2/pb`). v1 code and v1 user docs are removed from the mainline but preserved in git (`master`, `origin/master`, `v1.3.0`, plus a `v1-legacy` pointer).

**Tech Stack:** Go 1.27.1, gRPC, go-ethereum, protobuf; GitHub Actions.

**Spec:** `docs/superpowers/specs/2026-09-29-tronlib-single-module-mainline-design.md`

## Global Constraints

- Module path is `github.com/kslamph/tronlib/v2`; the release tag will be `v2.0.0` — **not created in this work**.
- `go` directive stays `1.27.1`.
- **No push, no tag, no merge.** Local commits only.
- Every command runs from the repo root as a single module (no `go -C v2`).
- Coverage floor: 80%.
- Do not change public API, behaviour, or protobuf generation.

## Review Focus

- A stale `github.com/kslamph/tronlib/pb` import surviving the rewrite (build break).
- `protos/` submodule contents being swept into `go build ./...`.
- Coverage denominator changing because `pb/` is now in-module.
- The docgen drift gate failing after the docs move.
- Untracked scratch files (`REPORT.md`, `REVIEW.md`, `.pi/`, …) being accidentally committed.

---

### Task 1: Cut over to the single root module (code)

**Files:**
- Delete: `pkg/`, `example/`, `internal/`, `cmd/`, `integration_test/`, `v2/cmd/migrate/`
- Move: `v2/{contract,event,internal,key,rpc,token,tron,tx,cmd}` → root; `v2/{doc.go,tronlib.go,network.go,energy_cache_test.go,example_test.go,facade_test.go,network_test.go,PHASE2.md,go.mod,go.sum}` → root; `v2/docs/{errors,examples,verification}.md` → `docs/`
- Modify: 67 `.go` files (pb import path), `scripts/proto-gen.sh`, `go.mod`

**Interfaces:**
- Produces: root module `github.com/kslamph/tronlib/v2`; package import paths `.../v2/{contract,key,rpc,tx,tron,token,event}` and `.../v2/pb/{api,core}`.

- [ ] **Step 1: Archive pointer + baseline**

```bash
cd /home/kslam/goproj/tronlib
git branch v1-legacy v1.3.0 2>/dev/null || true   # v1 pinned for reference
go -C v2 build ./... && go -C v2 test -short ./... -count=1
```
Expected: build OK, all packages `ok`.

- [ ] **Step 2: Remove v1 code and the migrate tool**

```bash
git rm -rq pkg example internal cmd integration_test v2/cmd/migrate
```

- [ ] **Step 3: Replace module files**

```bash
git rm -q go.mod go.sum
git mv v2/go.mod go.mod
git mv v2/go.sum go.sum
```

- [ ] **Step 4: Lift v2 packages and files to the root**

```bash
for d in contract event internal key rpc token tron tx cmd; do git mv "v2/$d" .; done
for f in doc.go tronlib.go network.go energy_cache_test.go example_test.go facade_test.go network_test.go PHASE2.md; do git mv "v2/$f" .; done
git mv v2/docs/errors.md docs/errors.md
git mv v2/docs/examples.md docs/examples.md
git mv v2/docs/verification.md docs/verification.md
git rm -q v2/docs/migration.md
```

- [ ] **Step 5: Drop the self-dependency from go.mod**

Remove the `github.com/kslamph/tronlib v1.3.0` require line and the surrounding
"consumes the root module's protobuf types" comment block.

- [ ] **Step 6: Re-point the protobuf imports**

```bash
grep -rl '"github.com/kslamph/tronlib/pb/' --include=*.go . \
  | xargs sed -i 's#github.com/kslamph/tronlib/pb#github.com/kslamph/tronlib/v2/pb#g'
sed -i 's#github.com/kslamph/tronlib/pb#github.com/kslamph/tronlib/v2/pb#g' scripts/proto-gen.sh
grep -rn '"github.com/kslamph/tronlib/pb/' --include=*.go . || echo CLEAN
```
Expected: `CLEAN`.

- [ ] **Step 7: Tidy and verify**

```bash
go mod tidy
go build ./... && go vet ./...
go test -short ./... -count=1
go list -m
```
Expected: build/vet clean, all packages `ok`, `go list -m` prints `github.com/kslamph/tronlib/v2`.

- [ ] **Step 8: Commit**

```bash
git add -A -- . ':!.agents' ':!.pi' ':!REPORT.md' ':!REVIEW.md' ':!docs/llms.txt' ':!docs/TRON_SDK_DESIGN.md' ':!skills-lock.json'
git commit -m "refactor: single root module .../v2 (remove v1, repoint pb)"
```

---

### Task 2: CI workflow for the single module

**Files:**
- Modify: `.github/workflows/test-coverage.yml`

- [ ] **Step 1: Rewrite the workflow**

One module: drop all `go -C v2`, the `/pkg` + `/example` steps, the migrate
gate, and `working-directory: v2`. Keep: setup-go 1.27.1, `go mod download`,
`go build ./...`, a coverage run over `go list ./... | grep -v '/pb/'` (pb is
generated and must not enter the coverage denominator), 80% floor, docgen
`-check`, govulncheck `./...`, codecov upload.

- [ ] **Step 2: Verify locally (the same commands CI runs)**

```bash
go run ./cmd/docgen sync-docs -pkg ./tron -example-pkg . -docs ./docs/errors.md -docs ./docs/examples.md -check
pkgs=$(go list ./... | grep -v '/pb/')
go test -short -coverprofile=coverage.txt -covermode=atomic $pkgs
go tool cover -func=coverage.txt | tail -1
```
Expected: docgen `-check` exits 0; tests pass; total coverage ≥ 80%.

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/test-coverage.yml
git commit -m "ci: single-module workflow"
```

---

### Task 3: v2-only documentation

**Files:**
- Modify: `README.md`, `docs/verification.md`, `PHASE2.md`, `codecov.yml`, `.gitignore`, `CODING_STANDARDS.md`
- Delete: v1 user docs under `docs/*.md` (keeping `docs/superpowers/**`, `docs/reviews/**`)

- [ ] **Step 1: Delete v1 user docs**

```bash
git rm -q docs/API_REFERENCE.md docs/account.md docs/architecture.md docs/client.md \
  docs/eventdecoder.md docs/network.md docs/quickstart.md docs/resources.md \
  docs/shielded.md docs/signer.md docs/smartcontract.md docs/trc10.md docs/trc20.md docs/types.md
```

- [ ] **Step 2: Rewrite README.md v2-only**

Cover: what tronlib is, `go get github.com/kslamph/tronlib/v2@v2.0.0`, Go
>= 1.27.1, the package map (`contract`, `key`, `rpc`, `tx`, `tron`, `token`,
`event`), a short quickstart, and links to `docs/`. No v1 wording.

- [ ] **Step 3: Fix `go -C v2` references and stale v1 bits**

```bash
sed -i 's#go -C v2 run#go run#g; s#go -C v2 #go #g' PHASE2.md docs/verification.md docs/errors.md docs/examples.md
```
Remove the `pkg/client/lowlevel/**` ignore from `codecov.yml`. Drop the stale
v1 `.gitignore` lines (`cmd/setup_nile_testnet/test.env`, `example/shielded/shielded_keys.json`).

- [ ] **Step 4: Verify docs gates + no v1 refs**

```bash
go run ./cmd/docgen sync-docs -pkg ./tron -example-pkg . -docs ./docs/errors.md -docs ./docs/examples.md -check
grep -rn 'go -C v2' --exclude-dir=.git . || echo CLEAN
grep -rln 'github.com/kslamph/tronlib/pb/' --include=*.md . || echo CLEAN
```
Expected: docgen `-check` exits 0; both greps `CLEAN`.

- [ ] **Step 5: Commit**

```bash
git add -A -- README.md docs PHASE2.md codecov.yml .gitignore CODING_STANDARDS.md
git commit -m "docs: v2-only README and docs; drop v1 docs and migration guide"
```

---

### Task 4: Full verification

- [ ] **Step 1: Gates**

```bash
gofmt -l . | grep -v 'cmd/docgen/testdata/brokenpkg/broken.go' || true
go build ./... && go vet ./...
go test ./... -count=1 -race
go test -short -coverprofile=coverage.txt -covermode=atomic ./...
go tool cover -func=coverage.txt | tail -1
govulncheck ./...
```
Expected: gofmt clean; build/vet/race pass; coverage >= 80%; govulncheck 0 reachable.

- [ ] **Step 2: Identity + leftovers**

```bash
go list -m
grep -rn '"github.com/kslamph/tronlib/pb/' --include=*.go . || echo NO_STALE_PB
go list ./... | grep -E '/protos(/|$)' && echo PROTO_SWEPT || echo NO_PROTO_PKGS
git status --short
```
Expected: `github.com/kslamph/tronlib/v2`; `NO_STALE_PB`; `NO_PROTO_PKGS`; only the pre-existing untracked scratch files remain.

- [ ] **Step 3: Commit any stragglers**

```bash
git add -A -- . ':!REPORT.md' ':!REVIEW.md' ':!docs/llms.txt' ':!docs/TRON_SDK_DESIGN.md' ':!skills-lock.json' ':!.agents' ':!.pi'
git diff --cached --quiet || git commit -m "chore: finish single-module mainline cleanup"
```
