// Copyright (c) 2025 github.com/kslamph
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

// Package types provides shared types and utilities for the TRON SDK
package types

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	eCommon "github.com/ethereum/go-ethereum/common"
	"github.com/mr-tron/base58"
)

const (
	BlackHoleAddress = "T9yD14Nj9j7xAB4dbGeiX9h8unkKHxuWwb" // Black hole address prefix
)

// Address represents a TRON address that can be stored in different formats.
// Always construct via the NewAddress[...] helpers to ensure validation.
type Address struct {
	base58Addr string //T prefixed 34 chars base58 representation
	bytesAddr  []byte // 0x41 prefixed 21 bytes address
}

// NewAddress creates an Address from a string, []byte, or base58 string
// it will try to parse the address as base58 first, then hex, then bytes
// performance penalty is expected
type addressAllowed interface {
	~string | ~[]byte | *Address | *eCommon.Address | [20]byte | [21]byte
}

func NewAddress[T addressAllowed](v T) (*Address, error) {
	switch any(v).(type) {
	case string:
		s := any(v).(string)
		addr, err := NewAddressFromBase58(s)
		if err == nil {
			return addr, nil
		}

		return NewAddressFromHex(s)

	case []byte:
		b := any(v).([]byte)
		return NewAddressFromBytes(b)

	case *Address:
		a := any(v).(*Address)
		if a == nil {
			return nil, fmt.Errorf("invalid address: nil Address")
		}
		return a, nil

	case *eCommon.Address:
		ea := any(v).(eCommon.Address)
		return NewAddressFromBytes(ea.Bytes())

	case [20]byte:
		b := any(v).([20]byte)
		return NewAddressFromBytes(b[:])

	case [21]byte:
		b := any(v).([21]byte)
		return NewAddressFromBytes(b[:])

	default:
		return nil, fmt.Errorf("invalid address: %v", v)
	}
}

// NewAddressFromBase58 creates an Address from a Base58Check string.
// The string must be length 34, T-prefixed.
func NewAddressFromBase58(base58Addr string) (*Address, error) {
	// Address must start with T
	if !strings.HasPrefix(base58Addr, "T") {
		return nil, fmt.Errorf("invalid address: must start with T")
	}
	// Address must be 34 chars long
	if len(base58Addr) != AddressBase58Length {
		return nil, fmt.Errorf("invalid address length: expected %d, got %d", AddressBase58Length, len(base58Addr))
	}

	decoded, err := base58.Decode(base58Addr)
	// Address must be valid base58
	if err != nil {
		return nil, fmt.Errorf("invalid base58 encoding: %w", err)
	}
	// Address hex must be prefixed with AddressPrefixByte
	if decoded[0] != AddressPrefixByte {
		return nil, fmt.Errorf("invalid address prefix: expected 0x%x, got 0x%x", AddressPrefixByte, decoded[0])
	}
	// Address hex must be 21 bytes long
	if len(decoded) != AddressLength+4 { // 21 bytes address + 4 bytes checksum
		return nil, errors.New("invalid decoded address length")
	}

	addressBytes := decoded[:AddressLength]
	checksum := decoded[AddressLength:]

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

// NewAddressFromHex creates an Address from a hex string. Supported forms:
//   - 0x41-prefixed 21-byte TRON hex
//   - 41-prefixed 21-byte TRON hex (without 0x)
//   - 20-byte hex (0x-optional) which will be promoted by adding 0x41 prefix
func NewAddressFromHex(hexAddr string) (*Address, error) {
	// Remove 0x prefix if present
	hexAddr = strings.TrimPrefix(strings.ToLower(hexAddr), "0x")
	decoded, err := hex.DecodeString(hexAddr)

	if err != nil {
		return nil, fmt.Errorf("invalid hex encoding: %w", err)
	}

	switch len(decoded) {
	case 21:
		if decoded[0] != AddressPrefixByte {
			return nil, fmt.Errorf("invalid hex address: must start with %x", AddressPrefixByte)
		}
	case 20:
		// Valid address - add prefix
		prefixed := make([]byte, 21)
		prefixed[0] = AddressPrefixByte
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

// NewAddressFromBytes creates an Address from bytes. Supported lengths:
//   - 21 bytes (0x41-prefixed TRON address)
//   - 20 bytes (EVM address), which will be promoted by adding 0x41 prefix
func NewAddressFromBytes(byteAddress []byte) (*Address, error) {
	switch len(byteAddress) {
	case 21:
		if byteAddress[0] != AddressPrefixByte {
			return nil, fmt.Errorf("invalid address prefix: expected 0x%x, got 0x%x", AddressPrefixByte, byteAddress[0])
		}
	case 20:
		// Valid address - add prefix
		prefixed := make([]byte, 21)
		prefixed[0] = AddressPrefixByte
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

// BytesEVM returns the raw bytes of the address (20 bytes without prefix)
func (a *Address) BytesEVM() []byte {
	if a == nil {
		return nil
	}
	return a.bytesAddr[1:]
}

// Hex returns the address as 41-prefixed, 42-character hex string.
func (a *Address) Hex() string {
	if a == nil {
		return ""
	}
	return hex.EncodeToString(a.bytesAddr)
}

// HexEVM returns the EVM-style 0x-prefixed, 40-character hex string.
func (a *Address) HexEVM() string {
	if a == nil {
		return ""
	}
	return "0x" + hex.EncodeToString(a.bytesAddr[1:])
}

// IsValid checks if the address is valid
func (a *Address) IsValid() bool {
	return a != nil && len(a.bytesAddr) == AddressLength && a.bytesAddr[0] == AddressPrefixByte &&
		len(a.base58Addr) == AddressBase58Length && strings.HasPrefix(a.base58Addr, "T")
}

// Equal checks if two addresses are equal
func (a *Address) Equal(other *Address) bool {
	if a == nil || other == nil {
		return a == other
	}
	return bytes.Equal(a.bytesAddr, other.bytesAddr)
}

// EVMAddress converts the TRON address to an Ethereum compatible address
// It panics if the address is nil
func (a *Address) EVMAddress() eCommon.Address {
	if a == nil {
		panic("nil Address cannot be converted to EVM address")
	}
	return eCommon.BytesToAddress(a.BytesEVM())
}

func NewAddressFromEVM(evmAddr eCommon.Address) (*Address, error) {

	return NewAddressFromBytes(evmAddr.Bytes())
	// Convert EVM address to TRON address format

}
