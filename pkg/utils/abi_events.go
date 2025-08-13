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

package utils

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	eABI "github.com/ethereum/go-ethereum/accounts/abi"
	eCommon "github.com/ethereum/go-ethereum/common"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
	"golang.org/x/crypto/sha3"
)

// DecodeEventLog decodes a single event log using the processor's ABI.
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

// DecodeEventSignature decodes 4- or 32-byte event signatures to the canonical
// signature string if known.
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

// buildEventSignatureCache pre-computes full 32-byte event signatures for O(1) lookup.
func (p *ABIProcessor) buildEventSignatureCache() {
	p.eventSignatureCache = make(map[[32]byte]*core.SmartContract_ABI_Entry)

	for _, entry := range p.abi.Entrys {
		if entry.Type != core.SmartContract_ABI_Entry_Event {
			continue
		}

		// Build event signature string
		args := make([]eABI.Argument, len(entry.Inputs))
		inputs := make([]string, len(entry.Inputs))
		for i, input := range entry.Inputs {
			// ensure types valid for ABI parsing too (matching original behavior of using names/types)
			inputs[i] = input.Type
			// keep eABI import used so file compiles; arg slice not used further but harmless
			_ = eABI.Argument{Name: input.Name}
			args[i] = eABI.Argument{Name: input.Name}
		}
		_ = args

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

// buildEvent4ByteSignatureCache pre-computes 4-byte event signatures for O(1) lookup.
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

// decodeTopicValue decodes a topic value based on its ABI type.
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
