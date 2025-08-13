// Package types provides shared types and utilities for the TRON SDK
package types

import (
	"errors"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
)

// Common error definitions for the TRON SDK.

var (
	// ErrInvalidAddress indicates an invalid address format or value
	ErrInvalidAddress = errors.New("invalid address")

	// ErrInvalidAmount indicates an invalid amount value
	ErrInvalidAmount = errors.New("invalid amount")

	// ErrInvalidContract indicates an invalid contract
	ErrInvalidContract = errors.New("invalid contract")

	// ErrInvalidTransaction indicates an invalid transaction
	ErrInvalidTransaction = errors.New("invalid transaction")

	// ErrInsufficientBalance indicates insufficient balance for operation
	ErrInsufficientBalance = errors.New("insufficient balance")

	// ErrInsufficientEnergy indicates insufficient energy for contract execution
	ErrInsufficientEnergy = errors.New("insufficient energy")

	// ErrInsufficientBandwidth indicates insufficient bandwidth for transaction
	ErrInsufficientBandwidth = errors.New("insufficient bandwidth")

	// ErrTransactionFailed indicates transaction execution failed
	ErrTransactionFailed = errors.New("transaction failed")

	// ErrContractExecutionFailed indicates contract execution failed
	ErrContractExecutionFailed = errors.New("contract execution failed")

	// ErrNetworkError indicates a network-related error
	ErrNetworkError = errors.New("network error")

	// ErrTimeout indicates operation timeout
	ErrTimeout = errors.New("operation timeout")

	// ErrNotFound indicates resource not found
	ErrNotFound = errors.New("not found")

	// ErrAlreadyExists indicates resource already exists
	ErrAlreadyExists = errors.New("already exists")

	// ErrPermissionDenied indicates insufficient permissions
	ErrPermissionDenied = errors.New("permission denied")

	// ErrInvalidParameter indicates invalid parameter value
	ErrInvalidParameter = errors.New("invalid parameter")
)

// TronError wraps TRON-specific errors with additional context.
type TronError struct {
	Code    int32
	Message string
	Cause   error
}

// Error implements error.
func (e *TronError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("TRON error %d: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("TRON error %d: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause.
func (e *TronError) Unwrap() error {
	return e.Cause
}

// NewTronError creates a new TronError.
func NewTronError(code int32, message string, cause error) *TronError {
	return &TronError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// TransactionError represents transaction-specific errors.
type TransactionError struct {
	TxID    string
	Message string
	Cause   error
}

// Error implements error.
func (e *TransactionError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("transaction %s failed: %s (caused by: %v)", e.TxID, e.Message, e.Cause)
	}
	return fmt.Sprintf("transaction %s failed: %s", e.TxID, e.Message)
}

// Unwrap returns the underlying cause.
func (e *TransactionError) Unwrap() error {
	return e.Cause
}

// NewTransactionError creates a new TransactionError.
func NewTransactionError(txID, message string, cause error) *TransactionError {
	return &TransactionError{
		TxID:    txID,
		Message: message,
		Cause:   cause,
	}
}

// ContractError represents smart contract execution errors.
type ContractError struct {
	ContractAddress string
	Method          string
	Message         string
	Cause           error
}

// Error implements error.
func (e *ContractError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("contract %s method %s failed: %s (caused by: %v)",
			e.ContractAddress, e.Method, e.Message, e.Cause)
	}
	return fmt.Sprintf("contract %s method %s failed: %s",
		e.ContractAddress, e.Method, e.Message)
}

// Unwrap returns the underlying cause.
func (e *ContractError) Unwrap() error {
	return e.Cause
}

// NewContractError creates a new ContractError.
func NewContractError(contractAddress, method, message string, cause error) *ContractError {
	return &ContractError{
		ContractAddress: contractAddress,
		Method:          method,
		Message:         message,
		Cause:           cause,
	}
}

// WrapTransactionResult wraps transaction result errors with context.
func WrapTransactionResult(result *api.Return, operation string) error {
	if result == nil {
		return NewTronError(0, fmt.Sprintf("nil result for %s", operation), nil)
	}

	if result.Result {
		return nil // Success
	}

	message := string(result.Message)
	if message == "" {
		message = "unknown error"
	}

	return NewTronError(int32(result.Code),
		fmt.Sprintf("%s failed: %s", operation, message), nil)
}
