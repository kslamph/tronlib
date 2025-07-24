// Package workflow provides transaction workflow management with action chaining
package workflow

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/client/lowlevel"
	"github.com/kslamph/tronlib/pkg/signer"
	"github.com/kslamph/tronlib/pkg/types"
)

// WorkflowState represents the current state of the transaction workflow
type WorkflowState int

const (
	StateUnsigned WorkflowState = iota
	StateSigned
	StateBroadcasted
	StateError
)

// String returns string representation of workflow state
func (s WorkflowState) String() string {
	switch s {
	case StateUnsigned:
		return "unsigned"
	case StateSigned:
		return "signed"
	case StateBroadcasted:
		return "broadcasted"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

// TransactionWorkflow manages the complete lifecycle of a TRON transaction
type TransactionWorkflow struct {
	mu          sync.RWMutex
	client      *client.Client
	transaction *core.Transaction
	signatures  []*core.Transaction_Contract
	state       WorkflowState
	error       error
	txid        string
	broadcastResult *BroadcastResult
}

// BroadcastResult contains the result of broadcasting a transaction
type BroadcastResult struct {
	TxID            string
	Success         bool
	TransactionInfo *core.TransactionInfo
	Error           error
}

// NewTransactionWorkflow creates a new transaction workflow
// Accepts either unsigned or signed transactions (for multi-signature scenarios)
func NewTransactionWorkflow(client *client.Client, tx interface{}) *TransactionWorkflow {
	workflow := &TransactionWorkflow{
		client: client,
		state:  StateUnsigned,
	}

	switch t := tx.(type) {
	case *core.Transaction:
		workflow.transaction = t
		// Check if transaction is already signed
		if len(t.Signature) > 0 {
			workflow.state = StateSigned
			workflow.txid = fmt.Sprintf("%x", types.GetTransactionID(t))
		}
	case *api.TransactionExtention:
		if t.Transaction != nil {
			workflow.transaction = t.Transaction
			if len(t.Transaction.Signature) > 0 {
				workflow.state = StateSigned
				workflow.txid = fmt.Sprintf("%x", types.GetTransactionID(t.Transaction))
			}
		} else {
			workflow.error = fmt.Errorf("transaction extension contains nil transaction")
			workflow.state = StateError
		}
	default:
		workflow.error = fmt.Errorf("invalid transaction type: expected *core.Transaction or *api.TransactionExtention")
		workflow.state = StateError
	}

	// Validate transaction structure
	if workflow.error == nil && workflow.transaction != nil {
		if workflow.transaction.RawData == nil {
			workflow.error = fmt.Errorf("transaction raw data is nil")
			workflow.state = StateError
		}
	}

	return workflow
}

// GetError returns any error that occurred during the workflow
func (w *TransactionWorkflow) GetError() error {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.error
}

// GetState returns the current state of the workflow
func (w *TransactionWorkflow) GetState() WorkflowState {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.state
}

// SetTimeout sets the expiration timestamp for the transaction (unsigned transactions only)
// timestamp is Unix timestamp in milliseconds when the transaction should expire
func (w *TransactionWorkflow) SetTimeout(timestamp int64) *TransactionWorkflow {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.error != nil {
		return w
	}

	if w.state != StateUnsigned {
		w.error = fmt.Errorf("cannot set timeout on %s transaction", w.state.String())
		w.state = StateError
		return w
	}

	if w.transaction.RawData != nil {
		w.transaction.RawData.Expiration = timestamp
	}

	return w
}

// SetFeeLimit sets the maximum fee limit for the transaction (unsigned transactions only)
// feeLimit is in SUN (1 TRX = 1,000,000 SUN)
func (w *TransactionWorkflow) SetFeeLimit(feeLimit int64) *TransactionWorkflow {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.error != nil {
		return w
	}

	if w.state != StateUnsigned {
		w.error = fmt.Errorf("cannot set fee limit on %s transaction", w.state.String())
		w.state = StateError
		return w
	}

	if w.transaction.RawData != nil {
		w.transaction.RawData.FeeLimit = feeLimit
	}

	return w
}

// Sign signs the transaction with the provided signer
// Can be called multiple times to accumulate signatures
func (w *TransactionWorkflow) Sign(s *signer.PrivateKeySigner) *TransactionWorkflow {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.error != nil {
		return w
	}

	if w.state == StateBroadcasted {
		w.error = fmt.Errorf("cannot sign broadcasted transaction")
		w.state = StateError
		return w
	}

	signedTx, err := s.Sign(w.transaction)
	if err != nil {
		w.error = fmt.Errorf("failed to sign transaction: %w", err)
		w.state = StateError
		return w
	}

	// Update transaction with new signature
	w.transaction = signedTx
	w.state = StateSigned
	w.txid = fmt.Sprintf("%x", types.GetTransactionID(signedTx))

	return w
}

// MultiSign signs the transaction with the provided signer and permission ID
// Can be called multiple times to accumulate multi-signatures
func (w *TransactionWorkflow) MultiSign(s *signer.PrivateKeySigner, permissionID int32) *TransactionWorkflow {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.error != nil {
		return w
	}

	if w.state == StateBroadcasted {
		w.error = fmt.Errorf("cannot multi-sign broadcasted transaction")
		w.state = StateError
		return w
	}

	// For multi-signature, we need to implement permission-based signing
	// This is a simplified implementation - in practice, multi-sig requires
	// more complex permission validation and signature aggregation
	signedTx, err := s.Sign(w.transaction)
	if err != nil {
		w.error = fmt.Errorf("failed to multi-sign transaction: %w", err)
		w.state = StateError
		return w
	}

	// Update transaction with new signature
	w.transaction = signedTx
	w.state = StateSigned
	w.txid = fmt.Sprintf("%x", types.GetTransactionID(signedTx))

	return w
}

// GetSignedTransaction returns a copy of the signed transaction (signed transactions only)
// Returns (txid, transaction_extension, error)
func (w *TransactionWorkflow) GetSignedTransaction() (string, *api.TransactionExtention, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.error != nil {
		return "", nil, w.error
	}

	if w.state == StateUnsigned {
		return "", nil, fmt.Errorf("transaction is not signed yet")
	}

	// Create a copy of the transaction
	txExt := &api.TransactionExtention{
		Transaction: w.transaction,
		Txid:        types.GetTransactionID(w.transaction),
	}

	return w.txid, txExt, nil
}

// GetTxid returns the transaction ID as hex string (signed transactions only)
// Returns empty string if transaction is not signed yet
func (w *TransactionWorkflow) GetTxid() string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.error != nil || w.state == StateUnsigned {
		return ""
	}

	return w.txid
}

// Broadcast broadcasts the signed transaction to the network (signed transactions only)
// waitSeconds: 0 means no waiting, >0 means wait up to this many seconds for receipt
// Returns (txid, broadcast_success, transaction_info, error)
// transaction_info can be nil if waiting not specified/applicable/failed
func (w *TransactionWorkflow) Broadcast(ctx context.Context, waitSeconds int64) (string, bool, *core.TransactionInfo, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.error != nil {
		return "", false, nil, w.error
	}

	if w.state == StateUnsigned {
		return "", false, nil, fmt.Errorf("cannot broadcast unsigned transaction")
	}

	// Broadcast the transaction
	result, err := lowlevel.BroadcastTransaction(w.client, ctx, w.transaction)
	if err != nil {
		w.error = fmt.Errorf("failed to broadcast transaction: %w", err)
		w.state = StateError
		return w.txid, false, nil, w.error
	}

	success := result != nil && result.Result
	w.state = StateBroadcasted

	// Store broadcast result
	w.broadcastResult = &BroadcastResult{
		TxID:    w.txid,
		Success: success,
		Error:   err,
	}

	var txInfo *core.TransactionInfo
	
	// If waiting is requested and this is a smart contract transaction, wait for receipt
	if waitSeconds > 0 && success && w.isSmartContractTransaction() {
		txInfo = w.waitForTransactionInfo(ctx, waitSeconds)
		w.broadcastResult.TransactionInfo = txInfo
	}

	return w.txid, success, txInfo, nil
}

// EstimateFee estimates the fee for the transaction (placeholder implementation)
// This is a complex calculation depending on transaction type and contracts involved
func (w *TransactionWorkflow) EstimateFee() (int64, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.error != nil {
		return 0, w.error
	}

	// Placeholder implementation
	// In practice, this would involve complex calculations based on:
	// - Transaction type
	// - Contract complexity
	// - Current network conditions
	// - Energy/bandwidth requirements
	
	return 0, fmt.Errorf("fee estimation not implemented yet")
}

// isSmartContractTransaction checks if the transaction involves smart contracts
func (w *TransactionWorkflow) isSmartContractTransaction() bool {
	if w.transaction == nil || w.transaction.RawData == nil {
		return false
	}

	for _, contract := range w.transaction.RawData.Contract {
		switch contract.Type {
		case core.Transaction_Contract_TriggerSmartContract,
			 core.Transaction_Contract_CreateSmartContract:
			return true
		}
	}
	return false
}

// waitForTransactionInfo waits for transaction confirmation and returns transaction info
func (w *TransactionWorkflow) waitForTransactionInfo(ctx context.Context, waitSeconds int64) *core.TransactionInfo {
	timeout := time.Duration(waitSeconds) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	txIDBytes := types.GetTransactionID(w.transaction)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			req := &api.BytesMessage{Value: txIDBytes}
			txInfo, err := lowlevel.GetTransactionInfoById(w.client, ctx, req)
			if err == nil && txInfo != nil {
				return txInfo
			}
		}
	}
}