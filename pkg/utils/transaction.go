// Package types provides shared types and utilities for the TRON SDK
package utils

import (
	"crypto/sha256"

	"github.com/kslamph/tronlib/pb/core"
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
