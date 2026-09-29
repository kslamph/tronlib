package tron

// Parity fault fixture: CodeBravo is in AllCodes but the Action switch has
// no case for it. generateCodes must refuse to let it fall to the default
// arm, naming CodeBravo.

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
	case CodeAlpha:
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
