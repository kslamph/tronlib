package tron

// Parity fault fixture: CodeGhost appears in AllCodes but is never declared
// as a Code constant. generateCodes must refuse to generate, naming it.

type Code string

type Action int

const (
	ActionBug Action = iota + 1
	ActionRetry
)

const (
	CodeAlpha Code = "alpha.one"
)

var AllCodes = []Code{
	CodeAlpha,
	CodeGhost,
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
	case CodeAlpha:
		return "alpha doc"
	default:
		return "unknown code: " + string(c)
	}
}
