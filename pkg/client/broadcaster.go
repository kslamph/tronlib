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
	Success bool                   `json:"success"`
	Code    api.ReturnResponseCode `json:"returnCode"`    // TRON return code
	Message string                 `json:"returnMessage"` // TRON return message concat with contract return message

	// ConstantReturn has the details of the contract returned error message or result
	ConstantReturn [][]byte //test if nil before use

	// Fields primarily populated by simulation (TriggerConstantContract)
	EnergyUsage int64                       `json:"energyUsed,omitempty"`
	NetUsage    int64                       `json:"netUsage,omitempty"`
	Logs        []*core.TransactionInfo_Log `json:"logs,omitempty"`
	// DebugExt   *api.TransactionExtention   `json:"debugExt,omitempty"`
}

func (c *Client) Simulate(ctx context.Context, anytx any) (*BroadcastResult, error) {
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

	// Perform constant call (simulation)
	ext, err := c.TriggerConstantContract(ctx, decodedTx)
	if err != nil {
		return nil, err
	}

	br := &BroadcastResult{}
	if ext != nil {
		if txid := ext.GetTxid(); len(txid) > 0 {
			br.TxID = hex.EncodeToString(txid)
		}
		if ret := ext.GetResult(); ret != nil {
			br.Success = ret.GetResult() && ext.GetTransaction().GetRet()[0].GetRet() == core.Transaction_Result_SUCESS
			br.Code = ret.GetCode()
			br.Message = string(ret.GetMessage()) + string(ext.GetResult().GetMessage())
		}
		br.ConstantReturn = ext.GetConstantResult()
		br.EnergyUsage = ext.GetEnergyUsed()
		br.Logs = ext.GetLogs()
	}

	return br, nil
}

func (c *Client) SignAndBroadcast(ctx context.Context, anytx any, opt BroadcastOptions, signers ...types.Signer) (*BroadcastResult, error) {
	// Apply defaults for zero-values without breaking explicit non-zero caller values.
	def := DefaultBroadcastOptions()
	if opt.FeeLimit == 0 {
		opt.FeeLimit = def.FeeLimit
	}
	// PermissionID: default is 0; honor explicit 0 provided by caller.
	// WaitForReceipt: honor explicit false (no defaulting needed here).
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

	result.Success = result.Success && txInfo.GetResult() == core.TransactionInfo_SUCESS

	result.Message = result.Message + string(txInfo.GetResMessage())

	result.ConstantReturn = txInfo.GetContractResult()
	result.EnergyUsage = txInfo.GetReceipt().GetEnergyUsageTotal()
	result.NetUsage = txInfo.GetReceipt().GetNetUsage()
	result.Logs = txInfo.GetLog()

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
