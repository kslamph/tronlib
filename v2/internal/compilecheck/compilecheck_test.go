package compilecheck

// The tests here run go/types over fixture snippets against the REAL tron
// package (loaded with golang.org/x/tools/go/packages, which is module-aware;
// go/importer.Default() is not and fails with "cannot find import").
// They assert type-level guarantees that no ordinary unit test can express:
// the Whole constraint must admit only the predeclared signed integer types.
//
// Spec section 5.1 documents two money-loss bugs verified by execution:
//   - with ~int64 in the constraint, SUN's own underlying type satisfies it,
//     so tron.TRX(someSUN) compiles and re-scales an already-scaled value
//     (1 TRX silently becomes 1,000,000 TRX);
//   - with unsigned kinds in the constraint, a uint64 above MaxInt64 converts
//     to a negative int64 inside TRX and slips past the overflow guard.
//
// The mustCompile positive control is what keeps this suite honest: if the
// loader broke and every snippet failed to type-check, the negative assertions
// would pass vacuously. The positive control fails loudly in that case.

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

const tronImportPath = "github.com/kslamph/tronlib/v2/tron"

// mustFail holds statements that must NOT type-check. Every fixture binds x
// and assigns it to _, so a failure can only come from the TRX call itself,
// never from go/types' "declared and not used" check.
var mustFail = map[string]string{
	"float literal": `x := tron.TRX(1.6); _ = x`,
	// Not `tron.TRX(tron.TRX(1))`-style: with ~int64 the inner TRX(1) itself
	// stops compiling (untyped 1 infers T=int), which would mask the bug. A
	// parsed amount is the realistic double-scaling path.
	"already-scaled": `v, _ := tron.ParseTRX("1"); x := tron.TRX(v); _ = x`,
	"unsigned":       `var u uint64 = 3; x := tron.TRX(u); _ = x`,
	"string":         `x := tron.TRX("1"); _ = x`,
	"any":            `var a any = 1; x := tron.TRX(a); _ = x`,
}

// mustCompile is the positive control: these must type-check, or the loader
// is broken and the negative assertions prove nothing.
var mustCompile = map[string]string{
	"int literal":    `x := tron.TRX(1); _ = x`,
	"int64 value":    `x := tron.TRX(int64(5)); _ = x`,
	"int8 value":     `x := tron.TRX(int8(1)); _ = x`,
	"int32 value":    `x := tron.TRX(int32(1)); _ = x`,
	"negative value": `x := tron.TRX(-2); _ = x`,
}

func TestAmountInputsThatMustNotCompile(t *testing.T) {
	tron := loadTron(t)
	for name, stmt := range mustFail {
		if err := typecheck(t, tron, stmt); err == nil {
			t.Errorf("%s: compiled but MUST NOT: %s", name, stmt)
		}
	}
}

func TestAmountInputsThatMustCompile(t *testing.T) {
	tron := loadTron(t)
	for name, stmt := range mustCompile {
		if err := typecheck(t, tron, stmt); err != nil {
			t.Errorf("%s: failed to type-check but MUST compile: %s\n%v", name, stmt, err)
		}
	}
}

// typecheck type-checks a one-function fixture against the real tron package
// and returns the first type error, or nil if the snippet compiles.
func typecheck(t *testing.T, tron *types.Package, stmt string) error {
	t.Helper()
	src := "package p\n\nimport \"github.com/kslamph/tronlib/v2/tron\"\n\nfunc f() {\n\t" + stmt + "\n}\n"
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatalf("fixture does not parse: %s: %v", stmt, err)
	}
	conf := types.Config{Importer: stubImporter{p: tron}, Error: func(error) {}}
	_, typeErr := conf.Check("p", fset, []*ast.File{f}, nil)
	return typeErr
}

// loadTron loads the real tron package from source with full type information.
// It hard-fails unless the SUN/TRX/ParseTRX API is present in the scope, so a
// broken load can never make the negative assertions pass vacuously.
func loadTron(t *testing.T) *types.Package {
	t.Helper()
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedTypes | packages.NeedDeps,
		Dir:  "../..", // the v2 module root, relative to this test's directory
	}, tronImportPath)
	if err != nil {
		t.Fatalf("load tron: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatalf("tron package not found at %s", tronImportPath)
	}
	if errs := pkgs[0].Errors; len(errs) > 0 {
		t.Fatalf("tron package has load errors: %v", errs)
	}
	p := pkgs[0].Types
	if p == nil || len(p.Scope().Names()) == 0 {
		t.Fatalf("tron loaded empty — does the package exist and compile?")
	}
	scope := p.Scope()
	for _, want := range []string{"SUN", "TRX", "ParseTRX"} {
		if scope.Lookup(want) == nil {
			t.Fatalf("tron loaded but missing %s; scope names: %s",
				want, strings.Join(scope.Names(), ", "))
		}
	}
	t.Logf("compilecheck loaded the real tron package; scope names: %s", strings.Join(scope.Names(), ", "))
	return p
}

type stubImporter struct{ p *types.Package }

func (s stubImporter) Import(path string) (*types.Package, error) {
	if path == tronImportPath {
		return s.p, nil
	}
	return nil, fmt.Errorf("unexpected import %q", path)
}
