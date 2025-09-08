package utils

import (
	"math/big"
	"testing"

	eCommon "github.com/ethereum/go-ethereum/common"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestABIEncoder_Comprehensive tests the EncodeMethod function with various scenarios
func TestABIEncoder_Comprehensive(t *testing.T) {
	processor := NewABIProcessor(nil)
	// Create a sample ABI with a constructor
	abi := &core.SmartContract_ABI{
		Entrys: []*core.SmartContract_ABI_Entry{
			{
				Type: core.SmartContract_ABI_Entry_Constructor,
				Inputs: []*core.SmartContract_ABI_Entry_Param{
					{Name: "initialValue", Type: "uint256"},
				},
			},
		},
	}

	inputTypes, err := processor.GetConstructorTypes(abi)
	require.NoError(t, err)
	assert.Equal(t, []string{"uint256"}, inputTypes)

	// Test convertBytes with different fixed sizes
	t.Run("convertBytes with fixed sizes", func(t *testing.T) {
		// Test bytes32
		result, err := processor.convertBytes("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", 32)
		require.NoError(t, err)
		assert.IsType(t, [32]byte{}, result)

		// Test bytes16
		result, err = processor.convertBytes("0x1234567890abcdef1234567890abcdef", 16)
		require.NoError(t, err)
		assert.IsType(t, [16]byte{}, result)

		// Test bytes8
		result, err = processor.convertBytes("0x1234567890abcdef", 8)
		require.NoError(t, err)
		assert.IsType(t, [8]byte{}, result)
	})

	// Test convertArrayElements with different types
	t.Run("convertArrayElements with different types", func(t *testing.T) {
		// Test with bool array
		elements := []interface{}{true, false, true}
		result, err := processor.convertArrayElements(elements, "bool")
		require.NoError(t, err)
		bools, ok := result.([]bool)
		assert.True(t, ok)
		assert.Equal(t, []bool{true, false, true}, bools)

		// Test with string array
		elements = []interface{}{"hello", "world"}
		result, err = processor.convertArrayElements(elements, "string")
		require.NoError(t, err)
		strings, ok := result.([]string)
		assert.True(t, ok)
		assert.Equal(t, []string{"hello", "world"}, strings)
	})

	// Test case 1: Constructor with parameters
	t.Run("Constructor with parameters", func(t *testing.T) {
		paramTypes := []string{"address", "uint256"}
		params := []interface{}{
			"TTgbn3yTSzVDP3BrH9EonLNLvhbfyT3TXh", // TRON address
			big.NewInt(1000),
		}

		encoded, err := processor.EncodeMethod("", paramTypes, params)
		require.NoError(t, err)
		assert.NotEmpty(t, encoded)
	})

	// Test case 2: Method with various parameter types
	t.Run("Method with various parameter types", func(t *testing.T) {
		method := "transfer"
		paramTypes := []string{"address", "uint256", "bool", "string"}
		params := []interface{}{
			"TTgbn3yTSzVDP3BrH9EonLNLvhbfyT3TXh",
			big.NewInt(1000),
			true,
			"test message",
		}

		encoded, err := processor.EncodeMethod(method, paramTypes, params)
		require.NoError(t, err)
		assert.NotEqual(t, []byte{0, 0, 0, 0}, encoded[:4]) // Method ID should not be zero
		// Don't check exact length as it can vary based on encoding
	})

	// Test case 3: Method with array parameters
	t.Run("Method with array parameters", func(t *testing.T) {
		method := "batchTransfer"
		paramTypes := []string{"address[]", "uint256[]"}
		// Use EVM addresses (hex format without 0x prefix)
		params := []interface{}{
			[]string{"c24e347a3d32f348aef332e819fb87586d4fb77c", "1234567890abcdef1234567890abcdef12345678"},
			[]*big.Int{big.NewInt(100), big.NewInt(200)},
		}

		encoded, err := processor.EncodeMethod(method, paramTypes, params)
		require.NoError(t, err)
		assert.NotEmpty(t, encoded)
	})

	// Test case 4: Error case - parameter count mismatch
	t.Run("Error case - parameter count mismatch", func(t *testing.T) {
		paramTypes := []string{"address", "uint256"}
		params := []interface{}{"TTgbn3yTSzVDP3BrH9EonLNLvhbfyT3TXh"} // Only one parameter

		_, err := processor.EncodeMethod("test", paramTypes, params)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "parameter count mismatch")
	})

	// Test case 5: Error case - invalid parameter type
	t.Run("Error case - invalid parameter type", func(t *testing.T) {
		paramTypes := []string{"invalidType"}
		params := []interface{}{"test"}

		_, err := processor.EncodeMethod("test", paramTypes, params)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create ABI type")
	})
}

