package tron

// Minimal stand-in for v2/tron/codes.go used by the generator tests.

type Code string

type Action int

const (
	ActionBug Action = iota + 1
	ActionRetry
)

const (
	CodeAlpha Code = "alpha.one"
	CodeBravo Code = "bravo.two"
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
