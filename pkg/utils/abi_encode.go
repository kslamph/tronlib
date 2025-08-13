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
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	eABI "github.com/ethereum/go-ethereum/accounts/abi"
	eCommon "github.com/ethereum/go-ethereum/common"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
	"golang.org/x/crypto/sha3"
)

// GetMethodTypes returns input and output type names for the given method.
func (p *ABIProcessor) GetMethodTypes(methodName string) ([]string, []string, error) {
	for _, entry := range p.abi.Entrys {
		if entry.Name == methodName && entry.Type == core.SmartContract_ABI_Entry_Function {
			inputTypes := make([]string, len(entry.Inputs))
			for i, input := range entry.Inputs {
				inputTypes[i] = input.Type
			}

			outputTypes := make([]string, len(entry.Outputs))
			for i, output := range entry.Outputs {
				outputTypes[i] = output.Type
			}

			return inputTypes, outputTypes, nil
		}
	}
	return nil, nil, fmt.Errorf("method %s not found", methodName)
}

// GetConstructorTypes returns the constructor input type names.
func (p *ABIProcessor) GetConstructorTypes(abi *core.SmartContract_ABI) ([]string, error) {
	for _, entry := range abi.Entrys {
		if entry.Type == core.SmartContract_ABI_Entry_Constructor {
			inputTypes := make([]string, len(entry.Inputs))
			for i, input := range entry.Inputs {
				inputTypes[i] = input.Type
			}
			return inputTypes, nil
		}
	}
	return nil, fmt.Errorf("constructor not found")
}

// EncodeMethod encodes a method call with parameters. For constructors, pass
// method="" to encode only parameters (no 4-byte method ID).
func (p *ABIProcessor) EncodeMethod(method string, paramTypes []string, params []interface{}) ([]byte, error) {
	// For constructors (empty method name), encode parameters without method ID
	if method == "" {
		if len(params) == 0 {
			return []byte{}, nil
		}
		// Encode parameters only (no method ID for constructors)
		return p.encodeParameters(paramTypes, params)
	}

	// Create method signature for regular methods
	methodSig := fmt.Sprintf("%s(%s)", method, strings.Join(paramTypes, ","))

	// Get method ID (first 4 bytes of keccak256 hash)
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write([]byte(methodSig))
	methodID := hasher.Sum(nil)[:4]

	if len(params) == 0 {
		return methodID, nil
	}

	// Encode parameters
	encoded, err := p.encodeParameters(paramTypes, params)
	if err != nil {
		return nil, fmt.Errorf("failed to encode parameters: %v", err)
	}

	return append(methodID, encoded...), nil
}

// encodeParameters encodes function parameters (internal method)
func (p *ABIProcessor) encodeParameters(paramTypes []string, params []interface{}) ([]byte, error) {
	if len(paramTypes) != len(params) {
		return nil, fmt.Errorf("parameter count mismatch: expected %d, got %d", len(paramTypes), len(params))
	}

	// Create ethereum ABI arguments
	args := make([]eABI.Argument, len(paramTypes))
	values := make([]interface{}, len(params))

	for i, paramType := range paramTypes {
		abiType, err := eABI.NewType(paramType, "", nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create ABI type for %s: %v", paramType, err)
		}
		args[i] = eABI.Argument{Type: abiType}

		// Convert parameter to appropriate type
		convertedValue, err := p.convertParameter(params[i], paramType)
		if err != nil {
			return nil, fmt.Errorf("failed to convert parameter %d: %v", i, err)
		}
		values[i] = convertedValue
	}

	// Pack the arguments
	packed, err := eABI.Arguments(args).Pack(values...)
	if err != nil {
		return nil, fmt.Errorf("failed to pack parameters: %v", err)
	}

	return packed, nil
}

