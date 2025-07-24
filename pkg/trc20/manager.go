// Package trc20 provides high-level TRC20 token functionality
package trc20

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/smartcontract"
	"github.com/kslamph/tronlib/pkg/utils"
	"github.com/shopspring/decimal"
)

// Manager handles TRC20 token operations
type Manager struct {
	client            *client.Client
	smartcontractMgr  *smartcontract.Manager
}

// NewManager creates a new TRC20 manager instance
func NewManager(client *client.Client) *Manager {
	return &Manager{
		client:           client,
		smartcontractMgr: smartcontract.NewManager(client),
	}
}

// Contract represents a TRC20 token contract
type Contract struct {
	manager *Manager
	address string
}

// NewContract creates a new TRC20 contract instance
func (m *Manager) NewContract(contractAddress string) (*Contract, error) {
	_, err := utils.ValidateAddress(contractAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid contract address: %v", err)
	}

	return &Contract{
		manager: m,
		address: contractAddress,
	}, nil
}

// Helper function to create method signature hash
func methodSignature(method string) []byte {
	// For TRC20, we use simplified method signatures
	// In a real implementation, this would use proper ABI encoding
	switch method {
	case "name()":
		return []byte{0x06, 0xfd, 0xde, 0x03} // name()
	case "symbol()":
		return []byte{0x95, 0xd8, 0x9b, 0x41} // symbol()
	case "decimals()":
		return []byte{0x31, 0x3c, 0xe5, 0x67} // decimals()
	case "totalSupply()":
		return []byte{0x18, 0x16, 0x0d, 0xdd} // totalSupply()
	case "balanceOf(address)":
		return []byte{0x70, 0xa0, 0x82, 0x31} // balanceOf(address)
	case "allowance(address,address)":
		return []byte{0xdd, 0x62, 0xed, 0x3e} // allowance(address,address)
	case "transfer(address,uint256)":
		return []byte{0xa9, 0x05, 0x9c, 0xbb} // transfer(address,uint256)
	case "transferFrom(address,address,uint256)":
		return []byte{0x23, 0xb8, 0x72, 0xdd} // transferFrom(address,address,uint256)
	case "approve(address,uint256)":
		return []byte{0x09, 0x5e, 0xa7, 0xb3} // approve(address,uint256)
	default:
		return nil
	}
}

// encodeAddress encodes a TRON address for contract calls
func encodeAddress(address string) ([]byte, error) {
	addr, err := utils.ValidateAddress(address)
	if err != nil {
		return nil, err
	}
	
	// Pad to 32 bytes for ABI encoding
	result := make([]byte, 32)
	copy(result[12:], addr.Bytes()) // Address goes in the last 20 bytes
	return result, nil
}

// encodeUint256 encodes a big.Int as uint256 for contract calls
func encodeUint256(value *big.Int) []byte {
	result := make([]byte, 32)
	value.FillBytes(result)
	return result
}

// Name returns the name of the token
func (c *Contract) Name(ctx context.Context) (string, error) {
	data := methodSignature("name()")
	if data == nil {
		return "", fmt.Errorf("failed to create method signature")
	}

	tx, err := c.manager.smartcontractMgr.TriggerConstantContract(ctx, c.address, c.address, data, 0)
	if err != nil {
		return "", fmt.Errorf("failed to call name: %v", err)
	}

	// Simplified result parsing - in real implementation would use proper ABI decoding
	if len(tx.ConstantResult) > 0 && len(tx.ConstantResult[0]) > 64 {
		// Skip the first 64 bytes (offset and length) and decode the string
		return string(tx.ConstantResult[0][64:]), nil
	}
	return "", fmt.Errorf("invalid response format")
}

// Symbol returns the symbol of the token
func (c *Contract) Symbol(ctx context.Context) (string, error) {
	data := methodSignature("symbol()")
	if data == nil {
		return "", fmt.Errorf("failed to create method signature")
	}

	tx, err := c.manager.smartcontractMgr.TriggerConstantContract(ctx, c.address, c.address, data, 0)
	if err != nil {
		return "", fmt.Errorf("failed to call symbol: %v", err)
	}

	// Simplified result parsing
	if len(tx.ConstantResult) > 0 && len(tx.ConstantResult[0]) > 64 {
		return string(tx.ConstantResult[0][64:]), nil
	}
	return "", fmt.Errorf("invalid response format")
}

