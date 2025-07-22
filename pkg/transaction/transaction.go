package transaction

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
	"google.golang.org/protobuf/proto"
)

const (
	// DefaultFeeLimit is the default fee limit for transactions
	DefaultFeeLimit = 100_000_000
	// DefaultExpiration is the default expiration time in seconds
	DefaultExpiration = 60
)

// Transaction represents a Tron transaction with its extension data
type Transaction struct {
	client      *client.Client
	owner       *types.Address
	txExtension *api.TransactionExtention
	receipt     *Receipt
}

// Receipt represents a transaction receipt
type Receipt struct {
	TxID    string
	Result  bool
	Message string
	Err     error
}

// NewTransaction creates a new transaction instance
func NewTransaction(client *client.Client) *Transaction {
	blackHoleAddr, _ := types.NewAddress(types.BlackHoleAddress)
	return &Transaction{
		client: client,
		owner:  blackHoleAddr,
		receipt: &Receipt{
			TxID:    "",
			Result:  false,
			Message: "",
			Err:     nil},
	}
}

func (tx *Transaction) SetOwner(owner *types.Address) *Transaction {
	if tx.receipt.Err != nil {
		return tx
	}
	tx.owner = owner
	return tx
}

// SetFeelimit sets the fee limit for the transaction
func (tx *Transaction) SetFeelimit(limit int64) *Transaction {
	if tx.receipt.Err != nil {
		return tx
	}
	if limit <= 0 {
		limit = DefaultFeeLimit
	}
	if tx.txExtension.GetTransaction() != nil {
		tx.txExtension.GetTransaction().RawData.FeeLimit = limit
	}
	return tx
}

// SetExpiration sets the expiration time in seconds from now
func (tx *Transaction) SetExpiration(seconds int64) *Transaction {
	if tx.receipt.Err != nil {
		return tx
	}
	if seconds <= 0 {
		seconds = DefaultExpiration
	}
	// Set the expiration time in milliseconds
	// Note: Tron uses milliseconds for expiration
	// and the default is 60 seconds
	// so we multiply by 1000 to convert to milliseconds
	if tx.txExtension.GetTransaction() != nil {
		tx.txExtension.GetTransaction().RawData.Expiration = time.Now().UnixMilli() + seconds*1000
	}
	return tx
}

func (tx *Transaction) SetError(err error) *Transaction {
	tx.receipt.Err = err
	return tx
}

// Sign signs the transaction with the signer
func (tx *Transaction) Sign(signer types.Signer) *Transaction {
	if tx.receipt.Err != nil {
		return tx
	}
	return tx.MultiSign(signer, 2)
}

// MultiSign signs the transaction with the specified permission ID
func (tx *Transaction) MultiSign(signer types.Signer, permissionID int32) *Transaction {
	if tx.receipt.Err != nil {
		return tx
	}

	if tx.txExtension.GetTransaction() == nil {
		tx.receipt.Err = fmt.Errorf("no transaction to sign")
		return tx
	}

	// Sign the transaction using the signer
	signedTx, err := signer.MultiSign(tx.txExtension.GetTransaction(), permissionID)
	if err != nil {
		tx.receipt.Err = fmt.Errorf("failed to sign transaction: %v", err)
		return tx
	}

	tx.txExtension.Transaction = signedTx
	if err := tx.updateTxID(); err != nil {
		tx.receipt.Err = fmt.Errorf("failed to update transaction ID: %v", err)
		return tx
	}
	return tx
}

// Broadcast broadcasts the signed transaction to the network
func (tx *Transaction) Broadcast() *Transaction {
	if tx.receipt.Err != nil {
		return tx // Return early if there's already an error
	}

	if tx.txExtension == nil {
		tx.receipt.Err = fmt.Errorf("broadcast failed: no transaction extension")
		return tx
	}

	if tx.txExtension.GetTransaction() == nil {
		tx.receipt.Err = fmt.Errorf("broadcast failed: no transaction to broadcast")
		return tx
	}

	// Ensure Txid is set before broadcasting
	if len(tx.txExtension.GetTxid()) == 0 {
		if err := tx.updateTxID(); err != nil {
			tx.receipt.Err = fmt.Errorf("failed to update transaction ID before broadcast: %w", err)
			return tx
		}
	}

	// Preserve the TxID that was set either by signing or by updateTxID
	finalTxID := hex.EncodeToString(tx.txExtension.GetTxid())

	// Create context with client timeout
	ctx, cancel := context.WithTimeout(context.Background(), tx.client.GetTimeout())
	defer cancel()

	// Get connection from pool
	conn, err := tx.client.GetConnection(ctx)
	if err != nil {
		tx.receipt.Result = false
		tx.receipt.Message = fmt.Sprintf("connection error: %v", err)
		tx.receipt.Err = fmt.Errorf("failed to get connection for broadcast: %w", err)
		return tx
	}

	// Ensure connection is returned to pool
	defer tx.client.ReturnConnection(conn)

	// Create wallet client
	client := api.NewWalletClient(conn)

	// Broadcast the transaction
	resp, grpcErr := client.BroadcastTransaction(ctx, tx.txExtension.GetTransaction())

	// Initialize receipt fields that are certain
	tx.receipt.TxID = finalTxID // Use the finalTxID captured before broadcast

	if grpcErr != nil {
		tx.receipt.Result = false
		tx.receipt.Message = fmt.Sprintf("gRPC call to BroadcastTransaction failed: %v", grpcErr)
		// If there was no prior error, set this as the error. Otherwise, append.
		if tx.receipt.Err == nil {
			tx.receipt.Err = fmt.Errorf("broadcast failed: %w", grpcErr)
		} else {
			tx.receipt.Err = fmt.Errorf("%w; additionally, broadcast failed: %w", tx.receipt.Err, grpcErr)
		}
		return tx
	}

	if resp == nil {
		tx.receipt.Result = false
		tx.receipt.Message = "broadcast failed: nil response"
		tx.receipt.Err = fmt.Errorf("broadcast failed: nil response from server")
		return tx
	}

	tx.receipt.Result = resp.GetResult()
	tx.receipt.Message = string(resp.GetMessage())

	if !resp.GetResult() {
		chainError := fmt.Errorf("transaction broadcast to chain failed: %s", string(resp.GetMessage()))
		if tx.receipt.Err == nil {
			tx.receipt.Err = chainError
		} else {
			// Append chain error to existing error
			tx.receipt.Err = fmt.Errorf("%w; additionally, %w", tx.receipt.Err, chainError)
		}
	}
	// If resp.GetResult() is true and tx.receipt.Err was nil, it remains nil (success)

	return tx
}

