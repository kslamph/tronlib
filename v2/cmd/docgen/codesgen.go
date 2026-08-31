package main

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// generateCodes parses pkgDir/codes.go and emits the source of the package's
// codes_gen.go: func (c Code) Action() Action and func (c Code) Doc() string
// with an explicit case per code in AllCodes.
//
// Nothing is invented here. The constants come from the Code-typed const
// block(s) and the case list from the AllCodes var, both in codes.go. The
// per-code Action and Doc values come from the existing (c Code) switches:
// codes.go's hand-written ones on first generation, and — once those have
// been deleted after migration — the generated ones in codes_gen.go. The
// two files use the same switch shape (one return per case, the same
// default arms), so regeneration after migration is a byte-identical fixed
// point. The generator owns no mapping table of its own.
//
// R-3 parity is enforced before anything is emitted, so a forgotten constant
// fails generation instead of escaping every test through the default arm:
//
//   - every Code constant appears in AllCodes exactly once
//   - every AllCodes entry is a declared Code constant (no duplicates)
//   - every AllCodes entry has an explicit Action case and an explicit Doc
//     case in codes.go; docgen does not generate default-arm fallbacks
func generateCodes(pkgDir string) (string, error) {
	constPath := filepath.Join(pkgDir, "codes.go")
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, constPath, nil, 0)
	if err != nil {
		return "", fmt.Errorf("%s: %w", constPath, err)
	}
	if f.Name.Name != "tron" {
		return "", fmt.Errorf("%s: package %q, want %q (codes_gen.go is generated into package tron)", constPath, f.Name.Name, "tron")
	}

	// codes_gen.go is the switch source once the hand-written switches have
	// been deleted from codes.go; it may not exist yet on first generation.
	genFile, err := parseOptional(fset, filepath.Join(pkgDir, "codes_gen.go"))
	if err != nil {
		return "", err
	}

	consts, err := codeConsts(f)
	if err != nil {
		return "", err
	}
	all, err := allCodesList(f)
	if err != nil {
		return "", err
	}
	action, err := switchMapping(f, genFile, "Action")
	if err != nil {
		return "", err
	}
	doc, err := switchMapping(f, genFile, "Doc")
	if err != nil {
		return "", err
	}

	if err := checkParity(constPath, consts, all, action, doc); err != nil {
		return "", err
	}
	return emitCodesGen(all, consts, action, doc)
}

// parseOptional parses path if it exists; a missing file yields (nil, nil).
func parseOptional(fset *token.FileSet, path string) (*ast.File, error) {
	src, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	f, err := parser.ParseFile(fset, path, src, 0)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return f, nil
}

// codeConst is one constant declared as type Code.
type codeConst struct {
	Name  string
	Value string // unquoted string value
}

// codeConsts returns every constant of type Code declared in f, in
// declaration order, with its unquoted string value. Constants whose type is
// inherited from the previous spec inside a const block are included.
// Iota-derived or non-literal values are an error: docgen needs the literal.
func codeConsts(f *ast.File) ([]codeConst, error) {
	var out []codeConst
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			continue
		}
		inCode := false // the previous spec in this block was a Code constant
		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			name := vs.Names[0].Name
			switch t := vs.Type.(type) {
			case *ast.Ident:
				inCode = t.Name == "Code"
			case nil:
				// type inherited from the previous spec in the block
			default:
				return nil, fmt.Errorf("constant %s: unsupported type expression %T; Code constants must be declared as `Name Code = \"value\"`", name, vs.Type)
			}
			if !inCode {
				continue
			}
			if len(vs.Names) != 1 {
				return nil, fmt.Errorf("const spec declares %d names; Code constants must be declared one per spec", len(vs.Names))
			}
			if len(vs.Values) != 1 {
				return nil, fmt.Errorf("constant %s: no literal value (iota-derived constants are not supported); every Code constant must carry a literal string", name)
			}
			lit, ok := vs.Values[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return nil, fmt.Errorf("constant %s: value is not a string literal; every Code constant must carry a literal string", name)
			}
			val, err := strconv.Unquote(lit.Value)
			if err != nil {
				return nil, fmt.Errorf("constant %s: %w", name, err)
			}
			out = append(out, codeConst{Name: name, Value: val})
		}
	}
	return out, nil
}

