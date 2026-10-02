// Command docgen keeps v2 documentation mechanically in sync with code.
//
// It fills two kinds of markers in Markdown files:
//
//   - <!-- go:errors --> ... <!-- /go:errors --> with the error-code table
//   - <!-- go:example NAME --> ... <!-- /go:example --> with the body of
//     the named example
//
// checkFile is the -check side: it reports which blocks no longer match
// their generated content. AST extraction lives in examples.go and
// codes_gen.go generation lives in codesgen.go/emit.go.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
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
// the codes, sorted by Code so re-renders are diff-stable. A file without
// an errStart marker is returned unchanged: docs files that carry only
// example blocks (e.g. an example index) are valid sync targets too.
func renderErrorList(md string, codes []codeDoc) (string, error) {
	if !strings.Contains(md, errStart) {
		return md, nil
	}
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

// multiFlag collects a repeated -flag into a slice (flag.Set overwrites a
// plain string value, but sync-docs takes multiple -docs files).
type multiFlag []string

func (m *multiFlag) String() string     { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error { *m = append(*m, v); return nil }

// main is an exit-code shim over run. Everything the CLI does lives in run so
// that flag parsing, the exit codes and the error text are testable without
// forking a subprocess (CODING_STANDARDS.md 6.5: a command's pure helpers carry
// tests; the process boundary itself is the one line that cannot).
func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run implements the docgen CLI and returns the process exit code rather than
// exiting: 0 on success, 1 when the requested operation failed, 2 on a usage
// error. stdout and stderr are injected so tests can assert on what the user
// sees. args is os.Args[1:] (the command word first), matching how main calls
// it.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	switch args[0] {
	case "generate-codes":
		fs := flag.NewFlagSet("generate-codes", flag.ContinueOnError)
		fs.SetOutput(stderr)
		pkgDir := fs.String("pkg", "", "package directory containing codes.go")
		out := fs.String("out", "", "output file for the generated source")
		if code, decided := parseFlags(fs, args[1:], stderr); decided {
			return code
		}
		if *pkgDir == "" || *out == "" {
			fmt.Fprintln(stderr, "generate-codes requires -pkg and -out")
			return 2
		}
		src, err := generateCodes(*pkgDir)
		if err != nil {
			fmt.Fprintf(stderr, "docgen: %v\n", err)
			return 1
		}
		if err := os.WriteFile(*out, []byte(src), 0o644); err != nil {
			fmt.Fprintf(stderr, "docgen: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "docgen: wrote %s (%d bytes)\n", *out, len(src))
		return 0
	case "sync-docs":
		fs := flag.NewFlagSet("sync-docs", flag.ContinueOnError)
		fs.SetOutput(stderr)
		pkgDir := fs.String("pkg", "", "package directory to read codes and Example functions from")
		var examplePkgs multiFlag
		fs.Var(&examplePkgs, "example-pkg", "additional package directory to extract Example functions from; repeatable")
		var docs multiFlag
		fs.Var(&docs, "docs", "docs file to sync; repeatable")
		check := fs.Bool("check", false, "do not write; byte-compare rendered output against each docs file and fail on drift")
		if code, decided := parseFlags(fs, args[1:], stderr); decided {
			return code
		}
		if *pkgDir == "" || len(docs) == 0 {
			fmt.Fprintln(stderr, "sync-docs requires -pkg and at least one -docs")
			return 2
		}
		if err := runSync(*pkgDir, examplePkgs, docs, *check, stdout); err != nil {
			fmt.Fprintf(stderr, "docgen: %v\n", err)
			return 1
		}
		return 0
	default:
		fmt.Fprintf(stderr, "docgen: unknown command %q\n%s", args[0], usage)
		return 2
	}
}

// parseFlags parses args for a command's FlagSet and decides whether the CLI is
// done. Under flag.ContinueOnError Parse returns its errors instead of exiting
// in-process, so this is where the old flag.ExitOnError behaviour is
// reconstructed: -h/-help printed the usage and exited 0, a bad flag printed an
// error and exited 2.
//
// The second result reports whether a decision was made; when it is false the
// flags parsed cleanly and the command should proceed with the zero code.
func parseFlags(fs *flag.FlagSet, args []string, stderr io.Writer) (int, bool) {
	err := fs.Parse(args)
	if err == nil {
		return 0, false
	}
	if errors.Is(err, flag.ErrHelp) {
		// flag already wrote the per-command usage to stderr.
		return 0, true
	}
	fmt.Fprint(stderr, usage)
	return 2, true
}

const usage = `usage: docgen <command> [flags]

commands:
  generate-codes -pkg <dir> -out <file>
      regenerate the tron package's codes_gen.go from codes.go

  sync-docs -pkg <dir> [-example-pkg <dir> ...] -docs <file> [-docs <file> ...] [-check]
      fill the go:errors table and go:example blocks in each docs file from
      the package source. -pkg supplies the error-code table and its own
      Examples; each -example-pkg adds another package whose Examples are
      namespaced <package>.<ExampleFunc> (e.g. tron.ExampleTRX,
      tronlib.ExampleClient_token). With -check, write nothing: re-render
      every docs file and byte-compare; any difference (including an Example
      function with no marker, or a marker naming no real Example) exits 1.
      This is the CI drift gate.
`
