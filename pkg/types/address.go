// Package types provides shared types and utilities for the TRON SDK
package types

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/mr-tron/base58"
	"golang.org/x/crypto/sha3"
)

const (
	addressPrefixByte = 0x41
	addressLength     = 21                                   // Raw address length in bytes (1 byte prefix + 20 bytes address)
	tRONAddressLength = 34                                   // TRON base58 address length
	BlackHoleAddress  = "T9yD14Nj9j7xAB4dbGeiX9h8unkKHxuWwb" // Black hole address prefix
)

// Address represents a TRON address that can be stored in different formats
type Address struct {
	base58Addr string
	bytesAddr  []byte
}

// NewAddress creates an Address from a string, []byte, or base58 string
// it will try to parse the address as base58 first, then hex, then bytes
// performance penalty is expected
func NewAddress(address any) (*Address, error) {
	switch v := address.(type) {
	case string:
		addr, err := NewAddressFromBase58(v)
		if err == nil {
			return addr, nil
		}
		addr, err = NewAddressFromHex(v)
		if err == nil {
			return addr, nil
		}
		return nil, fmt.Errorf("invalid address: %v", err)
	case []byte:
		addr, err := NewAddressFromBytes(v)
		if err == nil {
			return addr, nil
		}
		return nil, fmt.Errorf("invalid address: %v", err)
	case [20]byte:
		addr, err := NewAddressFromBytes(v[:])
		if err == nil {
			return addr, nil
		}
		return nil, fmt.Errorf("invalid address: %v", err)
	case [21]byte:
		addr, err := NewAddressFromBytes(v[:])
		if err == nil {
			return addr, nil
		}
		return nil, fmt.Errorf("invalid address: %v", err)
	default:
		return nil, fmt.Errorf("invalid address: %v", address)
	}
}

// NewAddressFromBase58 creates an Address from a base58 string, it must be a length 34 base58 string prefixed by "T"
func NewAddressFromBase58(base58Addr string) (*Address, error) {
	// Address must start with T
	if !strings.HasPrefix(base58Addr, "T") {
		return nil, fmt.Errorf("invalid address: must start with T")
	}
	// Address must be 34 chars long
	if len(base58Addr) != tRONAddressLength {
		return nil, fmt.Errorf("invalid address length: expected %d, got %d", tRONAddressLength, len(base58Addr))
	}

	decoded, err := base58.Decode(base58Addr)
	// Address must be valid base58
	if err != nil {
		return nil, fmt.Errorf("invalid base58 encoding: %w", err)
	}
	// Address hex must be prefixed with 0x41
	if decoded[0] != 0x41 {
		return nil, fmt.Errorf("invalid address prefix: expected 0x41, got 0x%x", decoded[0])
	}
	// Address hex must be 21 bytes long
	if len(decoded) != addressLength+4 { // 21 bytes address + 4 bytes checksum
		return nil, errors.New("invalid decoded address length")
	}

	addressBytes := decoded[:addressLength]
	checksum := decoded[addressLength:]

	// Verify checksum: first 4 bytes of sha256(sha256(addressBytes)) must be equal to checksum
	h1 := sha256.Sum256(addressBytes)
	h2 := sha256.Sum256(h1[:])
	if !bytes.Equal(h2[:4], checksum) {
		return nil, errors.New("invalid checksum")
	}

	return &Address{
		base58Addr: base58Addr,
		bytesAddr:  addressBytes,
	}, nil
}

// NewAddressFromHex creates an Address from a hex string,
// it must prefixed with 0x41 or 41 followed by 40 hex chars
// or 20 bytes hex string
func NewAddressFromHex(hexAddr string) (*Address, error) {
	// Remove 0x prefix if present
	hexAddr = strings.TrimPrefix(strings.ToLower(hexAddr), "0x")
	decoded, err := hex.DecodeString(hexAddr)

	if err != nil {
		return nil, fmt.Errorf("invalid hex encoding: %w", err)
	}

	switch len(decoded) {
	case 21:
		if decoded[0] != addressPrefixByte {
			return nil, fmt.Errorf("invalid hex address: must start with %x", addressPrefixByte)
		}
	case 20:
		// Valid address - add prefix
		prefixed := make([]byte, 21)
		prefixed[0] = addressPrefixByte
		copy(prefixed[1:], decoded)
		decoded = prefixed
	default:
		return nil, fmt.Errorf("invalid hex address length: expected 40 or 42, got %d", len(hexAddr))
	}

	return &Address{
		bytesAddr:  decoded,
		base58Addr: encodeBase58Addr(decoded),
	}, nil
}

