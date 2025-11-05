// Package types provides shared types and utilities for the TRON SDK
package utils

import (
	"crypto/sha256"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
	"google.golang.org/protobuf/proto"
)

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

func ExtractSigners(tx *core.Transaction) ([]*types.Address, error) {
	if tx == nil {
		return nil, nil
	}

	// Get the raw transaction data that was signed
	rawData, err := proto.Marshal(tx.GetRawData())
	if err != nil {
		return nil, err
	}

	// Calculate the hash that was signed (SHA256 of raw data)
	h256h := sha256.New()
	h256h.Write(rawData)
	hash := h256h.Sum(nil)

	// Get signatures from the transaction
	signatures := tx.GetSignature()
	if len(signatures) == 0 {
		return nil, fmt.Errorf("no signatures found in transaction")
	}

	signers := make([]*types.Address, 0, len(signatures))
	for _, sig := range signatures {
		if len(sig) < 64 {
			continue
		}

		// TRON signatures should be properly formatted for recovery
		// Recover the public key from the signature and hash
		pubKey, err := crypto.SigToPub(hash, sig)
		if err != nil {
			continue // Skip invalid signatures
		}

		// Convert public key to TRON address
		ethAddress := crypto.PubkeyToAddress(*pubKey)

		// Convert Ethereum address to TRON address format
		tronAddr := types.MustNewAddressFromHex(ethAddress.Hex())
		signers = append(signers, tronAddr)
	}

	return signers, nil
}

// SetPermissionID sets the permission ID for a transaction contract.
// IMPORTANT: This must be called before signing the transaction, as the signature
// is calculated based on the transaction's raw data including the permission ID.
//
// The function supports both *core.Transaction and *api.TransactionExtention types.
// For multi-signature transactions, ensure the permission ID is set before calling
// signer.SignTx() with any signers.
//
// Example:
//
//	tx, err := cli.Account().TransferTRX(ctx, from, to, amount)
//	if err != nil {
//	    // handle error
//	}
//
//	// Set permission ID BEFORE signing
//	err = utils.SetPermissionID(tx, 3)
//	if err != nil {
//	    // handle error
//	}
//
//	// Now sign the transaction
//	err = signer.SignTx(signer1, tx)
//	if err != nil {
//	    // handle error
//	}
func SetPermissionID(tx any, permissionID int32) error {
	if tx == nil {
		return fmt.Errorf("transaction cannot be nil")
	}

	switch t := tx.(type) {
	case *core.Transaction:
		if t.RawData == nil {
			return fmt.Errorf("transaction raw data cannot be nil")
		}
		if len(t.RawData.Contract) == 0 {
			return fmt.Errorf("transaction must have at least one contract")
		}
		t.RawData.Contract[0].PermissionId = permissionID
		return nil

	case *api.TransactionExtention:
		if t.Transaction == nil {
			return fmt.Errorf("transaction extension contains nil transaction")
		}
		if t.Transaction.RawData == nil {
			return fmt.Errorf("transaction raw data cannot be nil")
		}
		if len(t.Transaction.RawData.Contract) == 0 {
			return fmt.Errorf("transaction must have at least one contract")
		}
		t.Transaction.RawData.Contract[0].PermissionId = permissionID
		return nil

	default:
		return fmt.Errorf("unsupported transaction type: %T, expected *core.Transaction or *api.TransactionExtention", tx)
	}
}

