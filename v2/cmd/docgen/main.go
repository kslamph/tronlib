// Command docgen keeps v2 documentation mechanically in sync with code.
//
// It fills two kinds of markers in Markdown files:
//
//   - <!-- go:errors --> ... <!-- /go:errors --> with the error-code table
//   - <!-- go:example NAME --> ... <!-- /go:example --> with the body of
//     the named example
//
// checkFile is the -check side: it reports which blocks no longer match
// their generated content. AST extraction and codes_gen.go generation land
// in a later change; this file is only the renderer.
package main

import (
	"fmt"
	"sort"
	"strings"
)

// codeDoc is one row of the rendered error-code table.
type codeDoc struct {
	Code   string
	Action string
	Doc    string
}

const (
	errStart = "<!-- go:errors -->"
	errEnd   = "<!-- /go:errors -->"

	exampleStartPrefix = "<!-- go:example "
	exampleStartSuffix = " -->"
	exampleEnd         = "<!-- /go:example -->"
)

// renderErrorList fills the errStart/errEnd block with a Markdown table of
// the codes, sorted by Code so re-renders are diff-stable.
func renderErrorList(md string, codes []codeDoc) (string, error) {
	sorted := append([]codeDoc(nil), codes...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Code < sorted[j].Code })
	var b strings.Builder
	for _, c := range sorted {
		fmt.Fprintf(&b, "| `%s` | `%s` | %s |\n", c.Code, c.Action, c.Doc)
	}
	return replaceBetween(md, errStart, errEnd, b.String())
}

// replaceBetween replaces the content between the first start marker and the
// following end marker with body, keeping both markers. The body is placed
// on its own line: the result is start + "\n" + body + end + tail.
func replaceBetween(s, start, end, body string) (string, error) {
	i := strings.Index(s, start)
	if i < 0 {
		return "", fmt.Errorf("marker %q not found", start)
	}
	j := strings.Index(s[i:], end)
	if j < 0 {
		return "", fmt.Errorf("closing marker %q not found", end)
	}
	j += i + len(end)
	return s[:i+len(start)] + "\n" + body + s[j-len(end):], nil
}

// exampleStartMarker is the literal open marker for a named example.
func exampleStartMarker(name string) string {
	return exampleStartPrefix + name + exampleStartSuffix
}

// exampleNames lists, in document order, every NAME that has a
// "<!-- go:example NAME -->" marker in md.
func exampleNames(md string) ([]string, error) {
	var names []string
	for i := 0; ; {
		j := strings.Index(md[i:], exampleStartPrefix)
		if j < 0 {
			return names, nil
		}
		j = i + j + len(exampleStartPrefix)
		k := strings.Index(md[j:], exampleStartSuffix)
		if k < 0 {
			return nil, fmt.Errorf("unclosed example marker at offset %d: missing %q", j-len(exampleStartPrefix), exampleStartSuffix)
		}
		k += j
		names = append(names, md[j:k])
		i = k + len(exampleStartSuffix)
	}
}

// fillExamples replaces the content of every "<!-- go:example NAME -->"
// block in md with examples[NAME]. A marker whose NAME is absent from the
// map is an error; map entries without a marker in md are ignored.
func fillExamples(md string, examples map[string]string) (string, error) {
	names, err := exampleNames(md)
	if err != nil {
		return "", err
	}
	for _, name := range names {
		body, ok := examples[name]
		if !ok {
			return "", fmt.Errorf("example %q has a marker in the document but no entry in the examples map", name)
		}
		md, err = replaceBetween(md, exampleStartMarker(name), exampleEnd, body)
		if err != nil {
			return "", err
		}
	}
	return md, nil
}

// checkFile reports whether content still matches what docgen would render.
// replaceBetween makes a rendered block a fixed point exactly when the
// content between its markers is "\n"+body, so this per-marker comparison is
// equivalent to re-rendering the whole file, and the error names the drift.
func checkFile(content string, examples map[string]string) error {
	names, err := exampleNames(content)
	if err != nil {
		return err
	}
	var stale []string
	for _, name := range names {
		body, ok := examples[name]
		if !ok {
			return fmt.Errorf("example %q has a marker in the document but no entry in the examples map", name)
		}
		got, err := betweenContent(content, exampleStartMarker(name), exampleEnd)
		if err != nil {
			return err
		}
		if got != "\n"+body {
			stale = append(stale, name)
		}
	}
	if len(stale) > 0 {
		return fmt.Errorf("stale documentation for %s; run docgen to regenerate", strings.Join(stale, ", "))
	}
	return nil
}

// betweenContent returns the raw content between the first start marker and
// the following end marker.
func betweenContent(s, start, end string) (string, error) {
	i := strings.Index(s, start)
	if i < 0 {
		return "", fmt.Errorf("marker %q not found", start)
	}
	j := strings.Index(s[i:], end)
	if j < 0 {
		return "", fmt.Errorf("closing marker %q not found", end)
	}
	return s[i+len(start) : i+j], nil
}

func main() {}
