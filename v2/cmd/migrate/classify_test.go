package main

import "testing"

func hasPair(pairs []Pair, v1, v2 string) bool {
	for _, p := range pairs {
		if p.V1 == v1 && p.V2 == v2 {
			return true
		}
	}
	return false
}

func hasRename(rs []Rename, v1, v2 string) bool {
	for _, r := range rs {
		if r.V1 == v1 && r.V2 == v2 {
			return true
		}
	}
	return false
}

func hasRemoval(rs []Removal, v1, spec string) bool {
	for _, r := range rs {
		if r.V1 == v1 && r.Spec == spec {
			return true
		}
	}
	return false
}

func hasUnmapped(us []Symbol, key string) bool {
	for _, s := range us {
		if s.Key == key {
			return true
		}
	}
	return false
}

func TestClassify(t *testing.T) {
	v1 := []Symbol{
		{Pkg: "a", Name: "DoThing", Kind: "func", Key: "a.DoThing", File: "a/a.go"},
		{Pkg: "a", Name: "Transfer", Recv: "Manager", Kind: "method", Key: "a.Manager.Transfer", File: "a/a.go"},
		{Pkg: "a", Name: "OnlyHere", Kind: "func", Key: "a.OnlyHere", File: "a/a.go"},
		{Pkg: "a", Name: "OldName", Kind: "func", Key: "a.OldName", File: "a/a.go"},
	}
	v2 := []Symbol{
		{Pkg: "b", Name: "DoThing", Kind: "func", Key: "b.DoThing"},
		{Pkg: "b", Name: "Transfer", Kind: "func", Key: "b.Transfer"},
		{Pkg: "b", Name: "Renamed", Kind: "func", Key: "b.Renamed"},
	}
	r, err := classify(v1, v2, Inputs{Renames: map[string]string{"a.OldName": "b.Renamed"}})
	if err != nil {
		t.Fatalf("classify: %v", err)
	}
	if !hasPair(r.Moved, "a.DoThing", "b.DoThing") {
		t.Errorf("Moved = %v, want a.DoThing -> b.DoThing", r.Moved)
	}
	if !hasPair(r.Candidate, "a.Manager.Transfer", "b.Transfer") {
		t.Errorf("Candidate = %v, want a.Manager.Transfer -> b.Transfer (shape changed)", r.Candidate)
	}
	if !hasRename(r.Renamed, "a.OldName", "b.Renamed") {
		t.Errorf("Renamed = %v, want a.OldName -> b.Renamed", r.Renamed)
	}
	if !hasUnmapped(r.Unmapped, "a.OnlyHere") {
		t.Errorf("Unmapped = %v, want a.OnlyHere", r.Unmapped)
	}
}

func TestClassifyRemovedByPathAndPrefix(t *testing.T) {
	v1 := []Symbol{
		{Pkg: "lowlevel", Name: "ShieldedSend", Kind: "func", Key: "lowlevel.ShieldedSend", File: "pkg/client/lowlevel/shielded.go"},
		{Pkg: "trc10", Name: "AssetIssueCreate", Kind: "func", Key: "trc10.AssetIssueCreate", File: "pkg/trc10/manager.go"},
		{Pkg: "trc10", Name: "TransferAsset", Kind: "func", Key: "trc10.TransferAsset", File: "pkg/trc10/manager.go"},
	}
	r, err := classify(v1, nil, Inputs{
		RemovedPaths:    []string{"lowlevel/shielded.go"},
		RemovedPrefixes: []string{"AssetIssue"},
	})
	if err != nil {
		t.Fatalf("classify: %v", err)
	}
	if !hasRemoval(r.Removed, "lowlevel.ShieldedSend", "spec §13 (C3)") {
		t.Errorf("Removed = %v, want lowlevel.ShieldedSend removed by path", r.Removed)
	}
	if !hasRemoval(r.Removed, "trc10.AssetIssueCreate", "spec §13 (C3) TRC-10 issuance") {
		t.Errorf("Removed = %v, want trc10.AssetIssueCreate removed by prefix", r.Removed)
	}
	if !hasUnmapped(r.Unmapped, "trc10.TransferAsset") {
		t.Errorf("Unmapped = %v, want trc10.TransferAsset to survive (transfer is in scope)", r.Unmapped)
	}
}

func TestClassifyRenameTargetMissingIsError(t *testing.T) {
	_, err := classify(
		[]Symbol{{Pkg: "a", Name: "X", Kind: "func", Key: "a.X"}},
		nil,
		Inputs{Renames: map[string]string{"a.X": "b.Gone"}},
	)
	if err == nil {
		t.Fatal("classify with a dead renames target = nil, want a config error")
	}
}