// allCodesList returns the identifier list of the AllCodes var, in order.
func allCodesList(f *ast.File) ([]string, error) {
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.VAR {
			continue
		}
		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok || len(vs.Names) != 1 || vs.Names[0].Name != "AllCodes" {
				continue
			}
			if len(vs.Values) != 1 {
				return nil, fmt.Errorf("AllCodes: must be a single slice literal")
			}
			cl, ok := vs.Values[0].(*ast.CompositeLit)
			if !ok {
				return nil, fmt.Errorf("AllCodes: must be a []Code{...} literal, got %T", vs.Values[0])
			}
			var out []string
			for _, elt := range cl.Elts {
				id, ok := elt.(*ast.Ident)
				if !ok {
					return nil, fmt.Errorf("AllCodes: entry is %T, not a bare Code constant identifier; every entry must be a declared constant", elt)
				}
				out = append(out, id.Name)
			}
			return out, nil
		}
	}
	return nil, fmt.Errorf("no `var AllCodes = []Code{...}` found")
}

// switchMapping extracts the per-code data of one of the (c Code) methods
// (Action or Doc) from the switch in its body. The switch is looked up in
// primary (codes.go) first and, failing that, in fallback (codes_gen.go,
// present once the hand-written switches have been deleted).
//
// Each case arm must be a single return whose result is, for Action, a bare
// Action identifier and, for Doc, a string literal. The default arm must
// return ActionBug / "unknown code: " + string(c): generation re-emits that
// default verbatim, so a different default would be a silent behavior change.
func switchMapping(primary, fallback *ast.File, method string) (map[string]string, error) {
	fn := codeMethod(primary, method)
	if fn == nil && fallback != nil {
		fn = codeMethod(fallback, method)
	}
	if fn == nil {
		return nil, fmt.Errorf("no func (c Code) %s() found in codes.go or codes_gen.go", method)
	}
	var sw *ast.SwitchStmt
	for _, stmt := range fn.Body.List {
		if s, ok := stmt.(*ast.SwitchStmt); ok {
			sw = s
			break
		}
	}
	if sw == nil {
		return nil, fmt.Errorf("func (c Code) %s: body has no switch statement", method)
	}

	out := make(map[string]string)
	for _, stmt := range sw.Body.List {
		cc, ok := stmt.(*ast.CaseClause)
		if !ok {
			continue
		}
		ret, err := singleReturn(cc, method)
		if err != nil {
			return nil, err
		}
		if cc.List == nil { // default arm
			if err := checkDefaultArm(ret, method); err != nil {
				return nil, err
			}
			continue
		}
		var value string
		switch method {
		case "Action":
			id, ok := ret.(*ast.Ident)
			if !ok {
				return nil, fmt.Errorf("func (c Code) Action: case returns %s — must be a bare Action identifier", renderExpr(ret))
			}
			value = id.Name
		case "Doc":
			lit, ok := ret.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return nil, fmt.Errorf("func (c Code) Doc: case returns %s — must be a string literal", renderExpr(ret))
			}
			s, err := strconv.Unquote(lit.Value)
			if err != nil {
				return nil, fmt.Errorf("func (c Code) Doc: %w", err)
			}
			value = s
		}
		for _, e := range cc.List {
			id, ok := e.(*ast.Ident)
			if !ok {
				return nil, fmt.Errorf("func (c Code) %s: case expression is %s — must be a bare Code constant identifier", method, renderExpr(e))
			}
			if _, dup := out[id.Name]; dup {
				return nil, fmt.Errorf("func (c Code) %s: code %s appears in more than one case", method, id.Name)
			}
			out[id.Name] = value
		}
	}
	return out, nil
}

// codeMethod finds the value-receiver (c Code) method with the given name.
func codeMethod(f *ast.File, name string) *ast.FuncDecl {
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != name || fn.Recv == nil || len(fn.Recv.List) != 1 {
			continue
		}
		switch t := fn.Recv.List[0].Type.(type) {
		case *ast.Ident:
			if t.Name == "Code" {
				return fn
			}
		case *ast.StarExpr:
			if id, ok := t.X.(*ast.Ident); ok && id.Name == "Code" {
				return fn
			}
		}
	}
	return nil
}