// TestABIEncoder_AdditionalCoverage tests additional functions to improve coverage
func TestABIEncoder_AdditionalCoverage(t *testing.T) {
	processor := NewABIProcessor(nil)

	// Test GetConstructorTypes
	t.Run("GetConstructorTypes", func(t *testing.T) {
		// Create a sample ABI with a constructor
		abi := &core.SmartContract_ABI{
			Entrys: []*core.SmartContract_ABI_Entry{
				{
					Type: core.SmartContract_ABI_Entry_Constructor,
					Inputs: []*core.SmartContract_ABI_Entry_Param{
						{Name: "initialValue", Type: "uint256"},
					},
				},
			},
		}

		inputTypes, err := processor.GetConstructorTypes(abi)
		require.NoError(t, err)
		assert.Equal(t, []string{"uint256"}, inputTypes)
	})

	// Test convertBytes with different fixed sizes
	t.Run("convertBytes with fixed sizes", func(t *testing.T) {
		// Test bytes32
		result, err := processor.convertBytes("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", 32)
		require.NoError(t, err)
		assert.IsType(t, [32]byte{}, result)

		// Test bytes16
		result, err = processor.convertBytes("0x1234567890abcdef1234567890abcdef", 16)
		require.NoError(t, err)
		assert.IsType(t, [16]byte{}, result)

		// Test bytes8
		result, err = processor.convertBytes("0x1234567890abcdef", 8)
		require.NoError(t, err)
		assert.IsType(t, [8]byte{}, result)
	})

	// Test convertArrayElements with different types
	t.Run("convertArrayElements with different types", func(t *testing.T) {
		// Test with bool array
		elements := []interface{}{true, false, true}
		result, err := processor.convertArrayElements(elements, "bool")
		require.NoError(t, err)
		bools, ok := result.([]bool)
		assert.True(t, ok)
		assert.Equal(t, []bool{true, false, true}, bools)

		// Test with string array
		elements = []interface{}{"hello", "world"}
		result, err = processor.convertArrayElements(elements, "string")
		require.NoError(t, err)
		strings, ok := result.([]string)
		assert.True(t, ok)
		assert.Equal(t, []string{"hello", "world"}, strings)
	})
}

