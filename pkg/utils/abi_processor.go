package utils

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"sync"

	eABI "github.com/ethereum/go-ethereum/accounts/abi"
	eCommon "github.com/ethereum/go-ethereum/common"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
	"golang.org/x/crypto/sha3"
)

// ABIProcessor handles all smart contract ABI operations including encoding, decoding, parsing, and event processing
type ABIProcessor struct {
	abi *core.SmartContract_ABI

	// Event signature caches using sync.Once pattern
	eventCacheOnce      sync.Once
	eventSignatureCache map[[32]byte]*core.SmartContract_ABI_Entry

	event4ByteCacheOnce      sync.Once
	event4ByteSignatureCache map[[4]byte]*core.SmartContract_ABI_Entry
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

// DecodedEvent represents a decoded event
type DecodedEvent struct {
	EventName  string                  `json:"eventName"`
	Parameters []DecodedEventParameter `json:"parameters"`
}

// DecodedEventParameter represents a decoded event parameter
type DecodedEventParameter struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Value   string `json:"value"`
	Indexed bool   `json:"indexed"`
}

// NewABIProcessor creates a new ABI processor instance
func NewABIProcessor(abi *core.SmartContract_ABI) *ABIProcessor {
	return &ABIProcessor{
		abi: abi,
	}
}

// ParseABI decodes the ABI string into a core.SmartContract_ABI object
func (p *ABIProcessor) ParseABI(abi string) (*core.SmartContract_ABI, error) {
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
		abiEntry, err := p.parseABIEntry(entry)
		if err != nil {
			return nil, fmt.Errorf("failed to parse ABI entry: %v", err)
		}
		contractABI.Entrys = append(contractABI.Entrys, abiEntry)
	}

	return contractABI, nil
}

// parseABIEntry parses a single ABI entry
func (p *ABIProcessor) parseABIEntry(entry map[string]interface{}) (*core.SmartContract_ABI_Entry, error) {
	abiEntry := &core.SmartContract_ABI_Entry{}

	// Parse name
	if name, ok := entry["name"].(string); ok {
		abiEntry.Name = name
	}

	// Parse type
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

	// Parse inputs
	if inputs, ok := entry["inputs"].([]interface{}); ok {
		parsedInputs, err := p.parseParameters(inputs)
		if err != nil {
			return nil, fmt.Errorf("failed to parse inputs: %v", err)
		}
		abiEntry.Inputs = parsedInputs
	}

	// Parse outputs
	if outputs, ok := entry["outputs"].([]interface{}); ok {
		parsedOutputs, err := p.parseParameters(outputs)
		if err != nil {
			return nil, fmt.Errorf("failed to parse outputs: %v", err)
		}
		abiEntry.Outputs = parsedOutputs
	}

	// Parse payable
	if payable, ok := entry["payable"].(bool); ok {
		abiEntry.Payable = payable
	}

	// Parse stateMutability
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

	// Parse constant
	if constant, ok := entry["constant"].(bool); ok {
		abiEntry.Constant = constant
	}

	return abiEntry, nil
}

// parseParameters parses ABI parameters (inputs or outputs)
func (p *ABIProcessor) parseParameters(params []interface{}) ([]*core.SmartContract_ABI_Entry_Param, error) {
	result := make([]*core.SmartContract_ABI_Entry_Param, 0)

	for _, param := range params {
		paramMap, ok := param.(map[string]interface{})
		if !ok {
			continue
		}

		abiParam := &core.SmartContract_ABI_Entry_Param{}

		if name, ok := paramMap["name"].(string); ok {
			abiParam.Name = name
		}

		if type_, ok := paramMap["type"].(string); ok {
			abiParam.Type = type_
		}

		if indexed, ok := paramMap["indexed"].(bool); ok {
			abiParam.Indexed = indexed
		}

		result = append(result, abiParam)
	}

	return result, nil
}

// GetMethodTypes extracts parameter types for a method
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

// GetConstructorTypes extracts parameter types for constructor
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

// EncodeMethod encodes method call with parameters
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

// DecodeInputData decodes contract input data
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

// DecodeResult decodes contract call result
func (p *ABIProcessor) DecodeResult(data []byte, outputs []*core.SmartContract_ABI_Entry_Param) ([]interface{}, error) {
	if len(outputs) == 0 {
		return []interface{}{}, nil
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

	// Format the decoded values and put them in a slice
	result := make([]interface{}, len(outputs))
	for i, output := range outputs {
		if i < len(values) {
			result[i] = p.formatDecodedValue(values[i], output.Type)
		}
	}

	return result, nil
}

// decodeSingleValue decodes a single return value
func (p *ABIProcessor) decodeSingleValue(data []byte, output *core.SmartContract_ABI_Entry_Param) (interface{}, error) {
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

	return p.formatDecodedValue(values[0], output.Type), nil
}

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
			return tronAddr.String()
		}
		return value

	// case "uint256", "uint128", "uint64", "uint32", "uint16", "uint8":
	// 	if bigInt, ok := value.(*big.Int); ok {
	// 		// For uint types, if the value fits within uint64, return it as uint64
	// 		// Otherwise, return the value as *big.Int
	// 		if bigInt.IsUint64() {
	// 			return bigInt.Uint64()
	// 		}
	// 		return bigInt
	// 	}
	// 	return value

	// case "int256", "int128", "int64", "int32", "int16", "int8":
	// 	if bigInt, ok := value.(*big.Int); ok {
	// 		// For int types, if the value fits within int64, return it as int64
	// 		// Otherwise, return the value as *big.Int
	// 		if bigInt.IsInt64() {
	// 			return bigInt.Int64()
	// 		}
	// 		return bigInt
	// 	}
	// 	return value

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
			// If the value is not a slice, return it as is
			// This could happen if the ABI type doesn't match the actual value type
			// TODO: Implement full tuple support for array types
		}
		return value
	}
}

