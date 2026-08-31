package tron

// Parity fault fixture: CodeBravo is declared but missing from AllCodes.
// generateCodes must refuse to generate, naming CodeBravo.

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
