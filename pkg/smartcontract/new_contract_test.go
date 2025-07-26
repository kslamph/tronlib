package smartcontract

import (
	"testing"

	"github.com/kslamph/tronlib/pkg/utils"
)

func TestNewContractSignature(t *testing.T) {
	mockClient := createMockClient()
	mockAddress := createMockAddress()

	// Test 1: Create contract with ABI string
	contract1, err := NewContract(mockClient, mockAddress, testERC20ABI)
	if err != nil {
		t.Fatalf("Failed to create contract with ABI string: %v", err)
	}
	if contract1.Client != mockClient {
		t.Error("Client not properly set")
	}
	if contract1.Address != mockAddress {
		t.Error("Address not properly set")
	}

	// Test 2: Create contract with parsed ABI
	parser := utils.NewABIParser()
	parsedABI, err := parser.ParseABI(testERC20ABI)
	if err != nil {
		t.Fatalf("Failed to parse ABI: %v", err)
	}
	
	contract2, err := NewContract(mockClient, mockAddress, parsedABI)
	if err != nil {
		t.Fatalf("Failed to create contract with parsed ABI: %v", err)
	}
	if contract2.ABI == nil {
		t.Error("ABI not properly set")
	}

	// Test 3: Create contract without ABI (would retrieve from network in real scenario)
	// This will fail in test because we don't have a real network connection
	_, err = NewContract(mockClient, mockAddress)
	if err == nil {
		t.Error("Expected error when trying to retrieve ABI from network with mock client")
	}
}

func TestNewContractVariadicABI(t *testing.T) {
	mockClient := createMockClient()
	mockAddress := createMockAddress()

	// Test with no ABI arguments (should try to retrieve from network)
	_, err := NewContract(mockClient, mockAddress)
	if err == nil {
		t.Error("Expected error with mock client trying to retrieve from network")
	}

	// Test with one ABI argument (string)
	_, err = NewContract(mockClient, mockAddress, testERC20ABI)
	if err != nil {
		t.Errorf("Unexpected error with one ABI argument: %v", err)
	}

	// Test with too many ABI arguments
	_, err = NewContract(mockClient, mockAddress, testERC20ABI, testERC20ABI)
	if err == nil {
		t.Error("Expected error with too many ABI arguments")
	}

	// Test with invalid ABI type
	_, err = NewContract(mockClient, mockAddress, 123)
	if err == nil {
		t.Error("Expected error with invalid ABI type")
	}
}