package main

import (
	"fmt"
	"math/big"

	eABI "github.com/ethereum/go-ethereum/accounts/abi"
	eCommon "github.com/ethereum/go-ethereum/common"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/kslamph/tronlib/pkg/utils"
)

func main() {
	fmt.Println("ABI Type Mapping Test Program")
	fmt.Println("==============================")

	// Run the comprehensive test
	runComprehensiveTest()
}

func runComprehensiveTest() {
	fmt.Println("Comprehensive ABI Type Mapping Test")
	fmt.Println("===================================")

	// Test all the type mappings documented in ABI_TYPE_MAPPINGS.md
	testAllTypeMappings()
}

func testAllTypeMappings() {
	// Test address types
	testAddressMappings()

	// Test integer types
	testIntegerMappings()

	// Test boolean types
	testBoolMappings()

	// Test string types
	testStringMappings()

	// Test bytes types
	testBytesMappings()

	// Test fixed-size bytes types
	testFixedBytesMappings()

	// Test array types
	testArrayMappings()
}

func testAddressMappings() {
	fmt.Println("\n--- Testing Address Type Mappings ---")

	processor := &utils.ABIProcessor{}
	paramTypes := []string{"address"}

	// Test string
	tronAddrStr := "41598F46D7838183A664307841598F46D7838183A6"
	_, err := processor.EncodeMethod("", paramTypes, []interface{}{tronAddrStr})
	if err != nil {
		fmt.Printf("❌ Error encoding address as string: %v\n", err)
	} else {
		fmt.Println("✅ Successfully encoded address as string")
	}

	//Test T prefixed string
	tronAddrStrWithT := "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
	_, err = processor.EncodeMethod("", paramTypes, []interface{}{tronAddrStrWithT})
	if err != nil {
		fmt.Printf("❌ Error encoding address with T prefix: %v\n", err)
	} else {
		fmt.Println("✅ Successfully encoded address with T prefix")
	}

	// Test []byte
	tronAddr, _ := types.NewAddressFromHex(tronAddrStr)
	addrBytes := tronAddr.BytesEVM()
	_, err = processor.EncodeMethod("", paramTypes, []interface{}{addrBytes})
	if err != nil {
		fmt.Printf("❌ Error encoding address as []byte: %v\n", err)
	} else {
		fmt.Println("✅ Successfully encoded address as []byte")
	}

	// Test eCommon.Address
	ethAddr := eCommon.BytesToAddress(addrBytes)
	ethAddrEncoded, err := processor.EncodeMethod("", paramTypes, []interface{}{ethAddr})
	if err != nil {
		fmt.Printf("❌ Error encoding address as eCommon.Address: %v\n", err)
	} else {
		fmt.Println("✅ Successfully encoded address as eCommon.Address")
	}
	decoded, err := decodeParams("address", ethAddrEncoded, paramTypes)
	if err != nil {
		fmt.Printf("❌ Error decoding eCommon.Address: %v\n", err)
	} else {
		fmt.Printf("✅ Successfully decoded eCommon.Address: %s\n", decoded.(*types.Address).String())
	}

	// Test types.Address
	_, err = processor.EncodeMethod("", paramTypes, []interface{}{tronAddr})
	if err != nil {
		fmt.Printf("❌ Error encoding address as types.Address: %v\n", err)
	} else {
		fmt.Println("✅ Successfully encoded address as types.Address")
	}
}

func testIntegerMappings() {
	fmt.Println("\n--- Testing Integer Type Mappings ---")

	processor := &utils.ABIProcessor{}

	// Test uint8
	_, err := processor.EncodeMethod("", []string{"uint8"}, []interface{}{uint8(42)})
	if err != nil {
		fmt.Printf("❌ Error encoding uint8: %v\n", err)
	} else {
		fmt.Println("✅ Successfully encoded uint8")
	}

	// Test uint256 with *big.Int
	_, err = processor.EncodeMethod("", []string{"uint256"}, []interface{}{big.NewInt(1234567890)})
	if err != nil {
		fmt.Printf("❌ Error encoding uint256 with *big.Int: %v\n", err)
	} else {
		fmt.Println("✅ Successfully encoded uint256 with *big.Int")
	}

	// Test int256 with *big.Int
	_, err = processor.EncodeMethod("", []string{"int256"}, []interface{}{new(big.Int).Neg(big.NewInt(1234567890))})
	if err != nil {
		fmt.Printf("❌ Error encoding int256 with *big.Int: %v\n", err)
	} else {
		fmt.Println("✅ Successfully encoded int256 with *big.Int")
	}
}

func testBoolMappings() {
	fmt.Println("\n--- Testing Boolean Type Mappings ---")

	processor := &utils.ABIProcessor{}

	// Test bool
	_, err := processor.EncodeMethod("", []string{"bool"}, []interface{}{true})
	if err != nil {
		fmt.Printf("❌ Error encoding bool: %v\n", err)
	} else {
		fmt.Println("✅ Successfully encoded bool")
	}
}

func testStringMappings() {
	fmt.Println("\n--- Testing String Type Mappings ---")

	processor := &utils.ABIProcessor{}

	// Test string
	_, err := processor.EncodeMethod("", []string{"string"}, []interface{}{"Hello, World!"})
	if err != nil {
		fmt.Printf("❌ Error encoding string: %v\n", err)
	} else {
		fmt.Println("✅ Successfully encoded string")
	}
}

