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

// Package utils provides encoding and decoding utilities for the TRON SDK
package utils

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	types "github.com/kslamph/tronlib/pkg/types"
)

// HexToBytes converts hex string to bytes
func HexToBytes(hexStr string) ([]byte, error) {
	// Remove 0x prefix if present
	hexStr = strings.TrimPrefix(hexStr, "0x")

	// Ensure even length
	if len(hexStr)%2 != 0 {
		hexStr = "0" + hexStr
	}

	return hex.DecodeString(hexStr)
}

// BytesToHex converts bytes to hex string with 0x prefix
func BytesToHex(data []byte) string {
	return "0x" + hex.EncodeToString(data)
}

// PadLeft pads data to the left with zeros to reach the specified length
func PadLeft(data []byte, length int) []byte {
	if len(data) >= length {
		return data
	}

	padded := make([]byte, length)
	copy(padded[length-len(data):], data)
	return padded
}

// PadRight pads data to the right with zeros to reach the specified length
func PadRight(data []byte, length int) []byte {
	if len(data) >= length {
		return data
	}

	padded := make([]byte, length)
	copy(padded, data)
	return padded
}

// EncodeMethodSignature encodes a method signature for smart contract calls
func EncodeMethodSignature(method string) []byte {
	// This is a simplified implementation
	// In a real implementation, you would use keccak256 hash
	methodBytes := []byte(method)
	if len(methodBytes) >= 4 {
		return methodBytes[:4]
	}
	return PadRight(methodBytes, 4)
}

// EncodeParameters encodes parameters for smart contract calls using ABI
func EncodeParameters(abiJSON string, method string, params ...interface{}) ([]byte, error) {
	// Parse ABI
	contractABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %v", err)
	}

	// Find method
	methodABI, exists := contractABI.Methods[method]
	if !exists {
		return nil, fmt.Errorf("method %s not found in ABI", method)
	}

	// Encode parameters
	data, err := methodABI.Inputs.Pack(params...)
	if err != nil {
		return nil, fmt.Errorf("failed to encode parameters: %v", err)
	}

	// Prepend method signature
	methodSig := methodABI.ID
	result := make([]byte, 4+len(data))
	copy(result[:4], methodSig)
	copy(result[4:], data)

	return result, nil
}

// DecodeParameters decodes parameters from smart contract call result using ABI
func DecodeParameters(abiJSON string, method string, data []byte) ([]interface{}, error) {
	// Parse ABI
	contractABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %v", err)
	}

	// Find method
	methodABI, exists := contractABI.Methods[method]
	if !exists {
		return nil, fmt.Errorf("method %s not found in ABI", method)
	}

	// Decode parameters
	results, err := methodABI.Outputs.Unpack(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode parameters: %v", err)
	}

	return results, nil
}

// EncodeTRC20Transfer encodes a TRC20 transfer call
func EncodeTRC20Transfer(to string, amount *big.Int) ([]byte, error) {
	// Method signature for transfer(address,uint256)
	methodSig, _ := HexToBytes(types.TRC20TransferMethodID)

	// Encode address (32 bytes, left-padded)
	toAddr, err := HexToBytes(to)
	if err != nil {
		return nil, fmt.Errorf("invalid to address: %v", err)
	}
	encodedTo := PadLeft(toAddr, 32)

	// Encode amount (32 bytes, left-padded)
	amountBytes := amount.Bytes()
	encodedAmount := PadLeft(amountBytes, 32)

	// Combine all parts
	result := make([]byte, 4+32+32)
	copy(result[:4], methodSig)
	copy(result[4:36], encodedTo)
	copy(result[36:68], encodedAmount)

	return result, nil
}

// EncodeTRC20BalanceOf encodes a TRC20 balanceOf call
func EncodeTRC20BalanceOf(address string) ([]byte, error) {
	// Method signature for balanceOf(address)
	methodSig, _ := HexToBytes(types.TRC20BalanceOfMethodID)

	// Encode address (32 bytes, left-padded)
	addr, err := HexToBytes(address)
	if err != nil {
		return nil, fmt.Errorf("invalid address: %v", err)
	}
	encodedAddr := PadLeft(addr, 32)

	// Combine method signature and address
	result := make([]byte, 4+32)
	copy(result[:4], methodSig)
	copy(result[4:36], encodedAddr)

	return result, nil
}