// DecodeEventLog decodes an event log
func (p *ABIProcessor) DecodeEventLog(topics [][]byte, data []byte) (*DecodedEvent, error) {
	if len(topics) == 0 {
		return nil, fmt.Errorf("no topics provided")
	}

	// Ensure cache is built (thread-safe, one-time only)
	p.eventCacheOnce.Do(p.buildEventSignatureCache)

	// First topic is the event signature (32 bytes)
	eventSignature := topics[0]

	// Convert to fixed-size array for O(1) lookup
	var sigArray [32]byte
	copy(sigArray[:], eventSignature)

	// O(1) lookup instead of O(n) iteration
	matchedEvent, exists := p.eventSignatureCache[sigArray]
	if !exists {
		return &DecodedEvent{
			EventName:  fmt.Sprintf("unknown_event(0x%s)", hex.EncodeToString(eventSignature)),
			Parameters: []DecodedEventParameter{},
		}, nil
	}

	// Separate indexed and non-indexed parameters
	var indexedParams []*core.SmartContract_ABI_Entry_Param
	var nonIndexedParams []*core.SmartContract_ABI_Entry_Param

	for _, input := range matchedEvent.Inputs {
		if input.Indexed {
			indexedParams = append(indexedParams, input)
		} else {
			nonIndexedParams = append(nonIndexedParams, input)
		}
	}

	// Decode indexed parameters (from topics[1:])
	indexedValues := make([]DecodedEventParameter, 0)
	for i, param := range indexedParams {
		if i+1 >= len(topics) {
			break
		}

		value := p.decodeTopicValue(topics[i+1], param.Type)
		indexedValues = append(indexedValues, DecodedEventParameter{
			Name:    param.Name,
			Type:    param.Type,
			Value:   value,
			Indexed: true,
		})
	}

	// Decode non-indexed parameters (from data)
	var nonIndexedValues []DecodedEventParameter
	if len(nonIndexedParams) > 0 && len(data) > 0 {
		decoded, err := p.decodeEventData(data, nonIndexedParams)
		if err != nil {
			return nil, fmt.Errorf("failed to decode event data: %v", err)
		}
		nonIndexedValues = decoded
	}

	// Combine all parameters in original order
	allParams := make([]DecodedEventParameter, 0)
	for _, input := range matchedEvent.Inputs {
		if input.Indexed {
			// Find corresponding indexed parameter
			for _, indexed := range indexedValues {
				if indexed.Name == input.Name {
					allParams = append(allParams, indexed)
					break
				}
			}
		} else {
			// Find corresponding non-indexed parameter
			for _, nonIndexed := range nonIndexedValues {
				if nonIndexed.Name == input.Name {
					allParams = append(allParams, nonIndexed)
					break
				}
			}
		}
	}

	return &DecodedEvent{
		EventName:  matchedEvent.Name,
		Parameters: allParams,
	}, nil
}

// DecodeEventSignature decodes event signature bytes and returns the event name
func (p *ABIProcessor) DecodeEventSignature(signature []byte) (string, error) {
	if len(signature) < 4 {
		return "", fmt.Errorf("event signature too short, need at least 4 bytes")
	}

	// Ensure 4-byte cache is built (thread-safe, one-time only)
	p.event4ByteCacheOnce.Do(p.buildEvent4ByteSignatureCache)

	// Extract event signature (first 4 bytes)
	eventSig := signature[:4]

	// Convert to fixed-size array for O(1) lookup
	var sigArray [4]byte
	copy(sigArray[:], eventSig)

	// O(1) lookup instead of O(n) iteration
	matchedEvent, exists := p.event4ByteSignatureCache[sigArray]
	if !exists {
		return fmt.Sprintf("unknown_event(0x%s)", hex.EncodeToString(eventSig)), nil
	}

	// Build event signature string for the matched event
	inputs := make([]string, len(matchedEvent.Inputs))
	for i, input := range matchedEvent.Inputs {
		inputs[i] = input.Type
	}
	eventSigStr := fmt.Sprintf("%s(%s)", matchedEvent.Name, strings.Join(inputs, ","))

	return eventSigStr, nil
}

