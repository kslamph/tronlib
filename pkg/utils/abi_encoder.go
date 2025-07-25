package utils

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"strings"

	eABI "github.com/ethereum/go-ethereum/accounts/abi"
	eCommon "github.com/ethereum/go-ethereum/common"
	"github.com/kslamph/tronlib/pkg/types"
	"golang.org/x/crypto/sha3"
)

// ABIEncoder handles smart contract ABI encoding operations
type ABIEncoder struct{}

// NewABIEncoder creates a new ABI encoder instance
func NewABIEncoder() *ABIEncoder {
	return &ABIEncoder{}
}

// EncodeMethod encodes method call with parameters
func (e *ABIEncoder) EncodeMethod(method string, paramTypes []string, params []interface{}) ([]byte, error) {
	// For constructors (empty method name), encode parameters without method ID
	if method == "" {
		if len(params) == 0 {
			return []byte{}, nil
		}
		// Encode parameters only (no method ID for constructors)
		return e.EncodeParameters(paramTypes, params)
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
	encoded, err := e.EncodeParameters(paramTypes, params)
	if err != nil {
		return nil, fmt.Errorf("failed to encode parameters: %v", err)
	}

	return append(methodID, encoded...), nil
}

// EncodeParameters encodes function parameters
func (e *ABIEncoder) EncodeParameters(paramTypes []string, params []interface{}) ([]byte, error) {
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
		convertedValue, err := e.convertParameter(params[i], paramType)
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
func (e *ABIEncoder) convertParameter(param interface{}, paramType string) (interface{}, error) {
	if param == nil {
		return nil, fmt.Errorf("nil parameter not allowed")
	}

	// Handle array types
	if strings.HasSuffix(paramType, "[]") {
		baseType := strings.TrimSuffix(paramType, "[]")
		return e.convertArrayParameter(param, baseType)
	}

	// Handle scalar types
	switch paramType {
	case "address":
		return e.convertAddress(param)
	case "uint8":
		return e.convertUint8(param)
	case "uint256", "uint128", "uint64", "uint32", "uint16":
		return e.convertUint(param)
	case "int256", "int128", "int64", "int32", "int16", "int8":
		return e.convertInt(param)
	case "bool":
		return e.convertBool(param)
	case "string":
		return e.convertString(param)
	case "bytes", "bytes32", "bytes16", "bytes8":
		return e.convertBytes(param)
	default:
		return nil, fmt.Errorf("unsupported parameter type: %s", paramType)
	}
}

// convertAddress converts address parameter
func (e *ABIEncoder) convertAddress(param interface{}) (eCommon.Address, error) {
	addrStr, ok := param.(string)
	if !ok {
		return eCommon.Address{}, fmt.Errorf("address parameter must be a string")
	}

	if strings.HasPrefix(addrStr, "0x") {
		return eCommon.HexToAddress(addrStr), nil
	}

	if strings.HasPrefix(addrStr, "T") {
		tronAddr, err := types.NewAddressFromBase58(addrStr)
		if err != nil {
			return eCommon.Address{}, fmt.Errorf("invalid Tron address: %v", err)
		}
		return eCommon.BytesToAddress(tronAddr.Bytes()[1:]), nil
	}

	// Try hex decoding
	decoded, err := hex.DecodeString(addrStr)
	if err != nil {
		return eCommon.Address{}, fmt.Errorf("invalid address format: %v", err)
	}
	if len(decoded) > 20 {
		decoded = decoded[len(decoded)-20:]
	}
	return eCommon.BytesToAddress(decoded), nil
}

// convertUint converts unsigned integer parameter
func (e *ABIEncoder) convertUint(param interface{}) (*big.Int, error) {
	switch v := param.(type) {
	case string:
		n, ok := new(big.Int).SetString(v, 0)
		if !ok {
			return nil, fmt.Errorf("invalid number string: %s", v)
		}
		return n, nil
	case *big.Int:
		return v, nil
	case int:
		return new(big.Int).SetInt64(int64(v)), nil
	case int64:
		return new(big.Int).SetInt64(v), nil
	case uint64:
		return new(big.Int).SetUint64(v), nil
	case int32:
		return new(big.Int).SetInt64(int64(v)), nil
	case uint32:
		return new(big.Int).SetUint64(uint64(v)), nil
	case int16:
		return new(big.Int).SetInt64(int64(v)), nil
	case uint16:
		return new(big.Int).SetUint64(uint64(v)), nil
	case int8:
		return new(big.Int).SetInt64(int64(v)), nil
	case uint8:
		return new(big.Int).SetUint64(uint64(v)), nil
	case float64:
		return new(big.Int).SetInt64(int64(v)), nil
	default:
		return nil, fmt.Errorf("unsupported uint type: %T", param)
	}
}

// convertInt converts signed integer parameter
func (e *ABIEncoder) convertInt(param interface{}) (*big.Int, error) {
	return e.convertUint(param) // Same conversion logic
}

// convertBool converts boolean parameter
func (e *ABIEncoder) convertBool(param interface{}) (bool, error) {
	if b, ok := param.(bool); ok {
		return b, nil
	}
	return false, fmt.Errorf("bool parameter must be a boolean")
}

// convertString converts string parameter
func (e *ABIEncoder) convertString(param interface{}) (string, error) {
	if s, ok := param.(string); ok {
		return s, nil
	}
	return "", fmt.Errorf("string parameter must be a string")
}

// convertBytes converts bytes parameter
func (e *ABIEncoder) convertBytes(param interface{}) ([]byte, error) {
	switch v := param.(type) {
	case string:
		if strings.HasPrefix(v, "0x") {
			return eCommon.FromHex(v), nil
		}
		return hex.DecodeString(v)
	case []byte:
		return v, nil
	default:
		return nil, fmt.Errorf("bytes parameter must be string or []byte")
	}
}

// convertArrayParameter converts array parameter
func (e *ABIEncoder) convertArrayParameter(param interface{}, baseType string) (interface{}, error) {
	// Handle JSON string arrays
	if jsonStr, ok := param.(string); ok {
		var jsonArray []interface{}
		if err := json.Unmarshal([]byte(jsonStr), &jsonArray); err != nil {
			return nil, fmt.Errorf("failed to parse array JSON: %v", err)
		}
		return e.convertArrayElements(jsonArray, baseType)
	}

	// Handle slice directly
	if reflect.TypeOf(param).Kind() == reflect.Slice {
		slice := reflect.ValueOf(param)
		elements := make([]interface{}, slice.Len())
		for i := 0; i < slice.Len(); i++ {
			elements[i] = slice.Index(i).Interface()
		}
		return e.convertArrayElements(elements, baseType)
	}

	return nil, fmt.Errorf("array parameter must be JSON string or slice")
}

// convertArrayElements converts array elements to appropriate types
func (e *ABIEncoder) convertArrayElements(elements []interface{}, baseType string) (interface{}, error) {
	switch baseType {
	case "address":
		addresses := make([]eCommon.Address, len(elements))
		for i, elem := range elements {
			addr, err := e.convertAddress(elem)
			if err != nil {
				return nil, fmt.Errorf("invalid address at index %d: %v", i, err)
			}
			addresses[i] = addr
		}
		return addresses, nil

	case "uint256", "uint128", "uint64", "uint32", "uint16", "uint8":
		ints := make([]*big.Int, len(elements))
		for i, elem := range elements {
			n, err := e.convertUint(elem)
			if err != nil {
				return nil, fmt.Errorf("invalid uint at index %d: %v", i, err)
			}
			ints[i] = n
		}
		return ints, nil

	case "int256", "int128", "int64", "int32", "int16", "int8":
		ints := make([]*big.Int, len(elements))
		for i, elem := range elements {
			n, err := e.convertInt(elem)
			if err != nil {
				return nil, fmt.Errorf("invalid int at index %d: %v", i, err)
			}
			ints[i] = n
		}
		return ints, nil

	case "bool":
		bools := make([]bool, len(elements))
		for i, elem := range elements {
			b, err := e.convertBool(elem)
			if err != nil {
				return nil, fmt.Errorf("invalid bool at index %d: %v", i, err)
			}
			bools[i] = b
		}
		return bools, nil

	case "string":
		strings := make([]string, len(elements))
		for i, elem := range elements {
			s, err := e.convertString(elem)
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

// GetMethodSignature creates method signature string
func (e *ABIEncoder) GetMethodSignature(method string, paramTypes []string) string {
	return fmt.Sprintf("%s(%s)", method, strings.Join(paramTypes, ","))
}

// GetMethodID calculates method ID (first 4 bytes of keccak256 hash)
func (e *ABIEncoder) GetMethodID(methodSig string) []byte {
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write([]byte(methodSig))
	return hasher.Sum(nil)[:4]
}

// convertUint8 converts uint8 parameter specifically
func (e *ABIEncoder) convertUint8(param interface{}) (uint8, error) {
	switch v := param.(type) {
	case uint8:
		return v, nil
	case int:
		if v < 0 || v > 255 {
			return 0, fmt.Errorf("value %d out of range for uint8", v)
		}
		return uint8(v), nil
	case int64:
		if v < 0 || v > 255 {
			return 0, fmt.Errorf("value %d out of range for uint8", v)
		}
		return uint8(v), nil
	case uint64:
		if v > 255 {
			return 0, fmt.Errorf("value %d out of range for uint8", v)
		}
		return uint8(v), nil
	case int32:
		if v < 0 || v > 255 {
			return 0, fmt.Errorf("value %d out of range for uint8", v)
		}
		return uint8(v), nil
	case uint32:
		if v > 255 {
			return 0, fmt.Errorf("value %d out of range for uint8", v)
		}
		return uint8(v), nil
	case int16:
		if v < 0 || v > 255 {
			return 0, fmt.Errorf("value %d out of range for uint8", v)
		}
		return uint8(v), nil
	case uint16:
		if v > 255 {
			return 0, fmt.Errorf("value %d out of range for uint8", v)
		}
		return uint8(v), nil
	case int8:
		if v < 0 {
			return 0, fmt.Errorf("value %d out of range for uint8", v)
		}
		return uint8(v), nil
	case float64:
		if v < 0 || v > 255 {
			return 0, fmt.Errorf("value %f out of range for uint8", v)
		}
		return uint8(v), nil
	case string:
		n, err := strconv.ParseUint(v, 0, 8)
		if err != nil {
			return 0, fmt.Errorf("invalid uint8 string: %s", v)
		}
		return uint8(n), nil
	default:
		return 0, fmt.Errorf("unsupported uint8 type: %T", param)
	}
}