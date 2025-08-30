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

package smartcontract

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client/lowlevel"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/kslamph/tronlib/pkg/utils"
)

// contractClient defines the minimal dependency required by Contract.
// Satisfied by *client.Client.
type contractClient interface {
	lowlevel.ConnProvider
}

// Instance represents a high-level client bound to a deployed smart contract
// address and ABI, providing helpers for encoding inputs, invoking methods,
// constant calls, and decoding results and events.
type Instance struct {
	ABI     *core.SmartContract_ABI
	Address *types.Address
	Client  contractClient

	// Utility instance for encoding/decoding
	abiProcessor *utils.ABIProcessor
}

// NewInstance constructs a contract instance for the given address using the
// provided TRON client. The ABI can be omitted to fetch from the network, or
// supplied as either a JSON string or a *core.SmartContract_ABI.
func NewInstance(tronClient contractClient, contractAddress *types.Address, abi ...any) (*Instance, error) {
	if tronClient == nil {
		return nil, fmt.Errorf("tron client cannot be nil")
	}

	if contractAddress == nil {
		return nil, fmt.Errorf("contract address cannot be nil")
	}

	var contractABI *core.SmartContract_ABI
	var err error

	// Process ABI parameter
	if len(abi) == 0 {
		// No ABI provided - retrieve from network
		ctx, cancel := context.WithTimeout(context.Background(), tronClient.GetTimeout())
		defer cancel()
		contractInfo, err := getContractFromNetwork(ctx, tronClient, contractAddress)
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

	return &Instance{
		ABI:     contractABI,
		Address: contractAddress,
		Client:  tronClient,

		abiProcessor: utils.NewABIProcessor(contractABI),
	}, nil
}

// getContractFromNetwork retrieves smart contract information from the network
func getContractFromNetwork(ctx context.Context, client contractClient, contractAddress *types.Address) (*core.SmartContract, error) {
	if contractAddress == nil {
		return nil, fmt.Errorf("contract address cannot be nil")
	}

	req := &api.BytesMessage{Value: contractAddress.Bytes()}
	return lowlevel.Call(client, ctx, "get contract", func(cl api.WalletClient, ctx context.Context) (*core.SmartContract, error) {
		return cl.GetContract(ctx, req)
	})
}

// TriggerSmartContract builds a transaction that calls a method on the contract.
// The result should be signed and broadcasted by the caller.
// Invoke builds a transaction that calls a state-changing method on the contract.
// The result should be signed and broadcasted by the caller.
func (i *Instance) Invoke(ctx context.Context, owner *types.Address, callValue int64, method string, params ...interface{}) (*api.TransactionExtention, error) {

	if owner == nil {
		return nil, fmt.Errorf("owner address cannot be nil")
	}
	if callValue < 0 {
		return nil, fmt.Errorf("call value cannot be negative")
	}

	// Encode method call data
	data, err := i.Encode(method, params...)
	if err != nil {
		return nil, fmt.Errorf("failed to encode input for method %s: %v", method, err)
	}

	// Create trigger smart contract request
	req := &core.TriggerSmartContract{
		OwnerAddress:    owner.Bytes(),
		ContractAddress: i.Address.Bytes(),
		Data:            data,
		CallValue:       callValue,
		CallTokenValue:  0,
		TokenId:         0,
	}

	return lowlevel.TxCall(i.Client, ctx, "trigger contract", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.TriggerContract(ctx, req)
	})

}

// Call performs a constant (read-only) method call and returns the decoded
// result value. If the method has multiple outputs, the return is a []interface{};
// if one output, it's that single value; if none, nil.
func (i *Instance) Call(ctx context.Context, owner *types.Address, method string, params ...interface{}) (interface{}, error) {

	if owner == nil {
		return nil, fmt.Errorf("owner address cannot be nil")
	}

	// Encode method call data
	data, err := i.Encode(method, params...)
	if err != nil {
		return nil, fmt.Errorf("failed to encode input for method %s: %v", method, err)
	}

	// Create trigger smart contract request
	req := &core.TriggerSmartContract{
		OwnerAddress:    owner.Bytes(),
		ContractAddress: i.Address.Bytes(),
		Data:            data,
		CallValue:       0,
	}

	// Call the constant contract
	result, err := lowlevel.Call(i.Client, ctx, "trigger constant contract", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.TriggerConstantContract(ctx, req)
	})
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

	decoded, err := i.DecodeResult(method, concatenatedResult)
	if err != nil {
		return nil, fmt.Errorf("failed to decode result for method %s: %v", method, err)
	}

	return decoded, nil
}

