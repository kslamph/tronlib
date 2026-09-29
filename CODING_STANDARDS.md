# tronlib Coding Standards

This document defines how code in `tronlib` is written and reviewed. It exists so
that contributors, reviewers, and automated tooling apply the same rules.

**Two kinds of rules:**

- **Enforced** — checked by tooling (`golangci-lint`, CI, `gofmt`). Fix these
  locally before opening a PR; a reviewer will not debate them.
- **Review-enforced** — judgement calls applied in code review. Where this
  document states a rule, follow it; deviations need a stated reason in the PR
  description.

Anything not covered here falls back to the official Go style:
[Effective Go](https://go.dev/doc/effective_go) and
[Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments).

---

## 1. Project layout

```
*/                   the public library (facade package at the repo root)
  contract/          ABI-driven contract calls and deploys
  event/             transaction log / event decoding
  key/               signers (private key, mnemonic) and message signing
  rpc/               gRPC client, connection pool, 1:1 wrappers
  token/             TRC-20/TRC-10 token handles
  tron/              core types: addresses, SUN, coded errors
  tx/                transaction builders, signing, broadcast, receipts
pb/                  GENERATED protobuf code — do not hand-edit (regenerate)
protos/              protobuf sources for pb/ (git submodule)
internal/format/     private formatting helpers
internal/compilecheck/ compile-time API guards
cmd/docgen/          docs + generated-code tooling
cmd/tip491probe/     live-network probe (manual, read-only by default)
docs/                user and process documentation
scripts/             proto generation and other tooling
```

Rules:

- One domain per package. A change that touches the same concern across many
  packages is fine; a change that makes one package serve two unrelated domains
  is not.
- The root package (`tronlib`) is the **facade**: it aliases (`type X = p.X`)
  and delegates, and never reimplements subpackage logic. Subpackages do not
  import the facade back, so the graph stays acyclic.
- Subpackages may import `tron`, `rpc`, and `internal/*`. Cross-domain imports
  must be one-directional and obvious; cycles are a design error.
- Nothing outside `cmd/` imports a `cmd/` package.
- New public packages need a `docs/<package>.md` page and a package doc comment.

## 2. Go style

**Enforced by tooling** (`.golangci.yml`): `gofmt`/`goimports` formatting with
local prefix `github.com/kslamph/tronlib` (std → external → `tronlib` import
groups), `govet`, `errcheck`, `staticcheck`, `unused`, `revive`, `gocyclo`
(min-complexity 15), `gosec`.

**Review-enforced:**

- **Naming.** Exported identifiers get doc comments starting with the name.
  A name that needs a paragraph of explanation is a design smell — rename or
  redesign. No abbreviations that aren't domain-standard (`ABI`, `TRC20`, `RPC`).
- **Constructors.** `NewXxx(...)` returns `(*Xxx, error)`; validate all inputs
  before returning a usable object. No half-initialized structs.
- **Contexts.** Every method that performs I/O takes a `context.Context` as its
  first parameter. Never store a context in a struct; pass it down.
- **Interfaces.** Define interfaces where they are *consumed*, not where they
  are implemented. Keep them one- or two-method. Mock interfaces in tests via
  small fakes, not reflection-based mock frameworks.
- **Concurrency.** Goroutines started by library code must have a documented
  shutdown path (`Close()`/`Stop()`). Never leak a goroutine past client close.
- **Receivers.** Pointer receivers consistently, or value receivers
  consistently — never mixed on the same type.
- **No speculative generality.** Don't add options, hooks, or abstraction
  layers for imagined future needs. Inline until a second real caller exists.

## 3. Error handling

tronlib's error contract is part of its public API.

- **Coded errors live in `tron`** — `tron.Error` carrying a `tron.Code`
  (defined in `tron/codes.go`, rendered in `tron/codes_gen.go`): e.g.
  `key.invalid`, `account.insufficient_bandwidth`, `contract.arg_mismatch`.
  Callers branch with `errors.Is` / `errors.As`. Add a code in `tron/codes.go`
  (then regenerate) when the condition is something a caller would want to
  branch on; give it a comment explaining the likely cause.
- **Wrap with context, preserve the chain:**
  ```go
  return nil, fmt.Errorf("failed to pre-fetch decimals: %w", err)
  ```
  Always `%w`, never `%v`, when wrapping. The wrapped message describes what
  *this layer* was doing.
- **Errors are values.** Return them; do not log-and-continue inside the
  library, do not panic on user input, do not swallow (`_ =`) except for
  explicitly best-effort cleanup calls.
- **Must\* carve-out:** `types.MustNewAddressFrom*` follow the stdlib
  `MustXxx` idiom — they panic by documented contract and are acceptable.
  Non-`Must` exported methods must not panic on any input, including nil
  receivers (`Address.EVMAddress`'s current nil panic is a known deviation;
  fix to return the zero address or an error — don't add more like it).
- Callers match with `errors.Is` / `errors.As`. Never compare error strings.
- Methods on gRPC results must distinguish transport errors (return them
  wrapped) from on-chain rejections (map to the appropriate `types` sentinel or
  a typed error with txid).

## 4. Dependencies

- The public dependency set is deliberately small (go-ethereum, grpc, protobuf,
  testify, base58, decimal, bip39-hdwallet). Adding a new module dependency is
  a design decision — open an issue first.
- `go.mod` pins the minimum Go version; CI tests that version. Don't raise it
  casually.
- Generated code (`pb/`) changes only via `scripts/proto-gen.sh`. Never
  hand-edit `pb/`; regenerate and commit the result separately from logic
  changes.

## 5. Public API stability

- Breaking changes to exported identifiers require a deprecation cycle: keep
  the old symbol with a `// Deprecated:` comment for at least one minor
  release, and note the migration in the changelog / release notes.
- Don't export something you're not prepared to support. Unexport until needed.
- New exported functions that do I/O over a live network are documented as
  such, and any *example* calling them is guarded with `testing.Short()` (see
  §6).

## 6. Testing standards

Tests are first-class code. They are reviewed with the same care as `pkg/`.

### 6.1 What to test

- Every exported function gets at least: one happy path, each sentinel-error
  path, and each boundary condition. Table-driven tests are the default shape:
  ```go
  func TestSetFeeLimit(t *testing.T) {
      tests := []struct {
          name    string
          fee     int64
          wantErr error
      }{
          {"zero", 0, types.ErrInvalidAmount},
          {"negative", -1, types.ErrInvalidAmount},
          {"typical", 1_000_000, nil},
      }
      for _, tt := range tests {
          t.Run(tt.name, func(t *testing.T) { ... })
      }
  }
  ```
- Use `testify` `require` for preconditions that make the rest of the test
  meaningless, `assert` for the behaviour under test.

### 6.2 Test smells (rejected in review)

These are concrete, recurring anti-patterns. Each one has appeared in this
repository and been flagged; don't reintroduce them:

- **Tautological / coverage-bump tests.** A test whose only assertion is
  `result != nil` or `err == nil` on a path that cannot fail, or whose comment
  admits it exists "for more coverage". If a branch is genuinely unreachable,
  delete it from the implementation instead of testing around it.
- **Line-number comments.** `// covers line 56` rots on every edit. Test names
  and assertions describe behaviour, not source layout.
- **Copy-pasted scaffolding.** Fake gRPC servers and bufconn setup live in the
  consuming package's `fakes_test.go`; reuse that fake instead of hand-rolling
  a second one (§6.3).
- **One-test-per-case sprawl.** `TestFoo_Bar1`, `TestFoo_Bar2`, ... differing
  in one input → one table-driven `TestFoo_Bar`.
- **Non-deterministic tests.** No `time.Sleep` to "wait for" anything in unit
  tests; no reliance on wall-clock ordering; seeds fixed for randomized tables.
- **Naming.** `bug3_test.go` tells nobody anything. Files and tests are named
  after the behaviour: `transfer_roundtrip_test.go`,
  `TestTransfer_RoundTrip`. Bug-regression tests reference the issue number:
  `TestIssue42_...`.

### 6.3 Test infrastructure: in-memory gRPC fakes

All gRPC-facing unit tests use an in-memory `bufconn` server, never a live
node. The fake lives in the package it serves, as `fakes_test.go`, and the
client is built with rpc's test-only dialer:

```go
fake := &testWalletServer{Handlers: ...}                    // package-local fake
lis := bufconn.Listen(bufSize)
srv := grpc.NewServer()
api.RegisterWalletServer(srv, fake)
go srv.Serve(lis)                                           // t.Cleanup closes srv/lis
cli, _ := rpc.NewClientWithDialer("passthrough:///bufnet", lis.Dial)
```

`rpc.NewClientWithDialer` is stripped from release builds (`//go:build !release`).

The scaffolding is per-package but follows one shape:

- **Fake wallet servers** embed `api.UnimplementedWalletServer` and override
  behaviour via optional function fields (`TransferAsset2Func`,
  `GetAccountFunc`, ...). Defaults return minimal valid responses. A fake
  that grows beyond ~8 function-field overrides is really a scenario — give
  it a named constructor, and promote shared ones into `testutil`.
- **`mockConnProvider`** — the fake for the connection pool interface. When a
  second package needs the same fake, move it to `testutil` rather than
  copying it again.

Rules:

- New tests reuse the package's existing `fakes_test.go` fake. A test that
  hand-rolls its own server-side fake alongside an existing one will be asked
  to reuse it. Shared fakes move up only when a second package genuinely needs
  them.
- Fakes return **programmed errors** (`status.Error(codes.X, ...)`) to
  exercise error mapping — they never simulate failure by returning Go `nil`
  protobufs, because real gRPC never does.

### 6.4 Live-network and integration tests

- Default `go test ./...` must be **hermetic**: no network, no disk writes
  outside `t.TempDir()`. CI runs with `-short`.
- Live checks against a real node are explicit and manual, read-only by
  default: use `go run ./cmd/tip491probe` (Nile testnet) and record the
  evidence in `docs/verification.md`. Never point a unit test at a node.
- `cmd/` probes are `package main` and are exercised by `go build ./...`;
  their pure logic (parsing, replay assembly) carries hermetic tests where it
  exists.
- Example functions (`example_test.go`) that dial a live node must early-return
  under `testing.Short()`:
  ```go
  func ExampleManager_Balance() {
      if testing.Short() {
          fmt.Println("skipped in -short mode")
          return
      }
      ...
  }
  ```

### 6.5 Coverage policy

- **CI enforces a hard floor of 80%** total statement coverage on the module
  (`go tool cover -func` total, `-short` mode), with generated `pb/` excluded
  from the denominator. A PR that drops the repo below 80% fails; Codecov's
  per-flag status remains informational.
- Coverage is a floor, not a target. Do not write tests *for* coverage: a PR
  described as "increase coverage" must still state which behaviours it
  verifies.
- Error paths count. A package whose error branches are untested is not done.
- Generated protobuf code (`pb/**`) is excluded from both the CI coverage run
  and Codecov (`codecov.yml`); it carries no tests.

## 7. Documentation

- Every exported symbol has a doc comment. Package-level comments go in
  `doc.go` when they exceed a few lines (see `rpc/doc.go`).
- Doc comments for I/O-performing functions say so: "Balance queries the
  network".
- Runnable examples in `example_test.go` beat prose. They must compile —
  `go test` runs them; broken examples fail CI.
- `docs/<package>.md` documents workflows and design notes; keep it in sync
  when public behaviour changes. README quickstart snippets mirror
  `example_test.go`, which `go test` compiles.
- Comments explain *why*, code explains *what*. Delete commented-out code and
  stale TODOs; an actionable TODO references an issue number.

## 8. Security

- **Never log, serialize, or echo private keys, mnemonics, or signed raw
  transactions** in library code, examples, tests, or error messages.
- Error messages include txids and addresses (public data), never secrets.
- Test keys must be **throwaway Nile testnet keys only**. Mainnet keys or
  personal keys must never be committed; `.env` is gitignored.
- **Test fixtures:** `*_test.go` files may hardcode clearly-labeled throwaway
  keys for deterministic signature vectors (comment them "test-only, no
  value"). `cmd/` programs must take keys from environment variables or flags
  — never hardcode them, even testnet keys.
- Anything parsing external input (ABI JSON, event logs, API responses) fails
  closed: malformed input returns an error, never a best-effort zero value.

---

## Quick checklist (what a reviewer will look for)

1. `gofmt`/`goimports` clean, `.golangci.yml` passes — CI enforces.
2. Errors: sentinels in `tron`, wrapped with `%w`, matched with
   `errors.Is`/`As`.
3. Tests: table-driven, hermetic, meaningful assertions, use the package's
   bufconn fake, no coverage-bump padding.
4. New/changed public API documented, examples compile, deprecations marked.
5. No secrets in code, logs, or test fixtures.
6. Coverage floor 80% maintained; PR explains what the tests verify.