// convertParameter converts a parameter to the appropriate type for ABI encoding
func (p *ABIProcessor) convertParameter(param interface{}, paramType string) (interface{}, error) {
	if param == nil {
		return nil, fmt.Errorf("nil parameter not allowed")
	}

	// Handle array types
	if strings.HasSuffix(paramType, "[]") {
		baseType := strings.TrimSuffix(paramType, "[]")
		// TODO: Implement full tuple support for array types
		// For now, we handle arrays of basic types
		return p.convertArrayParameter(param, baseType)
	}

	// Handle scalar types
	// Leverage go-ethereum/accounts/abi's native ability to handle various Go integer types
	// Directly pass the provided interface{} parameter for number types to eABI.Arguments.Pack
	switch paramType {
	case "address":
		return p.convertAddress(param)
	case "bool":
		return p.convertBool(param)
	case "string":
		return p.convertString(param)
	case "bytes":
		return p.convertBytes(param, 0) // 0 indicates dynamic bytes
	case "bytes32":
		return p.convertBytes(param, 32) // 32 indicates fixed-size 32 bytes
	case "bytes16":
		return p.convertBytes(param, 16) // 16 indicates fixed-size 16 bytes
	case "bytes8":
		return p.convertBytes(param, 8) // 8 indicates fixed-size 8 bytes
	default:
		// For integer types and other types, pass the parameter directly
		// go-ethereum/accounts/abi will handle the conversion and validation
		return param, nil
	}
}

// convertAddress converts address parameter
func (p *ABIProcessor) convertAddress(param interface{}) (eCommon.Address, error) {
	var decoded []byte
	switch v := param.(type) {
	case string:
		addr, err := types.NewAddress(v)
		if err != nil {
			return eCommon.Address{}, fmt.Errorf("invalid address string: %v", err)
		}
		decoded = addr.BytesEVM()
	case []byte:
		addr, err := types.NewAddressFromBytes(v)
		if err != nil {
			return eCommon.Address{}, fmt.Errorf("invalid address bytes: %v", err)
		}
		decoded = addr.BytesEVM()
	case eCommon.Address:
		return v, nil
	case types.Address:
		decoded = v.BytesEVM()
	case *types.Address:
		if v == nil {
			return eCommon.Address{}, fmt.Errorf("nil Address cannot be converted to EVM address")
		}
		decoded = v.BytesEVM()
	default:
		return eCommon.Address{}, fmt.Errorf("invalid address type: %T", param)
	}

	return eCommon.BytesToAddress(decoded), nil
}

// convertBool converts boolean parameter
func (p *ABIProcessor) convertBool(param interface{}) (bool, error) {
	if b, ok := param.(bool); ok {
		return b, nil
	}
	return false, fmt.Errorf("bool parameter must be a boolean")
}

// convertString converts string parameter
func (p *ABIProcessor) convertString(param interface{}) (string, error) {
	if s, ok := param.(string); ok {
		return s, nil
	}
	return "", fmt.Errorf("string parameter must be a string")
}

// convertBytes converts bytes parameter
func (p *ABIProcessor) convertBytes(param interface{}, fixedSize int) (interface{}, error) {
	switch v := param.(type) {
	case string:
		var data []byte
		var err error
		if strings.HasPrefix(v, "0x") {
			data = eCommon.FromHex(v)
		} else {
			data, err = hex.DecodeString(v)
			if err != nil {
				return nil, err
			}
		}

		// If fixedSize is specified, convert to fixed-size array
		if fixedSize > 0 {
			if len(data) != fixedSize {
				return nil, fmt.Errorf("bytes length mismatch: expected %d, got %d", fixedSize, len(data))
			}
			// Create fixed-size array
			switch fixedSize {
			case 32:
				var array [32]byte
				copy(array[:], data)
				return array, nil
			case 16:
				var array [16]byte
				copy(array[:], data)
				return array, nil
			case 8:
				var array [8]byte
				copy(array[:], data)
				return array, nil
			default:
				// For other sizes, create generic array
				arrayValue := reflect.New(reflect.ArrayOf(fixedSize, reflect.TypeOf(byte(0)))).Elem()
				for i := 0; i < len(data) && i < fixedSize; i++ {
					arrayValue.Index(i).Set(reflect.ValueOf(data[i]))
				}
				return arrayValue.Interface(), nil
			}
		}
		return data, nil
	case []byte:
		// If fixedSize is specified, convert to fixed-size array
		if fixedSize > 0 {
			if len(v) != fixedSize {
				return nil, fmt.Errorf("bytes length mismatch: expected %d, got %d", fixedSize, len(v))
			}
			// Create fixed-size array
			switch fixedSize {
			case 32:
				var array [32]byte
				copy(array[:], v)
				return array, nil
			case 16:
				var array [16]byte
				copy(array[:], v)
				return array, nil
			case 8:
				var array [8]byte
				copy(array[:], v)
				return array, nil
			default:
				// For other sizes, create generic array
				arrayValue := reflect.New(reflect.ArrayOf(fixedSize, reflect.TypeOf(byte(0)))).Elem()
				for i := 0; i < len(v) && i < fixedSize; i++ {
					arrayValue.Index(i).Set(reflect.ValueOf(v[i]))
				}
				return arrayValue.Interface(), nil
			}
		}
		return v, nil
	default:
		// Handle fixed-size byte arrays (e.g., [32]byte)
		// Check if it's an array of bytes
		val := reflect.ValueOf(param)
		if val.Kind() == reflect.Array && val.Type().Elem().Kind() == reflect.Uint8 {
			// For fixed-size byte arrays, return the original value
			// The go-ethereum ABI package expects the original fixed-size array type
			return param, nil
		}
		return nil, fmt.Errorf("bytes parameter must be string, []byte, or [N]byte")
	}
}

