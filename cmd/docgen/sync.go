package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// codeTable renders parsed package data as the error-table rows docgen
// writes between the go:errors markers. The rows are the same codeDoc data
// generateCodes emits into codes_gen.go (Code, Action, Doc), derived from
// the parsed switch data — codes_gen.go text is never re-parsed for the
// table. Rows follow AllCodes order, the same order the generated switch
// uses, so the table, the generated source, and the architecture doc's code numbering
// stay aligned. renderErrorList sorts by code string, so presentation order
// is diff-stable regardless of AllCodes order.
func codeTable(data codesData) []codeDoc {
	rows := make([]codeDoc, 0, len(data.all))
	for _, name := range data.all {
		rows = append(rows, codeDoc{
			Code:   constValue(data.consts, name),
			Action: actionName(data.action[name], data.actionStrings),
			Doc:    data.doc[name],
		})
	}
	return rows
}

// constValue returns the string value of the named Code constant.
func constValue(consts []codeConst, name string) string {
	for _, c := range consts {
		if c.Name == name {
			return c.Value
		}
	}
	return name // unreachable after checkParity: every AllCodes entry is a declared constant
}

// checkExampleCoverage closes the deleted-block drift that pure marker
// filling cannot see: a docs file that loses an example block still passes
// a fill-and-compare check, silently dropping the example from the docs.
// Both directions are asserted:
//
//   - every Example function in the package must have a
//     "<!-- go:example NAME -->" marker in at least one docs file;
//   - every example marker in the docs files must name a real Example
//     function.
//
// Errors name the orphaned name, so the fix is obvious from the message.
//
// The error-table direction needs no equivalent check: the table is
// generated from the parsed package, so it cannot omit a code that exists.
func checkExampleCoverage(examples map[string]string, docsMarkers map[string][]string) error {
	marked := make(map[string]bool)
	for _, names := range docsMarkers {
		for _, name := range names {
			if _, ok := examples[name]; !ok {
				return fmt.Errorf("example %q has a go:example marker in the docs but no Example function in the package", name)
			}
			marked[name] = true
		}
	}
	var orphans []string
	for name := range examples {
		if !marked[name] {
			orphans = append(orphans, name)
		}
	}
	if len(orphans) > 0 {
		sort.Strings(orphans)
		return fmt.Errorf("Example function(s) with no go:example marker in any docs file: %s; add <!-- go:example NAME --> blocks", strings.Join(orphans, ", "))
	}
	return nil
}

// packageNameOf returns the namespace examples from dir are documented under:
// the package clause of its example files, with a trailing "_test" stripped
// (an Example in package tron_test documents package tron). Example files are
// preferred; a package with no example file falls back to its regular source.
func packageNameOf(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, wantTest := range []bool{true, false} {
		for _, e := range entries {
			n := e.Name()
			if e.IsDir() || !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") != wantTest {
				continue
			}
			f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, n), nil, parser.PackageClauseOnly)
			if err != nil {
				return "", fmt.Errorf("%s: %w", filepath.Join(dir, n), err)
			}
			return strings.TrimSuffix(f.Name.Name, "_test"), nil
		}
	}
	return "", fmt.Errorf("%s: no .go files", dir)
}

// runSync is the sync-docs engine shared by the sync and -check paths.
// It renders every docs file — filling the go:errors table from the parsed
// package and every go:example block from the package's Example functions —
// then either writes the result back (sync) or byte-compares it against the
// file on disk and reports the first differing line (check).
//
// The example-coverage check (both marker directions) runs in -check mode
// across all docs files before the byte comparison, so the CI gate fails
// closed. Plain sync stays a permissive filler: it must be able to refresh
// a single docs file (e.g. only errors.md) without the full docs set.
func runSync(codesPkg string, examplePkgs []string, docsFiles []string, check bool) error {
	data, err := parseCodesData(codesPkg)
	if err != nil {
		return fmt.Errorf("parsing %s: %w", codesPkg, err)
	}
	// Examples are namespaced <package>.<ExampleFunc> so two packages may
	// define the same Example name without colliding (multiple -example-pkg).
	examples := map[string]string{}
	seen := map[string]bool{}
	for _, dir := range append([]string{codesPkg}, examplePkgs...) {
		if seen[dir] {
			continue
		}
		seen[dir] = true
		pkgName, err := packageNameOf(dir)
		if err != nil {
			return fmt.Errorf("example package %s: %w", dir, err)
		}
		ex, err := extractExamples(dir)
		if err != nil {
			return fmt.Errorf("extracting examples from %s: %w", dir, err)
		}
		for fn, body := range ex {
			key := pkgName + "." + fn
			if _, dup := examples[key]; dup {
				return fmt.Errorf("example %q extracted from more than one package", key)
			}
			examples[key] = body
		}
	}
	rows := codeTable(data)

	raw := make(map[string]string, len(docsFiles))
	markers := make(map[string][]string, len(docsFiles))
	for _, docPath := range docsFiles {
		b, err := os.ReadFile(docPath)
		if err != nil {
			return err
		}
		raw[docPath] = string(b)
		names, err := exampleNames(raw[docPath])
		if err != nil {
			return fmt.Errorf("%s: %w", docPath, err)
		}
		markers[docPath] = names
	}

	if check {
		if err := checkExampleCoverage(examples, markers); err != nil {
			return err
		}
	}

	for _, docPath := range docsFiles {
		out, err := renderErrorList(raw[docPath], rows)
		if err != nil {
			return fmt.Errorf("%s: %w", docPath, err)
		}
		out, err = fillExamples(out, examples)
		if err != nil {
			return fmt.Errorf("%s: %w", docPath, err)
		}
		if check {
			if raw[docPath] != out {
				line, text := firstDiffLine(out, raw[docPath])
				return fmt.Errorf("%s is stale (docs drifted from package source); first difference at line %d: %q; run: docgen sync-docs -pkg <dir> -docs <files>", docPath, line, text)
			}
		} else {
			if err := os.WriteFile(docPath, []byte(out), 0o644); err != nil {
				return err
			}
			fmt.Printf("docgen: synced %s\n", docPath)
		}
	}
	return nil
}

// firstDiffLine returns the 1-based line number and the offending line from
// got (the file on disk) at the first position where got differs from want
// (the rendered expectation), for drift diagnostics.
func firstDiffLine(want, got string) (int, string) {
	wantLines := strings.Split(want, "\n")
	gotLines := strings.Split(got, "\n")
	for i := 0; i < len(wantLines) || i < len(gotLines); i++ {
		var w, g string
		if i < len(wantLines) {
			w = wantLines[i]
		}
		if i < len(gotLines) {
			g = gotLines[i]
		}
		if w != g {
			return i + 1, g
		}
	}
	return 0, ""
}
