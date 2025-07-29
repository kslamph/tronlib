package utils

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"sync"

	eABI "github.com/ethereum/go-ethereum/accounts/abi"
	eCommon "github.com/ethereum/go-ethereum/common"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
	"golang.org/x/crypto/sha3"
)

// EventDecoder handles smart contract event decoding operations
type EventDecoder struct {
	abi *core.SmartContract_ABI

	// Event signature caches using sync.Once pattern
	eventCacheOnce      sync.Once
	eventSignatureCache map[[32]byte]*core.SmartContract_ABI_Entry

	event4ByteCacheOnce      sync.Once
	event4ByteSignatureCache map[[4]byte]*core.SmartContract_ABI_Entry
}

// NewEventDecoder creates a new event decoder instance
func NewEventDecoder(abi *core.SmartContract_ABI) *EventDecoder {
	return &EventDecoder{
		abi: abi,
	}
}

// DecodeEventLog decodes an event log
func (e *EventDecoder) DecodeEventLog(topics [][]byte, data []byte) (*DecodedEvent, error) {
	if len(topics) == 0 {
		return nil, fmt.Errorf("no topics provided")
	}

	// Ensure cache is built (thread-safe, one-time only)
	e.eventCacheOnce.Do(e.buildEventSignatureCache)

	// First topic is the event signature (32 bytes)
	eventSignature := topics[0]

	// Convert to fixed-size array for O(1) lookup
	var sigArray [32]byte
	copy(sigArray[:], eventSignature)

	// O(1) lookup instead of O(n) iteration
	matchedEvent, exists := e.eventSignatureCache[sigArray]
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

		value := e.decodeTopicValue(topics[i+1], param.Type)
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
		decoded, err := e.decodeEventData(data, nonIndexedParams)
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
func (e *EventDecoder) DecodeEventSignature(signature []byte) (string, error) {
	if len(signature) < 4 {
		return "", fmt.Errorf("event signature too short, need at least 4 bytes")
	}

	// Ensure 4-byte cache is built (thread-safe, one-time only)
	e.event4ByteCacheOnce.Do(e.buildEvent4ByteSignatureCache)

	// Extract event signature (first 4 bytes)
	eventSig := signature[:4]

	// Convert to fixed-size array for O(1) lookup
	var sigArray [4]byte
	copy(sigArray[:], eventSig)

	// O(1) lookup instead of O(n) iteration
	matchedEvent, exists := e.event4ByteSignatureCache[sigArray]
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
func (e *EventDecoder) buildEventSignatureCache() {
	e.eventSignatureCache = make(map[[32]byte]*core.SmartContract_ABI_Entry)

	for _, entry := range e.abi.Entrys {
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
		e.eventSignatureCache[sigArray] = entry
	}
}

// buildEvent4ByteSignatureCache pre-computes 4-byte event signature hashes for O(1) lookup
func (e *EventDecoder) buildEvent4ByteSignatureCache() {
	e.event4ByteSignatureCache = make(map[[4]byte]*core.SmartContract_ABI_Entry)

	for _, entry := range e.abi.Entrys {
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
		e.event4ByteSignatureCache[sigArray] = entry
	}
}

// decodeTopicValue decodes a topic value based on its type
func (e *EventDecoder) decodeTopicValue(topic []byte, paramType string) string {
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
func (e *EventDecoder) decodeEventData(data []byte, params []*core.SmartContract_ABI_Entry_Param) ([]DecodedEventParameter, error) {
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
			value = e.formatEventValue(values[i], param.Type)
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
func (e *EventDecoder) formatEventValue(value interface{}, paramType string) string {
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