// DecodeTRC20Balance decodes a TRC20 balance result
func DecodeTRC20Balance(data []byte) (*big.Int, error) {
	if len(data) < 32 {
		return nil, fmt.Errorf("invalid balance data length: %d", len(data))
	}

	// Balance is encoded as uint256 (32 bytes)
	balance := new(big.Int).SetBytes(data[:32])
	return balance, nil
}

// EncodeUint256 encodes a big integer as uint256 (32 bytes)
func EncodeUint256(value *big.Int) []byte {
	return PadLeft(value.Bytes(), 32)
}

// DecodeUint256 decodes uint256 (32 bytes) to big integer
func DecodeUint256(data []byte) *big.Int {
	if len(data) < 32 {
		// Pad with zeros if data is shorter
		padded := make([]byte, 32)
		copy(padded[32-len(data):], data)
		data = padded
	}
	return new(big.Int).SetBytes(data[:32])
}

// EncodeAddress encodes an address as bytes32
func EncodeAddress(address string) ([]byte, error) {
	addr, err := HexToBytes(address)
	if err != nil {
		return nil, err
	}
	return PadLeft(addr, 32), nil
}

// DecodeAddress decodes bytes32 to address
func DecodeAddress(data []byte) string {
	if len(data) < 32 {
		return ""
	}
	// Take last 20 bytes for address
	addr := data[12:32]
	return BytesToHex(addr)
}

// EncodeString encodes a string for smart contract calls
func EncodeString(str string) []byte {
	data := []byte(str)
	// String encoding: offset + length + data (padded to 32-byte boundary)
	length := len(data)

	// Calculate total size: 32 (offset) + 32 (length) + padded data
	paddedLength := ((length + 31) / 32) * 32
	result := make([]byte, 32+32+paddedLength)

	// Offset (pointing to length field)
	offset := EncodeUint256(big.NewInt(32))
	copy(result[:32], offset)

	// Length
	lengthBytes := EncodeUint256(big.NewInt(int64(length)))
	copy(result[32:64], lengthBytes)

	// Data (padded)
	copy(result[64:64+length], data)

	return result
}

// DecodeString decodes a string from smart contract result
func DecodeString(data []byte) (string, error) {
	if len(data) < 64 {
		return "", fmt.Errorf("insufficient data for string decoding")
	}

	// Read offset (should be 32 for standard encoding)
	offset := DecodeUint256(data[:32]).Int64()
	if offset != 32 {
		return "", fmt.Errorf("unexpected string offset: %d", offset)
	}

	// Read length
	length := DecodeUint256(data[32:64]).Int64()
	if length < 0 || int64(len(data)) < 64+length {
		return "", fmt.Errorf("invalid string length: %d", length)
	}

	// Read string data
	stringData := data[64 : 64+length]
	return string(stringData), nil
}

// JSONToMap converts JSON string to map
func JSONToMap(jsonStr string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &result)
	return result, err
}

// MapToJSON converts map to JSON string
func MapToJSON(data map[string]interface{}) (string, error) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// ParseBigInt parses a string to big.Int with support for different formats
func ParseBigInt(str string) (*big.Int, error) {
	// Remove any whitespace
	str = strings.TrimSpace(str)

	// Handle hex format
	if strings.HasPrefix(str, "0x") {
		result, ok := new(big.Int).SetString(str[2:], 16)
		if !ok {
			return nil, fmt.Errorf("invalid hex big integer format: %s", str)
		}
		return result, nil
	}

	// Handle decimal format
	result, ok := new(big.Int).SetString(str, 10)
	if !ok {
		return nil, fmt.Errorf("invalid big integer format: %s", str)
	}

	return result, nil
}

// FormatBigInt formats a big.Int to string with optional decimal places
func FormatBigInt(value *big.Int, decimals int) string {
	if decimals <= 0 {
		return value.String()
	}

	// Convert to float for decimal formatting
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	quotient := new(big.Int).Div(value, divisor)
	remainder := new(big.Int).Mod(value, divisor)

	if remainder.Sign() == 0 {
		return quotient.String()
	}

	// Format with decimals
	remainderStr := remainder.String()
	// Pad with leading zeros if necessary
	for len(remainderStr) < decimals {
		remainderStr = "0" + remainderStr
	}

	// Remove trailing zeros
	remainderStr = strings.TrimRight(remainderStr, "0")
	if remainderStr == "" {
		return quotient.String()
	}

	return quotient.String() + "." + remainderStr
}
