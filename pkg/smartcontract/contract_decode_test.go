package smartcontract

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/kslamph/tronlib/pkg/types"
)

const testERC20ABIWithMultiOut = `[
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
	},
	{
		"constant": true,
		"inputs": [],
		"name": "getValues",
		"outputs": [
			{"name": "s", "type": "string"},
			{"name": "i", "type": "uint256"}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	}
]`

func TestDecodeResult(t *testing.T) {
	contract, err := NewInstance(createMockClient(), createMockAddress(), testERC20ABIWithMultiOut)
	if err != nil {
		t.Fatalf("Failed to create contract: %v", err)
	}

	// Test decoding a single string output
	// This is the hex encoding of "MyToken" as a string
	nameData, _ := hex.DecodeString("000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000074d79546f6b656e00000000000000000000000000000000000000000000000000")
	decodedName, err := contract.DecodeResult("name", nameData)
	if err != nil {
		t.Fatalf("Failed to decode name result: %v", err)
	}
	if name, ok := decodedName.(string); !ok || name != "MyToken" {
		t.Errorf("Expected name 'MyToken', got '%v'", decodedName)
	}

	// Test decoding a single uint256 output
	balanceData, _ := hex.DecodeString("0000000000000000000000000000000000000000000000000de0b6b3a7640000")
	decodedBalance, err := contract.DecodeResult("balanceOf", balanceData)
	if err != nil {
		t.Fatalf("Failed to decode balanceOf result: %v", err)
	}
	expectedBalance := new(big.Int)
	expectedBalance.SetString("1000000000000000000", 10)
	if balance, ok := decodedBalance.(*big.Int); !ok || balance.Cmp(expectedBalance) != 0 {
		t.Errorf("Expected balance %v, got %v", expectedBalance, decodedBalance)
	}

	// Test decoding multiple outputs (string, uint256)
	multiOutData, _ := hex.DecodeString("00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000042000000000000000000000000000000000000000000000000000000000000000b48656c6c6f20576f726c64000000000000000000000000000000000000000000")
	decodedMulti, err := contract.DecodeResult("getValues", multiOutData)
	if err != nil {
		t.Fatalf("Failed to decode getValues result: %v", err)
	}
	results, ok := decodedMulti.([]interface{})
	if !ok || len(results) != 2 {
		t.Fatalf("Expected two results, got %T", decodedMulti)
	}
	if s, ok := results[0].(string); !ok || s != "Hello World" {
		t.Errorf("Expected string 'Hello World', got '%v'", results)
	}
	expectedInt := big.NewInt(66)
	if i, ok := results[1].(*big.Int); !ok || i.Cmp(expectedInt) != 0 {
		t.Errorf("Expected int 66, got %v", results)
	}
}

func TestDecodeInput(t *testing.T) {
	contract, err := NewInstance(createMockClient(), createMockAddress(), testERC20ABI)
	if err != nil {
		t.Fatalf("Failed to create contract: %v", err)
	}

	// Test decoding input for name()
	nameInput, _ := hex.DecodeString("06fdde03")
	decodedName, err := contract.DecodeInput(nameInput)
	if err != nil {
		t.Fatalf("Failed to decode name input: %v", err)
	}
	if decodedName.Method != "name()" {
		t.Errorf("Expected method name 'name()', got '%s'", decodedName.Method)
	}
	if len(decodedName.Parameters) != 0 {
		t.Errorf("Expected 0 parameters for name, got %d", len(decodedName.Parameters))
	}

	// Test decoding input for transfer(address,uint256)
	transferInput, _ := hex.DecodeString("a9059cbb00000000000000000000004144f9bddf503f9920a1a21f0810e57a27de970f1a00000000000000000000000000000000000000000000000000000000bf91bf80")
	decodedTransfer, err := contract.DecodeInput(transferInput)
	if err != nil {
		t.Fatalf("Failed to decode transfer input: %v", err)
	}
	if decodedTransfer.Method != "transfer(address,uint256)" {
		t.Errorf("Expected method name 'transfer(address,uint256)', got '%s'", decodedTransfer.Method)
	}
	if len(decodedTransfer.Parameters) != 2 {
		t.Fatalf("Expected 2 parameters for transfer, got %d", len(decodedTransfer.Parameters))
	}

	// Check transfer parameters
	expectedAddress := types.MustNewAddressFromBase58("TGFv8TePyCuky7h7zSUgJyE1LghTqTcfZa")

	if addr, ok := decodedTransfer.Parameters[0].Value.(*types.Address); !ok || !addr.Equal(expectedAddress) {
		t.Errorf("Expected address '%s', got '%v'", expectedAddress.String(), decodedTransfer.Parameters[0].Value)
	}
	expectedAmount := big.NewInt(3214000000)
	if amount, ok := decodedTransfer.Parameters[1].Value.(*big.Int); !ok || amount.Cmp(expectedAmount) != 0 {
		t.Errorf("Expected amount %v, got '%v'", expectedAmount, decodedTransfer.Parameters[1].Value)
	}
}
