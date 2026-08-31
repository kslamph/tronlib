package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const errListFixture = `# Errors

<!-- go:errors -->
<!-- /go:errors -->
`

func TestRenderErrorList(t *testing.T) {
	out, err := renderErrorList(errListFixture, []codeDoc{
		{Code: "chain.timeout", Action: "retry", Doc: "the node did not respond in time"},
		{Code: "amount.overflow", Action: "fix_call", Doc: "the amount exceeds the int64 SUN range"},
	})
	require.NoError(t, err)
	assert.Contains(t, out, "| `chain.timeout` | `retry` | the node did not respond in time |")
	assert.Contains(t, out, "| `amount.overflow` | `fix_call` | the amount exceeds the int64 SUN range |")
	assert.True(t, strings.Contains(out, "<!-- go:errors -->") && strings.Contains(out, "<!-- /go:errors -->"))
}

func TestRenderErrorListStable(t *testing.T) {
	in := []codeDoc{
		{Code: "b.two", Action: "retry", Doc: "b"},
		{Code: "a.one", Action: "retry", Doc: "a"},
	}
	out1, _ := renderErrorList(errListFixture, in)
	out2, _ := renderErrorList(errListFixture, in)
	assert.Equal(t, out1, out2, "output must be deterministic")
	// and sorted, so a diff is meaningful
	assert.Less(t, strings.Index(out1, "a.one"), strings.Index(out1, "b.two"))
}

func TestStaleMarkerFailsCheck(t *testing.T) {
	// file contains an example that does not match the Example function
	stale := "x\n<!-- go:example ExampleFoo -->\nold\n<!-- /go:example -->\n"
	err := checkFile(stale, map[string]string{"ExampleFoo": "new"})
	assert.Error(t, err, "stale content must fail -check")
}

func TestFillExamplesFillsMarker(t *testing.T) {
	in := "doc\n<!-- go:example ExampleFoo -->\nold\n<!-- /go:example -->\n"
	out, err := fillExamples(in, map[string]string{"ExampleFoo": "new body"})
	require.NoError(t, err)
	assert.Contains(t, out, "new body")
	assert.NotContains(t, out, "old")
	assert.Contains(t, out, "<!-- go:example ExampleFoo -->")
	assert.Contains(t, out, "<!-- /go:example -->")
}

func TestFillExamplesMissingNameErrors(t *testing.T) {
	in := "doc\n<!-- go:example ExampleBar -->\nx\n<!-- /go:example -->\n"
	_, err := fillExamples(in, map[string]string{"ExampleOther": "y"})
	assert.Error(t, err, "a marker whose NAME is absent from the map must fail")
}
