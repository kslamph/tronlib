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
- [ ] `go test ./pkg/... -short` passes (what CI runs)
- [ ] `golangci-lint run` is clean
- [ ] Tests added or updated — table-driven, hermetic (no live network), meaningful assertions
- [ ] Test scaffolding uses `internal/testutil` (no new hand-rolled bufconn servers)
- [ ] Total coverage stays ≥ 80% (CI enforces)
- [ ] Bug fixes include a regression test
- [ ] Public API changes are documented; removals go through a `// Deprecated:` cycle
- [ ] Runnable examples still compile (`go test` runs them)
- [ ] No private keys, mnemonics, or funded addresses in this diff
- [ ] Commit title follows Conventional Commits

## Notes for reviewers

<!-- Anything non-obvious: trade-offs, follow-ups, areas you'd like a second opinion on. -->
