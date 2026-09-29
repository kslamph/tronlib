package tx

// validContractName reports whether name is empty or contains only visible
// (non-control) characters — the port of v1's utils.IsValidContractName,
// which BuildDeploy mirrors. Empty names are allowed (v1 semantics).
func validContractName(name string) bool {
	if name == "" {
		return true
	}
	for _, r := range name {
		if r < 32 || r == 127 {
			return false
		}
	}
	return true
}
