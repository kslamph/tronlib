// Package workflow provides transaction workflow management for TRON blockchain operations
package workflow

import (
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
)

// NewWorkflow creates a new transaction workflow with the specified client and transaction
// This is the main entry point for creating transaction workflows
func NewWorkflow(client *client.Client, tx interface{}) *TransactionWorkflow {
	return NewTransactionWorkflow(client, tx)
}

// CreateWorkflowFromTransaction creates a workflow from a core.Transaction
func CreateWorkflowFromTransaction(client *client.Client, tx *core.Transaction) *TransactionWorkflow {
	return NewTransactionWorkflow(client, tx)
}