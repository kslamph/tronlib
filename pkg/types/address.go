// Package types provides shared types and utilities for the TRON SDK
package types

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/btcsuite/btcutil/base58"
	"golang.org/x/crypto/sha3"
)

// Address represents a TRON address
type Address struct {
	address []byte
}

// NewAddressFromBytes creates a new Address from byte slice
func NewAddressFromBytes(addr []byte) (*Address, error) {
	if len(addr) != 21 {
		return nil, errors.New("invalid address length")
	}
	return &Address{address: addr}, nil
}

// NewAddressFromHex creates a new Address from hex string
func NewAddressFromHex(hexAddr string) (*Address, error) {
	if len(hexAddr) == 42 && hexAddr[:2] == "0x" {
		hexAddr = hexAddr[2:]
	}
	if len(hexAddr) != 40 {
		return nil, errors.New("invalid hex address length")
	}
	
	addr, err := hex.DecodeString(hexAddr)
	if err != nil {
		return nil, fmt.Errorf("invalid hex address: %v", err)
	}
	
	if len(addr) != 20 {
		return nil, errors.New("decoded address must be 20 bytes")
	}
	
	// Add TRON prefix (0x41)
	tronAddr := make([]byte, 21)
	tronAddr[0] = 0x41
	copy(tronAddr[1:], addr)
	
	return &Address{address: tronAddr}, nil
}

// NewAddressFromBase58 creates a new Address from base58 string
func NewAddressFromBase58(base58Addr string) (*Address, error) {
	decoded := base58.Decode(base58Addr)
	if len(decoded) != 25 {
		return nil, errors.New("invalid base58 address length")
	}
	
	// Verify checksum
	addr := decoded[:21]
	checksum := decoded[21:]
	
	hash1 := sha256.Sum256(addr)
	hash2 := sha256.Sum256(hash1[:])
	
	if !bytesEqual(hash2[:4], checksum) {
		return nil, errors.New("invalid base58 address checksum")
	}
	
	return &Address{address: addr}, nil
}

// Bytes returns the address as byte slice
func (a *Address) Bytes() []byte {
	if a == nil {
		return nil
	}
	return a.address
}

// Hex returns the address as hex string (without 0x prefix)
func (a *Address) Hex() string {
	if a == nil {
		return ""
	}
	return hex.EncodeToString(a.address)
}

// HexWithPrefix returns the address as hex string with 0x prefix
func (a *Address) HexWithPrefix() string {
	if a == nil {
		return ""
	}
	return "0x" + hex.EncodeToString(a.address)
}

// Base58 returns the address as base58 string
func (a *Address) Base58() string {
	if a == nil {
		return ""
	}
	
	// Calculate checksum
	hash1 := sha256.Sum256(a.address)
	hash2 := sha256.Sum256(hash1[:])
	
	// Append checksum
	fullAddr := make([]byte, 25)
	copy(fullAddr[:21], a.address)
	copy(fullAddr[21:], hash2[:4])
	
	return base58.Encode(fullAddr)
}

// String returns the address as base58 string (default representation)
func (a *Address) String() string {
	return a.Base58()
}

// IsValid checks if the address is valid
func (a *Address) IsValid() bool {
	return a != nil && len(a.address) == 21 && a.address[0] == 0x41
}

// Equal checks if two addresses are equal
func (a *Address) Equal(other *Address) bool {
	if a == nil || other == nil {
		return a == other
	}
	return bytesEqual(a.address, other.address)
}

// Helper function to compare byte slices
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// GenerateContractAddress generates a contract address from creator address and nonce
func GenerateContractAddress(creatorAddr *Address, nonce uint64) (*Address, error) {
	if creatorAddr == nil {
		return nil, errors.New("creator address is nil")
	}
	
	// Convert nonce to bytes
	nonceBytes := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		nonceBytes[i] = byte(nonce)
		nonce >>= 8
	}
	
	// Concatenate creator address and nonce
	data := append(creatorAddr.Bytes(), nonceBytes...)
	
	// Calculate keccak256 hash
	hash := sha3.NewLegacyKeccak256()
	hash.Write(data)
	hashBytes := hash.Sum(nil)
	
	// Take last 20 bytes and add TRON prefix
	contractAddr := make([]byte, 21)
	contractAddr[0] = 0x41
	copy(contractAddr[1:], hashBytes[12:])
	
	return &Address{address: contractAddr}, nil
}