// SetFeeLimit sets the fee limit for a transaction.
// IMPORTANT: This must be called before signing the transaction, as the signature
// is calculated based on the transaction's raw data including the fee limit.
//
// The function supports both *core.Transaction and *api.TransactionExtention types.
//
// Example:
//
//	tx, err := cli.Account().TransferTRX(ctx, from, to, amount)
//	if err != nil {
//	    // handle error
//	}
//
//	// Set fee limit BEFORE signing
//	err = utils.SetFeeLimit(tx, 150_000_000)  // 0.15 TRX
//	if err != nil {
//	    // handle error
//	}
//
//	// Now sign the transaction
//	err = signer.SignTx(signer1, tx)
//	if err != nil {
//	    // handle error
//	}
func SetFeeLimit(tx any, feeLimit int64) error {
	if tx == nil {
		return fmt.Errorf("transaction cannot be nil")
	}

	switch t := tx.(type) {
	case *core.Transaction:
		if t.RawData == nil {
			return fmt.Errorf("transaction raw data cannot be nil")
		}
		t.RawData.FeeLimit = feeLimit
		return nil

	case *api.TransactionExtention:
		if t.Transaction == nil {
			return fmt.Errorf("transaction extension contains nil transaction")
		}
		if t.Transaction.RawData == nil {
			return fmt.Errorf("transaction raw data cannot be nil")
		}
		t.Transaction.RawData.FeeLimit = feeLimit
		return nil

	default:
		return fmt.Errorf("unsupported transaction type: %T, expected *core.Transaction or *api.TransactionExtention", tx)
	}
}

// SetTimestamp sets the timestamp for a transaction.
// IMPORTANT: This must be called before signing the transaction, as the signature
// is calculated based on the transaction's raw data including the timestamp.
//
// The function supports both *core.Transaction and *api.TransactionExtention types.
// Note that if not set, the TRON node will typically set this when processing the transaction.
//
// Example:
//
//	tx, err := cli.Account().TransferTRX(ctx, from, to, amount)
//	if err != nil {
//	    // handle error
//	}
//
//	// Set timestamp BEFORE signing (usually current time in milliseconds)
//	err = utils.SetTimestamp(tx, time.Now().UnixMilli())
//	if err != nil {
//	    // handle error
//	}
//
//	// Now sign the transaction
//	err = signer.SignTx(signer1, tx)
//	if err != nil {
//	    // handle error
//	}
func SetTimestamp(tx any, timestamp int64) error {
	if tx == nil {
		return fmt.Errorf("transaction cannot be nil")
	}

	switch t := tx.(type) {
	case *core.Transaction:
		if t.RawData == nil {
			return fmt.Errorf("transaction raw data cannot be nil")
		}
		t.RawData.Timestamp = timestamp
		return nil

	case *api.TransactionExtention:
		if t.Transaction == nil {
			return fmt.Errorf("transaction extension contains nil transaction")
		}
		if t.Transaction.RawData == nil {
			return fmt.Errorf("transaction raw data cannot be nil")
		}
		t.Transaction.RawData.Timestamp = timestamp
		return nil

	default:
		return fmt.Errorf("unsupported transaction type: %T, expected *core.Transaction or *api.TransactionExtention", tx)
	}
}

// SetExpiration sets the expiration time for a transaction.
// IMPORTANT: This must be called before signing the transaction, as the signature
// is calculated based on the transaction's raw data including the expiration time.
//
// The function supports both *core.Transaction and *api.TransactionExtention types.
// The expiration should be a Unix timestamp in milliseconds in the future.
//
// Example:
//
//	tx, err := cli.Account().TransferTRX(ctx, from, to, amount)
//	if err != nil {
//	    // handle error
//	}
//
//	// Set expiration BEFORE signing (e.g., 1 hour from now)
//	expiration := time.Now().Add(time.Hour).UnixMilli()
//	err = utils.SetExpiration(tx, expiration)
//	if err != nil {
//	    // handle error
//	}
//
//	// Now sign the transaction
//	err = signer.SignTx(signer1, tx)
//	if err != nil {
//	    // handle error
//	}
func SetExpiration(tx any, expiration int64) error {
	if tx == nil {
		return fmt.Errorf("transaction cannot be nil")
	}

	switch t := tx.(type) {
	case *core.Transaction:
		if t.RawData == nil {
			return fmt.Errorf("transaction raw data cannot be nil")
		}
		t.RawData.Expiration = expiration
		return nil

	case *api.TransactionExtention:
		if t.Transaction == nil {
			return fmt.Errorf("transaction extension contains nil transaction")
		}
		if t.Transaction.RawData == nil {
			return fmt.Errorf("transaction raw data cannot be nil")
		}
		t.Transaction.RawData.Expiration = expiration
		return nil

	default:
		return fmt.Errorf("unsupported transaction type: %T, expected *core.Transaction or *api.TransactionExtention", tx)
	}
}
