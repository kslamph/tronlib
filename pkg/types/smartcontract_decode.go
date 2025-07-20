package types

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"strings"

	eABI "github.com/ethereum/go-ethereum/accounts/abi"
	eCommon "github.com/ethereum/go-ethereum/common"
	"github.com/kslamph/tronlib/pb/core"
	"golang.org/x/crypto/sha3"
)

// DecodedInput represents the decoded input data
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

// DecodeInputData decodes the contract input data and returns method signature and parameters
func (c *Contract) DecodeInputData(data []byte) (*DecodedInput, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("input data too short, need at least 4 bytes for method signature")
	}

	// Extract method signature (first 4 bytes)
	methodSig := data[:4]

	// Find matching method in ABI by comparing method signatures
	var matchedEntry *core.SmartContract_ABI_Entry
	var methodSignature string

	for _, entry := range c.ABI.Entrys {
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

	// Create ethereum ABI arguments for decoding
	args := make([]eABI.Argument, len(matchedEntry.Inputs))
	for i, input := range matchedEntry.Inputs {
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
	values, err := eABI.Arguments(args).Unpack(paramData)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack parameters: %v", err)
	}

	// Build decoded parameters
	parameters := make([]DecodedInputParameter, len(matchedEntry.Inputs))
	for i, input := range matchedEntry.Inputs {
		var value interface{}
		if i < len(values) {
			value = formatDecodedValue(values[i], input.Type)
		}

		parameters[i] = DecodedInputParameter{
			Name:  input.Name,
			Type:  input.Type,
			Value: value,
		}
	}

	return &DecodedInput{
		Method:     methodSignature,
		Parameters: parameters,
	}, nil
}

// formatDecodedValue formats the decoded value based on its type
func formatDecodedValue(value interface{}, paramType string) interface{} {
	switch paramType {
	case "address":
		if addr, ok := value.(eCommon.Address); ok {
			return addr.Hex()
		}
		return value
	case "uint256", "uint128", "uint64", "uint32", "uint16", "uint8":
		if bigInt, ok := value.(*big.Int); ok {
			return bigInt.String()
		}
		return value
	case "bytes", "bytes32", "bytes16", "bytes8":
		if bytes, ok := value.([]byte); ok {
			return hex.EncodeToString(bytes)
		}
		return value
	default:
		// Handle array types
		if strings.HasSuffix(paramType, "[]") {
			baseType := strings.TrimSuffix(paramType, "[]")
			if reflect.TypeOf(value).Kind() == reflect.Slice {
				slice := reflect.ValueOf(value)
				result := make([]interface{}, slice.Len())
				for i := 0; i < slice.Len(); i++ {
					result[i] = formatDecodedValue(slice.Index(i).Interface(), baseType)
				}
				return result
			}
		}
		return value
	}
}

// DecodeResult decodes the contract call result bytes
func (c *Contract) DecodeResult(method string, result [][]byte) (interface{}, error) {
	if len(result) == 0 {
		return nil, fmt.Errorf("empty result")
	}

	// Get the output argument types from ABI
	outputs, err := c.getParser(method)
	if err != nil {
		return nil, fmt.Errorf("failed to get output parser: %v", err)
	}

	// For single output
	if len(outputs) == 1 {
		return decodeValue(result[0], outputs[0])
	}

	// For multiple outputs
	decoded := make(map[string]interface{})
	for i, output := range outputs {
		if i >= len(result) {
			break
		}
		for typeName := range output {
			value, err := decodeValue(result[i], output)
			if err != nil {
				return nil, err
			}
			decoded[typeName] = value
		}
	}
	return decoded, nil
}

