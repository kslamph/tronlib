// Package eventtool holds the reusable logic behind cmd/eventtool: the
// 32-byte event corpus store, the v1->v2 corpus migration, ABI ingestion, the
// TronScan contract ranking snapshot, contract ABI capture, and the
// builtin_gen.go generator.
//
// The logic lives here, not in cmd/eventtool, so it is hermetically testable:
// cmd/eventtool is flags and wiring only (precedent: internal/format,
// internal/compilecheck). Design:
// .superpowers/specs/2026-09-30-eventtool-design.md; plan:
// .superpowers/plans/2026-09-30-eventtool.md (both local, untracked).
package eventtool