// TestABIEncoder_ConvertParameter_Comprehensive tests the convertParameter function with various types
func TestABIEncoder_ConvertParameter_Comprehensive(t *testing.T) {
	processor := NewABIProcessor(nil)

	// Test case 1: Address conversion
	t.Run("Address conversion", func(t *testing.T) {
		// Test with TRON address string
		addrStr := "TTgbn3yTSzVDP3BrH9EonLNLvhbfyT3TXh"
		result, err := processor.convertParameter(addrStr, "address")
		require.NoError(t, err)
		assert.IsType(t, eCommon.Address{}, result)

		// Test with TRON address bytes
		addr2, _ := types.NewAddress(addrStr)
		result, err = processor.convertParameter(addr2.Bytes(), "address")
		require.NoError(t, err)
		assert.IsType(t, eCommon.Address{}, result)

		// Test with TRON address bytes
		addr, _ := types.NewAddress(addrStr)
		result, err = processor.convertParameter(addr.Bytes(), "address")
		require.NoError(t, err)
		assert.IsType(t, eCommon.Address{}, result)

		// Test with Ethereum address
		ethAddr := eCommon.HexToAddress("0xc24e347a3d32f348aef332e819fb87586d4fb77c")
		result, err = processor.convertParameter(ethAddr, "address")
		require.NoError(t, err)
		assert.Equal(t, ethAddr, result)

		// Test with TRON Address type
		result, err = processor.convertParameter(addr, "address")
		require.NoError(t, err)
		assert.IsType(t, eCommon.Address{}, result)

		// Test with nil Address pointer
		var nilAddr *types.Address
		_, err = processor.convertParameter(nilAddr, "address")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nil Address cannot be converted")
	})

	// Test case 2: Boolean conversion
	t.Run("Boolean conversion", func(t *testing.T) {
		result, err := processor.convertParameter(true, "bool")
		require.NoError(t, err)
		assert.Equal(t, true, result)

		_, err = processor.convertParameter("not bool", "bool")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "bool parameter must be a boolean")
	})

	// Test case 3: String conversion
	t.Run("String conversion", func(t *testing.T) {
		result, err := processor.convertParameter("test string", "string")
		require.NoError(t, err)
		assert.Equal(t, "test string", result)

		_, err = processor.convertParameter(123, "string")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "string parameter must be a string")
	})

	// Test case 4: Bytes conversion
	t.Run("Bytes conversion", func(t *testing.T) {
		// Test with hex string
		result, err := processor.convertParameter("0x12345678", "bytes")
		require.NoError(t, err)
		assert.Equal(t, []byte{0x12, 0x34, 0x56, 0x78}, result)

		// Test with byte slice
		result, err = processor.convertParameter([]byte{0x12, 0x34, 0x56, 0x78}, "bytes")
		require.NoError(t, err)
		assert.Equal(t, []byte{0x12, 0x34, 0x56, 0x78}, result)

		// Test with fixed-size bytes32 - need to provide 32 bytes
		result, err = processor.convertParameter("0x1234567800000000000000000000000000000000000000000000000000000000", "bytes32")
		require.NoError(t, err)
		assert.IsType(t, [32]byte{}, result)
	})

	// Test case 5: Array conversion
	t.Run("Array conversion", func(t *testing.T) {
		// Test with address array
		result, err := processor.convertParameter([]string{"TTgbn3yTSzVDP3BrH9EonLNLvhbfyT3TXh"}, "address[]")
		require.NoError(t, err)
		assert.IsType(t, []eCommon.Address{}, result)

		// Test with uint256 array
		result, err = processor.convertParameter([]*big.Int{big.NewInt(1), big.NewInt(2)}, "uint256[]")
		require.NoError(t, err)
		assert.IsType(t, []*big.Int{}, result)

		// Test with uint256 array
		result, err = processor.convertParameter([]*big.Int{big.NewInt(1), big.NewInt(2)}, "uint256[]")
		require.NoError(t, err)
		assert.IsType(t, []*big.Int{}, result)
	})
}

