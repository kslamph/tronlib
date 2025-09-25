// Package types provides shared types and utilities for the TRON SDK
package utils

import (
	"crypto/sha256"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
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
