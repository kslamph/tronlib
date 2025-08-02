package smartcontract

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/kslamph/tronlib/pkg/utils"
)

// Contract represents a smart contract interface with high-level abstraction
type Contract struct {
	ABI     *core.SmartContract_ABI
	Address *types.Address
	Client  *client.Client

	// Utility instance for encoding/decoding
	abiProcessor *utils.ABIProcessor
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
			processor := utils.NewABIProcessor(nil)
			contractABI, err = processor.ParseABI(v)
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

		abiProcessor: utils.NewABIProcessor(contractABI),
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
	return client.GetContract(ctx, req)
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
	data, err := c.Encode(method, params...)
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

	return c.Client.TriggerContract(ctx, req)

}

// TriggerConstantContract queries a smart contract method and returns decoded result
func (c *Contract) TriggerConstantContract(ctx context.Context, owner *types.Address, method string, params ...interface{}) (interface{}, error) {

	if owner == nil {
		return nil, fmt.Errorf("owner address cannot be nil")
	}

	// Encode method call data
	data, err := c.Encode(method, params...)
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
	result, err := c.Client.TriggerConstantContract(ctx, req)
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
	// The constant result is typically a single byte slice, but it's returned as a slice of byte slices
	// We concatenate all the byte slices to form a single byte slice for decoding
	// This handles cases where the result might be split across multiple slices
	var concatenatedResult []byte
	for _, result := range constantResult {
		concatenatedResult = append(concatenatedResult, result...)
	}

	decoded, err := c.DecodeResult(method, concatenatedResult)
	if err != nil {
		return nil, fmt.Errorf("failed to decode result for method %s: %v", method, err)
	}

	return decoded, nil
}

type SimulateResult struct {
	Energy    int64
	APIResult *api.Return
	Logs      []*core.TransactionInfo_Log
}

func (c *Contract) Simulate(ctx context.Context, owner *types.Address, callValue int64, method string, params ...interface{}) (*SimulateResult, error) {

	if owner == nil {
		return nil, fmt.Errorf("owner address cannot be nil")
	}

	// Encode method call data
	data, err := c.Encode(method, params...)
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
	result, err := c.Client.TriggerConstantContract(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to trigger constant contract: %v", err)
	}

	if result == nil {
		return nil, fmt.Errorf("nil result from constant contract call")
	}
	return &SimulateResult{
		Energy:    result.GetEnergyUsed(),
		APIResult: result.GetResult(),
		Logs:      result.GetLogs(),
	}, nil

}

// Encode creates contract call data from method name and parameters
func (c *Contract) Encode(method string, params ...interface{}) ([]byte, error) {
	// Special handling for constructors (empty method name)
	if method == "" {
		paramTypes, err := c.abiProcessor.GetConstructorTypes(c.ABI)
		if err != nil {
			return nil, fmt.Errorf("failed to get constructor types: %v", err)
		}
		// We need to create a temporary ABIProcessor to encode parameters
		// since the GetConstructorTypes doesn't return the ABI
		tempProcessor := utils.NewABIProcessor(c.ABI)
		// For constructors, we need to pass empty method name and get input types
		return tempProcessor.EncodeMethod("", paramTypes, params)
	}

	// Get method parameter types from ABI
	inputTypes, _, err := c.abiProcessor.GetMethodTypes(method)
	if err != nil {
		return nil, fmt.Errorf("failed to get method types: %v", err)
	}

	return c.abiProcessor.EncodeMethod(method, inputTypes, params)
}

// DecodeResult decodes contract call result
// Return type changed to interface{} to allow single value returns like *big.Int, []byte, int64, uint64, *types.Address, etc.
func (c *Contract) DecodeResult(method string, data []byte) (interface{}, error) {
	// Get method output types from ABI
	_, outputTypes, err := c.abiProcessor.GetMethodTypes(method)
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

	// Decode the result using the abiProcessor's DecodeResult method
	decoded, err := c.abiProcessor.DecodeResult(data, outputs)
	if err != nil {
		return nil, fmt.Errorf("failed to decode result: %v", err)
	}

	return decoded, nil
}

// DecodeInput decodes contract input data
func (c *Contract) DecodeInput(data []byte) (*utils.DecodedInput, error) {
	return c.abiProcessor.DecodeInputData(data, c.ABI)
}

// DecodeEventLog decodes an event log
func (c *Contract) DecodeEventLog(topics [][]byte, data []byte) (*utils.DecodedEvent, error) {
	return c.abiProcessor.DecodeEventLog(topics, data)
}

// DecodeEventSignature decodes event signature bytes
func (c *Contract) DecodeEventSignature(signature []byte) (string, error) {
	return c.abiProcessor.DecodeEventSignature(signature)
}
