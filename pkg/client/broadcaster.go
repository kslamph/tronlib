package client

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client/lowlevel"
	"github.com/kslamph/tronlib/pkg/signer"
	"github.com/kslamph/tronlib/pkg/utils"
	"google.golang.org/protobuf/proto"
)

// BroadcastOptions controls high-level signing and broadcasting workflows.
// Fields with zero values are defaulted by DefaultBroadcastOptions unless
// explicitly documented otherwise.
//
// These options control how transactions are signed, broadcast, and confirmed.
// Use DefaultBroadcastOptions() to get sensible defaults, then modify as needed.
type BroadcastOptions struct {
	FeeLimit       int64         // Fee limit for the transaction
	PermissionID   int32         // Permission ID for the transaction
	WaitForReceipt bool          // Wait for transaction receipt
	WaitTimeout    time.Duration // Timeout for waiting for receipt
	PollInterval   time.Duration // Polling interval when waiting for receipt
}

// DefaultBroadcastOptions returns sane defaults for broadcasting transactions.
//
// The default options are:
//   - FeeLimit: 150,000,000 SUN (0.15 TRX)
//   - PermissionID: 0 (owner permission)
//   - WaitForReceipt: true (wait for transaction confirmation)
//   - WaitTimeout: 15 seconds
//   - PollInterval: 3 seconds
func DefaultBroadcastOptions() BroadcastOptions {
	return BroadcastOptions{
		FeeLimit:       150_000_000,
		PermissionID:   0,
		WaitForReceipt: true,
		WaitTimeout:    15 * time.Second, // seconds
		PollInterval:   3 * time.Second,  // polling cadence
	}
}

// BroadcastResult summarizes the outcome of a simulation or a broadcasted
// transaction, including TRON return status, and for smart contract transactions,
// resource usage and logs.
//
// This struct contains the results of either a Simulate or SignAndBroadcast operation.
// For smart contract transactions (CreateSmartContract and TriggerSmartContract),
// when WaitForReceipt is true in SignAndBroadcast, additional fields like EnergyUsage
// and Logs will be populated with data from the transaction receipt. For other
// transaction types (like TRX transfers), only the basic success/failure information
// will be available.
type BroadcastResult struct {
	TxID    string                 `json:"txID"`
	Success bool                   `json:"success"`
	Code    api.ReturnResponseCode `json:"returnCode"`    // TRON return code
	Message string                 `json:"returnMessage"` // TRON return message concat with contract return message

	// ConstantReturn has the details of the contract returned error message or result
	// Populated for smart contract transactions when WaitForReceipt is true
	ConstantReturn [][]byte //test if nil before use

	// Fields primarily populated by simulation (TriggerConstantContract) or
	// for smart contract transactions when WaitForReceipt is true
	EnergyUsage int64                       `json:"energyUsed,omitempty"`
	NetUsage    int64                       `json:"netUsage,omitempty"`
	Logs        []*core.TransactionInfo_Log `json:"logs,omitempty"`
	// DebugExt   *api.TransactionExtention   `json:"debugExt,omitempty"`
}

// Simulate performs a read-only execution of a single-contract transaction and
// returns a BroadcastResult with constant return data, energy usage, and logs.
//
// This method allows you to test a transaction without actually broadcasting it
// to the network. It's useful for estimating energy usage and checking if a
// transaction would succeed before actually sending it.
//
// Supported input types are *api.TransactionExtention and *core.Transaction.
// The transaction must contain exactly one contract and must not be expired.
//
// Example:
//
//	sim, err := cli.Simulate(ctx, txExt)
//	if err != nil {
//	    // handle error
//	}
//	if !sim.Success {
//	    // transaction would fail
//	}
//	fmt.Printf("Energy usage: %d\n", sim.EnergyUsage)
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
	ext, err := lowlevel.Call(c, ctx, "trigger constant contract", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.TriggerConstantContract(ctx, decodedTx)
	})
	if err != nil {
		return nil, err
	}

	br := &BroadcastResult{}
	if ext != nil {
		if txid := ext.GetTxid(); len(txid) > 0 {
			br.TxID = hex.EncodeToString(txid)
		}
		if ret := ext.GetResult(); ret != nil {

			br.Success = ret.GetResult()
			if ext.GetTransaction() != nil && len(ext.GetTransaction().GetRet()) > 0 {
				br.Success = br.Success && ext.GetTransaction().GetRet()[0].GetRet() == core.Transaction_Result_SUCESS
			}
			br.Code = ret.GetCode()
			br.Message = string(ret.GetMessage()) + string(ext.GetResult().GetMessage())
		}
		br.ConstantReturn = ext.GetConstantResult()
		br.EnergyUsage = ext.GetEnergyUsed()
		br.Logs = ext.GetLogs()
	}

	return br, nil
}

