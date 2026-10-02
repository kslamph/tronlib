package main

// Error-path tests for codesgen.go. docgen's job is to refuse to generate:
// every rejection here exists because a silent acceptance would put a wrong
// error-code table or a wrong Action/Doc mapping into package tron, where it
// would be indistinguishable from a correct one. So each rejection is asserted
// to name the construct it rejects.
//
// The fixtures are synthesized `package tron` sources rather than directories
// under testdata/, because most of these are one-line mutations of an otherwise
// valid file; keeping them inline shows the defect next to the assertion.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mustParse parses a synthetic codes.go body into an *ast.File. The source is
// never type-checked, so fixtures can name identifiers that do not exist - the
// extraction is purely syntactic.
func mustParse(t *testing.T, src string) *ast.File {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), "codes.go", src, 0)
	require.NoError(t, err, "fixture did not parse")
	return f
}

// codesPreamble is the smallest source that passes codeConsts and
// allCodesList. Each fixture below appends or mutates one construct.
const codesPreamble = `package tron

type Code string

type Action int

const (
	CodeAlpha Code = "alpha.one"
)

var AllCodes = []Code{CodeAlpha}
`

// actionSwitch and docSwitch are the conforming forms, used as the base for
// mutations of the switch data.
const actionSwitch = `
func (c Code) Action() Action {
	switch c {
	case CodeAlpha:
		return ActionRetry
	default:
		return ActionBug
	}
}
`

const docSwitch = `
func (c Code) Doc() string {
	switch c {
	case CodeAlpha:
		return "alpha doc"
	default:
		return "unknown code: " + string(c)
	}
}
`

// pkgDir writes files into a temp dir and returns its path.
func pkgDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, src := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644))
	}
	return dir
}

func TestParseCodesDataRejectsBadFiles(t *testing.T) {
	for _, tt := range []struct {
		name    string
		files   map[string]string
		wantSub string
	}{
		{
			name:    "syntax error in codes.go",
			files:   map[string]string{"codes.go": "package tron\n\nfunc { ("},
			wantSub: "expected",
		},
		{
			name:    "codes.go is not package tron",
			files:   map[string]string{"codes.go": "package main\n\nvar AllCodes = []Code{}\n"},
			wantSub: `package "main", want "tron"`,
		},
		{
			name: "syntax error in codes_gen.go",
			files: map[string]string{
				"codes.go":     codesPreamble + actionSwitch + docSwitch,
				"codes_gen.go": "package tron\n\nfunc (c Code) Action() Action { switch {",
			},
			wantSub: "codes_gen.go",
		},
		{
			name: "unsupported constant type",
			files: map[string]string{"codes.go": `package tron

const CodeAlpha otherpkg.Code = "alpha.one"

var AllCodes = []Code{CodeAlpha}
` + actionSwitch + docSwitch},
			wantSub: "unsupported type expression",
		},
		{
			name: "two names in one const spec",
			files: map[string]string{"codes.go": `package tron

const (
	CodeAlpha, CodeBeta Code = "alpha", "beta"
)

var AllCodes = []Code{CodeAlpha}
` + actionSwitch + docSwitch},
			wantSub: "must be declared one per spec",
		},
		{
			name: "inherited constant has no literal",
			files: map[string]string{"codes.go": `package tron

const (
	CodeAlpha Code = "alpha.one"
	CodeBeta
)

var AllCodes = []Code{CodeAlpha}
` + actionSwitch + docSwitch},
			wantSub: "no literal value",
		},
		{
			name: "iota-derived constant",
			files: map[string]string{"codes.go": `package tron

const CodeAlpha Code = iota

var AllCodes = []Code{CodeAlpha}
` + actionSwitch + docSwitch},
			wantSub: "not a string literal",
		},
		{
			name:    "AllCodes missing entirely",
			files:   map[string]string{"codes.go": codesPreambleNoAll + actionSwitch + docSwitch},
			wantSub: "no `var AllCodes",
		},
		{
			// A VAR declaration that is not AllCodes must be stepped over, not
			// rejected: package tron declares other vars alongside the table.
			name: "unrelated var before AllCodes",
			files: map[string]string{"codes.go": `package tron

const CodeAlpha Code = "alpha.one"

var SomeOtherThing = 1

var AllCodes = []Code{CodeAlpha}
` + actionSwitch + docSwitch},
			wantSub: "", // accepted; the point is that it does not error
		},
		{
			// `var AllCodes []Code` is valid Go with no initializer, so the
			// single-value guard is reachable from real source.
			name: "AllCodes declared without a value",
			files: map[string]string{"codes.go": `package tron

const CodeAlpha Code = "alpha.one"

var AllCodes []Code
` + actionSwitch + docSwitch},
			wantSub: "must be a single slice literal",
		},
		{
			name:    "no Action switch",
			files:   map[string]string{"codes.go": codesPreamble + docSwitch},
			wantSub: "no func (c Code) Action() found",
		},
		{
			name:    "no Doc switch",
			files:   map[string]string{"codes.go": codesPreamble + actionSwitch},
			wantSub: "no func (c Code) Doc() found",
		},
		{
			name: "AllCodes entry is not an identifier",
			files: map[string]string{"codes.go": `package tron

const CodeAlpha Code = "alpha.one"

var AllCodes = []Code{"alpha.one"}
` + actionSwitch + docSwitch},
			wantSub: "not a bare Code constant identifier",
		},
		{
			name: "AllCodes is not a slice literal",
			files: map[string]string{"codes.go": `package tron

const CodeAlpha Code = "alpha.one"

var AllCodes = buildAllCodes()
` + actionSwitch + docSwitch},
			wantSub: "must be a []Code{...} literal",
		},
		{
			name:    "codes.go absent",
			files:   map[string]string{"other.go": "package tron\n"},
			wantSub: "no such file or directory",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseCodesData(pkgDir(t, tt.files))
			if tt.wantSub == "" {
				require.NoError(t, err, "a package docgen cannot reject must parse")
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantSub, "the rejection must name the construct")
		})
	}
}

