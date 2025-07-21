package integration_test

import (
	"encoding/hex"
	"testing"

	"github.com/kslamph/tronlib/pkg/types"
)

var validBase58Addresses = []string{
	"TXBwCB1RxvMPZTZE79aJn9KjLbdSXMax55",
	"TRv7JdQNybFd7FEgJ75NvWUo822gRMax66",
}

var invalidBase58Addresses = []string{
	"INVALIDADDRESS1234567890123456789012",
	"T123",
}

func TestAddressCreationAndAccessors(t *testing.T) {
	for _, addrStr := range validBase58Addresses {
		addr, err := types.NewAddress(addrStr)
		if err != nil {
			t.Errorf("Failed to create address %s: %v", addrStr, err)
			continue
		}
		if addr.String() != addrStr {
			t.Errorf("String() = %s, want %s", addr.String(), addrStr)
		}
		if len(addr.Bytes()) != 21 {
			t.Errorf("Bytes() length = %d, want 21", len(addr.Bytes()))
		}
		hexStr := addr.Hex()
		if len(hexStr) != 42 {
			t.Errorf("Hex() length = %d, want 42", len(hexStr))
		}
		if _, err := hex.DecodeString(hexStr); err != nil {
			t.Errorf("Hex() is not valid hex: %v", err)
		}
	}
}

func TestInvalidBase58Address(t *testing.T) {
	for _, addrStr := range invalidBase58Addresses {
		_, err := types.NewAddress(addrStr)
		if err == nil {
			t.Errorf("Expected error for invalid address %s, got nil", addrStr)
		}
	}
}

func TestAddressFromHexAndBytes(t *testing.T) {
	base58Addr := validBase58Addresses[0]
	addr, err := types.NewAddress(base58Addr)
	if err != nil {
		t.Fatalf("Failed to create address: %v", err)
	}
	hexStr := addr.Hex()
	addrFromHex, err := types.NewAddressFromHex(hexStr)
	if err != nil {
		t.Errorf("Failed to create from hex: %v", err)
	}
	if addrFromHex.String() != base58Addr {
		t.Errorf("FromHex String() = %s, want %s", addrFromHex.String(), base58Addr)
	}
	addrFromBytes, err := types.NewAddressFromBytes(addr.Bytes())
	if err != nil {
		t.Errorf("Failed to create from bytes: %v", err)
	}
	if addrFromBytes.String() != base58Addr {
		t.Errorf("FromBytes String() = %s, want %s", addrFromBytes.String(), base58Addr)
	}
}
