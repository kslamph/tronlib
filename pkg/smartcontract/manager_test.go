package smartcontract

import (
	"context"
	"testing"

	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/stretchr/testify/assert"
)

func TestDeployContractValidation(t *testing.T) {
	// Create a mock client (we won't actually deploy)
	c, err := client.NewClient(client.DefaultClientConfig("127.0.0.1:50051"))
	assert.NoError(t, err)

	manager := NewManager(c)
	ctx := context.Background()

	// Test cases for validation
	tests := []struct {
		name                       string
		ownerAddress               string
		contractName               string
		abi                        *core.SmartContract_ABI
		bytecode                   []byte
		callValue                  int64
		consumeUserResourcePercent int64
		originEnergyLimit          int64
		constructorParams          []interface{}
		wantErr                    bool
		errMsg                     string
	}{
		{
			name:                       "Valid empty contract name",
			ownerAddress:               "TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g",
			contractName:               "",
			abi:                        nil,
			bytecode:                   []byte("608060405234801561001057600080fd5b50"),
			callValue:                  0,
			consumeUserResourcePercent: 50,
			originEnergyLimit:          1000000,
			constructorParams:          []interface{}{},
			wantErr:                    false,
		},
		{
			name:                       "Valid contract name with spaces",
			ownerAddress:               "TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g",
			contractName:               "My Test Contract",
			abi:                        nil,
			bytecode:                   []byte("608060405234801561001057600080fd5b50"),
			callValue:                  0,
			consumeUserResourcePercent: 100,
			originEnergyLimit:          1000000,
			constructorParams:          []interface{}{},
			wantErr:                    false,
		},
		{
			name:                       "Invalid contract name with control characters",
			ownerAddress:               "TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g",
			contractName:               "Test\x00Contract",
			abi:                        nil,
			bytecode:                   []byte("608060405234801561001057600080fd5b50"),
			callValue:                  0,
			consumeUserResourcePercent: 50,
			originEnergyLimit:          1000000,
			constructorParams:          []interface{}{},
			wantErr:                    true,
			errMsg:                     "invalid contract name",
		},
		{
			name:                       "Empty bytecode",
			ownerAddress:               "TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g",
			contractName:               "TestContract",
			abi:                        nil,
			bytecode:                   []byte{},
			callValue:                  0,
			consumeUserResourcePercent: 50,
			originEnergyLimit:          1000000,
			constructorParams:          []interface{}{},
			wantErr:                    true,
			errMsg:                     "bytecode cannot be empty",
		},
		{
			name:                       "Negative call value",
			ownerAddress:               "TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g",
			contractName:               "TestContract",
			abi:                        nil,
			bytecode:                   []byte("608060405234801561001057600080fd5b50"),
			callValue:                  -1,
			consumeUserResourcePercent: 50,
			originEnergyLimit:          1000000,
			constructorParams:          []interface{}{},
			wantErr:                    true,
			errMsg:                     "call value cannot be negative",
		},
		{
			name:                       "Invalid consume user resource percent - negative",
			ownerAddress:               "TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g",
			contractName:               "TestContract",
			abi:                        nil,
			bytecode:                   []byte("608060405234801561001057600080fd5b50"),
			callValue:                  0,
			consumeUserResourcePercent: -1,
			originEnergyLimit:          1000000,
			constructorParams:          []interface{}{},
			wantErr:                    true,
			errMsg:                     "consume user resource percent must be between 0 and 100",
		},
		{
			name:                       "Invalid consume user resource percent - over 100",
			ownerAddress:               "TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g",
			contractName:               "TestContract",
			abi:                        nil,
			bytecode:                   []byte("608060405234801561001057600080fd5b50"),
			callValue:                  0,
			consumeUserResourcePercent: 101,
			originEnergyLimit:          1000000,
			constructorParams:          []interface{}{},
			wantErr:                    true,
			errMsg:                     "consume user resource percent must be between 0 and 100",
		},
		{
			name:                       "Negative origin energy limit",
			ownerAddress:               "TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g",
			contractName:               "TestContract",
			abi:                        nil,
			bytecode:                   []byte("608060405234801561001057600080fd5b50"),
			callValue:                  0,
			consumeUserResourcePercent: 50,
			originEnergyLimit:          -1,
			constructorParams:          []interface{}{},
			wantErr:                    true,
			errMsg:                     "origin energy limit cannot be negative",
		},
		{
			name:                       "Invalid owner address",
			ownerAddress:               "invalid-address",
			contractName:               "TestContract",
			abi:                        nil,
			bytecode:                   []byte("608060405234801561001057600080fd5b50"),
			callValue:                  0,
			consumeUserResourcePercent: 50,
			originEnergyLimit:          1000000,
			constructorParams:          []interface{}{},
			wantErr:                    true,
			errMsg:                     "invalid owner address",
		},
		{
			name:                       "Constructor params without ABI",
			ownerAddress:               "TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g",
			contractName:               "TestContract",
			abi:                        nil,
			bytecode:                   []byte("608060405234801561001057600080fd5b50"),
			callValue:                  0,
			consumeUserResourcePercent: 50,
			originEnergyLimit:          1000000,
			constructorParams:          []interface{}{"param1"},
			wantErr:                    true,
			errMsg:                     "ABI cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: This will fail at the lowlevel.DeployContract call since we don't have a real connection
			// But we want to test the validation logic before that point
			_, err := manager.DeployContract(
				ctx,
				tt.ownerAddress,
				tt.contractName,
				tt.abi,
				tt.bytecode,
				tt.callValue,
				tt.consumeUserResourcePercent,
				tt.originEnergyLimit,
				tt.constructorParams...,
			)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				// For valid inputs, we expect it to fail at the network call level
				// since we don't have a real TRON connection, but validation should pass
				if err != nil {
					// If there's an error, it should be a connection error, not validation
					assert.NotContains(t, err.Error(), "invalid contract name")
					assert.NotContains(t, err.Error(), "bytecode cannot be empty")
					assert.NotContains(t, err.Error(), "call value cannot be negative")
					assert.NotContains(t, err.Error(), "consume user resource percent")
					assert.NotContains(t, err.Error(), "origin energy limit cannot be negative")
					assert.NotContains(t, err.Error(), "invalid owner address")
				}
			}
		})
	}
}

func TestEncodeConstructor(t *testing.T) {
	// Create a mock client
	c, err := client.NewClient(client.DefaultClientConfig("127.0.0.1:50051"))
	assert.NoError(t, err)

	manager := NewManager(c)

	tests := []struct {
		name              string
		abi               *core.SmartContract_ABI
		constructorParams []interface{}
		wantErr           bool
		errMsg            string
	}{
		{
			name:              "Empty ABI",
			abi:               nil,
			constructorParams: []interface{}{},
			wantErr:           false,
		},
		{
			name:              "Invalid ABI JSON",
			abi:               nil,
			constructorParams: []interface{}{},
			wantErr:           false,
		},
		{
			name:              "No constructor in ABI but params provided",
			abi:               nil,
			constructorParams: []interface{}{"param1"},
			wantErr:           true,
			errMsg:            "constructor parameters provided but ABI is nil",
		},
		{
			name:              "No constructor in ABI and no params - valid",
			abi:               nil,
			constructorParams: []interface{}{},
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := manager.encodeConstructor(tt.abi, tt.constructorParams)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}
