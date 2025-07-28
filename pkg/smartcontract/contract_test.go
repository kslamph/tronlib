package smartcontract

import (
	"testing"

	"github.com/kslamph/tronlib/pb/core"
)

// Test ERC20 ABI for testing
const testERC20ABI = `[
	{
		"constant": true,
		"inputs": [],
		"name": "name",
		"outputs": [{"name": "", "type": "string"}],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [{"name": "_owner", "type": "address"}],
		"name": "balanceOf",
		"outputs": [{"name": "balance", "type": "uint256"}],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{"name": "_to", "type": "address"},
			{"name": "_value", "type": "uint256"}
		],
		"name": "transfer",
		"outputs": [{"name": "", "type": "bool"}],
		"payable": false,
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"anonymous": false,
		"inputs": [
			{"indexed": true, "name": "from", "type": "address"},
			{"indexed": true, "name": "to", "type": "address"},
			{"indexed": false, "name": "value", "type": "uint256"}
		],
		"name": "Transfer",
		"type": "event"
	}
]`

func TestNewContract(t *testing.T) {
	// Create a mock client for testing
	mockClient := createMockClient(t)
	mockAddress := createMockAddress()

	// Test contract creation from ABI string
	contract, err1 := NewContract(mockClient, mockAddress, testERC20ABI)
	if err1 != nil {
		t.Fatalf("Failed to create contract: %v", err1)
	}

	if contract.Address.String() != "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t" {
		t.Errorf("Expected address TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t, got %s", contract.Address)
	}

	if contract.ABI == nil {
		t.Error("ABI should not be nil")
	}

	if len(contract.Address.Bytes()) == 0 {
		t.Error("AddressBytes should not be empty")
	}
}

func TestEncodeInput(t *testing.T) {
	contract, err1 := NewContract(createMockClient(t), createMockAddress(), testERC20ABI)
	if err1 != nil {
		t.Fatalf("Failed to create contract: %v", err1)
	}

	// Test encoding name() method (no parameters)
	data, err2 := contract.Encode("name")
	if err2 != nil {
		t.Fatalf("Failed to encode name method: %v", err2)
	}

	// name() method signature should be 0x06fdde03
	expected := []byte{0x06, 0xfd, 0xde, 0x03}
	if len(data) < 4 {
		t.Fatalf("Encoded data too short: %d bytes", len(data))
	}

	for i := 0; i < 4; i++ {
		if data[i] != expected[i] {
			t.Errorf("Method signature mismatch at byte %d: expected 0x%02x, got 0x%02x", i, expected[i], data[i])
		}
	}

	// Test encoding balanceOf(address) method
	testAddr := "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
	data, err3 := contract.Encode("balanceOf", testAddr)
	if err3 != nil {
		t.Fatalf("Failed to encode balanceOf method: %v", err3)
	}

	// balanceOf(address) should have method signature + 32 bytes for address
	if len(data) != 4+32 {
		t.Errorf("Expected 36 bytes for balanceOf encoding, got %d", len(data))
	}

	// Test encoding transfer(address,uint256) method
	data, err4 := contract.Encode("transfer", testAddr, "1000000000000000000") // 1 token with 18 decimals
	if err4 != nil {
		t.Fatalf("Failed to encode transfer method: %v", err4)
	}

	// transfer(address,uint256) should have method signature + 32 bytes for address + 32 bytes for amount
	if len(data) != 4+32+32 {
		t.Errorf("Expected 68 bytes for transfer encoding, got %d", len(data))
	}
}

func TestDecodeABI(t *testing.T) {
	abi, err5 := DecodeABI(testERC20ABI)
	if err5 != nil {
		t.Fatalf("Failed to decode ABI: %v", err5)
	}

	if len(abi.Entrys) != 4 {
		t.Errorf("Expected 4 ABI entries, got %d", len(abi.Entrys))
	}

	// Check that we have the expected methods and events
	var foundName, foundBalanceOf, foundTransfer, foundTransferEvent bool
	for _, entry := range abi.Entrys {
		switch entry.Name {
		case "name":
			foundName = true
		case "balanceOf":
			foundBalanceOf = true
		case "transfer":
			foundTransfer = true
		case "Transfer":
			foundTransferEvent = true
		}
	}

	if !foundName {
		t.Error("name method not found in ABI")
	}
	if !foundBalanceOf {
		t.Error("balanceOf method not found in ABI")
	}
	if !foundTransfer {
		t.Error("transfer method not found in ABI")
	}
	if !foundTransferEvent {
		t.Error("Transfer event not found in ABI")
	}
}

func TestInvalidInputs(t *testing.T) {
	mockClient := createMockClient(t)
	mockAddress := createMockAddress()

	// Test nil client
	_, err1 := NewContract(nil, mockAddress, testERC20ABI)
	if err1 == nil {
		t.Error("Expected error for nil client")
	}

	// Test nil address
	_, err2 := NewContract(mockClient, nil, testERC20ABI)
	if err2 == nil {
		t.Error("Expected error for nil address")
	}

	// Test empty ABI string
	_, err3 := NewContract(mockClient, mockAddress, "")
	if err3 == nil {
		t.Error("Expected error for empty ABI")
	}

	// Test invalid ABI string
	_, err4 := NewContract(mockClient, mockAddress, "invalid json")
	if err4 == nil {
		t.Error("Expected error for invalid ABI")
	}

	// Test nil ABI object
	_, err5 := NewContract(mockClient, mockAddress, (*core.SmartContract_ABI)(nil))
	if err5 == nil {
		t.Error("Expected error for nil ABI object")
	}

	// Test too many ABI arguments
	_, err6 := NewContract(mockClient, mockAddress, testERC20ABI, testERC20ABI)
	if err6 == nil {
		t.Error("Expected error with too many ABI arguments")
	}

	// Test invalid ABI type
	_, err7 := NewContract(mockClient, mockAddress, 123)
	if err7 == nil {
		t.Error("Expected error with invalid ABI type")
	}
}
