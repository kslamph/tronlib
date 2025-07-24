package smartcontract

import (
	"fmt"

	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/kslamph/tronlib/pkg/utils"
)

// Contract represents a smart contract interface with high-level abstraction
type Contract struct {
	ABI          *core.SmartContract_ABI
	Address      string
	AddressBytes []byte

	// Utility instances for encoding/decoding
	encoder      *utils.ABIEncoder
	decoder      *utils.ABIDecoder
	eventDecoder *utils.EventDecoder
	parser       *utils.ABIParser
}

// NewContract creates a new contract instance from ABI string and address
func NewContract(abi string, address string) (*Contract, error) {
	if abi == "" {
		return nil, fmt.Errorf("empty ABI string")
	}
	if address == "" {
		return nil, fmt.Errorf("empty contract address")
	}
	
	parser := utils.NewABIParser()
	decodedABI, err := parser.ParseABI(abi)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ABI: %v", err)
	}

	// Convert address to bytes
	addr, err := types.NewAddressFromBase58(address)
	if err != nil {
		return nil, fmt.Errorf("failed to parse address: %v", err)
	}

	return &Contract{
		ABI:          decodedABI,
		Address:      address,
		AddressBytes: addr.Bytes(),
		encoder:      utils.NewABIEncoder(),
		decoder:      utils.NewABIDecoder(),
		eventDecoder: utils.NewEventDecoder(decodedABI),
		parser:       parser,
	}, nil
}

// NewContractFromABI creates a new contract instance from ABI object and address
func NewContractFromABI(abi *core.SmartContract_ABI, address string) (*Contract, error) {
	if abi == nil {
		return nil, fmt.Errorf("ABI cannot be nil")
	}
	if address == "" {
		return nil, fmt.Errorf("empty contract address")
	}
	
	addr, err := types.NewAddressFromBase58(address)
	if err != nil {
		return nil, fmt.Errorf("failed to parse address: %v", err)
	}
	
	return &Contract{
		ABI:          abi,
		Address:      address,
		AddressBytes: addr.Bytes(),
		encoder:      utils.NewABIEncoder(),
		decoder:      utils.NewABIDecoder(),
		eventDecoder: utils.NewEventDecoder(abi),
		parser:       utils.NewABIParser(),
	}, nil
}

// EncodeInput creates contract call data from method name and parameters
func (c *Contract) EncodeInput(method string, params ...interface{}) ([]byte, error) {
	// Special handling for constructors (empty method name)
	if method == "" {
		paramTypes, err := c.parser.GetConstructorTypes(c.ABI)
		if err != nil {
			return nil, fmt.Errorf("failed to get constructor types: %v", err)
		}
		return c.encoder.EncodeParameters(paramTypes, params)
	}

	// Get method parameter types from ABI
	inputTypes, _, err := c.parser.GetMethodTypes(c.ABI, method)
	if err != nil {
		return nil, fmt.Errorf("failed to get method types: %v", err)
	}

	return c.encoder.EncodeMethod(method, inputTypes, params)
}

// DecodeResult decodes contract call result
func (c *Contract) DecodeResult(method string, data []byte) (interface{}, error) {
	// Get method output types from ABI
	_, outputTypes, err := c.parser.GetMethodTypes(c.ABI, method)
	if err != nil {
		return nil, fmt.Errorf("failed to get method types: %v", err)
	}

	// Convert output types to ABI entry params
	outputs := make([]*core.SmartContract_ABI_Entry_Param, len(outputTypes))
	for i, outputType := range outputTypes {
		outputs[i] = &core.SmartContract_ABI_Entry_Param{
			Type: outputType,
			Name: fmt.Sprintf("output%d", i),
		}
	}

	return c.decoder.DecodeResult(data, outputs)
}

// DecodeInputData decodes contract input data
func (c *Contract) DecodeInputData(data []byte) (*utils.DecodedInput, error) {
	return c.decoder.DecodeInputData(data, c.ABI)
}

// DecodeEventLog decodes an event log
func (c *Contract) DecodeEventLog(topics [][]byte, data []byte) (*utils.DecodedEvent, error) {
	return c.eventDecoder.DecodeEventLog(topics, data)
}

// DecodeEventSignature decodes event signature bytes
func (c *Contract) DecodeEventSignature(signature []byte) (string, error) {
	return c.eventDecoder.DecodeEventSignature(signature)
}