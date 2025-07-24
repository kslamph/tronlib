// Package types provides shared types and utilities for the TRON SDK
package types

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
)

// Transaction represents a TRON transaction with additional metadata
type Transaction struct {
	*core.Transaction
	TxID      string
	Timestamp time.Time
}

// TransactionInfo represents detailed transaction information
type TransactionInfo struct {
	*core.TransactionInfo
	Transaction *Transaction
}

// TransactionResult represents the result of a transaction operation
type TransactionResult struct {
	Transaction *api.TransactionExtention
	TxID        string
	Success     bool
	Message     string
	Error       error
}

// NewTransaction creates a new Transaction from core.Transaction
func NewTransaction(tx *core.Transaction) *Transaction {
	if tx == nil {
		return nil
	}
	
	txIDBytes := GetTransactionID(tx)
	var txIDStr string
	if txIDBytes != nil {
		txIDStr = hex.EncodeToString(txIDBytes)
	}
	
	return &Transaction{
		Transaction: tx,
		TxID:        txIDStr,
		Timestamp:   time.Unix(tx.RawData.Timestamp/1000, 0),
	}
}

// GetTransactionID calculates the transaction ID from a transaction
func GetTransactionID(tx *core.Transaction) []byte {
	if tx == nil || tx.RawData == nil {
		return nil
	}
	
	// Marshal raw data for hashing
	rawData, err := proto.Marshal(tx.RawData)
	if err != nil {
		return nil
	}
	
	// Calculate SHA256 hash
	hasher := sha256.New()
	hasher.Write(rawData)
	return hasher.Sum(nil)
}

// IsValid checks if the transaction is valid
func (t *Transaction) IsValid() bool {
	return t != nil && t.Transaction != nil && t.Transaction.RawData != nil
}

// GetContractType returns the type of the first contract in the transaction
func (t *Transaction) GetContractType() core.Transaction_Contract_ContractType {
	if !t.IsValid() || len(t.Transaction.RawData.Contract) == 0 {
		return core.Transaction_Contract_AccountCreateContract
	}
	return t.Transaction.RawData.Contract[0].Type
}

// GetContractAddress returns the contract address for contract-related transactions
func (t *Transaction) GetContractAddress() *Address {
	if !t.IsValid() || len(t.Transaction.RawData.Contract) == 0 {
		return nil
	}
	
	contract := t.Transaction.RawData.Contract[0]
	switch contract.Type {
	case core.Transaction_Contract_TriggerSmartContract:
		// Extract contract address from TriggerSmartContract
		// TODO: Implement proper contract address extraction
		return nil
	case core.Transaction_Contract_CreateSmartContract:
		// Extract contract address from CreateSmartContract
		// TODO: Implement proper contract address extraction
		return nil
	default:
		return nil
	}
}

// TransactionBuilder provides a fluent interface for building transactions
type TransactionBuilder struct {
	tx *core.Transaction
}

// NewTransactionBuilder creates a new TransactionBuilder
func NewTransactionBuilder() *TransactionBuilder {
	return &TransactionBuilder{
		tx: &core.Transaction{
			RawData: &core.TransactionRaw{
				Contract:   make([]*core.Transaction_Contract, 0),
				Timestamp:  time.Now().UnixMilli(),
				Expiration: time.Now().Add(10 * time.Minute).UnixMilli(),
			},
		},
	}
}

// WithRefBlock sets the reference block for the transaction
func (b *TransactionBuilder) WithRefBlock(refBlockHash []byte, refBlockNum int64) *TransactionBuilder {
	if b.tx.RawData != nil {
		b.tx.RawData.RefBlockHash = refBlockHash[8:16]
		b.tx.RawData.RefBlockBytes = []byte{byte(refBlockNum & 0xff), byte((refBlockNum >> 8) & 0xff)}
	}
	return b
}

// WithExpiration sets the expiration time for the transaction
func (b *TransactionBuilder) WithExpiration(expiration time.Time) *TransactionBuilder {
	if b.tx.RawData != nil {
		b.tx.RawData.Expiration = expiration.UnixMilli()
	}
	return b
}

// WithTimestamp sets the timestamp for the transaction
func (b *TransactionBuilder) WithTimestamp(timestamp time.Time) *TransactionBuilder {
	if b.tx.RawData != nil {
		b.tx.RawData.Timestamp = timestamp.UnixMilli()
	}
	return b
}

// WithContract adds a contract to the transaction
func (b *TransactionBuilder) WithContract(contractType core.Transaction_Contract_ContractType, parameter []byte) *TransactionBuilder {
	if b.tx.RawData != nil {
		contract := &core.Transaction_Contract{
			Type:      contractType,
			Parameter: &anypb.Any{Value: parameter},
		}
		b.tx.RawData.Contract = append(b.tx.RawData.Contract, contract)
	}
	return b
}

// Build returns the built transaction
func (b *TransactionBuilder) Build() *core.Transaction {
	return b.tx
}

// TransactionOptions represents options for transaction operations
type TransactionOptions struct {
	FeeLimit       int64
	CallValue      int64
	TokenID        int64
	TokenValue     int64
	PermissionID   int32
	Memo           string
	ExtraData      []byte
}

// DefaultTransactionOptions returns default transaction options
func DefaultTransactionOptions() *TransactionOptions {
	return &TransactionOptions{
		FeeLimit:     1000000, // 1 TRX
		CallValue:    0,
		TokenID:      0,
		TokenValue:   0,
		PermissionID: 0,
		Memo:         "",
		ExtraData:    nil,
	}
}