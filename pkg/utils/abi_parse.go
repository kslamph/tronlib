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
	"encoding/json"
	"fmt"

	"github.com/kslamph/tronlib/pb/core"
)

// NewABIProcessor creates an ABIProcessor bound to the provided ABI. The
// processor exposes helpers to parse ABI JSON, encode inputs, decode outputs,
// and decode events.
func NewABIProcessor(abi *core.SmartContract_ABI) *ABIProcessor {
	return &ABIProcessor{
		abi: abi,
	}
}

// ParseABI decodes a standard ABI JSON string into a *core.SmartContract_ABI.
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