// codesPreambleNoAll is codesPreamble without the AllCodes var.
const codesPreambleNoAll = `package tron

type Code string

type Action int

const (
	CodeAlpha Code = "alpha.one"
)
`

// TestParseCodesDataUnreadableCodesGen: parseOptional treats a missing
// codes_gen.go as "first generation", but any other read failure must stop it.
func TestParseCodesDataUnreadableCodesGen(t *testing.T) {
	dir := pkgDir(t, map[string]string{"codes.go": codesPreamble + actionSwitch + docSwitch})
	require.NoError(t, os.Mkdir(filepath.Join(dir, "codes_gen.go"), 0o755))
	_, err := parseCodesData(dir)
	require.Error(t, err, "a codes_gen.go that cannot be read as a file must fail, not be treated as absent")
}

func TestSwitchMappingRejections(t *testing.T) {
	for _, tt := range []struct {
		name    string
		method  string
		src     string
		wantSub string
	}{
		{
			name:    "no Action method anywhere",
			method:  "Action",
			src:     codesPreamble + docSwitch,
			wantSub: "no func (c Code) Action() found",
		},
		{
			name:    "body has no switch",
			method:  "Action",
			src:     codesPreamble + "\nfunc (c Code) Action() Action { return ActionBug }\n",
			wantSub: "body has no switch statement",
		},
		{
			name:   "Action case returns a literal",
			method: "Action",
			src: codesPreamble + `
func (c Code) Action() Action {
	switch c {
	case CodeAlpha:
		return "retry"
	default:
		return ActionBug
	}
}
`,
			wantSub: `case returns "retry" — must be a bare Action identifier`,
		},
		{
			name:   "Doc case returns an identifier",
			method: "Doc",
			src: codesPreamble + `
func (c Code) Doc() string {
	switch c {
	case CodeAlpha:
		return someDoc
	default:
		return "unknown code: " + string(c)
	}
}
`,
			wantSub: "case returns someDoc — must be a string literal",
		},
		{
			name:   "case expression is not an identifier",
			method: "Action",
			src: codesPreamble + `
func (c Code) Action() Action {
	switch c {
	case "alpha.one":
		return ActionRetry
	default:
		return ActionBug
	}
}
`,
			wantSub: `case expression is "alpha.one" — must be a bare Code constant`,
		},
		{
			name:   "same code in two cases",
			method: "Action",
			src: codesPreamble + `
func (c Code) Action() Action {
	switch c {
	case CodeAlpha:
		return ActionRetry
	case CodeAlpha:
		return ActionAbort
	default:
		return ActionBug
	}
}
`,
			wantSub: "CodeAlpha appears in more than one case",
		},
		{
			name:   "case arm has two statements",
			method: "Action",
			src: codesPreamble + `
func (c Code) Action() Action {
	switch c {
	case CodeAlpha:
		x := 1
		return ActionRetry
	default:
		return ActionBug
	}
}
`,
			wantSub: "single return statement, got 2 statements",
		},
		{
			name:   "case arm is not a return",
			method: "Action",
			src: codesPreamble + `
func (c Code) Action() Action {
	switch c {
	case CodeAlpha:
		println()
	default:
		return ActionBug
	}
}
`,
			wantSub: "must be a return statement, got *ast.ExprStmt",
		},
		{
			name:   "return has two results",
			method: "Action",
			src: codesPreamble + `
func (c Code) Action() Action {
	switch c {
	case CodeAlpha:
		return ActionRetry, ActionBug
	default:
		return ActionBug
	}
}
`,
			wantSub: "exactly one result, got 2",
		},
		{
			name:    "Action default is not ActionBug",
			method:  "Action",
			src:     codesPreamble + actionDefaultWrong,
			wantSub: "default arm returns ActionRetry, want ActionBug",
		},
		{
			name:   "Doc default is not a concatenation",
			method: "Doc",
			src: codesPreamble + `
func (c Code) Doc() string {
	switch c {
	case CodeAlpha:
		return "d"
	default:
		return "unknown"
	}
}
`,
			wantSub: "default arm returns \"unknown\"",
		},
		{
			name:   "Doc default has a non-literal left operand",
			method: "Doc",
			src: codesPreamble + `
func (c Code) Doc() string {
	switch c {
	case CodeAlpha:
		return "d"
	default:
		return c + string(c)
	}
}
`,
			wantSub: "default arm returns c + string(c)",
		},
		{
			name:   "Doc default concatenates a call with no argument",
			method: "Doc",
			src: codesPreamble + `
func (c Code) Doc() string {
	switch c {
	case CodeAlpha:
		return "d"
	default:
		return "unknown code: " + string()
	}
}
`,
			wantSub: `default arm returns "unknown code: " + string()`,
		},
		{
			name:   "Doc default uses a qualified conversion",
			method: "Doc",
			src: codesPreamble + `
func (c Code) Doc() string {
	switch c {
	case CodeAlpha:
		return "d"
	default:
		return "unknown code: " + pkgString(c)
	}
}
`,
			wantSub: `default arm returns "unknown code: " + pkgString(c)`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := switchMapping(mustParse(t, tt.src), nil, tt.method)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantSub,
				"the rejection must quote the offending expression as Go source")
		})
	}
}

