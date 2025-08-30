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

package utils

import (
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"

	eABI "github.com/ethereum/go-ethereum/accounts/abi"
	eCommon "github.com/ethereum/go-ethereum/common"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
	"golang.org/x/crypto/sha3"
)

// DecodeInputData decodes call data into a DecodedInput using the provided ABI.
func (p *ABIProcessor) DecodeInputData(data []byte, abi *core.SmartContract_ABI) (*DecodedInput, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("input data too short, need at least 4 bytes for method signature")
	}

	// Extract method signature (first 4 bytes)
	methodSig := data[:4]

	// Find matching method in ABI
	var matchedEntry *core.SmartContract_ABI_Entry
	var methodSignature string

	for _, entry := range abi.Entrys {
		if entry.Type != core.SmartContract_ABI_Entry_Function {
			continue
		}

		// Build method signature string
		inputs := make([]string, len(entry.Inputs))
		for i, input := range entry.Inputs {
			inputs[i] = input.Type
		}
		methodSigStr := fmt.Sprintf("%s(%s)", entry.Name, strings.Join(inputs, ","))

		// Calculate method ID
		hasher := sha3.NewLegacyKeccak256()
		hasher.Write([]byte(methodSigStr))
		calculatedSig := hasher.Sum(nil)[:4]

		// Compare signatures
		if hex.EncodeToString(calculatedSig) == hex.EncodeToString(methodSig) {
			matchedEntry = entry
			methodSignature = methodSigStr
			break
		}
	}

	if matchedEntry == nil {
		return &DecodedInput{
			Method:     fmt.Sprintf("unknown(0x%s)", hex.EncodeToString(methodSig)),
			Parameters: []DecodedInputParameter{},
		}, nil
	}

	// If no parameters, return method signature only
	if len(matchedEntry.Inputs) == 0 {
		return &DecodedInput{
			Method:     methodSignature,
			Parameters: []DecodedInputParameter{},
		}, nil
	}

	// Decode parameters
	paramData := data[4:]
	parameters, err := p.decodeParameters(paramData, matchedEntry.Inputs)
	if err != nil {
		return nil, fmt.Errorf("failed to decode parameters: %v", err)
	}

	return &DecodedInput{
		Method:     methodSignature,
		Parameters: parameters,
	}, nil
}

// decodeParameters decodes function parameters (internal method)
func (p *ABIProcessor) decodeParameters(data []byte, inputs []*core.SmartContract_ABI_Entry_Param) ([]DecodedInputParameter, error) {
	if len(inputs) == 0 {
		return []DecodedInputParameter{}, nil
	}

	// Create ethereum ABI arguments for decoding
	args := make([]eABI.Argument, len(inputs))
	for i, input := range inputs {
		abiType, err := eABI.NewType(input.Type, "", nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create ABI type for %s: %v", input.Type, err)
		}
		args[i] = eABI.Argument{
			Name: input.Name,
			Type: abiType,
		}
	}

	// Unpack the parameters
	values, err := eABI.Arguments(args).Unpack(data)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack parameters: %v", err)
	}

	// Build decoded parameters
	parameters := make([]DecodedInputParameter, len(inputs))
	for i, input := range inputs {
		var value interface{}
		if i < len(values) {
			value = p.formatDecodedValue(values[i], input.Type)
		}

		parameters[i] = DecodedInputParameter{
			Name:  input.Name,
			Type:  input.Type,
			Value: value,
		}
	}

	return parameters, nil
}

// DecodeResult decodes method return bytes. Behavior:
//   - no outputs: returns nil
//   - one output: returns the value directly
//   - many outputs: returns []interface{}
func (p *ABIProcessor) DecodeResult(data []byte, outputs []*core.SmartContract_ABI_Entry_Param) (interface{}, error) {
	if len(outputs) == 0 {
		return nil, nil
	}

	// Create ethereum ABI arguments for decoding
	args := make([]eABI.Argument, len(outputs))
	for i, output := range outputs {
		abiType, err := eABI.NewType(output.Type, "", nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create ABI type for %s: %v", output.Type, err)
		}
		args[i] = eABI.Argument{
			Name: output.Name,
			Type: abiType,
		}
	}

	// Unpack the values
	values, err := eABI.Arguments(args).Unpack(data)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack result: %v", err)
	}

	// Single output: return the single decoded value directly
	if len(outputs) == 1 {
		if len(values) == 0 {
			return nil, nil
		}
		return p.formatDecodedValue(values[0], outputs[0].Type), nil
	}

	// Multiple outputs: return as []interface{} (still satisfies interface{})
	result := make([]interface{}, len(outputs))
	for i, output := range outputs {
		if i < len(values) {
			result[i] = p.formatDecodedValue(values[i], output.Type)
		}
	}
	return result, nil
}

// decodeSingleValue decodes a single return value

// formatDecodedValue formats decoded value based on type
func (p *ABIProcessor) formatDecodedValue(value interface{}, paramType string) interface{} {
	switch paramType {
	case "address":
		if addr, ok := value.(eCommon.Address); ok {
			// Convert to TRON address format
			tronAddr, err := types.NewAddressFromEVM(addr)
			if err != nil {
				// If conversion fails, return the original Ethereum address
				// This could happen if the address is not a valid TRON address format
				return value
			}
			return tronAddr
		}
		return value

	case "bytes", "bytes32", "bytes16", "bytes8":
		// For bytes types, return []byte directly
		if bytes, ok := value.([]byte); ok {
			return bytes
		}
		return value

	case "string":
		return value

	case "bool":
		return value

	default:
		// Handle array types
		if strings.HasSuffix(paramType, "[]") {
			baseType := strings.TrimSuffix(paramType, "[]")
			if reflect.TypeOf(value).Kind() == reflect.Slice {
				slice := reflect.ValueOf(value)
				result := make([]interface{}, slice.Len())
				for i := 0; i < slice.Len(); i++ {
					result[i] = p.formatDecodedValue(slice.Index(i).Interface(), baseType)
				}
				return result
			}
		}
		return value
	}
}
