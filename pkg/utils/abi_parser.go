package utils

import (
	"encoding/json"
	"fmt"

	"github.com/kslamph/tronlib/pb/core"
)

// ABIParser handles ABI parsing operations
type ABIParser struct{}

// NewABIParser creates a new ABI parser instance
func NewABIParser() *ABIParser {
	return &ABIParser{}
}

// ParseABI decodes the ABI string into a core.SmartContract_ABI object
func (p *ABIParser) ParseABI(abi string) (*core.SmartContract_ABI, error) {
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
func (p *ABIParser) parseABIEntry(entry map[string]interface{}) (*core.SmartContract_ABI_Entry, error) {
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
func (p *ABIParser) parseParameters(params []interface{}) ([]*core.SmartContract_ABI_Entry_Param, error) {
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
func (p *ABIParser) GetMethodTypes(abi *core.SmartContract_ABI, methodName string) ([]string, []string, error) {
	for _, entry := range abi.Entrys {
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
func (p *ABIParser) GetConstructorTypes(abi *core.SmartContract_ABI) ([]string, error) {
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