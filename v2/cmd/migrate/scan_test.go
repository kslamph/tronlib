package main

import (
	"sort"
	"testing"
)

// summarize renders one symbol as "kind|recv|key" so a test can pin the
// exact set without depending on absolute file paths.
func summarize(s Symbol) string {
	return s.Kind + "|" + s.Recv + "|" + s.Key
}

func TestScanDirExportsExactSet(t *testing.T) {
	syms, err := scanDir("testdata/v1/account")
	if err != nil {
		t.Fatalf("scanDir: %v", err)
	}
	var got []string
	for _, s := range syms {
		got = append(got, summarize(s))
	}
	want := []string{
		"type||account.Manager",
		"method|Manager|account.Manager.Transfer",
		"const||account.MaxRetries",
		"func||account.NewManager",
	}
	if len(got) != len(want) {
		t.Fatalf("scanDir symbols = %v, want %v (unexported helper and _test.go must be absent)", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("symbol[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestScanTreeFindsNestedAndSkipsTests(t *testing.T) {
	syms, err := scanTree("testdata/v1")
	if err != nil {
		t.Fatalf("scanTree: %v", err)
	}
	got := map[string]bool{}
	for _, s := range syms {
		got[summarize(s)] = true
		if s.File == "" {
			t.Errorf("symbol %s has an empty File; path-based removal needs it", s.Key)
		}
	}
	for _, want := range []string{
		"func||lowlevel.ShieldedSend",
		"func||trc10.AssetIssueCreate",
		"func||account.NewManager",
	} {
		if !got[want] {
			t.Errorf("scanTree missing %q; got %v", want, keys(got))
		}
	}
	if got["func||account.ExportedTestHelper"] {
		t.Error("scanTree included a _test.go symbol; tests are not API")
	}
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// TestScanDirIsSorted proves scanDir output is deterministic, which the
// renderer's stability depends on.
func TestScanDirIsSorted(t *testing.T) {
	syms, err := scanDir("testdata/v1")
	if err != nil {
		t.Fatalf("scanDir: %v", err)
	}
	if !sort.SliceIsSorted(syms, func(i, j int) bool { return syms[i].Key < syms[j].Key }) {
		t.Errorf("scanDir output is not sorted by Key: %v", syms)
	}
}
