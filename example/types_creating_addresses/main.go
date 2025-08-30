// This snippet is from docs/types.md
// Creating Addresses
package main

import (
	"fmt"

	"github.com/kslamph/tronlib/pkg/types"
)

func main() {
	// From Base58 String
	addr, err := types.NewAddress("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")
	if err != nil {
		fmt.Printf("invalid address: %v\n", err)
		return
	}
	fmt.Printf("Address from Base58: %s\n", addr.String())

	// With validation (panics on error - use only with trusted input)
	addr = types.MustNewAddressFromBase58("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")
	fmt.Printf("Address from MustNewAddressFromBase58: %s\n", addr.String())

	// From Hex String
	// TRON format (41-prefixed)
	addr, err = types.NewAddressFromHex("41a614f803b6fd780986a42c78ec9c7f77e6ded13c")
	if err != nil {
		fmt.Printf("invalid address from hex: %v\n", err)
		return
	}
	fmt.Printf("Address from Hex: %s\n", addr.String())

	// EVM format (0x-prefixed, 20 bytes)
	addr, err = types.NewAddressFromHex("0xa614f803b6fd780986a42c78ec9c7f77e6ded13c")
	if err != nil {
		fmt.Printf("invalid address from EVM hex: %v\n", err)
		return
	}
	fmt.Printf("Address from EVM Hex: %s\n", addr.String())

	// From Raw Bytes
	// TRON format (21 bytes, 0x41 prefix)
	ronBytes := addr.Bytes()
	addr, err = types.NewAddressFromBytes(ronBytes)
	if err != nil {
		fmt.Printf("invalid address from bytes: %v\n", err)
		return
	}
	fmt.Printf("Address from Bytes: %s\n", addr.String())

	// EVM format (20 bytes, no prefix)
	evmBytes := addr.BytesEVM()
	addr, err = types.NewAddressFromBytes(evmBytes)
	if err != nil {
		fmt.Printf("invalid address from EVM bytes: %v\n", err)
		return
	}
	fmt.Printf("Address from EVM Bytes: %s\n", addr.String())
}
