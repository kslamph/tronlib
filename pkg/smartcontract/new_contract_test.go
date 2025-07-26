package smartcontract

import (
	"testing"

	"github.com/kslamph/tronlib/pkg/utils"
)

// No changes, just empty lines to replace the removed content
func TestNewContractSignature(t *testing.T) {
	mockClient := createMockClient(t)
	mockAddress := createMockAddress()

	// Test 1: Create contract with ABI string
	contract1, err1 := NewContract(mockClient, mockAddress, testERC20ABI)
	if err1 != nil {
		t.Fatalf("Failed to create contract with ABI string: %v", err1)
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

	contract2, err2 := NewContract(mockClient, mockAddress, parsedABI)
	if err2 != nil {
		t.Fatalf("Failed to create contract with parsed ABI: %v", err2)
	}
	if contract2.ABI == nil {
		t.Error("ABI not properly set")
	}

}

func TestNewContractVariadicABI(t *testing.T) {
	mockClient := createMockClient(t)
	mockAddress := createMockAddress()

	// Test with one ABI argument (string)
	_, err1 := NewContract(mockClient, mockAddress, testERC20ABI)
	if err1 != nil {
		t.Errorf("Unexpected error with one ABI argument: %v", err1)
	}

	// Test with too many ABI arguments
	_, err2 := NewContract(mockClient, mockAddress, testERC20ABI, testERC20ABI)
	if err2 == nil {
		t.Error("Expected error with too many ABI arguments")
	}

	// Test with invalid ABI type
	_, err3 := NewContract(mockClient, mockAddress, 123)
	if err3 == nil {
		t.Error("Expected error with invalid ABI type")
	}
}
