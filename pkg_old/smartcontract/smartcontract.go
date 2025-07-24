package smartcontract

import (
	"fmt"
	"sync"

	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
)

// Contract represents a smart contract interface
type Contract struct {
	ABI          *core.SmartContract_ABI
	Address      string
	AddressBytes []byte

	// Event signature cache using sync.Once pattern
	eventCacheOnce      sync.Once
	eventSignatureCache map[[32]byte]*core.SmartContract_ABI_Entry

	// 4-byte event signature cache for DecodeEventSignature
	event4ByteCacheOnce      sync.Once
	event4ByteSignatureCache map[[4]byte]*core.SmartContract_ABI_Entry
}

// Param list
type Param map[string]interface{}

// NewContract creates a new contract instance
func NewContract(abi string, address string) (*Contract, error) {
	if abi == "" {
		return nil, fmt.Errorf("empty ABI string")
	}
	if address == "" {
		return nil, fmt.Errorf("empty contract address")
	}
	decodedABI, err := DecodeABI(abi)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ABI: %v", err)
	}

	// Convert address to bytes
	addr, err := types.NewAddress(address)
	if err != nil {
		return nil, fmt.Errorf("failed to parse address: %v", err)
	}

	return &Contract{
		ABI:          decodedABI,
		Address:      address,
		AddressBytes: addr.Bytes(),
	}, nil
}

func NewContractFromABI(abi *core.SmartContract_ABI, address string) (*Contract, error) {
	addr, err := types.NewAddress(address)
	if err != nil {
		return nil, fmt.Errorf("failed to parse address: %v", err)
	}
	return &Contract{
		ABI:          abi,
		Address:      address,
		AddressBytes: addr.Bytes(),
	}, nil
}
