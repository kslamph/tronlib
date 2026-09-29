package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// extractExamples parses every *.go file in dir (regular and _test files;
// non-recursive — a Go package is flat by definition) and returns the
// gofmt-rendered body of every top-level example function: func Example,
// func ExampleXxx, func ExampleXxx_Yyy — with zero parameters and zero
// results. Non-example functions and examples with signatures are skipped.
//
// The map key is the example's function name; the value is the statements
// between the braces without the `func ExampleX() {` wrapper and without
// the trailing `}`, trimmed. Bodies are comment-free: files are parsed
// without comments so the // Output: harness line and narration comments
// never leak into a rendered doc marker.
//
// File read errors and syntax errors are returned mentioning the file.
// Two example functions with the same name in one package cannot compile,
// but nothing stops two files from parsing that way, so a collision is an
// error rather than a silent overwrite.
func extractExamples(dir string) (map[string]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(entries))
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".go" {
			continue
		}
		path := filepath.Join(dir, e.Name())
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Body == nil || !strings.HasPrefix(fn.Name.Name, "Example") {
				continue
			}
			if fn.Type.Params.NumFields() != 0 || fn.Type.Results.NumFields() != 0 {
				continue
			}
			body, err := renderBody(fset, fn.Body.List)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
			if _, dup := out[fn.Name.Name]; dup {
				return nil, fmt.Errorf("%s: example %q is defined more than once in the package", path, fn.Name.Name)
			}
			out[fn.Name.Name] = body
		}
	}
	return out, nil
}

// renderBody gofmt-renders a statement list at column 0 and trims the
// trailing newline.
func renderBody(fset *token.FileSet, stmts []ast.Stmt) (string, error) {
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, stmts); err != nil {
		return "", fmt.Errorf("rendering example body: %w", err)
	}
	return strings.TrimSpace(buf.String()), nil
}
