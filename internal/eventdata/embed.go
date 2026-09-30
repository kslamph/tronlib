// Package eventdata holds the tracked data the event tooling operates on: the
// curated event-signature corpus that generates event/builtin_gen.go, the seed
// ABI files, and the snapshotted TronScan contract ranking.
//
// The corpus used to live at tmp/events_registry.json — a gitignored path, so
// a fresh clone could not regenerate the built-in table and the provenance
// header on the generated file was unverifiable. It lives here, in the module,
// so the data and the generated code travel together.
//
// Design: docs/superpowers/specs/2026-09-30-eventtool-design.md.
package eventdata

import _ "embed"

// RegistryJSON is the curated event corpus: one entry per event definition,
// carrying its canonical signature and the full 32-byte sighash derived from
// it (managed by cmd/eventtool migrate/insert/capture; consumed by
// cmd/eventtool generate).
//
//go:embed events_registry.json
var RegistryJSON []byte