// TestABIDecoder_Comprehensive tests the DecodeInputData function with various scenarios
func TestABIDecoder_Comprehensive(t *testing.T) {
	// Create a sample ABI with a transfer function
	abi := &core.SmartContract_ABI{
		Entrys: []*core.SmartContract_ABI_Entry{
			{
				Name: "transfer",
				Type: core.SmartContract_ABI_Entry_Function,
				Inputs: []*core.SmartContract_ABI_Entry_Param{
					{Name: "to", Type: "address"},
					{Name: "value", Type: "uint256"},
				},
				Outputs: []*core.SmartContract_ABI_Entry_Param{
					{Name: "success", Type: "bool"},
				},
			},
		},
	}

	processor := NewABIProcessor(abi)

	// Test case 1: Successful decoding of known method
	t.Run("Successful decoding of known method", func(t *testing.T) {
		// Create encoded data for transfer function
		encoded, err := processor.EncodeMethod("transfer", []string{"address", "uint256"}, []interface{}{
			"TTgbn3yTSzVDP3BrH9EonLNLvhbfyT3TXh",
			big.NewInt(1000),
		})
		require.NoError(t, err)

		decoded, err := processor.DecodeInputData(encoded, abi)
		require.NoError(t, err)
		assert.NotNil(t, decoded)
		assert.Equal(t, "transfer(address,uint256)", decoded.Method)
		assert.Len(t, decoded.Parameters, 2)
		assert.Equal(t, "to", decoded.Parameters[0].Name)
		assert.Equal(t, "address", decoded.Parameters[0].Type)
		assert.Equal(t, "value", decoded.Parameters[1].Name)
		assert.Equal(t, "uint256", decoded.Parameters[1].Type)
	})

	// Test case 2: Unknown method
	t.Run("Unknown method", func(t *testing.T) {
		// Create data with unknown method signature
		unknownData := []byte{0x12, 0x34, 0x56, 0x78} // Random 4 bytes
		decoded, err := processor.DecodeInputData(unknownData, abi)
		require.NoError(t, err)
		assert.NotNil(t, decoded)
		assert.Contains(t, decoded.Method, "unknown(0x")
		assert.Empty(t, decoded.Parameters)
	})

	// Test case 3: Short data
	t.Run("Short data", func(t *testing.T) {
		shortData := []byte{0x01, 0x02, 0x03} // Less than 4 bytes
		_, err := processor.DecodeInputData(shortData, abi)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "input data too short")
	})

	// Test case 4: Method with no parameters
	t.Run("Method with no parameters", func(t *testing.T) {
		// Add a method with no parameters to ABI
		abi.Entrys = append(abi.Entrys, &core.SmartContract_ABI_Entry{
			Name:   "getValue",
			Type:   core.SmartContract_ABI_Entry_Function,
			Inputs: []*core.SmartContract_ABI_Entry_Param{},
		})

		// Encode the method
		encoded, err := processor.EncodeMethod("getValue", []string{}, []interface{}{})
		require.NoError(t, err)

		decoded, err := processor.DecodeInputData(encoded, abi)
		require.NoError(t, err)
		assert.NotNil(t, decoded)
		assert.Equal(t, "getValue()", decoded.Method)
		assert.Empty(t, decoded.Parameters)
	})
}

