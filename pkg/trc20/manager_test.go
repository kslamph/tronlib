package trc20

import (
	"context"
	"testing"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// Valid TRON addresses with proper checksums from pkg_old tests
const (
	validAddress1 = "TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb"
	validAddress2 = "TXNYeYdao7JL7wBtmzbk7mAie7UZsdgVjx"
	validAddress3 = "TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x"
	validAddress4 = "TNUC9Qb1rRpS5CbWLmNMxXBjyFoydXjWFR"
)

func TestNewManager(t *testing.T) {
	client := &client.Client{}
	manager := NewManager(client)
	assert.NotNil(t, manager)
	assert.Equal(t, client, manager.client)
}

func TestManager_NewContract_Validation(t *testing.T) {
	manager := NewManager(&client.Client{})

	// Test invalid contract address
	_, err := manager.NewContract("invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid contract address")

	// Test empty contract address
	_, err = manager.NewContract("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid contract address")

	// Test valid contract address
	contract, err := manager.NewContract(validAddress1)
	assert.NoError(t, err)
	assert.NotNil(t, contract)
	assert.Equal(t, validAddress1, contract.address)
}

func TestContract_BalanceOf_Validation(t *testing.T) {
	manager := NewManager(&client.Client{})
	contract := &Contract{
		manager: manager,
		address: validAddress1,
	}
	ctx := context.Background()

	// Test invalid address
	_, err := contract.BalanceOf(ctx, "invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid address")

	// Test empty address
	_, err = contract.BalanceOf(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid address")
}

func TestContract_Allowance_Validation(t *testing.T) {
	manager := NewManager(&client.Client{})
	contract := &Contract{
		manager: manager,
		address: validAddress1,
	}
	ctx := context.Background()

	// Test invalid owner address
	_, err := contract.Allowance(ctx, "invalid", validAddress2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid owner address")

	// Test invalid spender address
	_, err = contract.Allowance(ctx, validAddress1, "invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid spender address")

	// Test empty owner address
	_, err = contract.Allowance(ctx, "", validAddress2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid owner address")

	// Test empty spender address
	_, err = contract.Allowance(ctx, validAddress1, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid spender address")
}

func TestContract_Transfer_Validation(t *testing.T) {
	manager := NewManager(&client.Client{})
	contract := &Contract{
		manager: manager,
		address: validAddress1,
	}
	ctx := context.Background()
	amount := decimal.NewFromFloat(100.5)

	// Test invalid from address
	_, err := contract.Transfer(ctx, "invalid", validAddress2, amount)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid from address")

	// Test invalid to address
	_, err = contract.Transfer(ctx, validAddress1, "invalid", amount)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid to address")

	// Test zero amount
	_, err = contract.Transfer(ctx, validAddress1, validAddress2, decimal.Zero)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "amount must be positive")

	// Test negative amount
	_, err = contract.Transfer(ctx, validAddress1, validAddress2, decimal.NewFromFloat(-10))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "amount must be positive")

	// Test same from and to addresses
	_, err = contract.Transfer(ctx, validAddress1, validAddress1, amount)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "from and to addresses cannot be the same")

	// Test empty from address
	_, err = contract.Transfer(ctx, "", validAddress2, amount)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid from address")

	// Test empty to address
	_, err = contract.Transfer(ctx, validAddress1, "", amount)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid to address")
}

func TestContract_TransferFrom_Validation(t *testing.T) {
	manager := NewManager(&client.Client{})
	contract := &Contract{
		manager: manager,
		address: validAddress1,
	}
	ctx := context.Background()
	amount := decimal.NewFromFloat(100.5)

	// Test invalid spender address
	_, err := contract.TransferFrom(ctx, "invalid", validAddress2, validAddress3, amount)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid spender address")

	// Test invalid from address
	_, err = contract.TransferFrom(ctx, validAddress1, "invalid", validAddress3, amount)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid from address")

	// Test invalid to address
	_, err = contract.TransferFrom(ctx, validAddress1, validAddress2, "invalid", amount)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid to address")

	// Test zero amount
	_, err = contract.TransferFrom(ctx, validAddress1, validAddress2, validAddress3, decimal.Zero)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "amount must be positive")

	// Test negative amount
	_, err = contract.TransferFrom(ctx, validAddress1, validAddress2, validAddress3, decimal.NewFromFloat(-10))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "amount must be positive")

	// Test same from and to addresses
	_, err = contract.TransferFrom(ctx, validAddress1, validAddress2, validAddress2, amount)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "from and to addresses cannot be the same")
}

func TestContract_Approve_Validation(t *testing.T) {
	manager := NewManager(&client.Client{})
	contract := &Contract{
		manager: manager,
		address: validAddress1,
	}
	ctx := context.Background()
	amount := decimal.NewFromFloat(100.5)

	// Test invalid from address
	_, err := contract.Approve(ctx, "invalid", validAddress2, amount)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid from address")

	// Test invalid spender address
	_, err = contract.Approve(ctx, validAddress1, "invalid", amount)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid spender address")

	// Test negative amount
	_, err = contract.Approve(ctx, validAddress1, validAddress2, decimal.NewFromFloat(-10))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "amount cannot be negative")

	// Test same from and spender addresses
	_, err = contract.Approve(ctx, validAddress1, validAddress1, amount)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "from and spender addresses cannot be the same")

	// Test empty from address
	_, err = contract.Approve(ctx, "", validAddress2, amount)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid from address")

	// Test empty spender address
	_, err = contract.Approve(ctx, validAddress1, "", amount)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid spender address")

	// Test zero amount (should be allowed for approve)
	// This test would fail with network error but validation should pass
	// since zero amount is valid for approve (to revoke approval)
}

func TestDecimalAmounts(t *testing.T) {
	// Test decimal creation and manipulation
	amount1 := decimal.NewFromFloat(100.5)
	assert.Equal(t, "100.5", amount1.String())

	amount2 := decimal.NewFromInt(1000)
	assert.Equal(t, "1000", amount2.String())

	// Test zero
	assert.True(t, decimal.Zero.IsZero())
	assert.True(t, decimal.Zero.LessThanOrEqual(decimal.Zero))

	// Test negative
	negative := decimal.NewFromFloat(-10)
	assert.True(t, negative.LessThan(decimal.Zero))
}

func TestMethodSignatures(t *testing.T) {
	// Test that method signatures are generated correctly
	assert.NotNil(t, methodSignature("name()"))
	assert.NotNil(t, methodSignature("symbol()"))
	assert.NotNil(t, methodSignature("decimals()"))
	assert.NotNil(t, methodSignature("totalSupply()"))
	assert.NotNil(t, methodSignature("balanceOf(address)"))
	assert.NotNil(t, methodSignature("allowance(address,address)"))
	assert.NotNil(t, methodSignature("transfer(address,uint256)"))
	assert.NotNil(t, methodSignature("transferFrom(address,address,uint256)"))
	assert.NotNil(t, methodSignature("approve(address,uint256)"))
	
	// Test invalid method
	assert.Nil(t, methodSignature("invalidMethod()"))
}

func TestEncodeAddress(t *testing.T) {
	// Test valid address encoding
	encoded, err := encodeAddress(validAddress1)
	assert.NoError(t, err)
	assert.Equal(t, 32, len(encoded))

	// Test invalid address
	_, err = encodeAddress("invalid")
	assert.Error(t, err)

	// Test empty address
	_, err = encodeAddress("")
	assert.Error(t, err)
}