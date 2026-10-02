package tx

// validContractName reports whether name is empty or contains only visible
// (non-control) characters — the rule BuildDeploy enforces. Empty names are
// allowed.
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
