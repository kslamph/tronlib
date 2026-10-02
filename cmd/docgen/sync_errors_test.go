package main

// Error-path tests for the docs side of docgen: replaceBetween/betweenContent
// marker handling, exampleNames, checkFile, renderErrorList, fillExamples,
// runSync's per-stage failures, packageNameOf, constValue, firstDiffLine and
// emitCodesGen's self-validation guard.
//
// These matter because docgen is a CI gate: a docs file it cannot parse must
// fail loudly, because a silent skip would leave stale documentation passing.

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplaceBetweenErrors(t *testing.T) {
	t.Run("start marker missing", func(t *testing.T) {
		_, err := replaceBetween("no markers here", errStart, errEnd, "body")
		require.Error(t, err)
		assert.Contains(t, err.Error(), errStart, "the message must name the marker that was absent")
	})

	t.Run("end marker missing", func(t *testing.T) {
		_, err := replaceBetween("x\n"+errStart+"\n", errStart, errEnd, "body")
		require.Error(t, err)
		assert.Contains(t, err.Error(), errEnd)
	})
}

func TestBetweenContentErrors(t *testing.T) {
	_, err := betweenContent("nothing", errStart, errEnd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), errStart)

	_, err = betweenContent("x"+errStart+"\nbody", errStart, errEnd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), errEnd)
}

func TestExampleNamesUnclosedMarker(t *testing.T) {
	_, err := exampleNames("text <!-- go:example Foo")
	require.Error(t, err, "an unterminated marker must be an error, not a silently dropped name")
	assert.Contains(t, err.Error(), "unclosed example marker")
}

func TestFillExamplesErrorPaths(t *testing.T) {
	t.Run("propagates the marker scan failure", func(t *testing.T) {
		_, err := fillExamples("<!-- go:example Foo", map[string]string{"Foo": "b"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unclosed example marker")
	})

	t.Run("propagates the replacement failure", func(t *testing.T) {
		// The marker is closed (so exampleNames succeeds) but the block has no
		// end marker, so the replacement cannot be bounded.
		_, err := fillExamples("<!-- go:example Foo -->", map[string]string{"Foo": "body"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), exampleEnd)
	})
}

func TestCheckFileErrorPaths(t *testing.T) {
	t.Run("unclosed marker", func(t *testing.T) {
		err := checkFile("<!-- go:example Foo", map[string]string{"Foo": "body"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unclosed example marker")
	})

	t.Run("marker with no example of that name", func(t *testing.T) {
		in := "x\n<!-- go:example Gone -->\n\n<!-- /go:example -->\n"
		err := checkFile(in, map[string]string{"Other": "body"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), `"Gone"`)
	})

	t.Run("block with no end marker", func(t *testing.T) {
		in := "x\n<!-- go:example Foo -->"
		err := checkFile(in, map[string]string{"Foo": "body"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), exampleEnd)
	})

	t.Run("filled content passes", func(t *testing.T) {
		// checkFile's per-marker comparison is documented as equivalent to
		// re-rendering, so feeding it what fillExamples produces must pass. This
		// pins the fixed point rather than a hand-written layout that only looks
		// right.
		filled, err := fillExamples("x\n<!-- go:example Foo -->\n\n<!-- /go:example -->\n",
			map[string]string{"Foo": "body"})
		require.NoError(t, err)
		require.NoError(t, checkFile(filled, map[string]string{"Foo": "body"}))

		// A different body is stale.
		require.Error(t, checkFile(filled, map[string]string{"Foo": "other"}))
	})
}

func TestRenderErrorListMissingClose(t *testing.T) {
	_, err := renderErrorList("x\n"+errStart+"\n", []codeDoc{{Code: "a", Action: "retry", Doc: "d"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), errEnd)
}

func TestGoQuoteDocEscapesBacktickAndNewline(t *testing.T) {
	assert.Equal(t, "`plain doc`", goQuoteDoc("plain doc"))
	// A backtick cannot live inside a raw string literal, and an unescaped
	// newline would break the emitted source, so both fall back to quoting.
	assert.Equal(t, "\"with `tick`\"", goQuoteDoc("with `tick`"))
	assert.Equal(t, "\"line\\nfeed\"", goQuoteDoc("line\nfeed"))
	assert.Equal(t, `"carriage\r"`, goQuoteDoc("carriage\r"),
		"a carriage return must come out as the two-character escape, not a raw byte")
}

// TestEmitCodesGenRejectsUnparsableOutput: emitCodesGen formats its own output,
// so a bug that emits invalid Go is caught here rather than as a compile error
// in package tron. Reached with a hand-built action value that is not a Go
// expression, which is exactly what the guard is for.
func TestEmitCodesGenRejectsUnparsableOutput(t *testing.T) {
	consts := []codeConst{{Name: "CodeAlpha", Value: "alpha"}}
	all := []string{"CodeAlpha"}
	action := map[string]string{"CodeAlpha": "not an identifier!!"}
	doc := map[string]string{"CodeAlpha": "d"}

	_, err := emitCodesGen(all, consts, action, doc)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "docgen bug", "the message must say the guard fired, not that the input was bad")
}

func TestConstValueFallsBackToTheName(t *testing.T) {
	consts := []codeConst{{Name: "CodeAlpha", Value: "alpha.one"}}
	assert.Equal(t, "alpha.one", constValue(consts, "CodeAlpha"))
	// checkParity rejects an undeclared code before codeTable runs, so this
	// fallback only matters if the two are ever called out of order; returning
	// the name keeps the table rendering rather than emitting an empty cell.
	assert.Equal(t, "CodeGhost", constValue(consts, "CodeGhost"))
}

func TestFirstDiffLineIdenticalInputs(t *testing.T) {
	line, text := firstDiffLine("a\nb\n", "a\nb\n")
	assert.Zero(t, line, "no difference means no line to report")
	assert.Empty(t, text)
}

func TestPackageNameOfErrors(t *testing.T) {
	t.Run("missing directory", func(t *testing.T) {
		_, err := packageNameOf(filepath.Join(t.TempDir(), "absent"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no such file or directory")
	})

	t.Run("directory with no go files", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0o644))
		_, err := packageNameOf(dir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no .go files")
	})

	t.Run("go file with a broken package clause", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "broken_test.go"), []byte("!!! not go\n"), 0o644))
		_, err := packageNameOf(dir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "broken_test.go", "the error must name the file that failed to parse")
	})

	t.Run("test package name is preferred and stripped", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "a.go"), []byte("package syncpkg\n"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "b_test.go"), []byte("package syncpkg_test\n"), 0o644))
		name, err := packageNameOf(dir)
		require.NoError(t, err)
		assert.Equal(t, "syncpkg", name,
			"the _test suffix is stripped so example keys match the non-test package's own examples")
	})
}

