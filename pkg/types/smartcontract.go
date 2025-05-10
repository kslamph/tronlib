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

// Contract represents a smart contract interface
type Contract struct {
	ABI          *core.SmartContract_ABI
	Address      string
	AddressBytes []byte
}

// Param list
type Param map[string]interface{}

// NewContract creates a new contract instance
func NewContract(abi string, address string) (*Contract, error) {
	if abi == "" {
		return nil, fmt.Errorf("empty ABI string")
	}
	if address == "" {
		return nil, fmt.Errorf("empty contract address")
	}
	decodedABI, err := decodeABI(abi)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ABI: %v", err)
	}

	// Convert address to bytes
	addr, err := NewAddress(address)
	if err != nil {
		return nil, fmt.Errorf("failed to parse address: %v", err)
	}

	return &Contract{
		ABI:          decodedABI,
		Address:      address,
		AddressBytes: addr.Bytes(),
	}, nil
}

func NewContractFromABI(abi *core.SmartContract_ABI, address string) (*Contract, error) {
	addr, err := NewAddress(address)
	if err != nil {
		return nil, fmt.Errorf("failed to parse address: %v", err)
	}
	return &Contract{
		ABI:          abi,
		Address:      address,
		AddressBytes: addr.Bytes(),
	}, nil
}

// GetInputsParser returns input method parser arguments for the given method
func (c *Contract) GetInputsParser(method string) ([]Param, error) {
	for _, entry := range c.ABI.Entrys {
		if entry.Name == method {
			params := make([]Param, len(entry.Inputs))
			for i, input := range entry.Inputs {
				param := make(Param)
				param[input.Type] = nil
				params[i] = param
			}
			return params, nil
		}
	}
	return nil, fmt.Errorf("method %s not found", method)
}

// GetParser returns output arguments parser for the given method
func (c *Contract) GetParser(method string) ([]Param, error) {
	for _, entry := range c.ABI.Entrys {
		if entry.Name == method {
			params := make([]Param, len(entry.Outputs))
			for i, output := range entry.Outputs {
				param := make(Param)
				param[output.Type] = nil
				params[i] = param
			}
			return params, nil
		}
	}
	return nil, fmt.Errorf("method %s not found", method)
}

// Pack creates the encoded contract call data
func Pack(method string, params []Param) ([]byte, error) {
	// Create method signature
	methodSig := method
	if !strings.Contains(method, "(") {
		inputs := make([]string, 0, len(params))
		for _, p := range params {
			for t := range p {
				inputs = append(inputs, t)
			}
		}
		methodSig = fmt.Sprintf("%s(%s)", method, strings.Join(inputs, ","))
	}

	// Get method ID
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write([]byte(methodSig))
	methodID := hasher.Sum(nil)[:4]

	if len(params) == 0 {
		return methodID, nil
	}

	// Create args for ethereum ABI
	args := make([]eABI.Argument, len(params))
	values := make([]interface{}, len(params))

	for i, param := range params {
		for typ, val := range param {
			abiType, err := eABI.NewType(typ, "", nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create ABI type for %s: %v", typ, err)
			}
			args[i] = eABI.Argument{Type: abiType}
			values[i] = val
		}
	}

	// Pack the arguments
	packed, err := eABI.Arguments(args).Pack(values...)
	if err != nil {
		return nil, fmt.Errorf("failed to pack parameters: %v", err)
	}

	return append(methodID, packed...), nil
}

