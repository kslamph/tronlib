package transaction

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
	"google.golang.org/protobuf/proto"
)

// Transaction represents a Tron transaction with its extension data
type Transaction struct {
	client        *client.Client
	senderAccount *types.Account
	txExtension   *api.TransactionExtention
	receipt       *Receipt
}

// NewTransaction creates a new transaction instance
func NewTransaction(client *client.Client, sender *types.Account) *Transaction {
	return &Transaction{
		client:        client,
		senderAccount: sender,
	}
}

// SetFeelimit sets the fee limit for the transaction
func (tx *Transaction) SetFeelimit(limit int64) {
	if tx.txExtension.GetTransaction() != nil {
		tx.txExtension.GetTransaction().RawData.FeeLimit = limit
	}
}

// SetExpiration sets the expiration time in seconds from now
func (tx *Transaction) SetExpiration(seconds int64) {
	if tx.txExtension.GetTransaction() != nil {
		tx.txExtension.GetTransaction().RawData.Expiration = time.Now().UnixMilli() + seconds*1000
	}
}
func (tx *Transaction) Sign(signer *types.Account) error {
	return tx.MultiSign(signer, 2)
}

// Sign signs the transaction with the sender's private key
func (tx *Transaction) MultiSign(signer *types.Account, permissionID int32) error {

	if tx.txExtension.GetTransaction() == nil {
		return fmt.Errorf("no transaction to sign")
	}

	// Sign the transaction using the account's private key
	signedTx, err := signer.MultiSign(tx.txExtension.GetTransaction(), permissionID)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %v", err)
	}

	tx.txExtension.Transaction = signedTx
	if err := tx.UpdateTxID(); err != nil {
		return err
	}
	return nil
}

// Broadcast broadcasts the signed transaction to the network
func (tx *Transaction) Broadcast() error {
	if tx.txExtension.GetTransaction() == nil {
		return fmt.Errorf("no transaction to broadcast")
	}

	resp, err := tx.client.BroadcastTransaction(tx.txExtension.GetTransaction())
	if err != nil {
		return fmt.Errorf("failed to broadcast transaction: %v", err)
	}

	if !resp.GetResult() {
		return fmt.Errorf("broadcast failed: %s", string(resp.GetMessage()))
	}

	// Store receipt info
	tx.receipt = &Receipt{
		TxID:    tx.GetTxID(),
		Result:  resp.GetResult(),
		Message: string(resp.GetMessage()),
	}

	return nil
}

// GetTxID returns the transaction ID in hex format
func (tx *Transaction) UpdateTxID() error {
	rawData, err := proto.Marshal(tx.txExtension.GetTransaction().RawData)
	if err != nil {
		return fmt.Errorf("failed to marshal raw data: %v", err)
	}
	rawDataSHA256 := sha256.Sum256(rawData)
	tx.txExtension.Txid = rawDataSHA256[:]
	return nil
}
func (tx *Transaction) GetTxID() string {

	if tx.txExtension != nil {
		return hex.EncodeToString(tx.txExtension.GetTxid())
	}
	return ""
}

// GetReceipt returns the transaction receipt
func (tx *Transaction) GetReceipt() *Receipt {
	return tx.receipt
}

// Receipt represents a transaction receipt
type Receipt struct {
	TxID    string
	Result  bool
	Message string
}
