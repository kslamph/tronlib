package utils

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"reflect"
	"strings"

	eABI "github.com/ethereum/go-ethereum/accounts/abi"
	eCommon "github.com/ethereum/go-ethereum/common"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
	"golang.org/x/crypto/sha3"
)

// ABIDecoder handles smart contract ABI decoding operations
type ABIDecoder struct{}

// NewABIDecoder creates a new ABI decoder instance
func NewABIDecoder() *ABIDecoder {
	return &ABIDecoder{}
}

// DecodedInput represents decoded input data
type DecodedInput struct {
	Method     string                  `json:"method"`
	Parameters []DecodedInputParameter `json:"parameters"`
}

// DecodedInputParameter represents a decoded parameter
type DecodedInputParameter struct {
	Name  string      `json:"name"`
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

// DecodeInputData decodes contract input data
func (d *ABIDecoder) DecodeInputData(data []byte, abi *core.SmartContract_ABI) (*DecodedInput, error) {
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
	parameters, err := d.DecodeParameters(paramData, matchedEntry.Inputs)
	if err != nil {
		return nil, fmt.Errorf("failed to decode parameters: %v", err)
	}

	return &DecodedInput{
		Method:     methodSignature,
		Parameters: parameters,
	}, nil
}

// DecodeParameters decodes function parameters
func (d *ABIDecoder) DecodeParameters(data []byte, inputs []*core.SmartContract_ABI_Entry_Param) ([]DecodedInputParameter, error) {
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
			value = d.formatDecodedValue(values[i], input.Type)
		}

		parameters[i] = DecodedInputParameter{
			Name:  input.Name,
			Type:  input.Type,
			Value: value,
		}
	}

	return parameters, nil
}

// DecodeResult decodes contract call result
func (d *ABIDecoder) DecodeResult(data []byte, outputs []*core.SmartContract_ABI_Entry_Param) (interface{}, error) {
	if len(outputs) == 0 {
		return nil, nil
	}

	// For single output, return the value directly
	if len(outputs) == 1 {
		return d.decodeSingleValue(data, outputs[0])
	}

	// For multiple outputs, return a map
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

	values, err := eABI.Arguments(args).Unpack(data)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack result: %v", err)
	}

	result := make(map[string]interface{})
	for i, output := range outputs {
		if i < len(values) {
			result[output.Name] = d.formatDecodedValue(values[i], output.Type)
		}
	}

	return result, nil
}

// decodeSingleValue decodes a single return value
func (d *ABIDecoder) decodeSingleValue(data []byte, output *core.SmartContract_ABI_Entry_Param) (interface{}, error) {
	abiType, err := eABI.NewType(output.Type, "", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create ABI type for %s: %v", output.Type, err)
	}

	arg := eABI.Argument{Type: abiType}
	values, err := eABI.Arguments{arg}.Unpack(data)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack single value: %v", err)
	}

	if len(values) == 0 {
		return nil, nil
	}

	return d.formatDecodedValue(values[0], output.Type), nil
}

// formatDecodedValue formats decoded value based on type
func (d *ABIDecoder) formatDecodedValue(value interface{}, paramType string) interface{} {
	switch paramType {
	case "address":
		if addr, ok := value.(eCommon.Address); ok {
			// Convert to TRON address format
			tronAddr, err := types.NewAddressFromHex(addr.Hex())
			if err != nil {
				return value
			}
			return tronAddr.String()
		}
		return value

	case "uint256", "uint128", "uint64", "uint32", "uint16", "uint8",
		 "int256", "int128", "int64", "int32", "int16", "int8":
		if bigInt, ok := value.(*big.Int); ok {
			return bigInt.String()
		}
		return value

	case "bytes", "bytes32", "bytes16", "bytes8":
		if bytes, ok := value.([]byte); ok {
			return hex.EncodeToString(bytes)
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
					result[i] = d.formatDecodedValue(slice.Index(i).Interface(), baseType)
				}
				return result
			}
		}
		return value
	}
}