// actionDefaultWrong returns the wrong default for Action. docgen re-emits the
// default arm verbatim, so a changed default is a behaviour change smuggled into
// generated code.
const actionDefaultWrong = `
func (c Code) Action() Action {
	switch c {
	case CodeAlpha:
		return ActionRetry
	default:
		return ActionRetry
	}
}
`

// TestSwitchMappingFallsBackToCodesGen: once the hand-written switches are
// deleted from codes.go the data must still be found in codes_gen.go, and a
// pointer receiver must be recognised as a Code method.
func TestSwitchMappingFallsBackToCodesGen(t *testing.T) {
	primary := mustParse(t, codesPreamble)
	fallback := mustParse(t, codesPreamble+actionSwitch+docSwitch)

	got, err := switchMapping(primary, fallback, "Action")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"CodeAlpha": "ActionRetry"}, got)

	_, err = switchMapping(primary, primary, "Doc")
	require.Error(t, err, "a package with no Action/Doc switch in either file must be refused")

	star := mustParse(t, codesPreamble+`
func (c *Code) Action() Action {
	switch c {
	case CodeAlpha:
		return ActionRetry
	default:
		return ActionBug
	}
}
`)
	got, err = switchMapping(star, nil, "Action")
	require.NoError(t, err, "a pointer receiver is still a Code method")
	assert.Len(t, got, 1)
}

