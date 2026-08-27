# Contributing to tronlib

Thanks for helping improve tronlib! This guide covers everything from reporting
an issue to getting a PR merged.

- 📜 **Coding standards:** read [CODING_STANDARDS.md](CODING_STANDARDS.md)
  before writing code — it defines what review will ask of you.
- 📖 **Package docs:** [docs/](docs/) for design notes and workflows.
- 🧪 **Live-network testing:** [integration_test/TESTING_GUIDE.md](integration_test/TESTING_GUIDE.md).
- ❓ **Questions:** open a [Discussion](https://github.com/kslamph/tronlib/discussions),
  not an issue.

## Ways to help

- Fix a bug (look for issues labelled `bug` + `good first issue`)
- Improve test coverage or test quality (`test` + `help wanted`)
- Improve documentation (`documentation`)
- Add features that have an accepted proposal (see below)

## Prerequisites

| Tool | Version |
|---|---|
| Go | 1.25+ (must match `go.mod`) |
| golangci-lint | latest (`go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`) |
| protoc + plugins | only if changing `protos/` (see `scripts/proto-gen.sh`) |

## Setup

```bash
git clone https://github.com/<your-fork>/tronlib
cd tronlib
go build ./...
go test ./pkg/...          # hermetic unit tests, no network needed
golangci-lint run          # must pass clean
```

No environment variables or testnet accounts are required for unit tests.
Live-network integration tests are opt-in (`integration_test/TESTING_GUIDE.md`).

## Reporting issues

**Bugs, feature proposals, and documentation problems go through GitHub issue
templates** — pick the right one and fill it in:

| Template | Use when |
|---|---|
| [Bug report](https://github.com/kslamph/tronlib/issues/new?template=bug_report.yml) | Something behaves incorrectly or crashes |
| [Feature request](https://github.com/kslamph/tronlib/issues/new?template=feature_request.yml) | You want new capability or an API change |
| [Documentation](https://github.com/kslamph/tronlib/issues/new?template=documentation.yml) | Docs are wrong, unclear, or missing |

Issue rules:

1. **Search first** — duplicates get closed with a link to the original.
2. **One issue, one problem** — split unrelated problems into separate issues.
3. **Use the template** — "it doesn't work" reports without version, package,
   and reproduction steps will be closed as `needs-info` after 14 days.
4. **Never post private keys, mnemonics, or funded-account addresses.** If a
   reproduction requires a key, use a fresh throwaway Nile testnet account.
5. **Security vulnerabilities are not issues.** See
   [SECURITY.md](SECURITY.md) (report privately via GitHub security advisory).

### Issue lifecycle

1. New issues are triaged within a week and labelled (`bug`, `feature`,
   `documentation`, `good first issue`, `help wanted`, ...).
2. Feature requests need a maintainer's `accepted` label **before** you start
   coding the PR — this avoids wasted work on proposals that don't fit the
   roadmap.
3. An accepted issue is assigned or left open for pickup. Comment before
   starting work on an assigned issue; two people = one wasted PR.
4. Stale `needs-info` issues auto-close after 14 days without a reply.
   Reopening is fine once the info is added.

## Development workflow

### 1. Branch

Fork, then branch from `main`:

```bash
git checkout -b feat/trc20-allowance-query     # feature
git checkout -b fix/eventdecoder-bool-topic    # bug fix
git checkout -b test/trc10-table-tests         # test-only change
git checkout -b docs/trc20-page                # documentation
```

Branch names: `type/short-slug`. Keep one branch per issue.

### 2. Commit — Conventional Commits

```text
<type>(<scope>): <imperative summary, ≤72 chars>

[optional body: why, not what]
[optional footer: Fixes #123]
```

- **type**: `feat` | `fix` | `test` | `docs` | `refactor` | `perf` |
  `build` | `ci` | `chore`
- **scope**: the package or area — `trc20`, `client`, `eventdecoder`,
  `types`, `coverage`, `gate` ...
- Summary in the imperative: "add", not "added"/"adds".
- Reference the issue in the footer (`Fixes #123`) so the issue auto-closes on
  merge.

Examples from this repo's history:

```text
test(smartcontract): increase coverage from 75.6% to 95.9%
gate: add testing.Short() guard to live examples, update CI workflow
fix(utils): use Keccak256 for method signature encoding
```

### 3. Test

```bash
go test ./pkg/... -short          # what CI runs
go test ./pkg/trc20/... -v        # the package you touched
golangci-lint run
```

Expectations (details in [CODING_STANDARDS.md](CODING_STANDARDS.md) §6):

- New behaviour ships with tests in the same PR — table-driven, hermetic,
  using `internal/testutil` for fake gRPC servers.
- **CI enforces an 80% total coverage floor** on `./pkg/...`; a PR that drops
  below it fails.
- Bug fixes include a regression test named after the behaviour or the issue
  (`TestIssue42_...`), and the PR footer references the issue.

### 4. Open the PR

Fill in the [PR template](.github/PULL_REQUEST_TEMPLATE.md). The checklist:

- [ ] `go build ./...`, `go test ./pkg/... -short`, `golangci-lint run` all pass
- [ ] Tests added/updated; they assert behaviour, not just execution
- [ ] Conventional Commit title (`feat(trc20): ...`) — PRs are squash-merged,
      so the PR title becomes the commit
- [ ] Public API changes documented; deprecations marked `// Deprecated:`
- [ ] No secrets, keys, or mnemonics anywhere in the diff
- [ ] Linked the issue (`Fixes #123`) or explained why there is none

Review expectations:

- PRs stay small and single-purpose. Big changes split into stacked PRs.
- Reviews focus on correctness, API shape, and test quality (see
  CODING_STANDARDS.md). Style nits that tooling should catch get converted
  into `.golangci.yml` rules instead of review chatter.
- Respond to every comment, even if just with an emoji — silence reads as
  disagreement.
- Maintainers squash-merge onto `main` once CI is green and review is
  approved.

### 5. Integration tests (maintainers / network-touching changes)

If your change affects signing, broadcasting, or ABI encoding, run the Nile
testnet integration suite per
[integration_test/TESTING_GUIDE.md](integration_test/TESTING_GUIDE.md) before
requesting review, and note the result in the PR.

## Licensing

Contributions are made under the [MIT License](LICENSE) that covers this
repository. By submitting a PR you agree your contribution is licensed under
it. Don't copy code from other projects unless it is MIT/compatible — and say
where it came from.

## Recognition

Every contribution counts — issues, docs, tests, and code alike. Significant
contributors are credited in release notes.