// TestRunSyncStageFailures exercises each failure path of the sync engine:
// package parsing, example-package resolution, per-file marker scanning,
// rendering, and the write itself.
func TestRunSyncStageFailures(t *testing.T) {
	goodPkg := syncFixturePkg(t)

	docsWith := func(t *testing.T, body string) string {
		t.Helper()
		return writeSyncFixture(t, body)
	}

	t.Run("unparsable codes package", func(t *testing.T) {
		dir := pkgDir(t, map[string]string{"codes.go": "package main\n"})
		err := runSync(dir, nil, []string{docsWith(t, syncDocFixture)}, false, io.Discard)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parsing", "the failure must say which stage broke")
	})

	t.Run("example package that does not exist", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "absent")
		err := runSync(goodPkg, []string{missing}, []string{docsWith(t, syncDocFixture)}, false, io.Discard)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "example package "+missing)
	})

	t.Run("example package that fails extraction", func(t *testing.T) {
		dir := pkgDir(t, map[string]string{"x.go": "package broken\n\nfunc f( {\n"})
		err := runSync(goodPkg, []string{dir}, []string{docsWith(t, syncDocFixture)}, false, io.Discard)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "extracting examples from "+dir)
	})

	t.Run("duplicate example package is visited once", func(t *testing.T) {
		// The same directory twice must not be read twice: a second pass would
		// report every example as a duplicate.
		err := runSync(goodPkg, []string{goodPkg, goodPkg}, []string{docsWith(t, syncDocFixture)}, false, io.Discard)
		require.NoError(t, err, "a repeated -example-pkg must be deduplicated, not double-counted")
	})

	t.Run("unclosed marker in a docs file", func(t *testing.T) {
		err := runSync(goodPkg, nil, []string{docsWith(t, "text <!-- go:example Foo")}, true, io.Discard)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unclosed example marker")
	})

	t.Run("missing end marker in the errors block", func(t *testing.T) {
		err := runSync(goodPkg, nil, []string{docsWith(t, "x\n"+errStart+"\n")}, false, io.Discard)
		require.Error(t, err)
		assert.Contains(t, err.Error(), errEnd)
	})

	t.Run("marker naming no example", func(t *testing.T) {
		// Write mode stays permissive about coverage but cannot fill a block
		// whose example does not exist.
		in := "x\n<!-- go:errors -->\n<!-- /go:errors -->\n\n<!-- go:example syncpkg.ExampleNope -->\n\n<!-- /go:example -->\n"
		err := runSync(goodPkg, nil, []string{docsWith(t, in)}, false, io.Discard)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "syncpkg.ExampleNope")
	})

	t.Run("unreadable docs file", func(t *testing.T) {
		err := runSync(goodPkg, nil, []string{filepath.Join(t.TempDir(), "absent.md")}, false, io.Discard)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no such file or directory")
	})

	t.Run("unwritable docs file", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "doc.md"), []byte(syncDocFixture), 0o644))
		path := filepath.Join(dir, "doc.md")
		if os.Geteuid() == 0 {
			t.Skip("running as root: permission bits are not enforced")
		}
		// The file itself must be read-only. Tightening the directory is not
		// enough: O_TRUNC writes through the existing inode, so the directory's
		// write bit only controls creating or removing entries.
		require.NoError(t, os.Chmod(path, 0o400))
		t.Cleanup(func() { _ = os.Chmod(path, 0o644) })
		err := runSync(goodPkg, nil, []string{path}, false, io.Discard)
		require.Error(t, err, "a docs file that cannot be written must fail the sync")
		assert.Contains(t, err.Error(), path)
	})
}

// TestRunSyncDuplicateExampleAcrossPackages: two example packages that both
// declare the same Example name under the same package name would silently
// overwrite each other's body in the docs, so it must be refused.
func TestRunSyncDuplicateExampleAcrossPackages(t *testing.T) {
	src := "package syncpkg\n\nimport \"fmt\"\n\nfunc ExampleGreet() {\n\tfmt.Println(\"hi\")\n}\n"
	a := pkgDir(t, map[string]string{"a_test.go": src})
	b := pkgDir(t, map[string]string{"a_test.go": src})

	doc := writeSyncFixture(t, syncDocFixture)
	err := runSync(syncFixturePkg(t), []string{a, b}, []string{doc}, false, io.Discard)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "extracted from more than one package")
}