// convertArrayParameter converts array parameter
func (p *ABIProcessor) convertArrayParameter(param interface{}, baseType string) (interface{}, error) {
	// Handle JSON string arrays
	if jsonStr, ok := param.(string); ok {
		var jsonArray []interface{}
		if err := json.Unmarshal([]byte(jsonStr), &jsonArray); err != nil {
			return nil, fmt.Errorf("failed to parse array JSON: %v", err)
		}
		return p.convertArrayElements(jsonArray, baseType)
	}

	// Handle slice directly
	if reflect.TypeOf(param).Kind() == reflect.Slice {
		// For integer types, return the slice directly
		// The go-ethereum ABI package expects the actual slice type
		switch baseType {
		case "uint256", "uint128", "uint64", "uint32", "uint16", "uint8",
			"int256", "int128", "int64", "int32", "int16", "int8":
			return param, nil
		default:
			// For other types, convert elements individually
			slice := reflect.ValueOf(param)
			elements := make([]interface{}, slice.Len())
			for i := 0; i < slice.Len(); i++ {
				elements[i] = slice.Index(i).Interface()
			}
			return p.convertArrayElements(elements, baseType)
		}
	}

	return nil, fmt.Errorf("array parameter must be JSON string or slice")
}

// convertArrayElements converts array elements to appropriate types
func (p *ABIProcessor) convertArrayElements(elements []interface{}, baseType string) (interface{}, error) {
	switch baseType {
	case "address":
		addresses := make([]eCommon.Address, len(elements))
		for i, elem := range elements {
			addr, err := p.convertAddress(elem)
			if err != nil {
				return nil, fmt.Errorf("invalid address at index %d: %v", i, err)
			}
			addresses[i] = addr
		}
		return addresses, nil

	case "uint256", "uint128", "uint64", "uint32", "uint16", "uint8":
		// For integer types, return the elements directly
		// go-ethereum/accounts/abi expects the actual slice of values
		return elements, nil

	case "int256", "int128", "int64", "int32", "int16", "int8":
		// For integer types, return the elements directly
		// go-ethereum/accounts/abi expects the actual slice of values
		return elements, nil

	case "bool":
		bools := make([]bool, len(elements))
		for i, elem := range elements {
			b, err := p.convertBool(elem)
			if err != nil {
				return nil, fmt.Errorf("invalid bool at index %d: %v", i, err)
			}
			bools[i] = b
		}
		return bools, nil

	case "string":
		strings := make([]string, len(elements))
		for i, elem := range elements {
			s, err := p.convertString(elem)
			if err != nil {
				return nil, fmt.Errorf("invalid string at index %d: %v", i, err)
			}
			strings[i] = s
		}
		return strings, nil

	default:
		return nil, fmt.Errorf("unsupported array element type: %s", baseType)
	}
}
