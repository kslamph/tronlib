package smartcontract

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/client/lowlevel"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/kslamph/tronlib/pkg/utils"
)

// Contract represents a smart contract interface with high-level abstraction
type Contract struct {
	ABI     *core.SmartContract_ABI
	Address *types.Address

	// Utility instances for encoding/decoding
	encoder      *utils.ABIEncoder
	decoder      *utils.ABIDecoder
	eventDecoder *utils.EventDecoder
	parser       *utils.ABIParser
}

// NewContract creates a new contract instance from various input types
// abiOrClient can be:
// - string: ABI JSON string
// - *core.SmartContract_ABI: Parsed ABI object
// - *client.Client: Client to retrieve contract data from network
func NewContract(address any, abiOrClient interface{}) (*Contract, error) {
	if address == "" {
		return nil, fmt.Errorf("empty contract address")
	}

	// Convert address to bytes
	addr, err := types.NewAddress(address)
	if err != nil {
		return nil, fmt.Errorf("failed to parse address: %v", err)
	}

	var abi *core.SmartContract_ABI

	switch v := abiOrClient.(type) {
	case string:
		// Handle ABI string
		if v == "" {
			return nil, fmt.Errorf("empty ABI string")
		}
		parser := utils.NewABIParser()
		abi, err = parser.ParseABI(v)
		if err != nil {
			return nil, fmt.Errorf("failed to decode ABI: %v", err)
		}

	case *core.SmartContract_ABI:
		// Handle parsed ABI object
		if v == nil {
			return nil, fmt.Errorf("ABI cannot be nil")
		}
		abi = v

	case *client.Client:
		// Handle client - retrieve contract from network
		if v == nil {
			return nil, fmt.Errorf("client cannot be nil")
		}
		contractInfo, err := getContractFromNetwork(context.Background(), v, addr.String())
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve contract from network: %v", err)
		}
		if contractInfo.GetAbi() == nil {
			return nil, fmt.Errorf("contract has no ABI available on network")
		}
		abi = contractInfo.GetAbi()

	default:
		return nil, fmt.Errorf("abiOrClient must be string, *core.SmartContract_ABI, or *client.Client, got %T", abiOrClient)
	}

	return &Contract{
		ABI:     abi,
		Address: addr,

		encoder:      utils.NewABIEncoder(),
		decoder:      utils.NewABIDecoder(),
		eventDecoder: utils.NewEventDecoder(abi),
		parser:       utils.NewABIParser(),
	}, nil
}

// getContractFromNetwork retrieves smart contract information from the network
func getContractFromNetwork(ctx context.Context, client *client.Client, contractAddress string) (*core.SmartContract, error) {
	// Handle both hex and base58 addresses
	var contractAddressBytes []byte
	var err error

	// Try to parse as base58 first (standard TRON address)
	addr, err := types.NewAddress(contractAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to parse address: %v", err)
	}
	contractAddressBytes = addr.Bytes()

	req := &api.BytesMessage{
		Value: contractAddressBytes,
	}
	return lowlevel.GetContract(client, ctx, req)
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
