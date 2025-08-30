// This snippet is from docs/types.md
// Address Conversion
package main

import (
	"fmt"

	"github.com/kslamph/tronlib/pkg/types"
)

func main() {
	addr, _ := types.NewAddress("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")

	// Get different representations
	base58 := addr.String()     // "TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH"
	base58Alt := addr.Base58()  // Same as String()
	ronHex := addr.Hex()        // "41a614f803b6fd780986a42c78ec9c7f77e6ded13c"
	evmHex := addr.HexEVM()     // "0xa614f803b6fd780986a42c78ec9c7f77e6ded13c"
	ronBytes := addr.Bytes()    // []byte{0x41, 0xa6, ...} (21 bytes)
	evmBytes := addr.BytesEVM() // []byte{0xa6, 0x14, ...} (20 bytes)

	fmt.Printf("Base58: %s\n", base58)
	fmt.Printf("Base58 (alternative): %s\n", base58Alt)
	fmt.Printf("TRON Hex: %s\n", ronHex)
	fmt.Printf("EVM Hex: %s\n", evmHex)
	fmt.Printf("TRON Bytes: %x\n", ronBytes)
	fmt.Printf("EVM Bytes: %x\n", evmBytes)
}
