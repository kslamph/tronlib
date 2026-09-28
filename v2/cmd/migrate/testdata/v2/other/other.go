package other

// NewManager is the same shape as v1 account.NewManager in a different
// package: the Moved state.
func NewManager() {}

// Transfer is a free function where v1 had (*Manager).Transfer: the
// Candidate state (name matches, shape changed).
func Transfer() {}
