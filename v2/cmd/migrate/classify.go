package main

import (
	"fmt"
	"sort"
	"strings"
)

// Pair is a v1 symbol and its mechanical v2 counterpart.
type Pair struct{ V1, V2 string }

// Rename is a curated v1 -> v2 mapping, with the reason it is curated.
type Rename struct{ V1, V2, Why string }

// Removal records a v1 symbol excluded from v2 and the spec clause doing it.
type Removal struct{ V1, Spec string }

// Result is the classified migration surface.
type Result struct {
	Moved     []Pair
	Renamed   []Rename
	Removed   []Removal
	Candidate []Pair
	Unmapped  []Symbol
}

// Inputs is the curated input set from tables.go.
type Inputs struct {
	RemovedPaths    []string          // path substrings -> Removed (file-scoped C3/C4 exclusions)
	RemovedPrefixes []string          // name prefixes -> Removed (e.g. "AssetIssue")
	Renames         map[string]string // v1 key -> v2 key; every target must exist in v2
}

// classify sorts every v1 symbol into the spec §5.3 states, in that order:
// removed by path, removed by name prefix, explicitly renamed, moved (same
// shape, different package), candidate (name matches, shape changed), and
// unmapped. A Renames target absent from the v2 set is a config error: a
// dead mapping must never reach the guide.
func classify(v1, v2 []Symbol, in Inputs) (Result, error) {
	byKey := make(map[string]Symbol, len(v2))
	byName := make(map[string][]Symbol, len(v2))
	byShape := make(map[string]Symbol, len(v2))
	for _, s := range v2 {
		byKey[s.Key] = s
		byName[s.Name] = append(byName[s.Name], s)
		if _, ok := byShape[shapeKey(s)]; !ok {
			byShape[shapeKey(s)] = s
		}
	}
	for v1Key, v2Key := range in.Renames {
		if _, ok := byKey[v2Key]; !ok {
			return Result{}, fmt.Errorf("renames config: %q -> %q but %q is not a v2 symbol", v1Key, v2Key, v2Key)
		}
	}

	var r Result
	for _, s := range v1 {
		switch {
		case matchesPath(s, in.RemovedPaths):
			r.Removed = append(r.Removed, Removal{V1: s.Key, Spec: "spec §13 (C3)"})
		case matchesPrefix(s, in.RemovedPrefixes):
			r.Removed = append(r.Removed, Removal{V1: s.Key, Spec: "spec §13 (C3) TRC-10 issuance"})
		default:
			if target, ok := in.Renames[s.Key]; ok {
				r.Renamed = append(r.Renamed, Rename{V1: s.Key, V2: target, Why: "spec §14 step 12 relocation"})
				continue
			}
			if t, ok := byShape[shapeKey(s)]; ok {
				r.Moved = append(r.Moved, Pair{V1: s.Key, V2: t.Key})
				continue
			}
			if cands := byName[s.Name]; len(cands) > 0 {
				sort.Slice(cands, func(i, j int) bool { return cands[i].Key < cands[j].Key })
				r.Candidate = append(r.Candidate, Pair{V1: s.Key, V2: cands[0].Key})
				continue
			}
			r.Unmapped = append(r.Unmapped, s)
		}
	}
	return r, nil
}

// shapeKey matches a symbol to its mechanical counterpart: the same kind and
// name, and for methods the same receiver type name. Package is deliberately
// excluded — relocation across packages is the Moved state.
func shapeKey(s Symbol) string {
	if s.Kind == "method" {
		return "method|" + s.Recv + "|" + s.Name
	}
	return s.Kind + "||" + s.Name
}

func matchesPath(s Symbol, paths []string) bool {
	for _, p := range paths {
		if strings.Contains(s.File, p) {
			return true
		}
	}
	return false
}

func matchesPrefix(s Symbol, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s.Name, p) {
			return true
		}
	}
	return false
}
