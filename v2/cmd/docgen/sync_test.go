package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// syncDocFixture is the minimal docs file the sync tests fill. It mirrors
// the shape of v2/docs/errors.md and v2/docs/examples.md: a go:errors block
// and two go:example blocks.
const syncDocFixture = `# Fixture

<!-- go:errors -->
<!-- /go:errors -->

<!-- go:example ExampleGreet -->
<!-- /go:example -->

<!-- go:example ExampleFarewell -->
<!-- /go:example -->
`

func writeSyncFixture(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "doc.md")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func syncFixturePkg(t *testing.T) string {
	t.Helper()
	return filepath.Join("testdata", "syncpkg")
}

// TestSyncDocHAppyPath covers the whole sync pipeline on a fixture package:
// the error table is filled from the parsed switch data (not codes_gen.go
// text) and example bodies are extracted from the _test.go files.
func TestSyncDocHappyPath(t *testing.T) {
	path := writeSyncFixture(t, syncDocFixture)
	require.NoError(t, runSync(syncFixturePkg(t), []string{path}, false))

	out, err := os.ReadFile(path)
	require.NoError(t, err)
	s := string(out)
	// The table rows carry the fixture's codes, actions, and docs.
	assert.Contains(t, s, "| `alpha.one` | `retry` | alpha and bravo doc |")
	assert.Contains(t, s, "| `bravo.two` | `retry` | alpha and bravo doc |")
	// Example bodies were extracted from the fixture package's _test.go.
	assert.Contains(t, s, "fmt.Println(\"hello\")")
	assert.Contains(t, s, "fmt.Println(\"bye\")")
}

// TestSyncDocIdempotent: syncing an already-synced file is a fixed point.
func TestSyncDocIdempotent(t *testing.T) {
	path := writeSyncFixture(t, syncDocFixture)
	require.NoError(t, runSync(syncFixturePkg(t), []string{path}, false))
	first, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, runSync(syncFixturePkg(t), []string{path}, false))
	second, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, string(first), string(second))
}

// TestSyncDocCheckDetectsCellEdit: a hand-edited table cell is drift, and
// -check names the file.
func TestSyncDocCheckDetectsCellEdit(t *testing.T) {
	path := writeSyncFixture(t, syncDocFixture)
	require.NoError(t, runSync(syncFixturePkg(t), []string{path}, false))
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	mutated := strings.Replace(string(raw), "| `retry` |", "| `bug` |", 1)
	require.NotEqual(t, string(raw), mutated, "mutation must change the file")
	require.NoError(t, os.WriteFile(path, []byte(mutated), 0o644))

	err = runSync(syncFixturePkg(t), []string{path}, true)
	require.Error(t, err, "-check must fail on a hand-edited cell")
	assert.Contains(t, err.Error(), path, "the error must name the drifted file")
	assert.Contains(t, err.Error(), "stale")
}

// TestSyncDocCheckDetectsDeletedExampleBlock: removing a marker PAIR (both
// marker lines) is invisible to fill-and-compare (nothing left to fill) —
// the coverage check must still fail, naming the orphaned Example function.
func TestSyncDocCheckDetectsDeletedExampleBlock(t *testing.T) {
	path := writeSyncFixture(t, syncDocFixture)
	require.NoError(t, runSync(syncFixturePkg(t), []string{path}, false))
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	deleted := strings.Replace(string(raw),
		"<!-- go:example ExampleFarewell -->\nfmt.Println(\"bye\")<!-- /go:example -->\n", "", 1)
	require.NotEqual(t, string(raw), deleted, "mutation must change the file")
	require.NoError(t, os.WriteFile(path, []byte(deleted), 0o644))

	// Sync would also "pass" here — the block is gone. The check must not.
	err = runSync(syncFixturePkg(t), []string{path}, true)
	require.Error(t, err, "-check must fail when a marker pair is deleted")
	assert.Contains(t, err.Error(), "ExampleFarewell", "the error must name the orphaned Example")
}

// TestSyncDocCheckDetectsMarkerWithoutFunction: the other coverage
// direction — a marker naming an Example function the package does not have.
func TestSyncDocCheckDetectsMarkerWithoutFunction(t *testing.T) {
	broken := strings.Replace(syncDocFixture,
		"<!-- go:example ExampleFarewell -->",
		"<!-- go:example ExampleGhost -->", 1)
	path := writeSyncFixture(t, broken)

	err := runSync(syncFixturePkg(t), []string{path}, true)
	require.Error(t, err, "a marker without a real Example function must fail")
	assert.Contains(t, err.Error(), "ExampleGhost")
}

// TestSyncDocCoverageAcrossFiles: an example may be documented in one docs
// file while another docs file carries none of it; coverage is evaluated
// across the whole -docs set, not per file.
func TestSyncDocCoverageAcrossFiles(t *testing.T) {
	dir := t.TempDir()
	tableOnly := filepath.Join(dir, "errors.md")
	require.NoError(t, os.WriteFile(tableOnly, []byte("# Errors\n\n<!-- go:errors -->\n<!-- /go:errors -->\n"), 0o644))
	exampleOnly := filepath.Join(dir, "examples.md")
	require.NoError(t, os.WriteFile(exampleOnly, []byte("# Examples\n\n<!-- go:example ExampleGreet -->\n<!-- /go:example -->\n\n<!-- go:example ExampleFarewell -->\n<!-- /go:example -->\n"), 0o644))

	require.NoError(t, runSync(syncFixturePkg(t), []string{tableOnly, exampleOnly}, false))

	// And the check passes on the synced pair.
	require.NoError(t, runSync(syncFixturePkg(t), []string{tableOnly, exampleOnly}, true))

	// Deleting one block from examples.md orphans ExampleFarewell and the
	// check must name it even though errors.md is fine.
	raw, err := os.ReadFile(exampleOnly)
	require.NoError(t, err)
	deleted := strings.Replace(string(raw),
		"<!-- go:example ExampleFarewell -->\nfmt.Println(\"bye\")<!-- /go:example -->\n", "", 1)
	require.NotEqual(t, string(raw), deleted, "mutation must change the file")
	require.NoError(t, os.WriteFile(exampleOnly, []byte(deleted), 0o644))
	err = runSync(syncFixturePkg(t), []string{tableOnly, exampleOnly}, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ExampleFarewell")
}

// TestSyncDocRealTronIsAFixedPoint: the committed v2/docs files must be
// exactly what sync-docs renders from the live v2/tron package. This is the
// in-test shadow of the CI -check gate (the testdata fixtures above cover
// the fault injections without touching v2/tron).
func TestSyncDocRealTronIsAFixedPoint(t *testing.T) {
	tronDir := realTronDir(t)
	docsDir := filepath.Join("..", "..", "docs")
	files := []string{
		filepath.Join(docsDir, "errors.md"),
		filepath.Join(docsDir, "examples.md"),
	}
	for _, f := range files {
		if _, err := os.Stat(f); err != nil {
			t.Fatalf("docs target missing: %v", err)
		}
	}
	require.NoError(t, runSync(tronDir, files, true))
}
