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

// Test data migrated from pkg_old/types/address_test.go
func TestAddressValidation(t *testing.T) {
	validCases := []struct {
		name    string
		address string
	}{
		{
			name:    "Valid base58 address 1",
			address: "TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb",
		},
		{
			name:    "Valid base58 address 2",
			address: "TXNYeYdao7JL7wBtmzbk7mAie7UZsdgVjx",
		},
		{
			name:    "Valid hex address",
			address: "e28b3cfd4e0e909077821478e9fcb86b84be786e",
		},
		{
			name:    "Valid hex address with 0x prefix",
			address: "0xe28b3cfd4e0e909077821478e9fcb86b84be786e",
		},
	}

	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test IsValidTronAddress
			assert.True(t, IsValidTronAddress(tc.address))
			
			// Test ValidateAddress
			addr, err := ValidateAddress(tc.address)
			require.NoError(t, err)
			assert.NotNil(t, addr)
			assert.True(t, addr.IsValid())
		})
	}
}

func TestInvalidAddressValidation(t *testing.T) {
	invalidCases := []struct {
		name    string
		address string
	}{
		{
			name:    "Empty address",
			address: "",
		},
		{
			name:    "Wrong prefix base58",
			address: "AWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb",
		},
		{
			name:    "Wrong length base58",
			address: "TWd4WrZ9wn84f5x1hZhL4DHvk738ns5",
		},
		{
			name:    "Invalid hex characters",
			address: "x28b3cfd4e0e909077821478e9fcb86b84be786e",
		},
		{
			name:    "Wrong hex prefix",
			address: "1e28b3cfd4e0e909077821478e9fcb86b84be786e",
		},
	}

	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test IsValidTronAddress
			assert.False(t, IsValidTronAddress(tc.address))
			
			// Test ValidateAddress
			_, err := ValidateAddress(tc.address)
			assert.Error(t, err)
		})
	}
}

func TestAmountValidation(t *testing.T) {
	t.Run("Valid amounts", func(t *testing.T) {
		validAmounts := []*big.Int{
			big.NewInt(1),                    // 1 SUN
			big.NewInt(1000000),              // 1 TRX
			big.NewInt(1000000000),           // 1000 TRX
			new(big.Int).Mul(big.NewInt(1000000), big.NewInt(1000000)), // 1M TRX
		}

		for _, amount := range validAmounts {
			assert.True(t, IsValidAmount(amount))
			assert.NoError(t, ValidateAmount(amount, nil))
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
				assert.Error(t, ValidateAmount(tc.amount, nil))
			})
		}
	})

	t.Run("Minimum amount validation", func(t *testing.T) {
		minAmount := big.NewInt(1000000) // 1 TRX
		
		// Valid: above minimum
		assert.NoError(t, ValidateAmount(big.NewInt(2000000), minAmount))
		
		// Invalid: below minimum
		assert.Error(t, ValidateAmount(big.NewInt(500000), minAmount))
	})
}

func TestTRXAmountValidation(t *testing.T) {
	t.Run("Valid TRX amounts", func(t *testing.T) {
		validAmounts := []*big.Int{
			big.NewInt(1),        // 1 SUN
			big.NewInt(1000000),  // 1 TRX
			big.NewInt(10000000), // 10 TRX
		}

		for _, amount := range validAmounts {
			assert.NoError(t, ValidateTRXAmount(amount))
		}
	})

	t.Run("Invalid TRX amounts", func(t *testing.T) {
		invalidAmounts := []*big.Int{
			big.NewInt(0),  // Zero
			big.NewInt(-1), // Negative
		}

		for _, amount := range invalidAmounts {
			assert.Error(t, ValidateTRXAmount(amount))
		}
	})
}

func TestFreezeAmountValidation(t *testing.T) {
	t.Run("Valid freeze amounts", func(t *testing.T) {
		validAmounts := []*big.Int{
			big.NewInt(1000000),  // 1 TRX (minimum)
			big.NewInt(10000000), // 10 TRX
		}

		for _, amount := range validAmounts {
			assert.NoError(t, ValidateFreezeAmount(amount))
		}
	})

	t.Run("Invalid freeze amounts", func(t *testing.T) {
		invalidAmounts := []*big.Int{
			big.NewInt(500000), // Less than 1 TRX
			big.NewInt(0),      // Zero
			big.NewInt(-1),     // Negative
		}

		for _, amount := range invalidAmounts {
			assert.Error(t, ValidateFreezeAmount(amount))
		}
	})
}

func TestContractValidation(t *testing.T) {
	t.Run("Valid contract addresses", func(t *testing.T) {
		validAddresses := []string{
			"TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb",
			"e28b3cfd4e0e909077821478e9fcb86b84be786e",
		}

		for _, addr := range validAddresses {
			assert.True(t, IsValidContractAddress(addr))
		}
	})

	t.Run("Contract data validation", func(t *testing.T) {
		// Valid data with method signature
		validData := []byte{0xa9, 0x05, 0x9c, 0xbb} // transfer(address,uint256) signature
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
		"",           // Empty
		"123invalid", // Starts with number
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
		"",              // Empty
		"toolongsymbol", // Too long (>10 chars)
		"invalid-symbol", // Contains hyphen
		"symbol with space", // Contains space
		"symbol@",       // Contains special character
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
		"",                    // Empty
		"invalid-url",         // No port
		"http://example.com",  // Wrong protocol
        "127.0.0.1",           // Missing port
        "127.0.0.1:abc",       // Invalid port
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

func TestBatchValidation(t *testing.T) {
	t.Run("Valid addresses batch", func(t *testing.T) {
		addresses := []string{
			"TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb",
			"TXNYeYdao7JL7wBtmzbk7mAie7UZsdgVjx",
		}
		assert.NoError(t, ValidateAddresses(addresses))
	})

	t.Run("Invalid addresses batch", func(t *testing.T) {
		addresses := []string{
			"TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb",
			"invalid-address",
		}
		assert.Error(t, ValidateAddresses(addresses))
	})

	t.Run("Valid amounts batch", func(t *testing.T) {
		amounts := []*big.Int{
			big.NewInt(1000000),
			big.NewInt(2000000),
		}
		assert.NoError(t, ValidateAmounts(amounts, big.NewInt(1000000)))
	})

	t.Run("Invalid amounts batch", func(t *testing.T) {
		amounts := []*big.Int{
			big.NewInt(1000000),
			big.NewInt(500000), // Below minimum
		}
		assert.Error(t, ValidateAmounts(amounts, big.NewInt(1000000)))
	})
}