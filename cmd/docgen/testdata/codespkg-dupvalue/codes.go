package tron

// Parity fault fixture: CodeAlpha and CodeBravo are both declared, both in
// AllCodes exactly once, and both have Action/Doc cases — but they share the
// SAME string value. Two names for one code is invisible to the bijection
// checks and makes the rendered table ambiguous, so generateCodes must
// refuse it and name the collision.
//
// Note: only go/parser reads this file (never the compiler), so the shared
// value is legal here — the `case CodeAlpha, CodeBravo` arm is a syntactic
// duplicate the compiler would reject, which is exactly the point.

type Code string

type Action int

const (
	ActionBug Action = iota + 1
	ActionRetry
)

const (
	CodeAlpha Code = "same.value"
	CodeBravo Code = "same.value"
)

var AllCodes = []Code{
	CodeAlpha,
	CodeBravo,
}

func (c Code) Action() Action {
	switch c {
	case CodeAlpha, CodeBravo:
		return ActionRetry
	default:
		return ActionBug
	}
}

func (c Code) Doc() string {
	switch c {
	case CodeAlpha, CodeBravo:
		return "alpha and bravo doc"
	default:
		return "unknown code: " + string(c)
	}
}