// NewAddressFromBytes creates an Address from 21(prefixed with 0x41) or 20 bytes
func NewAddressFromBytes(byteAddress []byte) (*Address, error) {
	switch len(byteAddress) {
	case 21:
		if byteAddress[0] != 0x41 {
			return nil, fmt.Errorf("invalid address prefix: expected 0x41, got 0x%x", byteAddress[0])
		}
	case 20:
		// Valid address - add prefix
		prefixed := make([]byte, 21)
		prefixed[0] = addressPrefixByte
		copy(prefixed[1:], byteAddress)
		byteAddress = prefixed
	default:
		return nil, fmt.Errorf("invalid address length: expected 21 or 20, got %d", len(byteAddress))
	}

	return &Address{
		bytesAddr:  byteAddress,
		base58Addr: encodeBase58Addr(byteAddress),
	}, nil
}

// MustNewAddressFromBase58 is a wrapper for NewAddressFromBase58 that panics if the address is invalid
func MustNewAddressFromBase58(base58Addr string) *Address {
	addr, err := NewAddressFromBase58(base58Addr)
	if err != nil {
		panic(err)
	}
	return addr
}

// MustNewAddressFromHex is a wrapper for NewAddressFromHex that panics if the address is invalid
func MustNewAddressFromHex(hexAddr string) *Address {
	addr, err := NewAddressFromHex(hexAddr)
	if err != nil {
		panic(err)
	}
	return addr
}

// MustNewAddressFromBytes is a wrapper for NewAddressFromBytes that panics if the address is invalid
func MustNewAddressFromBytes(byteAddress []byte) *Address {
	addr, err := NewAddressFromBytes(byteAddress)
	if err != nil {
		panic(err)
	}
	return addr
}

// encodeBase58Addr returns the base58 representation of the address
func encodeBase58Addr(bytesAddr []byte) string {
	// Calculate checksum
	h1 := sha256.Sum256(bytesAddr)
	h2 := sha256.Sum256(h1[:])
	checksum := h2[:4]

	// Combine address with checksum
	combined := append([]byte{}, bytesAddr...)
	combined = append(combined, checksum...)

	// Encode to base58
	return base58.Encode(combined)
}

// String returns the T prefixed 34 chars base58 representation
func (a *Address) String() string {
	if a == nil {
		return ""
	}
	return a.base58Addr
}

// Base58 returns the T prefixed 34 chars base58 representation
func (a *Address) Base58() string {
	if a == nil {
		return ""
	}
	return a.base58Addr
}

// Bytes returns the raw bytes of the address (0x41 prefixed 21 bytes)
func (a *Address) Bytes() []byte {
	if a == nil {
		return nil
	}
	return a.bytesAddr
}

func (a *Address) BytesEVM() []byte {
	if a == nil {
		return nil
	}
	return a.bytesAddr[1:]
}

// Hex returns the hex string of the address (41 prefixed 42 chars)
func (a *Address) Hex() string {
	if a == nil {
		return ""
	}
	return hex.EncodeToString(a.bytesAddr)
}

// HexWithPrefix returns the address as hex string with 0x prefix
func (a *Address) HexEVM() string {
	if a == nil {
		return ""
	}
	return "0x" + hex.EncodeToString(a.bytesAddr[1:])
}

// IsValid checks if the address is valid
func (a *Address) IsValid() bool {
	return a != nil && len(a.bytesAddr) == 21 && a.bytesAddr[0] == 0x41
}

// Equal checks if two addresses are equal
func (a *Address) Equal(other *Address) bool {
	if a == nil || other == nil {
		return a == other
	}
	return bytes.Equal(a.bytesAddr, other.bytesAddr)
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

	return &Address{
		bytesAddr:  contractAddr,
		base58Addr: encodeBase58Addr(contractAddr),
	}, nil
}
