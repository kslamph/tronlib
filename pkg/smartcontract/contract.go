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
	Client  *client.Client

	// Utility instances for encoding/decoding
	encoder      *utils.ABIEncoder
	decoder      *utils.ABIDecoder
	eventDecoder *utils.EventDecoder
	parser       *utils.ABIParser
}

// NewContract creates a new contract instance
// tronClient: TRON client instance
// address: Contract address
// abi: Optional ABI - can be:
//   - string: ABI JSON string
//   - *core.SmartContract_ABI: Parsed ABI object
//   - omitted: ABI will be retrieved from network
func NewContract(tronClient *client.Client, address *types.Address, abi ...any) (*Contract, error) {
	if tronClient == nil {
		return nil, fmt.Errorf("tron client cannot be nil")
	}

	if address == nil {
		return nil, fmt.Errorf("contract address cannot be nil")
	}

	var contractABI *core.SmartContract_ABI
	var err error

	// Process ABI parameter
	if len(abi) == 0 {
		// No ABI provided - retrieve from network
		contractInfo, err := getContractFromNetwork(context.Background(), tronClient, address.String())
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve contract from network: %v", err)
		}
		if contractInfo.GetAbi() == nil {
			return nil, fmt.Errorf("contract has no ABI available on network")
		}
		contractABI = contractInfo.GetAbi()
	} else if len(abi) == 1 {
		// ABI provided - process based on type
		switch v := abi[0].(type) {
		case string:
			// Handle ABI JSON string
			if v == "" {
				return nil, fmt.Errorf("empty ABI string")
			}
			parser := utils.NewABIParser()
			contractABI, err = parser.ParseABI(v)
			if err != nil {
				return nil, fmt.Errorf("failed to parse ABI string: %v", err)
			}

		case *core.SmartContract_ABI:
			// Handle parsed ABI object
			if v == nil {
				return nil, fmt.Errorf("ABI cannot be nil")
			}
			contractABI = v

		default:
			return nil, fmt.Errorf("ABI must be string or *core.SmartContract_ABI, got %T", v)
		}
	} else {
		return nil, fmt.Errorf("too many ABI arguments provided, expected 0 or 1")
	}

	return &Contract{
		ABI:     contractABI,
		Address: address,
		Client:  tronClient,

		encoder:      utils.NewABIEncoder(),
		decoder:      utils.NewABIDecoder(),
		eventDecoder: utils.NewEventDecoder(contractABI),
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

// TriggerSmartContract builds a smart contract transaction, to be signed and broadcasted
func (c *Contract) TriggerSmartContract(ctx context.Context, owner *types.Address, callValue int64, method string, params ...interface{}) (*api.TransactionExtention, error) {

	if owner == nil {
		return nil, fmt.Errorf("owner address cannot be nil")
	}
	if callValue < 0 {
		return nil, fmt.Errorf("call value cannot be negative")
	}

	// Encode method call data
	data, err := c.EncodeInput(method, params...)
	if err != nil {
		return nil, fmt.Errorf("failed to encode input for method %s: %v", method, err)
	}

	// Create trigger smart contract request
	req := &core.TriggerSmartContract{
		OwnerAddress:    owner.Bytes(),
		ContractAddress: c.Address.Bytes(),
		Data:            data,
		CallValue:       callValue,
		CallTokenValue:  0,
		TokenId:         0,
	}

	return lowlevel.TriggerContract(c.Client, ctx, req)
}

// TriggerConstantContract queries a smart contract method and returns decoded result
func (c *Contract) TriggerConstantContract(ctx context.Context, owner *types.Address, method string, params ...interface{}) (interface{}, error) {

	if owner == nil {
		return nil, fmt.Errorf("owner address cannot be nil")
	}

	// Encode method call data
	data, err := c.EncodeInput(method, params...)
	if err != nil {
		return nil, fmt.Errorf("failed to encode input for method %s: %v", method, err)
	}

	// Create trigger smart contract request
	req := &core.TriggerSmartContract{
		OwnerAddress:    owner.Bytes(),
		ContractAddress: c.Address.Bytes(),
		Data:            data,
		CallValue:       0,
	}

	// Call the constant contract
	result, err := lowlevel.TriggerConstantContract(c.Client, ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to trigger constant contract: %v", err)
	}

	if result == nil {
		return nil, fmt.Errorf("nil result from constant contract call")
	}

	// Get the constant result bytes
	constantResult := result.GetConstantResult()
	if len(constantResult) == 0 {
		return nil, fmt.Errorf("empty constant result")
	}

	// Decode the result using the contract's DecodeResult method
	decoded, err := c.DecodeResult(method, constantResult)
	if err != nil {
		return nil, fmt.Errorf("failed to decode result for method %s: %v", method, err)
	}

	return decoded, nil
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
func (c *Contract) DecodeResult(method string, data [][]byte) (interface{}, error) {
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

	// If we have a single result item, decode it directly
	if len(data) == 1 {
		return c.decoder.DecodeResult(data[0], outputs)
	}

	// For multiple result items, decode each one
	results := make([]interface{}, len(data))
	for i, resultData := range data {
		if i < len(outputs) {
			decoded, err := c.decoder.DecodeResult(resultData, []*core.SmartContract_ABI_Entry_Param{outputs[i]})
			if err != nil {
				return nil, fmt.Errorf("failed to decode result %d: %v", i, err)
			}
			results[i] = decoded
		}
	}

	// If only one output type expected, return the single result
	if len(outputTypes) == 1 && len(results) > 0 {
		return results[0], nil
	}

	return results, nil
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