func testBytesMappings() {
	fmt.Println("\n--- Testing Bytes Type Mappings ---")

	processor := &utils.ABIProcessor{}

	// Test bytes with []byte
	_, err := processor.EncodeMethod("", []string{"bytes"}, []interface{}{[]byte("Hello, Bytes!")})
	if err != nil {
		fmt.Printf("❌ Error encoding bytes with []byte: %v\n", err)
	} else {
		fmt.Println("✅ Successfully encoded bytes with []byte")
	}

	// Test bytes with hex string
	_, err = processor.EncodeMethod("", []string{"bytes"}, []interface{}{"0x48656c6c6f2c20427974657321"})
	if err != nil {
		fmt.Printf("❌ Error encoding bytes with hex string: %v\n", err)
	} else {
		fmt.Println("✅ Successfully encoded bytes with hex string")
	}
}

func testFixedBytesMappings() {
	fmt.Println("\n--- Testing Fixed-Size Bytes Type Mappings ---")

	processor := &utils.ABIProcessor{}

	// Test bytes32 with [32]byte
	bytes32 := [32]byte{}
	copy(bytes32[:], []byte("Hello, Bytes32!"))
	_, err := processor.EncodeMethod("", []string{"bytes32"}, []interface{}{bytes32})
	if err != nil {
		fmt.Printf("❌ Error encoding bytes32 with [32]byte: %v\n", err)
	} else {
		fmt.Println("✅ Successfully encoded bytes32 with [32]byte")
	}

	// Test bytes32 with []byte
	bytes32Slice := make([]byte, 32)
	copy(bytes32Slice, []byte("Hello, Bytes32!"))
	_, err = processor.EncodeMethod("", []string{"bytes32"}, []interface{}{bytes32Slice})
	if err != nil {
		fmt.Printf("❌ Error encoding bytes32 with []byte: %v\n", err)
	} else {
		fmt.Println("✅ Successfully encoded bytes32 with []byte")
	}

	// Test bytes32 with hex string
	_, err = processor.EncodeMethod("", []string{"bytes32"}, []interface{}{"0x48656c6c6f2c2042797465733332210000000000000000000000000000000000"})
	if err != nil {
		fmt.Printf("❌ Error encoding bytes32 with hex string: %v\n", err)
	} else {
		fmt.Println("✅ Successfully encoded bytes32 with hex string")
	}
}

func testArrayMappings() {
	fmt.Println("\n--- Testing Array Type Mappings ---")

	processor := &utils.ABIProcessor{}

	// Test uint256[] with []*big.Int
	uint256Array := []*big.Int{big.NewInt(123), big.NewInt(456), big.NewInt(789)}
	_, err := processor.EncodeMethod("", []string{"uint256[]"}, []interface{}{uint256Array})
	if err != nil {
		fmt.Printf("❌ Error encoding uint256[] with []*big.Int: %v\n", err)
	} else {
		fmt.Println("✅ Successfully encoded uint256[] with []*big.Int")
	}

	// Test string[] with []string
	stringArray := []string{"Hello", "World", "Test"}
	_, err = processor.EncodeMethod("", []string{"string[]"}, []interface{}{stringArray})
	if err != nil {
		fmt.Printf("❌ Error encoding string[] with []string: %v\n", err)
	} else {
		fmt.Println("✅ Successfully encoded string[] with []string")
	}

	// Test uint256[3] with [3]*big.Int
	uint256FixedArray := [3]*big.Int{big.NewInt(123), big.NewInt(456), big.NewInt(789)}
	_, err = processor.EncodeMethod("", []string{"uint256[3]"}, []interface{}{uint256FixedArray})
	if err != nil {
		fmt.Printf("❌ Error encoding uint256[3] with [3]*big.Int: %v\n", err)
	} else {
		fmt.Println("✅ Successfully encoded uint256[3] with [3]*big.Int")
	}

	// Test bool[3] with [3]bool
	boolFixedArray := [3]bool{true, false, true}
	_, err = processor.EncodeMethod("", []string{"bool[3]"}, []interface{}{boolFixedArray})
	if err != nil {
		fmt.Printf("❌ Error encoding bool[3] with [3]bool: %v\n", err)
	} else {
		fmt.Println("✅ Successfully encoded bool[3] with [3]bool")
	}
}

// encodeParams uses the ABIProcessor to encode parameters
func encodeParams(goType string, paramTypes []string, params []interface{}) ([]byte, error) {
	fmt.Printf("Encoding with Go type %s\n", goType)

	processor := &utils.ABIProcessor{}
	return processor.EncodeMethod("", paramTypes, params)
}

// decodeParams uses the go-ethereum ABI to decode parameters
func decodeParams(solidityType string, data []byte, paramTypes []string) (interface{}, error) {
	// Create ethereum ABI arguments for decoding
	args := make([]eABI.Argument, len(paramTypes))
	for i, paramType := range paramTypes {
		abiType, err := eABI.NewType(paramType, "", nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create ABI type for %s: %v", paramType, err)
		}
		args[i] = eABI.Argument{Type: abiType}
	}

	// Unpack the parameters
	values, err := eABI.Arguments(args).Unpack(data)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack parameters: %v", err)
	}

	if len(values) > 0 {
		return values[0], nil
	}

	return nil, fmt.Errorf("no values decoded")
}