// TestCheckParityUndeclaredCases: a switch case that names no declared constant
// would be emitted into codes_gen.go as a case for an undefined identifier, so
// the generated file would not compile.
func TestCheckParityUndeclaredCases(t *testing.T) {
	consts := []codeConst{{Name: "CodeAlpha", Value: "alpha.one"}}
	all := []string{"CodeAlpha"}

	err := checkParity("codes.go", consts, all,
		map[string]string{"CodeAlpha": "ActionRetry", "CodeGhost": "ActionRetry"},
		map[string]string{"CodeAlpha": "d"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Action switch case CodeGhost is not a declared Code constant")

	err = checkParity("codes.go", consts, all,
		map[string]string{"CodeAlpha": "ActionRetry"},
		map[string]string{"CodeAlpha": "d", "CodeGhost": "ghost doc"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Doc switch case CodeGhost is not a declared Code constant")

	// Both problems are reported together, sorted, so one run shows the whole
	// list rather than forcing a fix-and-rerun loop.
	err = checkParity("codes.go", consts, []string{"CodeAlpha", "CodeMissing"},
		map[string]string{"CodeAlpha": "ActionRetry"}, map[string]string{"CodeAlpha": "d"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CodeMissing has no explicit case in the Action switch")
	assert.Contains(t, err.Error(), "CodeMissing has no explicit case in the Doc switch")
}

func TestActionStringMapping(t *testing.T) {
	t.Run("no String method yields nil without error", func(t *testing.T) {
		m, err := actionStringMapping(mustParse(t, codesPreamble))
		require.NoError(t, err)
		assert.Nil(t, m)
	})

	t.Run("String without a switch is ignored", func(t *testing.T) {
		src := codesPreamble + "\nfunc (a Action) String() string { return \"x\" }\n"
		m, err := actionStringMapping(mustParse(t, src))
		require.NoError(t, err, "String() is optional data, so a non-conforming one must not fail generation")
		assert.Nil(t, m)
	})

	// Non-conforming arms are skipped rather than rejected, and the remaining
	// good arms must still be extracted.
	src := codesPreamble + `
func (a Action) String() string {
	switch a {
	case ActionRetry, ActionWait:
		return "retry"
	case ActionAbort:
		return "abort"
	case ActionBug:
		return bareName
	case ActionNoReturn:
		println()
	case ActionUnused:
		println()
		return "unused"
	case ActionMore:
		return 7
	default:
		return "unknown"
	}
}
`
	m, err := actionStringMapping(mustParse(t, src))
	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"ActionRetry": "retry",
		"ActionWait":  "retry",
		"ActionAbort": "abort",
	}, m, "multi-identifier cases map every name; arms that are not a single string return are skipped")
	assert.NotContains(t, m, "ActionNoReturn", "an arm that is a single non-return statement is skipped")
}

func TestActionName(t *testing.T) {
	assert.Equal(t, "abort", actionName("ActionAbort", map[string]string{"ActionAbort": "abort"}),
		"the String() switch is the name source when present")
	assert.Equal(t, "Abort", actionName("ActionAbort", nil),
		"without String() the fallback strips the Action prefix")
	assert.Equal(t, "Abort", actionName("ActionAbort", map[string]string{"Other": "x"}))
}