// Decimals returns the number of decimals used to format token amounts
func (c *Contract) Decimals(ctx context.Context) (uint8, error) {
	data := methodSignature("decimals()")
	if data == nil {
		return 0, fmt.Errorf("failed to create method signature")
	}

	tx, err := c.manager.smartcontractMgr.TriggerConstantContract(ctx, c.address, c.address, data, 0)
	if err != nil {
		return 0, fmt.Errorf("failed to call decimals: %v", err)
	}

	// Simplified result parsing
	if len(tx.ConstantResult) > 0 && len(tx.ConstantResult[0]) >= 32 {
		result := new(big.Int).SetBytes(tx.ConstantResult[0][:32])
		return uint8(result.Uint64()), nil
	}
	return 0, fmt.Errorf("invalid response format")
}

// TotalSupply returns the total token supply as decimal
func (c *Contract) TotalSupply(ctx context.Context) (decimal.Decimal, error) {
	data := methodSignature("totalSupply()")
	if data == nil {
		return decimal.Zero, fmt.Errorf("failed to create method signature")
	}

	tx, err := c.manager.smartcontractMgr.TriggerConstantContract(ctx, c.address, c.address, data, 0)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to call totalSupply: %v", err)
	}

	if len(tx.ConstantResult) > 0 && len(tx.ConstantResult[0]) >= 32 {
		result := new(big.Int).SetBytes(tx.ConstantResult[0][:32])
		decimals, err := c.Decimals(ctx)
		if err != nil {
			return decimal.Zero, fmt.Errorf("failed to get decimals: %v", err)
		}
		return decimal.NewFromBigInt(result, -int32(decimals)), nil
	}
	return decimal.Zero, fmt.Errorf("invalid response format")
}

// BalanceOf returns the account balance of another account with address as decimal
func (c *Contract) BalanceOf(ctx context.Context, address string) (decimal.Decimal, error) {
	_, err := utils.ValidateAddress(address)
	if err != nil {
		return decimal.Zero, fmt.Errorf("invalid address: %v", err)
	}

	data := methodSignature("balanceOf(address)")
	if data == nil {
		return decimal.Zero, fmt.Errorf("failed to create method signature")
	}

	// Encode address parameter
	encodedAddr, err := encodeAddress(address)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to encode address: %v", err)
	}
	data = append(data, encodedAddr...)

	tx, err := c.manager.smartcontractMgr.TriggerConstantContract(ctx, c.address, c.address, data, 0)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to call balanceOf: %v", err)
	}

	if len(tx.ConstantResult) > 0 && len(tx.ConstantResult[0]) >= 32 {
		result := new(big.Int).SetBytes(tx.ConstantResult[0][:32])
		decimals, err := c.Decimals(ctx)
		if err != nil {
			return decimal.Zero, fmt.Errorf("failed to get decimals: %v", err)
		}
		return decimal.NewFromBigInt(result, -int32(decimals)), nil
	}
	return decimal.Zero, fmt.Errorf("invalid response format")
}

// Allowance returns the amount which spender is still allowed to withdraw from owner
func (c *Contract) Allowance(ctx context.Context, owner, spender string) (decimal.Decimal, error) {
	_, err := utils.ValidateAddress(owner)
	if err != nil {
		return decimal.Zero, fmt.Errorf("invalid owner address: %v", err)
	}
	_, err = utils.ValidateAddress(spender)
	if err != nil {
		return decimal.Zero, fmt.Errorf("invalid spender address: %v", err)
	}

	data := methodSignature("allowance(address,address)")
	if data == nil {
		return decimal.Zero, fmt.Errorf("failed to create method signature")
	}

	// Encode parameters
	encodedOwner, err := encodeAddress(owner)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to encode owner address: %v", err)
	}
	encodedSpender, err := encodeAddress(spender)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to encode spender address: %v", err)
	}
	data = append(data, encodedOwner...)
	data = append(data, encodedSpender...)

	tx, err := c.manager.smartcontractMgr.TriggerConstantContract(ctx, c.address, c.address, data, 0)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to call allowance: %v", err)
	}

	if len(tx.ConstantResult) > 0 && len(tx.ConstantResult[0]) >= 32 {
		result := new(big.Int).SetBytes(tx.ConstantResult[0][:32])
		decimals, err := c.Decimals(ctx)
		if err != nil {
			return decimal.Zero, fmt.Errorf("failed to get decimals: %v", err)
		}
		return decimal.NewFromBigInt(result, -int32(decimals)), nil
	}
	return decimal.Zero, fmt.Errorf("invalid response format")
}

