package client

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
	"google.golang.org/protobuf/proto"
)

// Provide high level sign and broadcast workflows
type BroadcastOptions struct {
	FeeLimit       int64         // Fee limit for the transaction
	PermissionID   int32         // Permission ID for the transaction
	WaitForReceipt bool          // Wait for transaction receipt
	WaitTimeout    time.Duration // Timeout for waiting for receipt
	PollInterval   time.Duration // Polling interval when waiting for receipt
}

// DefaultBroadcastOptions returns sane defaults for broadcasting transactions.
func DefaultBroadcastOptions() BroadcastOptions {
	return BroadcastOptions{
		FeeLimit:       150_000_000,
		PermissionID:   0,
		WaitForReceipt: true,
		WaitTimeout:    15,              // seconds
		PollInterval:   3 * time.Second, // polling cadence
	}
}

type BroadcastResult struct {
	TxID    string                 `json:"txID"`
	Success bool                   `json:"success"`       //indicate if the transaction was successfully broadcasted
	Code    api.ReturnResponseCode `json:"returnCode"`    // TRON return code
	Message string                 `json:"returnMessage"` // TRON return message

	//ContractReceipt is the receipt of the contract execution, reflect the resources usage
	ContractReceipt *core.ResourceReceipt //test if nil before use
	// ContractResult has the details of the contract returned error message or result
	ContractResult [][]byte //test if nil before use

	// Derived fields from TransactionInfo for easier consumption
	// ExecutionSuccess indicates whether the smart contract execution itself succeeded (state not reverted)
	ExecutionSuccess bool `json:"executionSuccess"`
	// TxInfoResult mirrors TransactionInfo.result (SUCESS/FAILED)
	TxInfoResult core.TransactionInfoCode `json:"txInfoResult"`
	// TxInfoResMessage is the decoded human-readable message (e.g., revert reason) if available
	TxInfoResMessage string `json:"txInfoResMessage"`
	// ContractRet mirrors receipt.result (SUCCESS/REVERT/OUT_OF_ENERGY/...)
	ContractRet core.Transaction_ResultContractResult `json:"contractRet"`
}

func (c *Client) Simulate(ctx context.Context, anytx any) (*api.TransactionExtention, error) {
	if anytx == nil {
		return nil, fmt.Errorf("transaction cannot be nil")
	}
	var coretx *core.Transaction

	switch tx := anytx.(type) {
	case *api.TransactionExtention:
		coretx = tx.GetTransaction()

	case *core.Transaction:
		coretx = tx
	default:
		return nil, fmt.Errorf("unsupported transaction type: %T", anytx)
	}
	// Validate transaction
	if coretx == nil {
		return nil, fmt.Errorf("transaction cannot be nil")
	}
	if coretx.GetRawData() == nil {
		return nil, fmt.Errorf("transaction raw data cannot be nil")
	}
	if len(coretx.GetRawData().GetContract()) != 1 {
		return nil, fmt.Errorf("transaction must have exactly one contract, got %d", len(coretx.GetRawData().GetContract()))
	}
	if coretx.GetRawData().GetExpiration() < time.Now().UnixMilli() {
		return nil, fmt.Errorf("transaction expiration must be in the future")
	}

	decodedTx := &core.TriggerSmartContract{}

	if err := proto.Unmarshal(coretx.GetRawData().GetContract()[0].GetParameter().GetValue(), decodedTx); err != nil {

		return nil, fmt.Errorf("failed to decode transaction: %v", err)
	}

	// fmt.Println("Decoded Transaction:", decodedTx)

	return c.TriggerConstantContract(ctx, decodedTx)
}

