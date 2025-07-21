package types_test

import (
	"encoding/hex"
	"testing"

	"github.com/kslamph/tronlib/pkg/types"
)

const testABI = `[{"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"guy","type":"address"},{"name":"sad","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"src","type":"address"},{"name":"dst","type":"address"},{"name":"sad","type":"uint256"}],"name":"transferFrom","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"sad","type":"uint256"}],"name":"withdraw","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"guy","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"dst","type":"address"},{"name":"sad","type":"uint256"}],"name":"transfer","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[],"name":"deposit","outputs":[],"payable":true,"stateMutability":"payable","type":"function"},{"constant":false,"inputs":[{"name":"guy","type":"address"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"src","type":"address"},{"name":"guy","type":"address"}],"name":"allowance","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"payable":true,"stateMutability":"payable","type":"fallback"},{"anonymous":false,"inputs":[{"indexed":true,"name":"src","type":"address"},{"indexed":true,"name":"guy","type":"address"},{"indexed":false,"name":"sad","type":"uint256"}],"name":"Approval","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"src","type":"address"},{"indexed":true,"name":"dst","type":"address"},{"indexed":false,"name":"sad","type":"uint256"}],"name":"Transfer","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"dst","type":"address"},{"indexed":false,"name":"sad","type":"uint256"}],"name":"Deposit","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"src","type":"address"},{"indexed":false,"name":"sad","type":"uint256"}],"name":"Withdrawal","type":"event"}]`
const testAddress = "TNUC9Qb1rRpS5CbWLmNMxXBjyFoydXjWFR"

func TestNewContract(t *testing.T) {
	tests := []struct {
		name    string
		abi     string
		address string
		wantErr bool
	}{
		{"Valid ABI and address", testABI, testAddress, false},
		{"Empty ABI", "", testAddress, true},
		{"Empty address", testABI, "", true},
		{"Invalid ABI", "[{]", testAddress, true},
		{"Invalid address", testABI, "invalid", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := types.NewContract(tt.abi, tt.address)
			if tt.wantErr && err == nil {
				t.Errorf("Expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestDecodeInputData(t *testing.T) {
	contract, err := types.NewContract(testABI, testAddress)
	if err != nil {
		t.Fatalf("Failed to create contract: %v", err)
	}
	// approve(address,uint256) method signature: keccak256("approve(address,uint256)")[:4] = 095ea7b3
	approveSig, _ := hex.DecodeString("095ea7b3")
	approveParam := make([]byte, 64)
	// address param (first 32 bytes): pad with zeros except last 20 bytes
	copy(approveParam[12:32], []byte{0x41, 0xe2, 0x8b, 0x3c, 0xfd, 0x4e, 0x0e, 0x90, 0x90, 0x77, 0x82, 0x14, 0x78, 0xe9, 0xfc, 0xb8, 0x6b, 0x84, 0xbe, 0x78})
	// uint256 param (second 32 bytes): value = 42
	approveParam[63] = 42
	approveInput := append(approveSig, approveParam...)

	// withdraw(uint256) method signature: keccak256("withdraw(uint256)")[:4] = 2e1a7d4d
	withdrawSig, _ := hex.DecodeString("2e1a7d4d")
	withdrawParam := make([]byte, 32)
	withdrawParam[31] = 42
	withdrawInput := append(withdrawSig, withdrawParam...)

	tests := []struct {
		name       string
		data       []byte
		wantMethod string
		wantErr    bool
	}{
		{"Valid approve input", approveInput, "approve(address,uint256)", false},
		{"Valid withdraw input", withdrawInput, "withdraw(uint256)", false},
		{"Too short input", []byte{0x01, 0x02}, "unknown(0x0102)", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoded, err := contract.DecodeInputData(tt.data)
			if tt.wantErr && err == nil {
				t.Errorf("Expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if decoded != nil && decoded.Method != tt.wantMethod {
				t.Errorf("Method = %v, want %v", decoded.Method, tt.wantMethod)
			}
		})
	}
}

func TestDecodeEventSignature(t *testing.T) {
	contract, err := types.NewContract(testABI, testAddress)
	if err != nil {
		t.Fatalf("Failed to create contract: %v", err)
	}
	// Deposit(address,uint256) event signature: keccak256("Deposit(address,uint256)")[:32]
	depositSig, _ := hex.DecodeString("e1fffcc4923d04b559f4d29a8bfc6cda04eb5b0d3c460751c2402c5c5cc9109c")
	// Withdrawal(address,uint256) event signature: keccak256("Withdrawal(address,uint256)")[:32]
	withdrawalSig, _ := hex.DecodeString("7fcf532c15f0a6db0bd6d0e038bea71d30d808c7d98cb3bf7268a95bf5081b65")
	// Approval(address,address,uint256) event signature: keccak256("Approval(address,address,uint256)")[:32]
	approvalSig, _ := hex.DecodeString("8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925")

	tests := []struct {
		name    string
		sig     []byte
		want    string
		wantErr bool
	}{
		{"Valid Deposit event signature", depositSig, "Deposit(address,uint256)", false},
		{"Valid Withdrawal event signature", withdrawalSig, "Withdrawal(address,uint256)", false},
		{"Valid Approval event signature", approvalSig, "Approval(address,address,uint256)", false},
		{"Unknown event signature", []byte{0x01, 0x02, 0x03, 0x04}, "unknown_event(0x01020304)", false},
		{"Too short signature", []byte{0x01}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, err := contract.DecodeEventSignature(tt.sig)
			if tt.wantErr && err == nil {
				t.Errorf("Expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if name != tt.want {
				t.Errorf("Event name = %v, want %v", name, tt.want)
			}
		})
	}
}

func TestDecodeEventLog(t *testing.T) {
	contract, err := types.NewContract(testABI, testAddress)
	if err != nil {
		t.Fatalf("Failed to create contract: %v", err)
	}
	// Deposit(address,uint256) event signature: keccak256("Deposit(address,uint256)")
	depositSig, _ := hex.DecodeString("e1fffcc4923d04b559f4d29a8bfc6cda04eb5b0d3c460751c2402c5c5cc9109c")
	// Indexed param dst (address): pad with zeros except last 20 bytes
	dst := make([]byte, 32)
	copy(dst[12:32], []byte{0x41, 0xea, 0xc4, 0x9b, 0xc7, 0x66, 0xbe, 0x29, 0xbe, 0x1b, 0x6d, 0x36, 0x61, 0x9e, 0xff, 0x8f, 0x86, 0xed, 0x4d, 0x04})
	// Non-indexed param sad (uint256): value = 42
	data := make([]byte, 32)
	data[31] = 42
	topics := [][]byte{depositSig, dst}

	tests := []struct {
		name      string
		topics    [][]byte
		data      []byte
		wantEvent string
		wantErr   bool
	}{
		{"Valid event log", topics, data, "Deposit", false},
		{"No topics", [][]byte{}, data, "unknown_event(0x)", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := contract.DecodeEventLog(tt.topics, tt.data)
			if tt.wantErr && err == nil {
				t.Errorf("Expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if event != nil && event.EventName != tt.wantEvent {
				t.Errorf("EventName = %v, want %v", event.EventName, tt.wantEvent)
			}
		})
	}
}

// For real onchain testcases, create a program under examples/contract/ to retrieve contract ABI and logs.
// See examples/contract/contract.go for template.
