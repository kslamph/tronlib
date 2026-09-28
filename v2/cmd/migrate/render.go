package main

import (
	"fmt"
	"strings"
)

// render produces the generated Markdown block: a summary line, then one
// section per non-empty classification state, in the spec §5.3 order.
func render(r Result) string {
	n := len(r.Moved) + len(r.Renamed) + len(r.Removed) + len(r.Candidate) + len(r.Unmapped)
	parts := []string{fmt.Sprintf(
		"**%d v1 symbols: %d moved, %d renamed, %d removed, %d candidates, %d unmapped.**",
		n, len(r.Moved), len(r.Renamed), len(r.Removed), len(r.Candidate), len(r.Unmapped),
	)}

	if len(r.Moved) > 0 {
		var s strings.Builder
		s.WriteString("## Moved\n\n| v1 | v2 |\n|---|---|")
		for _, p := range r.Moved {
			fmt.Fprintf(&s, "\n| `%s` | `%s` |", p.V1, p.V2)
		}
		parts = append(parts, s.String())
	}
	if len(r.Renamed) > 0 {
		var s strings.Builder
		s.WriteString("## Renamed\n\n| v1 | v2 | why |\n|---|---|---|")
		for _, x := range r.Renamed {
			fmt.Fprintf(&s, "\n| `%s` | `%s` | %s |", x.V1, x.V2, x.Why)
		}
		parts = append(parts, s.String())
	}
	if len(r.Removed) > 0 {
		var s strings.Builder
		s.WriteString("## Removed\n\n| v1 | spec |\n|---|---|")
		for _, x := range r.Removed {
			fmt.Fprintf(&s, "\n| `%s` | %s |", x.V1, x.Spec)
		}
		parts = append(parts, s.String())
	}
	if len(r.Candidate) > 0 {
		var s strings.Builder
		s.WriteString("## Needs review (mechanical candidate — verify)\n\n| v1 | v2 |\n|---|---|")
		for _, p := range r.Candidate {
			fmt.Fprintf(&s, "\n| `%s` | `%s` |", p.V1, p.V2)
		}
		parts = append(parts, s.String())
	}
	if len(r.Unmapped) > 0 {
		items := make([]string, 0, len(r.Unmapped))
		for _, s := range r.Unmapped {
			items = append(items, "- `"+s.Key+"`")
		}
		parts = append(parts, "## Unmapped (no mechanical v2 counterpart found)\n\n"+strings.Join(items, "\n"))
	}
	return strings.Join(parts, "\n\n")
}
