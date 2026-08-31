package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// goldString compare: the extracted body must be exactly the gofmt source
// between the braces, comment-free and trimmed.
func TestExtractExamples(t *testing.T) {
	got, err := extractExamples(filepath.Join("testdata", "examplepkg"))
	require.NoError(t, err)

	wantPrint := "fmt.Println(\"one\")\nfmt.Println(\"two\")"
	wantTwo := "x := 6\nfmt.Println(x)"
	assert.Equal(t, map[string]string{
		"ExamplePrint":     wantPrint,
		"ExamplePrint_Two": wantTwo,
	}, got)
}

func TestExtractExamplesSkipsNonExamples(t *testing.T) {
	got, err := extractExamples(filepath.Join("testdata", "examplepkg"))
	require.NoError(t, err)
	assert.NotContains(t, got, "helper")
	assert.NotContains(t, got, "ExampleWithParams", "examples with parameters are not doc examples")
}

func TestExtractExamplesCollisionErrors(t *testing.T) {
	_, err := extractExamples(filepath.Join("testdata", "duppkg"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ExampleDup", "the error must name the colliding example")
}

func TestExtractExamplesSyntaxErrorNamesFile(t *testing.T) {
	_, err := extractExamples(filepath.Join("testdata", "brokenpkg"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "broken.go")
}

func TestExtractExamplesMissingDirErrors(t *testing.T) {
	_, err := extractExamples(filepath.Join("testdata", "no_such_dir"))
	assert.Error(t, err)
}

// extractExamples must also see the real tron package examples — the same
// bodies that will be filled into docs markers in a later dispatch.
func TestExtractExamplesRealTron(t *testing.T) {
	got, err := extractExamples(realTronDir(t))
	require.NoError(t, err)
	for _, name := range []string{"ExampleParseAddress", "ExampleHasCode", "ExampleTRX", "ExampleParseTRX"} {
		assert.Contains(t, got, name)
		assert.NotContains(t, got[name], "Output:", "// Output is a test harness line, not doc content")
	}
	assert.Equal(t, "a, err := tron.ParseAddress(\"TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb\")\nif err != nil {\n\tfmt.Println(\"err:\", err)\n\treturn\n}\nfmt.Println(a.String())", got["ExampleParseAddress"])
}

func realTronDir(t *testing.T) string {
	t.Helper()
	// v2/cmd/docgen -> v2/tron
	dir := filepath.Join("..", "..", "tron")
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("tron package not found from test cwd: %v", err)
	}
	return dir
}