// singleReturn requires each case arm to be exactly one return statement
// with one result.
func singleReturn(cc *ast.CaseClause, method string) (ast.Expr, error) {
	if len(cc.Body) != 1 {
		return nil, fmt.Errorf("func (c Code) %s: case arm must be a single return statement, got %d statements", method, len(cc.Body))
	}
	ret, ok := cc.Body[0].(*ast.ReturnStmt)
	if !ok {
		return nil, fmt.Errorf("func (c Code) %s: case arm must be a return statement, got %T", method, cc.Body[0])
	}
	if len(ret.Results) != 1 {
		return nil, fmt.Errorf("func (c Code) %s: return must have exactly one result, got %d", method, len(ret.Results))
	}
	return ret.Results[0], nil
}

// checkDefaultArm pins the default arm the generated switch will re-emit.
func checkDefaultArm(ret ast.Expr, method string) error {
	switch method {
	case "Action":
		id, ok := ret.(*ast.Ident)
		if !ok || id.Name != "ActionBug" {
			return fmt.Errorf("func (c Code) Action: default arm returns %s, want ActionBug; docgen re-emits the default verbatim and would silently change behavior", renderExpr(ret))
		}
	case "Doc":
		bin, ok := ret.(*ast.BinaryExpr)
		if !ok || bin.Op != token.ADD {
			return fmt.Errorf("func (c Code) Doc: default arm returns %s, want \"unknown code: \" + string(c)", renderExpr(ret))
		}
		pfx, ok := bin.X.(*ast.BasicLit)
		if !ok || pfx.Kind != token.STRING || pfx.Value != `"unknown code: "` {
			return fmt.Errorf("func (c Code) Doc: default arm returns %s, want \"unknown code: \" + string(c)", renderExpr(ret))
		}
		call, ok2 := bin.Y.(*ast.CallExpr)
		if !ok2 || len(call.Args) != 1 {
			return fmt.Errorf("func (c Code) Doc: default arm returns %s, want \"unknown code: \" + string(c)", renderExpr(ret))
		}
		fn, ok3 := call.Fun.(*ast.Ident)
		arg, ok4 := call.Args[0].(*ast.Ident)
		if !ok3 || fn.Name != "string" || !ok4 || arg.Name != "c" {
			return fmt.Errorf("func (c Code) Doc: default arm returns %s, want \"unknown code: \" + string(c)", renderExpr(ret))
		}
	}
	return nil
}

// renderExpr renders an expression for an error message.
func renderExpr(e ast.Expr) string {
	var b strings.Builder
	if err := format.Node(&b, token.NewFileSet(), e); err != nil {
		return fmt.Sprintf("%T", e)
	}
	return b.String()
}

// checkParity enforces R-3: const <-> AllCodes is a bijection, and every
// code has explicit Action and Doc data. Errors name the offending constant.
func checkParity(path string, consts []codeConst, all []string, action, doc map[string]string) error {
	declared := make(map[string]bool, len(consts))
	for _, c := range consts {
		declared[c.Name] = true
	}
	var problems []string
	for _, name := range all {
		if !declared[name] {
			problems = append(problems, fmt.Sprintf("AllCodes entry %s is not a declared Code constant", name))
		}
	}
	counts := make(map[string]int, len(all))
	for _, name := range all {
		counts[name]++
	}
	for _, c := range consts {
		switch counts[c.Name] {
		case 0:
			problems = append(problems, fmt.Sprintf("constant %s (%q) is missing from AllCodes — add it, or it will escape every test through the default arm", c.Name, c.Value))
		case 1:
		default:
			problems = append(problems, fmt.Sprintf("AllCodes lists %s %d times; every code must appear exactly once", c.Name, counts[c.Name]))
		}
	}
	for _, name := range all {
		if _, ok := action[name]; !ok {
			problems = append(problems, fmt.Sprintf("code %s has no explicit case in the Action switch; docgen does not generate default-arm fallbacks", name))
		}
		if _, ok := doc[name]; !ok {
			problems = append(problems, fmt.Sprintf("code %s has no explicit case in the Doc switch; docgen does not generate default-arm fallbacks", name))
		}
	}
	for name := range action {
		if !declared[name] {
			problems = append(problems, fmt.Sprintf("Action switch case %s is not a declared Code constant", name))
		}
	}
	for name := range doc {
		if !declared[name] {
			problems = append(problems, fmt.Sprintf("Doc switch case %s is not a declared Code constant", name))
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("%s: codes.go parity violations:\n\t%s", path, strings.Join(problems, "\n\t"))
	}
	return nil
}
