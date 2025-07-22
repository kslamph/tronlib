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

// getInputsParser returns input method parser arguments for the given method
func (c *Contract) getInputsParser(method string) ([]Param, error) {
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

// getParser returns output arguments parser for the given method
func (c *Contract) getParser(method string) ([]Param, error) {
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

// pack creates the encoded contract call data
func pack(method string, params []Param) ([]byte, error) {
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
	// Special handling for constructors (empty method name)
	if method == "" {
		return c.encodeConstructor(params...)
	}

	// Get input parameters from ABI
	abiParams, err := c.getInputsParser(method)
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
					case uint64:
						param[typeName] = new(big.Int).SetUint64(v)
					case uint32:
						param[typeName] = new(big.Int).SetUint64(uint64(v))
					case uint16:
						param[typeName] = new(big.Int).SetUint64(uint64(v))
					case uint8:
						param[typeName] = new(big.Int).SetUint64(uint64(v))
					case int32:
						param[typeName] = new(big.Int).SetInt64(int64(v))
					case int16:
						param[typeName] = new(big.Int).SetInt64(int64(v))
					case int8:
						param[typeName] = new(big.Int).SetInt64(int64(v))
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

	return pack(method, abiParams)
}

// encodeConstructor handles constructor parameter encoding
func (c *Contract) encodeConstructor(params ...interface{}) ([]byte, error) {
	// Find constructor in ABI
	var constructorEntry *core.SmartContract_ABI_Entry
	for _, entry := range c.ABI.Entrys {
		if entry.Type == core.SmartContract_ABI_Entry_Constructor {
			constructorEntry = entry
			break
		}
	}

	if constructorEntry == nil {
		return nil, fmt.Errorf("constructor not found in ABI")
	}

	// Create params structure for constructor
	abiParams := make([]Param, len(constructorEntry.Inputs))
	for i, input := range constructorEntry.Inputs {
		param := make(Param)
		param[input.Type] = nil
		abiParams[i] = param
	}

	if len(params) != len(abiParams) {
		return nil, fmt.Errorf("constructor argument count mismatch: expected %d, got %d", len(abiParams), len(params))
	}

	// Convert parameters to proper format
	for i, paramValue := range params {
		if paramValue == nil {
			return nil, fmt.Errorf("nil value not allowed for constructor parameter %d", i)
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
					case uint64:
						param[typeName] = new(big.Int).SetUint64(v)
					case uint32:
						param[typeName] = new(big.Int).SetUint64(uint64(v))
					case uint16:
						param[typeName] = new(big.Int).SetUint64(uint64(v))
					case uint8:
						param[typeName] = new(big.Int).SetUint64(uint64(v))
					case int32:
						param[typeName] = new(big.Int).SetInt64(int64(v))
					case int16:
						param[typeName] = new(big.Int).SetInt64(int64(v))
					case int8:
						param[typeName] = new(big.Int).SetInt64(int64(v))
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

	// Pack constructor parameters (no method ID)
	return packConstructor(abiParams)
}

// packConstructor packs constructor parameters without method ID
func packConstructor(params []Param) ([]byte, error) {
	if len(params) == 0 {
		return []byte{}, nil
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

	// Pack the arguments (no method ID for constructors)
	packed, err := eABI.Arguments(args).Pack(values...)
	if err != nil {
		return nil, fmt.Errorf("failed to pack constructor parameters: %v", err)
	}

	return packed, nil
}
