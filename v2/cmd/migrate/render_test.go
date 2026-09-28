package main

import "testing"

func TestRenderGolden(t *testing.T) {
	r := Result{
		Moved:     []Pair{{V1: "a.DoThing", V2: "b.DoThing"}},
		Renamed:   []Rename{{V1: "a.OldName", V2: "b.Renamed", Why: "spec §14 step 12 relocation"}},
		Removed:   []Removal{{V1: "lowlevel.ShieldedSend", Spec: "spec §13 (C3)"}},
		Candidate: []Pair{{V1: "a.Manager.Transfer", V2: "b.Transfer"}},
		Unmapped:  []Symbol{{Key: "a.OnlyHere"}},
	}
	want := "**5 v1 symbols: 1 moved, 1 renamed, 1 removed, 1 candidates, 1 unmapped.**\n" +
		"\n## Moved\n\n| v1 | v2 |\n|---|---|\n| `a.DoThing` | `b.DoThing` |\n" +
		"\n## Renamed\n\n| v1 | v2 | why |\n|---|---|---|\n| `a.OldName` | `b.Renamed` | spec §14 step 12 relocation |\n" +
		"\n## Removed\n\n| v1 | spec |\n|---|---|\n| `lowlevel.ShieldedSend` | spec §13 (C3) |\n" +
		"\n## Needs review (mechanical candidate — verify)\n\n| v1 | v2 |\n|---|---|\n| `a.Manager.Transfer` | `b.Transfer` |\n" +
		"\n## Unmapped (no mechanical v2 counterpart found)\n\n- `a.OnlyHere`"
	if got := render(r); got != want {
		t.Errorf("render mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestRenderOmitsEmptySections(t *testing.T) {
	got := render(Result{Moved: []Pair{{V1: "a.X", V2: "b.X"}}})
	if want := "**1 v1 symbols: 1 moved, 0 renamed, 0 removed, 0 candidates, 0 unmapped.**\n\n## Moved\n\n| v1 | v2 |\n|---|---|\n| `a.X` | `b.X` |"; got != want {
		t.Errorf("render = %q, want %q", got, want)
	}
}
