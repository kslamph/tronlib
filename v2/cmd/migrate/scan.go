package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Symbol is one exported v1 or v2 API element.
type Symbol struct {
	Pkg  string // short package name, e.g. "account"
	Name string // "Manager", "Transfer", "TRX"
	Recv string // receiver type name for methods, "" otherwise
	Kind string // "func" | "type" | "method" | "const"
	Key  string // "pkg.Type.Method" | "pkg.Type" | "pkg.Func"
	File string // source path, for path-based removal
}

func symbolKey(pkg, recv, name string) string {
	if recv != "" {
		return pkg + "." + recv + "." + name
	}
	return pkg + "." + name
}

// scanDir parses every non-test *.go file directly in dir (non-recursive) and
// returns its exported top-level funcs, methods, types, and consts, sorted by
// Key.
func scanDir(dir string) ([]Symbol, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	var out []Symbol
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(dir, name)
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		out = append(out, symbolsInFile(f, path)...)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out, nil
}

// scanTree walks root and scans every directory that contains a non-test .go
// file, so a package tree can be scanned without naming each package.
func scanTree(root string) ([]Symbol, error) {
	var out []Symbol
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() || !hasNonTestGo(path) {
			return nil
		}
		syms, err := scanDir(path)
		if err != nil {
			return err
		}
		out = append(out, syms...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out, nil
}

func hasNonTestGo(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		n := e.Name()
		if !e.IsDir() && strings.HasSuffix(n, ".go") && !strings.HasSuffix(n, "_test.go") {
			return true
		}
	}
	return false
}

func symbolsInFile(f *ast.File, path string) []Symbol {
	pkg := f.Name.Name
	var out []Symbol
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if !ast.IsExported(d.Name.Name) {
				continue
			}
			recv := ""
			if d.Recv != nil && len(d.Recv.List) > 0 {
				recv = receiverName(d.Recv.List[0].Type)
			}
			kind := "func"
			if recv != "" {
				kind = "method"
			}
			out = append(out, Symbol{
				Pkg: pkg, Name: d.Name.Name, Recv: recv, Kind: kind,
				Key: symbolKey(pkg, recv, d.Name.Name), File: path,
			})
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if ast.IsExported(s.Name.Name) {
						out = append(out, Symbol{
							Pkg: pkg, Name: s.Name.Name, Kind: "type",
							Key: symbolKey(pkg, "", s.Name.Name), File: path,
						})
					}
				case *ast.ValueSpec:
					if d.Tok != token.CONST {
						continue
					}
					for _, n := range s.Names {
						if ast.IsExported(n.Name) {
							out = append(out, Symbol{
								Pkg: pkg, Name: n.Name, Kind: "const",
								Key: symbolKey(pkg, "", n.Name), File: path,
							})
						}
					}
				}
			}
		}
	}
	return out
}

// receiverName unwraps *T, T and generic T[P] receiver expressions to the
// receiver type name.
func receiverName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return receiverName(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		return receiverName(t.X)
	case *ast.IndexListExpr:
		return receiverName(t.X)
	}
	return ""
}
