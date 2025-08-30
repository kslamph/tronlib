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
	"testing"

	"github.com/kslamph/tronlib/pkg/utils"
)

// No changes, just empty lines to replace the removed content
func TestNewContractSignature(t *testing.T) {
	mockClient := createMockClient()
	mockAddress := createMockAddress()

	// Test 1: Create contract with ABI string
	contract1, err1 := NewInstance(mockClient, mockAddress, testERC20ABI)
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
	processor := utils.NewABIProcessor(nil)
	parsedABI, err := processor.ParseABI(testERC20ABI)
	if err != nil {
		t.Fatalf("Failed to parse ABI: %v", err)
	}

	contract2, err2 := NewInstance(mockClient, mockAddress, parsedABI)
	if err2 != nil {
		t.Fatalf("Failed to create contract with parsed ABI: %v", err2)
	}
	if contract2.ABI == nil {
		t.Error("ABI not properly set")
	}

}

func TestNewContractVariadicABI(t *testing.T) {
	mockClient := createMockClient()
	mockAddress := createMockAddress()

	// Test with one ABI argument (string)
	_, err1 := NewInstance(mockClient, mockAddress, testERC20ABI)
	if err1 != nil {
		t.Errorf("Unexpected error with one ABI argument: %v", err1)
	}

	// Test with too many ABI arguments
	_, err2 := NewInstance(mockClient, mockAddress, testERC20ABI, testERC20ABI)
	if err2 == nil {
		t.Error("Expected error with too many ABI arguments")
	}

	// Test with invalid ABI type
	_, err3 := NewInstance(mockClient, mockAddress, 123)
	if err3 == nil {
		t.Error("Expected error with invalid ABI type")
	}
}
