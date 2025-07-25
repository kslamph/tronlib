package types

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/mr-tron/base58"
)

const (
	AddressPrefix     = "41"                                 // TRON address prefix in hex
	AddressLength     = 21                                   // Raw address length in bytes (1 byte prefix + 20 bytes address)
	TRONAddressLength = 34                                   // TRON base58 address length
	BlackHoleAddress  = "T9yD14Nj9j7xAB4dbGeiX9h8unkKHxuWwb" // Black hole address prefix in hex
)

// Address represents a Tron address that can be stored in different formats
type Address struct {
	base58Addr string
	bytesAddr  []byte
	validated  bool // Private field to track if address was properly constructed
}

// NewAddress creates an Address from a base58 string, it must be a length 34 base58 string prefixed by "T"
func NewAddress(base58Addr string) (*Address, error) {
	if !strings.HasPrefix(base58Addr, "T") {
		return nil, fmt.Errorf("invalid address: must start with T")
	}
	if len(base58Addr) != TRONAddressLength {
		return nil, fmt.Errorf("invalid address length: expected %d, got %d", TRONAddressLength, len(base58Addr))
	}

	// Validate checksum
	decoded, err := base58.Decode(base58Addr)
	if err != nil {
		return nil, fmt.Errorf("invalid base58 encoding: %w", err)
	}

	if len(decoded) != AddressLength+4 { // 21 bytes address + 4 bytes checksum
		return nil, errors.New("invalid decoded address length")
	}

	if decoded[0] != 0x41 {
		return nil, fmt.Errorf("invalid address prefix: expected 0x41, got 0x%x", decoded[0])
	}

	addressBytes := decoded[:AddressLength]
	checksum := decoded[AddressLength:]

	// Verify checksum
	h1 := sha256.Sum256(addressBytes)
	h2 := sha256.Sum256(h1[:])
	if !bytes.Equal(h2[:4], checksum) {
		return nil, errors.New("invalid checksum")
	}

	return &Address{
		base58Addr: base58Addr,
		validated:  true,
	}, nil
}

// NewAddressFromHex creates an Address from a hex string, it must prefixed with 0x41 or 41 followed by 40 hex chars
func NewAddressFromHex(hexAddr string) (*Address, error) {
	// Remove 0x prefix if present
	hexAddr = strings.TrimPrefix(strings.ToLower(hexAddr), "0x")

	// Validate prefix and length
	if !strings.HasPrefix(hexAddr, AddressPrefix) {
		return nil, fmt.Errorf("invalid hex address: must start with %s", AddressPrefix)
	}

	if len(hexAddr) != 42 { // "41" + 40 hex chars
		return nil, fmt.Errorf("invalid hex address length: expected 42, got %d", len(hexAddr))
	}

	// Decode hex string
	decoded, err := hex.DecodeString(hexAddr)
	if err != nil {
		return nil, fmt.Errorf("invalid hex encoding: %w", err)
	}

	return &Address{
		bytesAddr: decoded,
		validated: true,
	}, nil
}

// NewAddressFromEVMHex creates an Address from 20 bytes HEX, optionally prefixed with 0x
func NewAddressFromEVMHex(hexAddr string) (*Address, error) {
	// Remove 0x prefix if present
	hexAddr = strings.TrimPrefix(strings.ToLower(hexAddr), "0x")

	if len(hexAddr) != 40 { // 40 hex chars
		return nil, fmt.Errorf("invalid hex address length: expected 40, got %d", len(hexAddr))
	}

	hexAddr = AddressPrefix + hexAddr // Add TRON prefix

	// Decode hex string
	decoded, err := hex.DecodeString(hexAddr)
	if err != nil {
		return nil, fmt.Errorf("invalid hex encoding: %w", err)
	}

	return &Address{
		bytesAddr: decoded,
		validated: true,
	}, nil
}

// NewAddressFromBytes creates an Address from bytes
func NewAddressFromBytes(byteAddress []byte) (*Address, error) {
	if len(byteAddress) != AddressLength {
		return nil, fmt.Errorf("invalid address length: expected %d, got %d", AddressLength, len(byteAddress))
	}

	if byteAddress[0] != 0x41 {
		return nil, fmt.Errorf("invalid address prefix: expected 0x41, got 0x%x", byteAddress[0])
	}

	return &Address{
		bytesAddr: byteAddress,
		validated: true,
	}, nil
}

// MustNewAddress is a wrapper for NewAddress that panics if the address is invalid
func MustNewAddress(base58Addr string) *Address {
	addr, err := NewAddress(base58Addr)
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

// MustNewAddressFromEVMHex is a wrapper for NewAddressFromEVMHex that panics if the address is invalid
func MustNewAddressFromEVMHex(hexAddr string) *Address {
	addr, err := NewAddressFromEVMHex(hexAddr)
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

// GetBase58Addr returns the base58 representation of the address
func (a *Address) GetBase58Addr() (string, error) {
	if !a.validated {
		return "", errors.New("address must be created using NewAddress* methods")
	}

	if a.base58Addr != "" {
		return a.base58Addr, nil
	}

	// Convert from bytes
	if a.bytesAddr == nil {
		return "", errors.New("address not initialized")
	}

	// Calculate checksum
	h1 := sha256.Sum256(a.bytesAddr)
	h2 := sha256.Sum256(h1[:])
	checksum := h2[:4]

	// Combine address with checksum
	combined := append([]byte{}, a.bytesAddr...)
	combined = append(combined, checksum...)

	// Encode to base58
	a.base58Addr = base58.Encode(combined)
	return a.base58Addr, nil
}

// GetBytes returns the raw bytes of the address
func (a *Address) GetBytes() ([]byte, error) {
	if !a.validated {
		return nil, errors.New("address must be created using NewAddress* methods")
	}

	if a.bytesAddr != nil {
		return a.bytesAddr, nil
	}

	// Convert from base58
	if a.base58Addr == "" {
		return nil, errors.New("address not initialized")
	}

	decoded, err := base58.Decode(a.base58Addr)
	if err != nil {
		return nil, fmt.Errorf("invalid base58 encoding: %w", err)
	}

	// Take only the address part without checksum
	if len(decoded) < AddressLength {
		return nil, errors.New("invalid decoded address length")
	}

	a.bytesAddr = decoded[:AddressLength]
	return a.bytesAddr, nil
}

// GetHex returns the hex string representation with 0x prefix
func (a *Address) GetHex() (string, error) {
	if !a.validated {
		return "", errors.New("address must be created using NewAddress* methods")
	}
	bytes, err := a.GetBytes()
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// String returns the base58 representation
func (a *Address) String() string {
	if !a.validated {
		return "<address must be created using NewAddress* methods>"
	}
	addr, err := a.GetBase58Addr()
	if err != nil {
		return "<invalid address>"
	}
	return addr
}

// Bytes returns the raw bytes of the address
func (a *Address) Bytes() []byte {
	if !a.validated {
		return nil
	}
	bytes, err := a.GetBytes()
	if err != nil {
		return nil
	}
	return bytes
}

// Hex returns the hex string representation with 0x prefix
func (a *Address) Hex() string {
	if !a.validated {
		return "<address must be created using NewAddress* methods>"
	}
	hex, err := a.GetHex()
	if err != nil {
		return "<invalid address>"
	}
	return hex
}
