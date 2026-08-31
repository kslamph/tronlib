package tron

import (
	"errors"
	"fmt"
)

// Error is the only error type v2 returns from exported functions.
//
// Code is the machine-parseable identity; Hint is the human/agent remediation.
// The two are deliberately separate because v1 fused them into one string
// ("invalid address: check format and ensure it's a valid TRON address"),
// which meant a machine could not get the fact without the advice and a human
// could not change the advice without changing the message. Error() never
// prints Hint.
type Error struct {
	Code  Code
	Op    string // "Dial", "Broadcast", "contract.Invoke"
	Hint  string // remediation; may be empty; never duplicated into Error()
	TxID  string // populated when a transaction id is known
	Cause error
	Next  Action // 0 (unset) means "use Code.Action()"; see Action docs
}

func (e *Error) Error() string {
	if e.Op != "" {
		return fmt.Sprintf("%s: %s", e.Op, e.Code)
	}
	return string(e.Code)
}

func (e *Error) Unwrap() error { return e.Cause }

// Is reports whether target is a *Error carrying the same Code.
// Defined as a method so errors.Is(err, &Error{Code: X}) works alongside HasCode.
// HasCode — not errors.Is against a bare Code — is the sanctioned verb: a bare
// Code is a string and errors.Is would silently return false for it.
func (e *Error) Is(target error) bool {
	var t *Error
	if errors.As(target, &t) {
		return t.Code == e.Code
	}
	return false
}

// HasCode reports whether err, or any error in its chain, is a *Error with code c.
func HasCode(err error, c Code) bool {
	var e *Error
	if errors.As(err, &e) {
		return e.Code == c
	}
	return false
}

// Action returns the remediation the caller should take.
// An explicit Next overrides the code-derived default, because some remedies
// depend on context the code cannot see: a chain.timeout before submission is
// retryable, after submission it is not. Next == 0 (the unset sentinel) means
// "derive from Code".
func (e *Error) Action() Action {
	if e.Next != 0 {
		return e.Next
	}
	return e.Code.Action()
}
