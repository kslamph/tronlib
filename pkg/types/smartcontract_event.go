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
	Name    string `json:"name"`
	Type    string `json:"type"`
	Value   string `json:"value"`
	Indexed bool   `json:"indexed"`
}

// buildEventSignatureCache pre-computes event signature hashes for O(1) lookup
func (c *Contract) buildEventSignatureCache() {
	c.eventSignatureCache = make(map[[32]byte]*core.SmartContract_ABI_Entry)

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

		// Convert to fixed-size array for map key
		var sigArray [32]byte
		copy(sigArray[:], calculatedSig)
		c.eventSignatureCache[sigArray] = entry
	}
}

// buildEvent4ByteSignatureCache pre-computes 4-byte event signature hashes for O(1) lookup
func (c *Contract) buildEvent4ByteSignatureCache() {
	c.event4ByteSignatureCache = make(map[[4]byte]*core.SmartContract_ABI_Entry)

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

		// Calculate event signature hash and take first 4 bytes
		hasher := sha3.NewLegacyKeccak256()
		hasher.Write([]byte(eventSigStr))
		calculatedSig := hasher.Sum(nil)

		// Convert to fixed-size array for map key (first 4 bytes)
		var sigArray [4]byte
		copy(sigArray[:], calculatedSig[:4])
		c.event4ByteSignatureCache[sigArray] = entry
	}
}

// DecodeEventLog decodes an event log and returns the event details
func (c *Contract) DecodeEventLog(topics [][]byte, data []byte) (*DecodedEvent, error) {
	if len(topics) == 0 {
		return nil, fmt.Errorf("no topics provided")
	}

	// Ensure cache is built (thread-safe, one-time only)
	c.eventCacheOnce.Do(c.buildEventSignatureCache)

	// First topic is the event signature (32 bytes)
	eventSignature := topics[0]

	// Convert to fixed-size array for O(1) lookup
	var sigArray [32]byte
	copy(sigArray[:], eventSignature)

	// O(1) lookup instead of O(n) iteration
	matchedEvent, exists := c.eventSignatureCache[sigArray]

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
			var value string
			if i < len(values) {
				formattedValue := formatDecodedValue(values[i], param.Type)
				// Handle different return types from formatDecodedValue
				switch v := formattedValue.(type) {
				case string:
					value = v
				case []interface{}:
					// Convert slice to string representation
					value = fmt.Sprintf("%v", v)
				default:
					value = fmt.Sprintf("%v", v)
				}
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
func decodeTopicValue(topic []byte, paramType string) string {
	switch paramType {
	case "address":
		ethaddr := eCommon.BytesToAddress(topic)
		addrBase58 := MustNewAddressFromEVMHex(ethaddr.Hex())
		return addrBase58.String()
	case "uint256", "uint128", "uint64", "uint32", "uint16", "uint8":
		return new(big.Int).SetBytes(topic).String()
	case "int256", "int128", "int64", "int32", "int16", "int8":
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

// DecodeEventSignature decodes event signature bytes and returns the event name
func (c *Contract) DecodeEventSignature(signature []byte) (string, error) {
	if len(signature) < 4 {
		return "", fmt.Errorf("event signature too short, need at least 4 bytes")
	}

	// Ensure 4-byte cache is built (thread-safe, one-time only)
	c.event4ByteCacheOnce.Do(c.buildEvent4ByteSignatureCache)

	// Extract event signature (first 4 bytes)
	eventSig := signature[:4]

	// Convert to fixed-size array for O(1) lookup
	var sigArray [4]byte
	copy(sigArray[:], eventSig)

	// O(1) lookup instead of O(n) iteration
	matchedEvent, exists := c.event4ByteSignatureCache[sigArray]

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