// SignAndBroadcast signs a single-contract transaction using the provided
// signers (if any), applies BroadcastOptions, broadcasts it to the network,
// and optionally waits for receipt. It returns a BroadcastResult with txid,
// TRON return code/message, and, if waiting for smart contract transactions,
// resource usage and logs.
//
// This is the primary method for sending transactions to the TRON network.
// It handles signing, broadcasting, and (optionally) waiting for the transaction
// to be confirmed.
//
// For smart contract transactions (CreateSmartContract and TriggerSmartContract),
// when WaitForReceipt is enabled, it will retrieve execution results including
// energy usage, logs, and contract return values. For other transaction types
// (like TRX transfers), it will only indicate success/failure status.
//
// Supported input types are *api.TransactionExtention and *core.Transaction.
//
// Example:
//
//	opts := client.DefaultBroadcastOptions()
//	opts.FeeLimit = 100_000_000
//	opts.WaitForReceipt = true
//
//	result, err := cli.SignAndBroadcast(ctx, txExt, opts, signer)
//	if err != nil {
//	    // handle error
//	}
//	if result.Success {
//	    fmt.Printf("Transaction successful: %s\n", result.TxID)
//	    // For smart contract transactions, additional info will be available:
//	    if result.EnergyUsage > 0 {
//	        fmt.Printf("Energy used: %d\n", result.EnergyUsage)
//	    }
//	}
func (c *Client) SignAndBroadcast(ctx context.Context, anytx any, opt BroadcastOptions, signers ...signer.Signer) (*BroadcastResult, error) {
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
	if coretx.GetRawData().GetExpiration() < time.Now().UnixMilli() {
		return nil, fmt.Errorf("transaction expiration must be in the future")
	}

	if len(signers) > 0 {
		if opt.PermissionID != 0 {
			coretx.RawData.GetContract()[0].PermissionId = opt.PermissionID
		}
		coretx.RawData.FeeLimit = opt.FeeLimit
		for _, s := range signers {
			if err := signer.SignTx(s, coretx); err != nil {
				return nil, fmt.Errorf("failed to sign transaction: %w", err)
			}
		}
	}

	txid := utils.GetTransactionID(coretx)

	result := &BroadcastResult{TxID: hex.EncodeToString(txid)}

	ret, err := lowlevel.Call(c, ctx, "broadcast transaction", func(cl api.WalletClient, ctx context.Context) (*api.Return, error) {
		return cl.BroadcastTransaction(ctx, coretx)
	})
	if err != nil {
		return result, fmt.Errorf("failed to broadcast transaction: %w", err)
	}
	result.Success = ret.GetResult()
	result.Code = ret.GetCode()
	result.Message = string(ret.GetMessage())

	// Check if this is a smart contract transaction (only applicable to CreateSmartContract and TriggerSmartContract)
	contractType := coretx.GetRawData().GetContract()[0].GetType()
	isSmartContractTx := contractType == core.Transaction_Contract_CreateSmartContract ||
		contractType == core.Transaction_Contract_TriggerSmartContract

	if !opt.WaitForReceipt || !isSmartContractTx {
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
			txInfo, err := lowlevel.Call(c, ctx, "get transaction info by id", func(cl api.WalletClient, ctx context.Context) (*core.TransactionInfo, error) {
				return cl.GetTransactionInfoById(ctx, req)
			})
			if err == nil && txInfo != nil && bytes.Equal(txInfo.Id, txid) {
				return txInfo
			}
		}
	}
}