// decodeValue decodes a single value based on its ABI type
func decodeValue(data []byte, param Param) (interface{}, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data")
	}

	for typeName := range param {
		switch typeName {
		case "address":
			return eCommon.BytesToAddress(data).Hex(), nil

		case "uint256", "uint128", "uint64", "uint32", "uint16", "uint8":
			return new(big.Int).SetBytes(data), nil

		case "string":
			return string(data), nil

		case "bool":
			return len(data) > 0 && data[0] != 0, nil

		default:
			return data, nil
		}
	}
	return nil, fmt.Errorf("no type specified in param")
}

// decodeABI decodes the ABI string into a core.SmartContract_ABI object
func decodeABI(abi string) (*core.SmartContract_ABI, error) {
	if abi == "" {
		return nil, fmt.Errorf("empty ABI string")
	}

	var abiData []map[string]interface{}
	if err := json.Unmarshal([]byte(abi), &abiData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ABI: %v", err)
	}

	contractABI := &core.SmartContract_ABI{
		Entrys: make([]*core.SmartContract_ABI_Entry, 0),
	}

	for _, entry := range abiData {
		abiEntry := &core.SmartContract_ABI_Entry{}

		if name, ok := entry["name"].(string); ok {
			abiEntry.Name = name
		}

		if type_, ok := entry["type"].(string); ok {
			switch type_ {
			case "function":
				abiEntry.Type = core.SmartContract_ABI_Entry_Function
			case "constructor":
				abiEntry.Type = core.SmartContract_ABI_Entry_Constructor
			case "event":
				abiEntry.Type = core.SmartContract_ABI_Entry_Event
			case "fallback":
				abiEntry.Type = core.SmartContract_ABI_Entry_Fallback
			default:
				abiEntry.Type = core.SmartContract_ABI_Entry_UnknownEntryType
			}
		}

		if inputs, ok := entry["inputs"].([]interface{}); ok {
			abiEntry.Inputs = make([]*core.SmartContract_ABI_Entry_Param, 0)
			for _, input := range inputs {
				inputMap, ok := input.(map[string]interface{})
				if !ok {
					continue
				}

				param := &core.SmartContract_ABI_Entry_Param{}
				if name, ok := inputMap["name"].(string); ok {
					param.Name = name
				}
				if type_, ok := inputMap["type"].(string); ok {
					param.Type = type_
				}
				if indexed, ok := inputMap["indexed"].(bool); ok {
					param.Indexed = indexed
				}
				abiEntry.Inputs = append(abiEntry.Inputs, param)
			}
		}

		if outputs, ok := entry["outputs"].([]interface{}); ok {
			abiEntry.Outputs = make([]*core.SmartContract_ABI_Entry_Param, 0)
			for _, output := range outputs {
				outputMap, ok := output.(map[string]interface{})
				if !ok {
					continue
				}

				param := &core.SmartContract_ABI_Entry_Param{}
				if name, ok := outputMap["name"].(string); ok {
					param.Name = name
				}
				if type_, ok := outputMap["type"].(string); ok {
					param.Type = type_
				}
				if indexed, ok := outputMap["indexed"].(bool); ok {
					param.Indexed = indexed
				}
				abiEntry.Outputs = append(abiEntry.Outputs, param)
			}
		}

		if payable, ok := entry["payable"].(bool); ok {
			abiEntry.Payable = payable
		}

		if stateMutability, ok := entry["stateMutability"].(string); ok {
			switch stateMutability {
			case "pure":
				abiEntry.StateMutability = core.SmartContract_ABI_Entry_Pure
			case "view":
				abiEntry.StateMutability = core.SmartContract_ABI_Entry_View
			case "nonpayable":
				abiEntry.StateMutability = core.SmartContract_ABI_Entry_Nonpayable
			case "payable":
				abiEntry.StateMutability = core.SmartContract_ABI_Entry_Payable
			default:
				abiEntry.StateMutability = core.SmartContract_ABI_Entry_UnknownMutabilityType
			}
		}
		if constant, ok := entry["constant"].(bool); ok {
			abiEntry.Constant = constant
		}

		contractABI.Entrys = append(contractABI.Entrys, abiEntry)
	}

	return contractABI, nil
}
