package utils

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test data migrated from pkg_old/crypto/verify_message_v2_test.go
func TestVerifyMessageV2(t *testing.T) {
	testCases := []struct {
		name      string
		address   string
		message   string
		signature string
		valid     bool
	}{
		{
			name:      "Valid message verification",
			address:   "TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x",
			message:   "sign message testing",
			signature: "0x88bacb8549cbe7c3e26d922b05e88757197b77410fb0db1fabb9f30480202c84691b7025e928d36be962cfd7b4a8d2353b97f36d64bdc14398e9568091b701201b",
			valid:     true,
		},
		{
			name:      "Wrong message for signature",
			address:   "TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x",
			message:   "wrong message",
			signature: "0x88bacb8549cbe7c3e26d922b05e88757197b77410fb0db1fabb9f30480202c84691b7025e928d36be962cfd7b4a8d2353b97f36d64bdc14398e9568091b701201b",
			valid:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			valid, err := VerifyMessageV2(tc.message, tc.signature, tc.address)
			require.NoError(t, err)
			assert.Equal(t, tc.valid, valid)
		})
	}
}

func TestVerifyMessageV2InvalidSignatures(t *testing.T) {
	address := "TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x"
	message := "test message"

	invalidCases := []struct {
		name      string
		signature string
		errorMsg  string
	}{
		{
			name:      "Invalid signature length",
			signature: "0x1234",
			errorMsg:  "signature must be 65 bytes",
		},
		{
			name:      "Missing 0x prefix",
			signature: "88bacb8549cbe7c3e26d922b05e88757197b77410fb0db1fabb9f30480202c84691b7025e928d36be962cfd7b4a8d2353b97f36d64bdc14398e9568091b701201b",
			errorMsg:  "signature must start with 0x",
		},
		{
			name:      "Invalid hex characters",
			signature: "0xZZ",
			errorMsg:  "signature must be 65 bytes",
		},
	}

	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := VerifyMessageV2(message, tc.signature, address)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.errorMsg)
		})
	}
}

func TestAmountValidation(t *testing.T) {
	t.Run("Valid amounts", func(t *testing.T) {
		validAmounts := []*big.Int{
			big.NewInt(1),          // 1 SUN
			big.NewInt(1000000),    // 1 TRX
			big.NewInt(1000000000), // 1000 TRX
			new(big.Int).Mul(big.NewInt(1000000), big.NewInt(1000000)), // 1M TRX
		}

		for _, amount := range validAmounts {
			assert.True(t, IsValidAmount(amount))
			assert.NoError(t, ValidateAmount(amount))
		}
	})

	t.Run("Invalid amounts", func(t *testing.T) {
		invalidCases := []struct {
			name   string
			amount *big.Int
		}{
			{
				name:   "Nil amount",
				amount: nil,
			},
			{
				name:   "Zero amount",
				amount: big.NewInt(0),
			},
			{
				name:   "Negative amount",
				amount: big.NewInt(-1),
			},
		}

		for _, tc := range invalidCases {
			t.Run(tc.name, func(t *testing.T) {
				assert.False(t, IsValidAmount(tc.amount))
				assert.Error(t, ValidateAmount(tc.amount))
			})
		}
	})

}

func TestContractValidation(t *testing.T) {

	t.Run("Contract data validation", func(t *testing.T) {
		// Valid data with method signature
		validData := []byte{0x15, 0x91, 0x69, 0x0b} // transfer(address,uint256) signature
		assert.NoError(t, ValidateContractData(validData))

		// Invalid: empty data
		assert.Error(t, ValidateContractData([]byte{}))

		// Invalid: too short (less than 4 bytes)
		assert.Error(t, ValidateContractData([]byte{0xa9, 0x05}))
	})
}

func TestMethodNameValidation(t *testing.T) {
	validNames := []string{
		"transfer",
		"balanceOf",
		"approve",
		"_burn",
		"symbol",
		"decimals",
	}

	for _, name := range validNames {
		t.Run("Valid: "+name, func(t *testing.T) {
			assert.True(t, IsValidMethodName(name))
			assert.NoError(t, ValidateMethodName(name))
		})
	}

	invalidNames := []string{
		"",               // Empty
		"123invalid",     // Starts with number
		"invalid-method", // Contains hyphen
		"invalid method", // Contains space
	}

	for _, name := range invalidNames {
		t.Run("Invalid: "+name, func(t *testing.T) {
			assert.False(t, IsValidMethodName(name))
			assert.Error(t, ValidateMethodName(name))
		})
	}
}

func TestTokenSymbolValidation(t *testing.T) {
	validSymbols := []string{
		"TRX",
		"USDT",
		"BTC",
		"ETH",
		"USDC123",
	}

	for _, symbol := range validSymbols {
		t.Run("Valid: "+symbol, func(t *testing.T) {
			assert.True(t, IsValidTokenSymbol(symbol))
			assert.NoError(t, ValidateTokenSymbol(symbol))
		})
	}

	invalidSymbols := []string{
		"",                  // Empty
		"toolongsymbol",     // Too long (>10 chars)
		"invalid-symbol",    // Contains hyphen
		"symbol with space", // Contains space
		"symbol@",           // Contains special character
	}

	for _, symbol := range invalidSymbols {
		t.Run("Invalid: "+symbol, func(t *testing.T) {
			assert.False(t, IsValidTokenSymbol(symbol))
			assert.Error(t, ValidateTokenSymbol(symbol))
		})
	}
}

func TestNodeURLValidation(t *testing.T) {
	validURLs := []string{
		"grpc://127.0.0.1:50051",
		"grpcs://grpc.trongrid.io:50051",
	}

	for _, url := range validURLs {
		t.Run("Valid: "+url, func(t *testing.T) {
			assert.True(t, IsValidNodeURL(url))
			assert.NoError(t, ValidateNodeURL(url))
		})
	}

	invalidURLs := []string{
		"",                       // Empty
		"invalid-url",            // No port
		"http://example.com",     // Wrong protocol
		"127.0.0.1",              // Missing port
		"127.0.0.1:abc",          // Invalid port
		"grpc.trongrid.io:50051", // Missing scheme now invalid
	}

	for _, url := range invalidURLs {
		t.Run("Invalid: "+url, func(t *testing.T) {
			assert.False(t, IsValidNodeURL(url))
			assert.Error(t, ValidateNodeURL(url))
		})
	}
}

func TestDecimalsValidation(t *testing.T) {
	validDecimals := []int{0, 6, 8, 18}

	for _, decimals := range validDecimals {
		t.Run("Valid decimals", func(t *testing.T) {
			assert.True(t, IsValidDecimals(decimals))
			assert.NoError(t, ValidateDecimals(decimals))
		})
	}

	invalidDecimals := []int{-1, 19, 100}

	for _, decimals := range invalidDecimals {
		t.Run("Invalid decimals", func(t *testing.T) {
			assert.False(t, IsValidDecimals(decimals))
			assert.Error(t, ValidateDecimals(decimals))
		})
	}
}

func TestPermissionIDValidation(t *testing.T) {
	validIDs := []int32{0, 1, 2, 255}

	for _, id := range validIDs {
		t.Run("Valid permission ID", func(t *testing.T) {
			assert.True(t, IsValidPermissionID(id))
			assert.NoError(t, ValidatePermissionID(id))
		})
	}

	invalidIDs := []int32{-1, 256, 1000}

	for _, id := range invalidIDs {
		t.Run("Invalid permission ID", func(t *testing.T) {
			assert.False(t, IsValidPermissionID(id))
			assert.Error(t, ValidatePermissionID(id))
		})
	}
}
