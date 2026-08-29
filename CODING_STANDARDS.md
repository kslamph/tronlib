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
cmd/                 standalone tools (event_abi_generator, setup_nile_testnet, ...)
pkg/                 the public library, one domain per package
  account/           account queries and permissions
  client/            gRPC client, connection pool, broadcaster
  eventdecoder/      transaction log / event decoding
  network/           node & network info
  resources/         bandwidth / energy queries
  signer/            private-key and HD-wallet signing
  smartcontract/     deploy, call, simulate
  trc10/             TRC10 asset operations
  trc20/             TRC20 token operations
  types/             shared domain types, sentinel errors, address
  utils/             ABI, encoding, transaction helpers
  voting/            vote & witness operations
pb/                  GENERATED protobuf code — do not hand-edit (regenerate)
protos/              protobuf sources for pb/
internal/            private packages (incl. internal/testutil for tests)
example/             runnable quickstart programs
integration_test/    live-network tests (build-tagged, opt-in)
docs/                package-level documentation
scripts/             proto generation and other tooling
```

Rules:

- One domain per package. A change that touches the same concern across many
  packages is fine; a change that makes one package serve two unrelated domains
  is not.
- `pkg/` packages may import `pkg/types`, `pkg/utils`, and `internal/*`. They
  must **not** import each other unless the dependency is one-directional and
  obvious (e.g. `trc20` → `smartcontract`). Cycles are a design error.
  **Exception — the facade:** `pkg/client/manager.go` deliberately imports
  every domain package to expose `Client.Account()`, `Client.TRC20()`, etc.;
  domain packages depend only on their local `gRPCClient`/`lowlevel`
  interfaces and never import `pkg/client` back, keeping the graph acyclic.
  Don't add imports to the facade without preserving that rule.
- Nothing under `pkg/` imports `cmd/`, `example/`, or `integration_test/`.
- New public packages need a `docs/<package>.md` page and a doc comment at the
  top of the package.

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

- **Sentinel errors live in `pkg/types/errors.go`** (`ErrInvalidAddress`,
  `ErrInvalidAmount`, `ErrInvalidContract`, `ErrInvalidTransaction`, ...).
  Add a new sentinel there when the condition is something a caller would want
  to branch on; give it a comment explaining the likely cause.
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
- **Copy-pasted scaffolding.** Fake gRPC servers, bufconn setup, and
  `mockConnProvider` implementations belong in `internal/testutil` (§6.3).
  A new test file that hand-rolls its own bufconn server will be asked to
  move it.
- **One-test-per-case sprawl.** `TestFoo_Bar1`, `TestFoo_Bar2`, ... differing
  in one input → one table-driven `TestFoo_Bar`.
- **Non-deterministic tests.** No `time.Sleep` to "wait for" anything in unit
  tests; no reliance on wall-clock ordering; seeds fixed for randomized tables.
- **Naming.** `bug3_test.go` tells nobody anything. Files and tests are named
  after the behaviour: `transfer_roundtrip_test.go`,
  `TestTransfer_RoundTrip`. Bug-regression tests reference the issue number:
  `TestIssue42_...`.

### 6.3 Test infrastructure: `internal/testutil`

All gRPC-facing unit tests use an in-memory `bufconn` server, never a live
node. The connection plumbing lives in `internal/testutil` and is mandatory:

```go
fake := &fakeWalletServer{GetAccountFunc: ...}       // per-package fake (below)
lis := testutil.NewBufconnServer(t, fake)           // listener + t.Cleanup
conn := testutil.DialBufconn(t, lis)                // *grpc.ClientConn + t.Cleanup
```

- `testutil.NewBufconnServer(t, impl)` — creates the bufconn listener,
  registers the gRPC server, and cleans up via `t.Cleanup`.
- `testutil.DialBufconn(t, lis)` — returns a `*grpc.ClientConn` wired to the
  listener, closed via `t.Cleanup`.

Remaining scaffolding is per-package but follows one shape:

- **Fake wallet servers** embed `api.UnimplementedWalletServer` and override
  behaviour via optional function fields (`TransferAsset2Func`,
  `GetAccountFunc`, ...). Defaults return minimal valid responses. A fake
  that grows beyond ~8 function-field overrides is really a scenario — give
  it a named constructor, and promote shared ones into `testutil`.
- **`mockConnProvider`** — the fake for the connection pool interface. When a
  second package needs the same fake, move it to `testutil` rather than
  copying it again.

Rules:

- New tests use `testutil.NewBufconnServer` / `testutil.DialBufconn`. A new
  test file that hand-rolls its own `bufconn.Listen(...)` will be asked to
  move it. The four remaining legacy copies
  (`client`, `trc20`, `voting`, `account` test fakes) are being migrated;
  if you touch one, migrate it in the same PR.
- Fakes return **programmed errors** (`status.Error(codes.X, ...)`) to
  exercise error mapping — they never simulate failure by returning Go `nil`
  protobufs, because real gRPC never does.

### 6.4 Live-network and integration tests

- Default `go test ./pkg/...` must be **hermetic**: no network, no disk writes
  outside `t.TempDir()`. CI runs with `-short`.
- Anything needing a real node lives in `integration_test/` behind the
  `integration` build tag, configured via `integration_test/test.env`
  (Nile testnet only). See `integration_test/TESTING_GUIDE.md`.
- `example/` programs are `package main`, so they are invisible to
  `go test ./pkg/...`. CI therefore also runs `go build ./...` and
  `go test -short ./example/...`: an example that stops compiling fails the
  build, and pure logic inside an example (amount scaling, note selection,
  ABI encoding) is expected to carry hermetic tests where it exists.
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

- **CI enforces a hard floor of 80%** total statement coverage on
  `./pkg/...` (`go tool cover -func` total, `-short` mode). A PR that drops
  the repo below 80% fails; Codecov's per-flag status remains informational.
- Coverage is a floor, not a target. Do not write tests *for* coverage: a PR
  described as "increase coverage" must still state which behaviours it
  verifies.
- Error paths count. A package whose error branches are untested is not done.
- `pkg/client/lowlevel/**` is ignored by Codecov (`codecov.yml`), as is generated
  code. The **CI floor is not**: it is `go tool cover -func` over
  `-coverpkg=./pkg/...`, so lowlevel's statements count toward the 80% and the
  two numbers will not agree. Treat the CI number as the gate and the Codecov
  flags as colour.

## 7. Documentation

- Every exported symbol has a doc comment. Package-level comments go in
  `doc.go` when they exceed a few lines (see `pkg/types/doc.go`).
- Doc comments for I/O-performing functions say so: "Balance queries the
  network".
- Runnable examples in `example_test.go` beat prose. They must compile —
  `go test` runs them; broken examples fail CI.
- `docs/<package>.md` documents workflows and design notes; keep it in sync
  when public behaviour changes. README quickstart snippets are tested via
  `example/readme_*`.
- Comments explain *why*, code explains *what*. Delete commented-out code and
  stale TODOs; an actionable TODO references an issue number.

## 8. Security

- **Never log, serialize, or echo private keys, mnemonics, or signed raw
  transactions** in library code, examples, tests, or error messages.
- Error messages include txids and addresses (public data), never secrets.
- `integration_test/test.env` may contain **throwaway Nile testnet keys only**
  (already the case). Mainnet keys or personal keys must never be committed;
  `.env` is gitignored.
- **Test fixtures:** `*_test.go` files may hardcode clearly-labeled throwaway
  keys for deterministic signature vectors (comment them "test-only, no
  value"). `example/` and `cmd/` programs must take keys from environment
  variables or flags — never hardcode them, even testnet keys.
- Anything parsing external input (ABI JSON, event logs, API responses) fails
  closed: malformed input returns an error, never a best-effort zero value.

---

## Quick checklist (what a reviewer will look for)

1. `gofmt`/`goimports` clean, `.golangci.yml` passes — CI enforces.
2. Errors: sentinels in `pkg/types`, wrapped with `%w`, matched with
   `errors.Is`/`As`.
3. Tests: table-driven, hermetic, meaningful assertions, use
   `internal/testutil`, no coverage-bump padding.
4. New/changed public API documented, examples compile, deprecations marked.
5. No secrets in code, logs, or test fixtures.
6. Coverage floor 80% maintained; PR explains what the tests verify.
