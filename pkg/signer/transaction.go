package signer

import (
	"crypto/sha256"
	"fmt"
	"strings" // Added strings import

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"google.golang.org/protobuf/proto"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// SignTx takes a transaction, hashes it, signs it using the provided signer,
// and attaches the signature to the transaction. It supports both *core.Transaction
// and *api.TransactionExtention types.
//
// IMPORTANT: Transaction parameters (such as permission ID, fee limit, etc.) must be set
// BEFORE calling this function, as the signature is calculated based on the transaction's
// raw data. Changing transaction parameters after signing will invalidate the signature.
//
// For multi-signature transactions, call this function multiple times with different
// signers on the same transaction object.
//
// Example:
//
//	tx, err := cli.Account().TransferTRX(ctx, from, to, amount)
//	if err != nil {
//	    // handle error
//	}
//
//	// Set any required parameters BEFORE signing
//	err = utils.SetPermissionID(tx, 3)  // For multi-sig scenarios
//	if err != nil {
//	    // handle error
//	}
//
//	// Now sign the transaction
//	pkSigner, _ := NewPrivateKeySigner("0x...")
//	err = SignTx(pkSigner, tx)
//	if err != nil {
//	    // Handle error
//	}
func SignTx(s Signer, tx any) error {
	if tx == nil {
		return fmt.Errorf("transaction cannot be nil")
	}

	var rawData []byte
	var err error

	switch t := tx.(type) {
	case *core.Transaction:
		rawData, err = proto.Marshal(t.GetRawData())
		if err != nil {
			return fmt.Errorf("failed to marshal transaction raw data: %w", err)
		}
		// Calculate hash
		h256h := sha256.New()
		h256h.Write(rawData)
		hash := h256h.Sum(nil)

		// Sign the hash
		signature, err := s.Sign(hash)
		if err != nil {
			return fmt.Errorf("failed to sign transaction: %w", err)
		}
		t.Signature = append(t.Signature, signature)

	case *api.TransactionExtention:
		rawData, err = proto.Marshal(t.GetTransaction().GetRawData())
		if err != nil {
			return fmt.Errorf("failed to marshal transaction raw data: %w", err)
		}
		// Calculate hash
		h256h := sha256.New()
		h256h.Write(rawData)
		hash := h256h.Sum(nil)

		// Sign the hash
		signature, err := s.Sign(hash)
		if err != nil {
			return fmt.Errorf("failed to sign transaction: %w", err)
		}
		t.Transaction.Signature = append(t.Transaction.Signature, signature)

	default:
		return fmt.Errorf("unsupported transaction type: %T", tx)
	}

	return nil
}

// SignMessageV2 signs a message using the TIP-191 format (v2) with the provided signer.
// It prefixes the message, hashes it, and then signs the hash.
//
// Example:
//
//	privateKey := "0x..."
//	message := "Hello Tron!"
//	pkSigner, _ := NewPrivateKeySigner(privateKey)
//	signature, err := SignMessageV2(pkSigner, message)
//	if err != nil {
//	    // Handle error
//	}
//	fmt.Printf("Signed Message Signature: %s\n", signature)
func SignMessageV2(s Signer, message string) (string, error) {
	var data []byte
	if strings.HasPrefix(message, "0x") {
		// Assume hex-encoded string
		data = common.FromHex(message)
	} else {
		data = []byte(message)
	}

	// Prefix the message
	messageLen := len(data)
	prefixedMessage := []byte(fmt.Sprintf("%s%d%s", TronMessagePrefix, messageLen, string(data)))

	// Hash the prefixed message (Keccak256)
	hash := crypto.Keccak256Hash(prefixedMessage)

	// Sign the hash
	signature, err := s.Sign(hash.Bytes())
	if err != nil {
		return "", fmt.Errorf("failed to sign message: %w", err)
	}

	// Adjust the recovery ID (v). go-ethereum's Sign function returns
	// the signature in [R || S || V] format, where V is 0 or 1. Tron
	// expects V to be 27 or 28, so we add 27.
	signature[64] += 27

	// Return the hex-encoded signature
	return "0x" + common.Bytes2Hex(signature), nil
}