// GetReceipt returns the transaction receipt
func (tx *Transaction) GetReceipt() *Receipt {
	return tx.receipt
}

func (tx *Transaction) GetError() error {
	return tx.receipt.Err
}

// updateTxID updates the transaction ID
func (tx *Transaction) updateTxID() error {
	rawData, err := proto.Marshal(tx.txExtension.GetTransaction().RawData)
	if err != nil {
		return fmt.Errorf("failed to marshal raw data: %v", err)
	}
	rawDataSHA256 := sha256.Sum256(rawData)
	tx.txExtension.Txid = rawDataSHA256[:]
	return nil
}

func (tx *Transaction) SetDefaultOptions() *Transaction {
	if tx.receipt.Err != nil {
		return tx
	}
	if tx.txExtension.GetTransaction() != nil {
		tx.txExtension.GetTransaction().RawData.Expiration = time.Now().UnixMilli() + DefaultExpiration*1000
		tx.txExtension.GetTransaction().RawData.FeeLimit = DefaultFeeLimit
		return tx
	}
	tx.receipt.Err = fmt.Errorf("no transaction to set default options")
	return tx
}

// DeployContract deploys a smart contract with the provided parameters
// The owner address must be set using SetOwner() before calling this method
func (tx *Transaction) DeployContract(ctx context.Context, bytecode []byte, abi []byte, name string, originEnergyLimit int64, consumeUserResourcePercent int64, constructorParams ...interface{}) *Transaction {
	if tx.receipt.Err != nil {
		return tx
	}

	// Validate required parameters
	if len(bytecode) == 0 {
		tx.receipt.Err = fmt.Errorf("bytecode cannot be empty")
		return tx
	}
	if len(abi) == 0 {
		tx.receipt.Err = fmt.Errorf("abi cannot be empty")
		return tx
	}
	if name == "" {
		tx.receipt.Err = fmt.Errorf("contract name cannot be empty")
		return tx
	}
	if originEnergyLimit <= 0 {
		tx.receipt.Err = fmt.Errorf("origin energy limit must be greater than 0")
		return tx
	}
	if consumeUserResourcePercent < 0 || consumeUserResourcePercent > 100 {
		tx.receipt.Err = fmt.Errorf("consume user resource percent must be between 0 and 100")
		return tx
	}

	// Check if owner is set
	if tx.owner == nil {
		tx.receipt.Err = fmt.Errorf("owner address must be set before deploying contract")
		return tx
	}

	// Prepare the final bytecode with constructor parameters if provided
	finalBytecode := bytecode
	if len(constructorParams) > 0 {
		// Decode ABI to handle constructor parameters
		contractABI, err := types.DecodeABI(string(abi))
		if err != nil {
			tx.receipt.Err = fmt.Errorf("failed to decode ABI: %v", err)
			return tx
		}

		// Create contract instance to encode constructor parameters
		contract, err := types.NewContractFromABI(contractABI, tx.owner.String())
		if err != nil {
			tx.receipt.Err = fmt.Errorf("failed to create contract instance: %v", err)
			return tx
		}

		// Encode constructor parameters
		encodedParams, err := contract.EncodeInput("", constructorParams...)
		if err != nil {
			tx.receipt.Err = fmt.Errorf("failed to encode constructor parameters: %v", err)
			return tx
		}

		// Append constructor parameters to bytecode
		finalBytecode = append(bytecode, encodedParams...)
	}

	// Decode ABI for the smart contract
	contractABI, err := types.DecodeABI(string(abi))
	if err != nil {
		tx.receipt.Err = fmt.Errorf("failed to decode ABI for smart contract: %v", err)
		return tx
	}

	// Create the smart contract request
	createReq := &core.CreateSmartContract{
		OwnerAddress: tx.owner.Bytes(),
		NewContract: &core.SmartContract{
			Name:                       name,
			Bytecode:                   finalBytecode,
			Abi:                        contractABI,
			OriginAddress:              tx.owner.Bytes(),
			OriginEnergyLimit:          originEnergyLimit,
			ConsumeUserResourcePercent: consumeUserResourcePercent,
		},
	}

	// Call the low-level client method
	txExt, err := tx.client.CreateDeployContractTransaction(ctx, createReq)
	if err != nil {
		tx.receipt.Err = fmt.Errorf("failed to create deploy contract transaction: %v", err)
		return tx
	}

	tx.txExtension = txExt
	tx.SetDefaultOptions()
	return tx
}
