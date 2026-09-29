<!--
PR title: Conventional Commit — type(scope): imperative summary
e.g. feat(trc20): add allowance query — PRs are squash-merged, so the title becomes the commit.
Read CODING_STANDARDS.md before requesting review.
-->

## What & why

<!-- What does this change do, and why? Link the issue: "Fixes #123".
For test-only PRs: name the *behaviours* the tests verify, not just "increase coverage". -->

Fixes #

## Changes

<!-- Bullet list of meaningful changes; skip generated files (pb/). -->

-

## Checklist

- [ ] `go build ./...` passes
- [ ] `go test ./... -short` passes (what CI runs)
- [ ] `golangci-lint run` is clean (CI enforces)
- [ ] Tests added or updated — table-driven, hermetic (no live network), meaningful assertions
- [ ] Reuse an existing bufconn fake (tx/fakes_test.go, contract/fakes_test.go, rpc's) where one fits; only hand-roll a new server when the shape genuinely differs
- [ ] Total coverage stays ≥ 80% (CI enforces)
- [ ] Bug fixes include a regression test
- [ ] Public API changes are documented; breaking changes are next-major material (v2 carries no shims — CODING_STANDARDS §5)
- [ ] Runnable examples still compile (`go test` builds them; docgen drift check regenerates docs/examples.md)
- [ ] No private keys, mnemonics, or funded addresses in this diff
- [ ] Commit title follows Conventional Commits

## Notes for reviewers

<!-- Anything non-obvious: trade-offs, follow-ups, areas you'd like a second opinion on. -->
