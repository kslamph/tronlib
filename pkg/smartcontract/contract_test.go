package smartcontract

import (
	"testing"
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
	// Test contract creation from ABI string
	contract, err := NewContract(testERC20ABI, "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	if err != nil {
		t.Fatalf("Failed to create contract: %v", err)
	}

	if contract.Address != "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t" {
		t.Errorf("Expected address TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t, got %s", contract.Address)
	}

	if contract.ABI == nil {
		t.Error("ABI should not be nil")
	}

	if len(contract.AddressBytes) == 0 {
		t.Error("AddressBytes should not be empty")
	}
}

func TestEncodeInput(t *testing.T) {
	contract, err := NewContract(testERC20ABI, "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	if err != nil {
		t.Fatalf("Failed to create contract: %v", err)
	}

	// Test encoding name() method (no parameters)
	data, err := contract.EncodeInput("name")
	if err != nil {
		t.Fatalf("Failed to encode name method: %v", err)
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
	data, err = contract.EncodeInput("balanceOf", testAddr)
	if err != nil {
		t.Fatalf("Failed to encode balanceOf method: %v", err)
	}

	// balanceOf(address) should have method signature + 32 bytes for address
	if len(data) != 4+32 {
		t.Errorf("Expected 36 bytes for balanceOf encoding, got %d", len(data))
	}

	// Test encoding transfer(address,uint256) method
	data, err = contract.EncodeInput("transfer", testAddr, "1000000000000000000") // 1 token with 18 decimals
	if err != nil {
		t.Fatalf("Failed to encode transfer method: %v", err)
	}

	// transfer(address,uint256) should have method signature + 32 bytes for address + 32 bytes for amount
	if len(data) != 4+32+32 {
		t.Errorf("Expected 68 bytes for transfer encoding, got %d", len(data))
	}
}

func TestDecodeABI(t *testing.T) {
	abi, err := DecodeABI(testERC20ABI)
	if err != nil {
		t.Fatalf("Failed to decode ABI: %v", err)
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
	// Test empty ABI
	_, err := NewContract("", "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	if err == nil {
		t.Error("Expected error for empty ABI")
	}

	// Test empty address
	_, err = NewContract(testERC20ABI, "")
	if err == nil {
		t.Error("Expected error for empty address")
	}

	// Test invalid address
	_, err = NewContract(testERC20ABI, "invalid_address")
	if err == nil {
		t.Error("Expected error for invalid address")
	}

	// Test invalid ABI
	_, err = NewContract("invalid json", "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	if err == nil {
		t.Error("Expected error for invalid ABI")
	}
}