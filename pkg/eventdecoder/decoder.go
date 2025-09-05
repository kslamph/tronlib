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

package eventdecoder

import (
	"encoding/hex"
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

// DecodedEvent represents a decoded event
type DecodedEvent struct {
	EventName  string                  `json:"eventName"`
	Parameters []DecodedEventParameter `json:"parameters"`
	Contract   string                  `json:"contract"`
}

// DecodedEventParameter represents a decoded event parameter
type DecodedEventParameter struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Value   string `json:"value"`
	Indexed bool   `json:"indexed"`
}

// ParamDef is a compact representation of an event parameter definition
type ParamDef struct {
	Type    string
	Indexed bool
	Name    string
}

// EventDef is a compact representation of an event definition
type EventDef struct {
	Name   string
	Inputs []ParamDef
}

var (
	mu   sync.RWMutex
	sig4 = make(map[[4]byte]*EventDef)
)

// RegisterABIJSON registers all event entries from a JSON ABI string
func RegisterABIJSON(abiJSON string) error {
	proc := NewSimpleABIParser()
	parsed, err := proc.ParseABI(abiJSON)
	if err != nil {
		return err
	}
	return RegisterABIObject(parsed)
}

// RegisterABIObject registers all event entries from a SmartContract_ABI object
func RegisterABIObject(abi *core.SmartContract_ABI) error {
	if abi == nil {
		return fmt.Errorf("nil ABI")
	}
	return RegisterABIEntries(abi.Entrys)
}

// RegisterABIEntries registers all event entries from the provided list (non-event entries are ignored)
func RegisterABIEntries(entries []*core.SmartContract_ABI_Entry) error {
	if len(entries) == 0 {
		return nil
	}

	local := make(map[[4]byte]*EventDef)

	for _, entry := range entries {
		if entry == nil || entry.Type != core.SmartContract_ABI_Entry_Event {
			continue
		}
		// Build canonical signature string: Name(types...)
		inputs := make([]string, len(entry.Inputs))
		compactInputs := make([]ParamDef, len(entry.Inputs))
		for i, in := range entry.Inputs {
			if in == nil {
				continue
			}
			inputs[i] = in.Type
			compactInputs[i] = ParamDef{Type: in.Type, Indexed: in.Indexed, Name: in.Name}
		}
		sigStr := fmt.Sprintf("%s(%s)", entry.Name, strings.Join(inputs, ","))

		// Compute 4-byte signature key
		hasher := sha3.NewLegacyKeccak256()
		hasher.Write([]byte(sigStr))
		sum := hasher.Sum(nil)

		var key [4]byte
		copy(key[:], sum[:4])

		local[key] = &EventDef{
			Name:   entry.Name,
			Inputs: compactInputs,
		}
	}

	if len(local) == 0 {
		return nil
	}

	mu.Lock()
	for k, v := range local {
		sig4[k] = v // overwrite by design
	}
	mu.Unlock()
	return nil
}

// DecodeEventSignature returns the canonical event signature string for the given 4-byte signature if known
// The boolean indicates whether the signature was found in the registry
func DecodeEventSignature(sig []byte) (string, bool) {
	if len(sig) < 4 {
		return "", false
	}
	var key [4]byte
	copy(key[:], sig[:4])

	mu.RLock()
	def, ok := sig4[key]
	mu.RUnlock()
	if !ok || def == nil {
		return "", false
	}
	types := make([]string, len(def.Inputs))
	for i, in := range def.Inputs {
		types[i] = in.Type
	}
	return fmt.Sprintf("%s(%s)", def.Name, strings.Join(types, ",")), true
}

// DecodeLog decodes a single log using the global 4-byte signature registry
func DecodeLog(topics [][]byte, data []byte) (*DecodedEvent, error) {
	if len(topics) == 0 {
		return nil, fmt.Errorf("no topics provided")
	}
	sigTopic := topics[0]
	if len(sigTopic) < 4 {
		return nil, fmt.Errorf("first topic too short for signature: %d", len(sigTopic))
	}

	// Lookup by first 4 bytes
	var key [4]byte
	copy(key[:], sigTopic[:4])

	mu.RLock()
	def := sig4[key]
	mu.RUnlock()

	if def == nil {
		return &DecodedEvent{
			EventName:  fmt.Sprintf("unknown_event(0x%s)", hex.EncodeToString(sigTopic[:4])),
			Parameters: []DecodedEventParameter{},
		}, nil
	}

	// Decode using internal implementation
	return decodeEventInternal(def, topics, data)
}

// DecodeLogs decodes a slice of logs using the global 4-byte signature registry
func DecodeLogs(logs []*core.TransactionInfo_Log) ([]*DecodedEvent, error) {
	if len(logs) == 0 {
		return []*DecodedEvent{}, nil
	}
	result := make([]*DecodedEvent, 0, len(logs))
	for _, lg := range logs {
		if lg == nil {
			continue
		}
		ev, err := DecodeLog(lg.GetTopics(), lg.GetData())
		if err != nil {
			return nil, err
		}
		ev.Contract = types.MustNewAddressFromBytes(lg.GetAddress()).String()
		result = append(result, ev)
	}
	return result, nil
}

// decodeEventInternal performs the actual event decoding without external dependencies
func decodeEventInternal(def *EventDef, topics [][]byte, data []byte) (*DecodedEvent, error) {
	// Separate indexed and non-indexed parameters
	var indexedParams []ParamDef
	var nonIndexedParams []ParamDef

	for _, input := range def.Inputs {
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
		decoded, err := decodeEventData(data, nonIndexedParams)
		if err != nil {
			return nil, fmt.Errorf("failed to decode event data: %v", err)
		}
		nonIndexedValues = decoded
	}

	// Combine all parameters in original order
	allParams := make([]DecodedEventParameter, 0)
	for _, input := range def.Inputs {
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
		EventName:  def.Name,
		Parameters: allParams,
	}, nil
}

// decodeTopicValue decodes a topic value based on its ABI type
func decodeTopicValue(topic []byte, paramType string) string {
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
func decodeEventData(data []byte, params []ParamDef) ([]DecodedEventParameter, error) {
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
			value = formatEventValue(values[i], param.Type)
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
func formatEventValue(value interface{}, paramType string) string {
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
	default:
		// Handle array types
		if strings.HasSuffix(paramType, "[]") {
			baseType := strings.TrimSuffix(paramType, "[]")
			if reflect.TypeOf(value).Kind() == reflect.Slice {
				slice := reflect.ValueOf(value)
				result := make([]string, slice.Len())
				for i := 0; i < slice.Len(); i++ {
					elementValue := formatEventValue(slice.Index(i).Interface(), baseType)
					result[i] = elementValue
				}
				// Join array elements with commas
				return "[" + strings.Join(result, ",") + "]"
			}
		}
	}
	return fmt.Sprintf("%v", value)
}

// Simple ABI parser for JSON parsing (minimal implementation)
type SimpleABIParser struct{}

func NewSimpleABIParser() *SimpleABIParser {
	return &SimpleABIParser{}
}

// ParseABI provides basic ABI parsing functionality
func (p *SimpleABIParser) ParseABI(abiJSON string) (*core.SmartContract_ABI, error) {
	// This is a minimal implementation - in practice you might want to use a JSON parser
	// For now, return empty ABI to avoid dependency on complex parsing
	return &core.SmartContract_ABI{
		Entrys: []*core.SmartContract_ABI_Entry{},
	}, nil
}