// EncodeInput creates contract call data
func (c *Contract) EncodeInput(method string, params ...interface{}) ([]byte, error) {
	if method == "" {
		return nil, fmt.Errorf("empty method name")
	}

	// Get input parameters from ABI
	abiParams, err := c.GetInputsParser(method)
	if err != nil {
		return nil, fmt.Errorf("failed to get input parser: %v", err)
	}

	if len(params) != len(abiParams) {
		return nil, fmt.Errorf("argument count mismatch: expected %d, got %d", len(abiParams), len(params))
	}

	// Convert parameters to proper format
	for i, paramValue := range params {
		if paramValue == nil {
			return nil, fmt.Errorf("nil value not allowed for parameter %d", i)
		}

		paramType := reflect.TypeOf(paramValue)
		param := abiParams[i]

		for typeName := range param {
			isArray := strings.HasSuffix(typeName, "[]")
			baseType := strings.TrimSuffix(typeName, "[]")

			if isArray {
				// Handle array types
				if paramType.Kind() != reflect.String {
					return nil, fmt.Errorf("array parameter must be provided as JSON string")
				}

				jsonStr, ok := paramValue.(string)
				if !ok {
					return nil, fmt.Errorf("failed to convert array parameter to string")
				}

				var jsonArray []interface{}
				if err := json.Unmarshal([]byte(jsonStr), &jsonArray); err != nil {
					return nil, fmt.Errorf("failed to parse array JSON: %v", err)
				}

				switch baseType {
				case "address":
					addresses := make([]eCommon.Address, len(jsonArray))
					for j, addr := range jsonArray {
						addrStr, ok := addr.(string)
						if !ok {
							return nil, fmt.Errorf("invalid address in array at index %d", j)
						}
						if strings.HasPrefix(addrStr, "0x") {
							addresses[j] = eCommon.HexToAddress(addrStr)
						} else {
							tronAddr, err := NewAddress(addrStr)
							if err != nil {
								return nil, fmt.Errorf("invalid Tron address at index %d: %v", j, err)
							}
							addresses[j] = eCommon.BytesToAddress(tronAddr.Bytes()[1:])
						}
					}
					param[typeName] = addresses

				case "uint256", "uint128", "uint64", "uint32", "uint16", "uint8":
					ints := make([]*big.Int, len(jsonArray))
					for j, val := range jsonArray {
						switch v := val.(type) {
						case float64:
							ints[j] = new(big.Int).SetInt64(int64(v))
						case string:
							n, ok := new(big.Int).SetString(v, 0)
							if !ok {
								return nil, fmt.Errorf("invalid number in array at index %d", j)
							}
							ints[j] = n
						default:
							return nil, fmt.Errorf("unsupported number type in array at index %d", j)
						}
					}
					param[typeName] = ints

				default:
					return nil, fmt.Errorf("unsupported array type: %s", baseType)
				}
			} else {
				// Handle scalar types
				switch typeName {
				case "address":
					addrStr, ok := paramValue.(string)
					if !ok {
						return nil, fmt.Errorf("address parameter must be a string")
					}

					if !strings.HasPrefix(addrStr, "0x") && !strings.HasPrefix(addrStr, "T") {
						decoded, err := hex.DecodeString(addrStr)
						if err != nil {
							return nil, fmt.Errorf("invalid hex string for address: %v", err)
						}
						if len(decoded) > 20 {
							decoded = decoded[len(decoded)-20:]
						}
						param[typeName] = eCommon.BytesToAddress(decoded)
					} else if strings.HasPrefix(addrStr, "T") {
						tronAddr, err := NewAddress(addrStr)
						if err != nil {
							return nil, fmt.Errorf("invalid Tron address: %v", err)
						}
						param[typeName] = eCommon.BytesToAddress(tronAddr.Bytes()[1:])
					} else {
						param[typeName] = eCommon.HexToAddress(addrStr)
					}

				case "uint256", "uint128", "uint64", "uint32", "uint16", "uint8":
					switch v := paramValue.(type) {
					case string:
						n, ok := new(big.Int).SetString(v, 0)
						if !ok {
							return nil, fmt.Errorf("invalid number string: %s", v)
						}
						param[typeName] = n
					case float64:
						param[typeName] = new(big.Int).SetInt64(int64(v))
					case int64:
						param[typeName] = new(big.Int).SetInt64(v)
					default:
						param[typeName] = paramValue
					}

				case "bool":
					if paramType.Kind() != reflect.Bool {
						return nil, fmt.Errorf("invalid type for bool parameter: expected bool")
					}
					param[typeName] = paramValue

				case "string":
					if paramType.Kind() != reflect.String {
						return nil, fmt.Errorf("invalid type for string parameter: expected string")
					}
					param[typeName] = paramValue

				case "bytes", "bytes32", "bytes16", "bytes8":
					switch v := paramValue.(type) {
					case string:
						if !strings.HasPrefix(v, "0x") {
							decoded, err := hex.DecodeString(v)
							if err != nil {
								return nil, fmt.Errorf("invalid hex string for bytes: %v", err)
							}
							param[typeName] = decoded
						} else {
							param[typeName] = eCommon.FromHex(v)
						}
					default:
						param[typeName] = paramValue
					}

				default:
					return nil, fmt.Errorf("unsupported parameter type %s", typeName)
				}
			}
		}
	}

	return Pack(method, abiParams)
}

// DecodeResult decodes the contract call result bytes
func (c *Contract) DecodeResult(method string, result [][]byte) (interface{}, error) {
	if len(result) == 0 {
		return nil, fmt.Errorf("empty result")
	}

	// Get the output argument types from ABI
	outputs, err := c.GetParser(method)
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