// TestABIDecoder_DecodeResult_Comprehensive tests the DecodeResult function
func TestABIDecoder_DecodeResult_Comprehensive(t *testing.T) {
	processor := NewABIProcessor(nil)

	// Test case 1: No outputs
	t.Run("No outputs", func(t *testing.T) {
		result, err := processor.DecodeResult([]byte{0x01}, []*core.SmartContract_ABI_Entry_Param{})
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	// Test case 2: Single output
	t.Run("Single output", func(t *testing.T) {
		outputs := []*core.SmartContract_ABI_Entry_Param{
			{Name: "value", Type: "uint256"},
		}

		// Encode a uint256 value (1000)
		encoded := make([]byte, 32)
		big.NewInt(1000).FillBytes(encoded[:])

		result, err := processor.DecodeResult(encoded, outputs)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, big.NewInt(1000).String(), result.(*big.Int).String())
	})

	// Test case 3: Multiple outputs
	t.Run("Multiple outputs", func(t *testing.T) {
		outputs := []*core.SmartContract_ABI_Entry_Param{
			{Name: "success", Type: "bool"},
			{Name: "value", Type: "uint256"},
		}

		// Encode bool true and uint256 1000
		encoded := make([]byte, 64)
		encoded[31] = 0x01 // true
		big.NewInt(1000).FillBytes(encoded[32:])

		result, err := processor.DecodeResult(encoded, outputs)
		require.NoError(t, err)
		assert.NotNil(t, result)
		results := result.([]interface{})
		assert.Len(t, results, 2)
		assert.Equal(t, true, results[0])
		assert.Equal(t, big.NewInt(1000).String(), results[1].(*big.Int).String())
	})

	// Test case 4: Address output
	t.Run("Address output", func(t *testing.T) {
		outputs := []*core.SmartContract_ABI_Entry_Param{
			{Name: "addr", Type: "address"},
		}

		// Encode an Ethereum address
		ethAddr := eCommon.HexToAddress("0xc24e347a3d32f348aef332e819fb87586d4fb77c")
		encoded := make([]byte, 32)
		copy(encoded[12:], ethAddr.Bytes())

		result, err := processor.DecodeResult(encoded, outputs)
		require.NoError(t, err)
		assert.NotNil(t, result)
		switch addr := result.(type) {
		case *types.Address:
			assert.Equal(t, "TTgbn3yTSzVDP3BrH9EonLNLvhbfyT3TXh", addr.String())
		case types.Address:
			assert.Equal(t, "TTgbn3yTSzVDP3BrH9EonLNLvhbfyT3TXh", addr.String())
		default:
			t.Errorf("Expected *types.Address or types.Address, got %T", result)
		}
	})
}

// TestABIDecoder_FormatDecodedValue_Comprehensive tests the formatDecodedValue function
func TestABIDecoder_FormatDecodedValue_Comprehensive(t *testing.T) {
	processor := NewABIProcessor(nil)

	// Test case 1: Address formatting
	t.Run("Address formatting", func(t *testing.T) {
		ethAddr := eCommon.HexToAddress("0xc24e347a3d32f348aef332e819fb87586d4fb77c")
		result := processor.formatDecodedValue(ethAddr, "address")
		switch addr := result.(type) {
		case *types.Address:
			assert.Equal(t, "TTgbn3yTSzVDP3BrH9EonLNLvhbfyT3TXh", addr.String())
		case types.Address:
			assert.Equal(t, "TTgbn3yTSzVDP3BrH9EonLNLvhbfyT3TXh", addr.String())
		default:
			t.Errorf("Expected *types.Address or types.Address, got %T", result)
		}
	})

	// Test case 2: Bytes formatting
	t.Run("Bytes formatting", func(t *testing.T) {
		bytesData := []byte{0x01, 0x02, 0x03, 0x04}
		result := processor.formatDecodedValue(bytesData, "bytes")
		assert.Equal(t, bytesData, result)

		result = processor.formatDecodedValue(bytesData, "bytes32")
		assert.Equal(t, bytesData, result)
	})

	// Test case 3: String formatting
	t.Run("String formatting", func(t *testing.T) {
		result := processor.formatDecodedValue("test string", "string")
		assert.Equal(t, "test string", result)
	})

	// Test case 4: Boolean formatting
	t.Run("Boolean formatting", func(t *testing.T) {
		result := processor.formatDecodedValue(true, "bool")
		assert.Equal(t, true, result)
	})

	// Test case 5: Array formatting
	t.Run("Array formatting", func(t *testing.T) {
		// Test address array
		ethAddr1 := eCommon.HexToAddress("0xc24e347a3d32f348aef332e819fb87586d4fb77c")
		ethAddr2 := eCommon.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
		addrArray := []eCommon.Address{ethAddr1, ethAddr2}
		result := processor.formatDecodedValue(addrArray, "address[]")
		results := result.([]interface{})
		assert.Len(t, results, 2)
		// Handle both pointer and value types for Address
		switch addr := results[0].(type) {
		case *types.Address:
			assert.Equal(t, "TTgbn3yTSzVDP3BrH9EonLNLvhbfyT3TXh", addr.String())
		case types.Address:
			assert.Equal(t, "TTgbn3yTSzVDP3BrH9EonLNLvhbfyT3TXh", addr.String())
		default:
			t.Errorf("Expected *types.Address or types.Address, got %T", results[0])
		}
	})

	// Test case 6: Default case (number types)
	t.Run("Default case - number types", func(t *testing.T) {
		result := processor.formatDecodedValue(big.NewInt(1000), "uint256")
		assert.Equal(t, big.NewInt(1000), result)
	})
}