// Transfer transfers tokens to a specified address
func (c *Contract) Transfer(ctx context.Context, from, to string, amount decimal.Decimal) (*api.TransactionExtention, error) {
	// Validate inputs
	_, err := utils.ValidateAddress(from)
	if err != nil {
		return nil, fmt.Errorf("invalid from address: %v", err)
	}
	_, err = utils.ValidateAddress(to)
	if err != nil {
		return nil, fmt.Errorf("invalid to address: %v", err)
	}
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("amount must be positive")
	}
	if strings.EqualFold(from, to) {
		return nil, fmt.Errorf("from and to addresses cannot be the same")
	}

	decimals, err := c.Decimals(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get decimals: %v", err)
	}

	// Convert decimal amount to big.Int with proper decimals
	rawAmount := amount.Shift(int32(decimals)).BigInt()

	data := methodSignature("transfer(address,uint256)")
	if data == nil {
		return nil, fmt.Errorf("failed to create method signature")
	}

	// Encode parameters
	encodedTo, err := encodeAddress(to)
	if err != nil {
		return nil, fmt.Errorf("failed to encode to address: %v", err)
	}
	encodedAmount := encodeUint256(rawAmount)
	data = append(data, encodedTo...)
	data = append(data, encodedAmount...)

	return c.manager.smartcontractMgr.TriggerContract(ctx, from, c.address, data, 0, 0, 0)
}

// TransferFrom transfers tokens from `from` to `to`
// `spender` approved by `from` is required
// transaction is expected to be signed by `spender`
func (c *Contract) TransferFrom(ctx context.Context, spender, from, to string, amount decimal.Decimal) (*api.TransactionExtention, error) {
	// Validate inputs
	_, err := utils.ValidateAddress(spender)
	if err != nil {
		return nil, fmt.Errorf("invalid spender address: %v", err)
	}
	_, err = utils.ValidateAddress(from)
	if err != nil {
		return nil, fmt.Errorf("invalid from address: %v", err)
	}
	_, err = utils.ValidateAddress(to)
	if err != nil {
		return nil, fmt.Errorf("invalid to address: %v", err)
	}
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("amount must be positive")
	}
	if strings.EqualFold(from, to) {
		return nil, fmt.Errorf("from and to addresses cannot be the same")
	}

	decimals, err := c.Decimals(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get decimals: %v", err)
	}

	rawAmount := amount.Shift(int32(decimals)).BigInt()

	data := methodSignature("transferFrom(address,address,uint256)")
	if data == nil {
		return nil, fmt.Errorf("failed to create method signature")
	}

	// Encode parameters
	encodedFrom, err := encodeAddress(from)
	if err != nil {
		return nil, fmt.Errorf("failed to encode from address: %v", err)
	}
	encodedTo, err := encodeAddress(to)
	if err != nil {
		return nil, fmt.Errorf("failed to encode to address: %v", err)
	}
	encodedAmount := encodeUint256(rawAmount)
	data = append(data, encodedFrom...)
	data = append(data, encodedTo...)
	data = append(data, encodedAmount...)

	return c.manager.smartcontractMgr.TriggerContract(ctx, spender, c.address, data, 0, 0, 0)
}

// Approve allows spender to withdraw from your account multiple times up to the amount
func (c *Contract) Approve(ctx context.Context, from, spender string, amount decimal.Decimal) (*api.TransactionExtention, error) {
	// Validate inputs
	_, err := utils.ValidateAddress(from)
	if err != nil {
		return nil, fmt.Errorf("invalid from address: %v", err)
	}
	_, err = utils.ValidateAddress(spender)
	if err != nil {
		return nil, fmt.Errorf("invalid spender address: %v", err)
	}
	if amount.LessThan(decimal.Zero) {
		return nil, fmt.Errorf("amount cannot be negative")
	}
	if strings.EqualFold(from, spender) {
		return nil, fmt.Errorf("from and spender addresses cannot be the same")
	}

	decimals, err := c.Decimals(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get decimals: %v", err)
	}

	// Convert decimal amount to big.Int with proper decimals
	rawAmount := amount.Shift(int32(decimals)).BigInt()

	data := methodSignature("approve(address,uint256)")
	if data == nil {
		return nil, fmt.Errorf("failed to create method signature")
	}

	// Encode parameters
	encodedSpender, err := encodeAddress(spender)
	if err != nil {
		return nil, fmt.Errorf("failed to encode spender address: %v", err)
	}
	encodedAmount := encodeUint256(rawAmount)
	data = append(data, encodedSpender...)
	data = append(data, encodedAmount...)

	return c.manager.smartcontractMgr.TriggerContract(ctx, from, c.address, data, 0, 0, 0)
}