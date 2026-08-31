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

// --- tx.Tx sealing fixtures (Task 6, spec §6.1) ---
//
// tx.Tx is sealed with the unexported txInternal method, so a foreign type
// implementing every exported Tx method is still NOT a Tx — a hand-rolled
// transaction that bypasses the builders (and their fee-limit, expiration
// and permission-id defaults) cannot be broadcast. The positive control is
// that the four real kinds DO satisfy the interface.

const txImportPath = "github.com/kslamph/tronlib/v2/tx"

// txForeignFixture implements every exported Tx method but, being outside
// the tx package, cannot implement txInternal.
const txForeignFixture = `package p

import (
	"time"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/v2/tx"
	"github.com/kslamph/tronlib/v2/tron"
)

type fakeTx struct{}

func (fakeTx) ID() string                           { return "" }
func (fakeTx) Kind() tx.Kind                        { return 0 }
func (fakeTx) Extension() *api.TransactionExtention { return nil }
func (fakeTx) Transaction() *core.Transaction       { return nil }
func (fakeTx) Signers() ([]tron.Address, error)     { return nil, nil }
func (fakeTx) IsSigned() bool                       { return false }
func (fakeTx) FeeLimit() tron.SUN                   { return 0 }
func (fakeTx) Expiration() time.Time                { return time.Time{} }
func (fakeTx) PermissionID() int32                  { return 0 }

var _ tx.Tx = fakeTx{}
`

// txPositiveFixture: the four built kinds satisfy Tx, and Simulate/
// EstimateEnergy compile on *ContractTx (the F1 fix positive control).
const txPositiveFixture = `package p

import (
	"context"

	"github.com/kslamph/tronlib/v2/tx"
)

var _ tx.Tx = (*tx.NativeTx)(nil)
var _ tx.Tx = (*tx.ContractTx)(nil)
var _ tx.Tx = (*tx.DeployTx)(nil)
var _ tx.Tx = (*tx.AssetTx)(nil)

func simulateContractTx(c *tx.ContractTx, ctx context.Context) {
	_, _ = c.Simulate(ctx)
	_, _ = c.EstimateEnergy(ctx)
}
`

// txF1NegativeFixture: Simulate and EstimateEnergy exist ONLY on
// *ContractTx (spec §6.2). Calling them on a *NativeTx is a compile error —
// the static kind replaces the runtime dispatch v1 could forget. One fixture
// exercises both methods; either alone would fail to type-check.
const txF1NegativeFixture = `package p

import (
	"context"

	"github.com/kslamph/tronlib/v2/tx"
)

func simulateNativeTx(n *tx.NativeTx, ctx context.Context) {
	_ = n.Simulate(ctx)
	_, _ = n.EstimateEnergy(ctx)
}
`

func TestForeignTypeCannotSatisfyTx(t *testing.T) {
	imp := loadTxGraph(t)
	if err := typecheckTx(t, imp, txForeignFixture); err == nil {
		t.Fatal("a foreign type with every exported Tx method compiled as tx.Tx; the seal (txInternal) failed")
	} else {
		t.Logf("foreign type rejected as expected: %v", err)
	}
}

func TestTxKindsSatisfyTxAndSimulateIsContractOnly(t *testing.T) {
	imp := loadTxGraph(t)
	if err := typecheckTx(t, imp, txPositiveFixture); err != nil {
		t.Fatalf("positive control failed: %v", err)
	}
	if err := typecheckTx(t, imp, txF1NegativeFixture); err == nil {
		t.Fatal("Simulate/EstimateEnergy on *NativeTx compiled; the F1 fix (ContractTx-only read paths) failed")
	} else {
		t.Logf("Simulate on *NativeTx rejected as expected: %v", err)
	}
}

// typecheckTx type-checks a full-source fixture with imp.
func typecheckTx(t *testing.T, imp types.Importer, src string) error {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatalf("fixture does not parse: %v", err)
	}
	conf := types.Config{Importer: imp, Error: func(error) {}}
	_, typeErr := conf.Check("p", fset, []*ast.File{f}, nil)
	return typeErr
}

// loadTxGraph loads the tx package with its full dependency graph and an
// importer serving every reachable package (tx, tron, pb, stdlib).
func loadTxGraph(t *testing.T) types.Importer {
	t.Helper()
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedName | packages.NeedImports | packages.NeedTypes | packages.NeedDeps,
		Dir:  "../..",
	}, txImportPath)
	if err != nil {
		t.Fatalf("load tx: %v", err)
	}
	if len(pkgs) == 0 || pkgs[0].Types == nil {
		t.Fatal("tx package not found or has no types")
	}
	if errs := pkgs[0].Errors; len(errs) > 0 {
		t.Fatalf("tx package has load errors: %v", errs)
	}
	graph := map[string]*types.Package{}
	var visit func(p *packages.Package)
	visit = func(p *packages.Package) {
		if p.Types != nil && graph[p.PkgPath] == nil {
			graph[p.PkgPath] = p.Types
			for _, imp := range p.Imports {
				visit(imp)
			}
		}
	}
	visit(pkgs[0])
	if graph[txImportPath] == nil || graph[tronImportPath] == nil {
		t.Fatal("tx graph incomplete: missing tx or tron types")
	}
	return graphImporter{graph}
}

type graphImporter struct{ graph map[string]*types.Package }

func (g graphImporter) Import(path string) (*types.Package, error) {
	if p, ok := g.graph[path]; ok && p.Complete() {
		return p, nil
	}
	return nil, fmt.Errorf("unexpected import %q", path)
}
