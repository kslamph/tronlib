package client

import (
	"fmt"
	"strings"
)

// ErrorCode represents different types of errors that can occur
type ErrorCode int

const (
	ErrNetwork ErrorCode = iota
	ErrValidation
	ErrTransaction
	ErrContract
	ErrAuthentication
	ErrResourceExhausted
)

// TronError represents an error from the Tron network
type TronError struct {
	Code    ErrorCode
	Message string
	Cause   error
}

func (e *TronError) Error() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("[%d] %s", e.Code, e.Message))
	if e.Cause != nil {
		sb.WriteString(fmt.Sprintf(": %v", e.Cause))
	}
	return sb.String()
}

// NewTronError creates a new TronError
func NewTronError(code ErrorCode, message string, cause error) *TronError {
	return &TronError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}