// SimulateResult captures details from a constant-call simulation.
type SimulateResult struct {
	Energy    int64
	APIResult *api.Return
	Logs      []*core.TransactionInfo_Log
}

// Simulate performs a read-only execution of the specified method and returns
// energy usage, raw API result, and logs without decoding the return value.
func (i *Instance) Simulate(ctx context.Context, owner *types.Address, callValue int64, method string, params ...interface{}) (*SimulateResult, error) {

	if owner == nil {
		return nil, fmt.Errorf("owner address cannot be nil")
	}

	// Encode method call data
	data, err := i.Encode(method, params...)
	if err != nil {
		return nil, fmt.Errorf("failed to encode input for method %s: %v", method, err)
	}

	// Create trigger smart contract request
	req := &core.TriggerSmartContract{
		OwnerAddress:    owner.Bytes(),
		ContractAddress: i.Address.Bytes(),
		Data:            data,
		CallValue:       0,
	}

	// Call the constant contract
	result, err := lowlevel.Call(i.Client, ctx, "trigger constant contract", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.TriggerConstantContract(ctx, req)
	})
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

// Encode encodes a method invocation into call data. For constructors, pass an
// empty method name and only parameters.
func (i *Instance) Encode(method string, params ...interface{}) ([]byte, error) {
	// Special handling for constructors (empty method name)
	if method == "" {
		paramTypes, err := i.abiProcessor.GetConstructorTypes(i.ABI)
		if err != nil {
			return nil, fmt.Errorf("failed to get constructor types: %v", err)
		}
		// We need to create a temporary ABIProcessor to encode parameters
		// since the GetConstructorTypes doesn't return the ABI
		tempProcessor := utils.NewABIProcessor(i.ABI)
		// For constructors, we need to pass empty method name and get input types
		return tempProcessor.EncodeMethod("", paramTypes, params)
	}

	// Get method parameter types from ABI
	inputTypes, _, err := i.abiProcessor.GetMethodTypes(method)
	if err != nil {
		return nil, fmt.Errorf("failed to get method types: %v", err)
	}

	return i.abiProcessor.EncodeMethod(method, inputTypes, params)
}

// DecodeResult decodes a method's return bytes into a Go value. Single-output
// methods return the value directly; multiple outputs return []interface{}.
func (i *Instance) DecodeResult(method string, data []byte) (interface{}, error) {
	// Get method output types from ABI
	_, outputTypes, err := i.abiProcessor.GetMethodTypes(method)
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
	decoded, err := i.abiProcessor.DecodeResult(data, outputs)
	if err != nil {
		return nil, fmt.Errorf("failed to decode result: %v", err)
	}

	return decoded, nil
}

// DecodeInput decodes input call data to a typed representation.
func (i *Instance) DecodeInput(data []byte) (*utils.DecodedInput, error) {
	return i.abiProcessor.DecodeInputData(data, i.ABI)
}

// DecodeEventLog decodes a single event log using the contract ABI.
// func (i *Instance) DecodeEventLog(topics [][]byte, data []byte) (*utils.DecodedEvent, error) {
// 	// Convert ABI entries to eventdecoder format
// 	if err := eventdecoder.RegisterABIEntries(i.ABI.Entrys); err != nil {
// 		return nil, fmt.Errorf("failed to register ABI entries: %v", err)
// 	}

// 	// Use eventdecoder for decoding
// 	event, err := eventdecoder.DecodeLog(topics, data)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Convert eventdecoder.DecodedEvent to utils.DecodedEvent for backward compatibility
// 	parameters := make([]utils.DecodedEventParameter, len(event.Parameters))
// 	for i, param := range event.Parameters {
// 		parameters[i] = utils.DecodedEventParameter{
// 			Name:    param.Name,
// 			Type:    param.Type,
// 			Value:   param.Value,
// 			Indexed: param.Indexed,
// 		}
// 	}

// 	return &utils.DecodedEvent{
// 		EventName:  event.EventName,
// 		Parameters: parameters,
// 		Contract:   event.Contract,
// 	}, nil
// }

// // DecodeEventSignature decodes 4- or 32-byte event signature to the canonical
// // signature string if known.
// func (i *Instance) DecodeEventSignature(signature []byte) (string, error) {
// 	return i.abiProcessor.DecodeEventSignature(signature)
// }
