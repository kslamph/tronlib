package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCheckDetectsDrift(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "migration.md")
	if err := os.WriteFile(out, []byte("# Migration\n\n<!-- go:migration -->\nSTALE\n<!-- /go:migration -->\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run("testdata/v1", "testdata/v2", []string{"other"}, out, true); err == nil {
		t.Fatal("check on a stale doc = nil, want a drift error")
	}
	if err := run("testdata/v1", "testdata/v2", []string{"other"}, out, false); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if err := run("testdata/v1", "testdata/v2", []string{"other"}, out, true); err != nil {
		t.Fatalf("check after sync = %v, want nil (fixed point)", err)
	}
}

func TestRunCreatesTemplateWhenMissing(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "nested", "migration.md")
	if err := run("testdata/v1", "testdata/v2", []string{"other"}, out, false); err != nil {
		t.Fatalf("sync into a missing file: %v", err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), migStart) || !strings.Contains(string(b), migEnd) {
		t.Fatalf("created doc lacks the migration markers:\n%s", b)
	}
	if err := run("testdata/v1", "testdata/v2", []string{"other"}, out, true); err != nil {
		t.Fatalf("check after create = %v, want nil", err)
	}
}

func TestRunPreservesProseOutsideMarkers(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "migration.md")
	prose := "# Migration\n\nRead this first.\n\n<!-- go:migration -->\n\x00\n<!-- /go:migration -->\n\nAppendix.\n"
	if err := os.WriteFile(out, []byte(prose), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run("testdata/v1", "testdata/v2", []string{"other"}, out, false); err != nil {
		t.Fatalf("sync: %v", err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.HasPrefix(s, "# Migration\n\nRead this first.\n") {
		t.Errorf("prose before the block changed:\n%s", s)
	}
	if !strings.HasSuffix(s, "Appendix.\n") {
		t.Errorf("prose after the block changed:\n%s", s)
	}
}