func (c *Client) SignAndBroadcast(ctx context.Context, anytx any, opt BroadcastOptions, signers ...types.Signer) (*BroadcastResult, error) {
	// Apply defaults for zero-values without breaking explicit non-zero caller values.
	def := DefaultBroadcastOptions()
	if opt.FeeLimit == 0 {
		opt.FeeLimit = def.FeeLimit
	}
	if opt.PermissionID == 0 {
		// default 0 already, nothing to change; keep for clarity
	}
	if !opt.WaitForReceipt && def.WaitForReceipt {
		// honor explicit false, only set default when zero value ambiguity matters; WaitForReceipt is boolean so
		// leave as is to respect caller. If caller provides zero value (false) intentionally, we keep it.
	}
	if opt.WaitTimeout == 0 {
		opt.WaitTimeout = def.WaitTimeout
	}
	if opt.PollInterval == 0 {
		opt.PollInterval = def.PollInterval
	}
	// Trigger the smart contract with the given parameters
	if anytx == nil {
		return nil, fmt.Errorf("transaction cannot be nil")
	}
	var coretx *core.Transaction

	switch tx := anytx.(type) {
	case *api.TransactionExtention:
		coretx = tx.GetTransaction()

	case *core.Transaction:
		coretx = tx
	default:
		return nil, fmt.Errorf("unsupported transaction type: %T", anytx)
	}
	// Validate transaction
	if coretx == nil {
		return nil, fmt.Errorf("transaction cannot be nil")
	}
	if coretx.GetRawData() == nil {
		return nil, fmt.Errorf("transaction raw data cannot be nil")
	}
	if len(coretx.GetRawData().GetContract()) != 1 {
		return nil, fmt.Errorf("transaction must have exactly one contract, got %d", len(coretx.GetRawData().GetContract()))
	}
	if coretx.GetRawData().GetExpiration() < time.Now().UnixNano() {
		return nil, fmt.Errorf("transaction expiration must be in the future")
	}

	if len(signers) > 0 {
		if opt.PermissionID != 0 {
			coretx.RawData.GetContract()[0].PermissionId = opt.PermissionID
		}
		coretx.RawData.FeeLimit = opt.FeeLimit
		for _, signer := range signers {
			if err := signer.Sign(coretx); err != nil {
				return nil, fmt.Errorf("failed to sign transaction: %w", err)
			}
		}
	}

	txid := types.GetTransactionID(coretx)
	// fmt.Printf("txid:%x\n", txid)
	result := &BroadcastResult{
		TxID: hex.EncodeToString(txid),
	}

	ret, err := c.BroadcastTransaction(ctx, coretx)
	if err != nil {
		return result, fmt.Errorf("failed to broadcast transaction: %w", err)
	}
	result.Success = ret.GetResult()
	result.Code = ret.GetCode()
	result.Message = string(ret.GetMessage())

	if !opt.WaitForReceipt {
		return result, nil

	}

	txInfo := c.waitForTransactionInfo(ctx, txid, opt.WaitTimeout, opt.PollInterval)
	if txInfo == nil {
		return result, nil
	}
	result.ContractResult = txInfo.GetContractResult()
	result.ContractReceipt = txInfo.GetReceipt()

	// Populate derived fields for execution status
	result.TxInfoResult = txInfo.GetResult()
	if resMsg := txInfo.GetResMessage(); len(resMsg) > 0 {
		result.TxInfoResMessage = string(resMsg)
	}
	if receipt := txInfo.GetReceipt(); receipt != nil {
		result.ContractRet = receipt.GetResult()
		// Consider execution successful only if receipt result is SUCCESS and TransactionInfo result is SUCESS
		if result.TxInfoResult == core.TransactionInfo_SUCESS && result.ContractRet == core.Transaction_Result_SUCCESS {
			result.ExecutionSuccess = true
		}
	}

	return result, nil
}

func (c *Client) waitForTransactionInfo(ctx context.Context, txid []byte, waitTimeout time.Duration, pollInterval time.Duration) *core.TransactionInfo {
	timeout := waitTimeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if pollInterval <= 0 {
		pollInterval = 3 * time.Second
	}
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			req := &api.BytesMessage{Value: txid}
			txInfo, err := c.GetTransactionInfoById(ctx, req)
			if err == nil && txInfo != nil && bytes.Equal(txInfo.Id, txid) {
				return txInfo
			}
		}
	}
}
