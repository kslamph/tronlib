package types

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	eABI "github.com/ethereum/go-ethereum/accounts/abi"
	eCommon "github.com/ethereum/go-ethereum/common"
	"github.com/kslamph/tronlib/pb/core"
	"golang.org/x/crypto/sha3"
)

// DecodedEvent represents a decoded event
type DecodedEvent struct {
	EventName  string                  `json:"eventName"`
	Parameters []DecodedEventParameter `json:"parameters"`
}

// DecodedEventParameter represents a decoded event parameter
type DecodedEventParameter struct {
	Name    string      `json:"name"`
	Type    string      `json:"type"`
	Value   interface{} `json:"value"`
	Indexed bool        `json:"indexed"`
}

// DecodeEventLog decodes an event log and returns the event details
func (c *Contract) DecodeEventLog(topics [][]byte, data []byte) (*DecodedEvent, error) {
	if len(topics) == 0 {
		return nil, fmt.Errorf("no topics provided")
	}

	// First topic is the event signature (32 bytes)
	eventSignature := topics[0]

	// Find the matching event in ABI
	var matchedEvent *core.SmartContract_ABI_Entry
	for _, entry := range c.ABI.Entrys {
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

		// Compare signatures - compare full 32-byte hashes
		if hex.EncodeToString(calculatedSig) == hex.EncodeToString(eventSignature) {
			matchedEvent = entry
			break
		}
	}

	if matchedEvent == nil {
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

		value := decodeTopicValue(topics[i+1], param.Type)
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
		// Create ethereum ABI arguments for decoding
		args := make([]eABI.Argument, len(nonIndexedParams))
		for i, param := range nonIndexedParams {
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

		nonIndexedValues = make([]DecodedEventParameter, len(nonIndexedParams))
		for i, param := range nonIndexedParams {
			var value interface{}
			if i < len(values) {
				value = formatDecodedValue(values[i], param.Type)
			}

			nonIndexedValues[i] = DecodedEventParameter{
				Name:    param.Name,
				Type:    param.Type,
				Value:   value,
				Indexed: false,
			}
		}
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

// decodeTopicValue decodes a topic value based on its type
func decodeTopicValue(topic []byte, paramType string) interface{} {
	switch paramType {
	case "address":
		return eCommon.BytesToAddress(topic).Hex()
	case "uint256", "uint128", "uint64", "uint32", "uint16", "uint8":
		return new(big.Int).SetBytes(topic).String()
	case "int256", "int128", "int64", "int32", "int16", "int8":
		return new(big.Int).SetBytes(topic).String()
	case "bool":
		return len(topic) > 0 && topic[0] != 0
	case "bytes32":
		return hex.EncodeToString(topic)
	default:
		return hex.EncodeToString(topic)
	}
}

// DecodeEventSignature decodes event signature bytes and returns the event name
func (c *Contract) DecodeEventSignature(signature []byte) (string, error) {
	if len(signature) < 4 {
		return "", fmt.Errorf("event signature too short, need at least 4 bytes")
	}

	// Extract event signature (first 4 bytes)
	eventSig := signature[:4]

	// Find matching event in ABI by comparing event signatures
	for _, entry := range c.ABI.Entrys {
		if entry.Type != core.SmartContract_ABI_Entry_Event {
			continue
		}

		// Build event signature string
		inputs := make([]string, len(entry.Inputs))
		for i, input := range entry.Inputs {
			inputs[i] = input.Type
		}
		eventSigStr := fmt.Sprintf("%s(%s)", entry.Name, strings.Join(inputs, ","))

		// Calculate event ID (same as method ID for events)
		hasher := sha3.NewLegacyKeccak256()
		hasher.Write([]byte(eventSigStr))
		calculatedSig := hasher.Sum(nil)[:4]

		// Compare signatures
		if hex.EncodeToString(calculatedSig) == hex.EncodeToString(eventSig) {
			return eventSigStr, nil
		}
	}

	return fmt.Sprintf("unknown_event(0x%s)", hex.EncodeToString(eventSig)), nil
}
