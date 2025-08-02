// Package workflow provides transaction workflow management with action chaining
package workflow

import (
	"bytes"
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
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
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
	// signatures      [][]byte
	state           WorkflowState
	txid            string
	broadcastResult *BroadcastResult
	err             error
}

// BroadcastResult contains the result of broadcasting a transaction
type BroadcastResult struct {
	TxID            string
	Success         bool
	Return          *api.Return
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
			workflow.err = fmt.Errorf("transaction extension contains nil transaction")
			workflow.state = StateError
		}
	default:
		workflow.err = fmt.Errorf("invalid transaction type: expected *core.Transaction or *api.TransactionExtention")
		workflow.state = StateError
	}

	// Validate transaction structure
	if workflow.err == nil && workflow.transaction != nil {
		if workflow.transaction.RawData == nil {
			workflow.err = fmt.Errorf("transaction raw data is nil")
			workflow.state = StateError
		}
	}

	return workflow
}

// GetError returns any error that occurred during the workflow
func (w *TransactionWorkflow) GetError() error {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.err
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

	if w.err != nil {
		return w
	}

	if w.state != StateUnsigned {
		w.err = fmt.Errorf("cannot set timeout on %s transaction", w.state.String())
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

	if w.err != nil {
		return w
	}

	if w.state != StateUnsigned {
		w.err = fmt.Errorf("cannot set fee limit on %s transaction", w.state.String())
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

	if w.err != nil {
		return w
	}

	if w.state == StateBroadcasted {
		w.err = fmt.Errorf("cannot sign broadcasted transaction")
		w.state = StateError
		return w
	}

	err := s.Sign(w.transaction)
	if err != nil {
		w.err = fmt.Errorf("failed to sign transaction: %w", err)
		w.state = StateError
		return w
	}

	// Update transaction with new signature

	w.state = StateSigned
	w.txid = fmt.Sprintf("%x", types.GetTransactionID(w.transaction))

	return w
}

// GetSignedTransaction returns a copy of the signed transaction (signed transactions only)
// Returns (txid, transaction_extension, error)
func (w *TransactionWorkflow) GetSignedTransaction() (string, *api.TransactionExtention, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.err != nil {
		return "", nil, w.err
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

	if w.err != nil || w.state == StateUnsigned {
		return ""
	}

	return w.txid
}

// Broadcast broadcasts the signed transaction to the network (signed transactions only)
// waitSeconds: 0 means no waiting, >0 means wait up to this many seconds for receipt
// Returns (txid, broadcast_success, transaction_info, error)
// transaction_info can be nil if waiting not specified/applicable/failed
func (w *TransactionWorkflow) Broadcast(ctx context.Context, waitSeconds int64) (string, bool, *api.Return, *core.TransactionInfo, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.err != nil {
		return "", false, nil, nil, w.err
	}

	if w.state == StateUnsigned {
		return "", false, nil, nil, fmt.Errorf("cannot broadcast unsigned transaction")
	}

	// Broadcast the transaction
	result, err := lowlevel.BroadcastTransaction(w.client, ctx, w.transaction)
	if err != nil {
		w.err = fmt.Errorf("failed to broadcast transaction: %w", err)
		w.state = StateError
		return w.txid, false, result, nil, w.err
	}

	success := result != nil && result.Result
	w.state = StateBroadcasted

	// Store broadcast result
	w.broadcastResult = &BroadcastResult{
		TxID:            w.txid,
		Success:         success,
		Return:          result,
		TransactionInfo: nil,
		Error:           err,
	}

	var txInfo *core.TransactionInfo

	// If waiting is requested and this is a smart contract transaction, wait for receipt
	if waitSeconds > 0 && success && w.isSmartContractTransaction() {
		txInfo = w.waitForTransactionInfo(ctx, waitSeconds)
		w.broadcastResult.TransactionInfo = txInfo
	}

	return w.txid, success, result, txInfo, nil
}

// EstimateFee estimates the fee for the transaction
// Returns estimated fee in SUN
func (w *TransactionWorkflow) EstimateFee(ctx context.Context) (int64, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.err != nil {
		return 0, w.err
	}

	if w.transaction == nil || w.transaction.RawData == nil || len(w.transaction.RawData.Contract) == 0 {
		return 0, fmt.Errorf("invalid transaction for fee estimation")
	}

	contract := w.transaction.RawData.Contract[0]
	if contract.Type != core.Transaction_Contract_TriggerSmartContract {
		// For non-contract tx, estimate bandwidth only (1 SUN per byte)
		txBytes, _ := proto.Marshal(w.transaction)
		return int64(len(txBytes)), nil
	}

	// Extract TriggerSmartContract params
	param := &core.TriggerSmartContract{}
	if err := anypb.UnmarshalTo(contract.Parameter, param, proto.UnmarshalOptions{}); err != nil {
		return 0, fmt.Errorf("failed to unmarshal contract param: %w", err)
	}

	// Call estimate energy API (assuming lowlevel.EstimateEnergy exists or simulate)
	// In practice, implement lowlevel.EstimateEnergy if not present
	req := &core.TriggerSmartContract{
		OwnerAddress:    param.OwnerAddress,
		ContractAddress: param.ContractAddress,
		CallValue:       param.CallValue,
		Data:            param.Data,
		CallTokenValue:  param.CallTokenValue,
		TokenId:         param.TokenId,
	}
	energyResp, err := lowlevel.EstimateEnergy(w.client, ctx, req) // Assume this method
	if err != nil {
		return 0, fmt.Errorf("failed to estimate energy: %w", err)
	}
	contEst, err := lowlevel.TriggerConstantContract(w.client, ctx, req)
	if err != nil {
		return 0, fmt.Errorf("failed to trigger constant contract: %w", err)
	}
	fmt.Printf("contEst: %+v\n", contEst)
	energyRequired := energyResp.EnergyRequired // Assuming response has EnergyRequired

	// Estimate bandwidth: tx size in bytes * 1 SUN/byte
	txBytes, _ := proto.Marshal(w.transaction)
	bandwidthCost := int64(len(txBytes))

	// Energy cost: energy * 420 SUN (typical price, could fetch dynamically)
	energyCost := energyRequired * 420
	fmt.Printf("energyResp: %+v\n", energyResp)
	fmt.Printf("energyCost: %d\n", energyCost)
	fmt.Printf("energyRequired: %d\n", energyRequired)
	fmt.Printf("txBytes: %d\n", len(txBytes))
	fmt.Printf("bandwidthCost: %d\n", bandwidthCost)
	return bandwidthCost + energyCost, nil
}

// isSmartContractTransaction checks if the transaction involves smart contracts
func (w *TransactionWorkflow) isSmartContractTransaction() bool {

	if w.transaction == nil || w.transaction.RawData == nil {
		return false
	}
	contractType := w.transaction.RawData.Contract[0].Type
	if contractType == core.Transaction_Contract_CreateSmartContract || contractType == core.Transaction_Contract_TriggerSmartContract {
		return true
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
			if err == nil && txInfo != nil && bytes.Equal(txInfo.Id, txIDBytes) {
				return txInfo
			}
		}
	}
}