// buildEventSignatureCache pre-computes event signature hashes for O(1) lookup
func (p *ABIProcessor) buildEventSignatureCache() {
	p.eventSignatureCache = make(map[[32]byte]*core.SmartContract_ABI_Entry)

	for _, entry := range p.abi.Entrys {
		if entry.Type != core.SmartContract_ABI_Entry_Event {
			continue
		}

		// Build event signature string
		inputs := make([]string, len(entry.Inputs))
		for i, input := range entry.Inputs {
			inputs[i] = input.Type
		}
		eventSigStr := fmt.Sprintf("%s(%s)", entry.Name, strings.Join(inputs, ","))

		// Calculate event signature hash (full 32 bytes)
		hasher := sha3.NewLegacyKeccak256()
		hasher.Write([]byte(eventSigStr))
		calculatedSig := hasher.Sum(nil)

		// Convert to fixed-size array for map key
		var sigArray [32]byte
		copy(sigArray[:], calculatedSig)
		p.eventSignatureCache[sigArray] = entry
	}
}

// buildEvent4ByteSignatureCache pre-computes 4-byte event signature hashes for O(1) lookup
func (p *ABIProcessor) buildEvent4ByteSignatureCache() {
	p.event4ByteSignatureCache = make(map[[4]byte]*core.SmartContract_ABI_Entry)

	for _, entry := range p.abi.Entrys {
		if entry.Type != core.SmartContract_ABI_Entry_Event {
			continue
		}

		// Build event signature string
		inputs := make([]string, len(entry.Inputs))
		for i, input := range entry.Inputs {
			inputs[i] = input.Type
		}
		eventSigStr := fmt.Sprintf("%s(%s)", entry.Name, strings.Join(inputs, ","))

		// Calculate event signature hash and take first 4 bytes
		hasher := sha3.NewLegacyKeccak256()
		hasher.Write([]byte(eventSigStr))
		calculatedSig := hasher.Sum(nil)

		// Convert to fixed-size array for map key (first 4 bytes)
		var sigArray [4]byte
		copy(sigArray[:], calculatedSig[:4])
		p.event4ByteSignatureCache[sigArray] = entry
	}
}

// decodeTopicValue decodes a topic value based on its type
func (p *ABIProcessor) decodeTopicValue(topic []byte, paramType string) string {
	switch paramType {
	case "address":
		ethaddr := eCommon.BytesToAddress(topic)
		tronAddr, err := types.NewAddressFromHex(ethaddr.Hex())
		if err != nil {
			return hex.EncodeToString(topic)
		}
		return tronAddr.String()
	case "uint256", "uint128", "uint64", "uint32", "uint16", "uint8",
		"int256", "int128", "int64", "int32", "int16", "int8":
		return new(big.Int).SetBytes(topic).String()
	case "bool":
		if len(topic) > 0 && topic[0] != 0 {
			return "true"
		}
		return "false"
	case "bytes32":
		return hex.EncodeToString(topic)
	default:
		return hex.EncodeToString(topic)
	}
}

// decodeEventData decodes non-indexed event parameters from data
func (p *ABIProcessor) decodeEventData(data []byte, params []*core.SmartContract_ABI_Entry_Param) ([]DecodedEventParameter, error) {
	// Create ethereum ABI arguments for decoding
	args := make([]eABI.Argument, len(params))
	for i, param := range params {
		abiType, err := eABI.NewType(param.Type, "", nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create ABI type for %s: %v", param.Type, err)
		}
		args[i] = eABI.Argument{
			Name: param.Name,
			Type: abiType,
		}
	}

	// Unpack the data
	values, err := eABI.Arguments(args).Unpack(data)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack event data: %v", err)
	}

	result := make([]DecodedEventParameter, len(params))
	for i, param := range params {
		var value string
		if i < len(values) {
			value = p.formatEventValue(values[i], param.Type)
		}

		result[i] = DecodedEventParameter{
			Name:    param.Name,
			Type:    param.Type,
			Value:   value,
			Indexed: false,
		}
	}

	return result, nil
}

// formatEventValue formats event value for display
func (p *ABIProcessor) formatEventValue(value interface{}, paramType string) string {
	switch paramType {
	case "address":
		if addr, ok := value.(eCommon.Address); ok {
			tronAddr, err := types.NewAddressFromHex(addr.Hex())
			if err != nil {
				return fmt.Sprintf("%v", value)
			}
			return tronAddr.String()
		}
	case "uint256", "uint128", "uint64", "uint32", "uint16", "uint8",
		"int256", "int128", "int64", "int32", "int16", "int8":
		if bigInt, ok := value.(*big.Int); ok {
			return bigInt.String()
		}
	case "bytes", "bytes32", "bytes16", "bytes8":
		if bytes, ok := value.([]byte); ok {
			return hex.EncodeToString(bytes)
		}
	case "string":
		if s, ok := value.(string); ok {
			return s
		}
	case "bool":
		if b, ok := value.(bool); ok {
			if b {
				return "true"
			}
			return "false"
		}
	}
	return fmt.Sprintf("%v", value)
}
