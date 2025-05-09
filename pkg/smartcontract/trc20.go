package smartcontract

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/shopspring/decimal"
)

// TRC20Contract represents a TRC20 token contract
type TRC20Contract struct {
	*types.Contract
	client *client.Client

	// Cached values
	symbolOnce    sync.Once
	symbolCache   string
	symbolErr     error
	decimalsOnce  sync.Once
	decimalsCache uint8
	decimalsErr   error
}

// NewTRC20Contract creates a new TRC20 contract instance
func NewTRC20Contract(address string, client *client.Client) (*TRC20Contract, error) {
	contract, err := types.NewContract(types.ERC20ABI, address)
	if err != nil {
		return nil, fmt.Errorf("failed to create TRC20 contract: %v", err)
	}

	return &TRC20Contract{
		Contract: contract,
		client:   client,
	}, nil
}

// Name returns the name of the token
func (t *TRC20Contract) Name() (string, error) {
	data, err := t.ContractTrigger("name")
	if err != nil {
		return "", fmt.Errorf("failed to create name call: %v", err)
	}

	ownerAddr, err := types.NewAddress(t.Address)
	if err != nil {
		return "", fmt.Errorf("failed to parse owner address: %v", err)
	}

	result, err := t.client.TriggerConstantSmartContract(t.Contract, ownerAddr, data)
	if err != nil {
		return "", fmt.Errorf("failed to call name: %v", err)
	}

	decoded, err := t.DecodeResult("name", result)
	if err != nil {
		return "", fmt.Errorf("failed to decode name result: %v", err)
	}

	return decoded.(string), nil
}

// Symbol returns the symbol of the token (cached)
func (t *TRC20Contract) Symbol() (string, error) {
	t.symbolOnce.Do(func() {
		data, err := t.ContractTrigger("symbol")
		if err != nil {
			t.symbolErr = fmt.Errorf("failed to create symbol call: %v", err)
			return
		}

		ownerAddr, err := types.NewAddress(t.Address)
		if err != nil {
			t.symbolErr = fmt.Errorf("failed to parse owner address: %v", err)
			return
		}

		result, err := t.client.TriggerConstantSmartContract(t.Contract, ownerAddr, data)
		if err != nil {
			t.symbolErr = fmt.Errorf("failed to call symbol: %v", err)
			return
		}

		decoded, err := t.DecodeResult("symbol", result)
		if err != nil {
			t.symbolErr = fmt.Errorf("failed to decode symbol result: %v", err)
			return
		}

		t.symbolCache = decoded.(string)
	})

	return t.symbolCache, t.symbolErr
}

// Decimals returns the number of decimals used to format token amounts (cached)
func (t *TRC20Contract) Decimals() (uint8, error) {
	t.decimalsOnce.Do(func() {
		data, err := t.ContractTrigger("decimals")
		if err != nil {
			t.decimalsErr = fmt.Errorf("failed to create decimals call: %v", err)
			return
		}

		ownerAddr, err := types.NewAddress(t.Address)
		if err != nil {
			t.decimalsErr = fmt.Errorf("failed to parse owner address: %v", err)
			return
		}

		result, err := t.client.TriggerConstantSmartContract(t.Contract, ownerAddr, data)
		if err != nil {
			t.decimalsErr = fmt.Errorf("failed to call decimals: %v", err)
			return
		}

		decoded, err := t.DecodeResult("decimals", result)
		if err != nil {
			t.decimalsErr = fmt.Errorf("failed to decode decimals result: %v", err)
			return
		}

		t.decimalsCache = uint8(decoded.(*big.Int).Uint64())
	})

	return t.decimalsCache, t.decimalsErr
}

// TotalSupply returns the total token supply as decimal
func (t *TRC20Contract) TotalSupply() (decimal.Decimal, error) {
	data, err := t.ContractTrigger("totalSupply")
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to create totalSupply call: %v", err)
	}

	ownerAddr, err := types.NewAddress(t.Address)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to parse owner address: %v", err)
	}

	result, err := t.client.TriggerConstantSmartContract(t.Contract, ownerAddr, data)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to call totalSupply: %v", err)
	}

	decoded, err := t.DecodeResult("totalSupply", result)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to decode totalSupply result: %v", err)
	}

	decimals, err := t.Decimals()
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to get decimals: %v", err)
	}

	// Convert big.Int to decimal and adjust for decimals
	return decimal.NewFromBigInt(decoded.(*big.Int), -int32(decimals)), nil
}

// BalanceOf returns the account balance of another account with address as decimal
func (t *TRC20Contract) BalanceOf(address string) (decimal.Decimal, error) {
	data, err := t.ContractTrigger("balanceOf", address)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to create balanceOf call: %v", err)
	}

	ownerAddr, err := types.NewAddress(t.Address)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to parse owner address: %v", err)
	}

	result, err := t.client.TriggerConstantSmartContract(t.Contract, ownerAddr, data)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to call balanceOf: %v", err)
	}

	decoded, err := t.DecodeResult("balanceOf", result)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to decode balanceOf result: %v", err)
	}

	decimals, err := t.Decimals()
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to get decimals: %v", err)
	}

	// Convert big.Int to decimal and adjust for decimals
	return decimal.NewFromBigInt(decoded.(*big.Int), -int32(decimals)), nil
}

// Transfer transfers tokens to a specified address
func (t *TRC20Contract) Transfer(to string, amount decimal.Decimal) ([]byte, error) {
	decimals, err := t.Decimals()
	if err != nil {
		return nil, fmt.Errorf("failed to get decimals: %v", err)
	}

	// Convert decimal amount to big.Int with proper decimals
	rawAmount := amount.Shift(int32(decimals)).BigInt()
	return t.ContractTrigger("transfer", to, rawAmount)
}

// TransferFrom transfers tokens from one address to another
func (t *TRC20Contract) TransferFrom(from, to string, amount decimal.Decimal) ([]byte, error) {
	decimals, err := t.Decimals()
	if err != nil {
		return nil, fmt.Errorf("failed to get decimals: %v", err)
	}

	// Convert decimal amount to big.Int with proper decimals
	rawAmount := amount.Shift(int32(decimals)).BigInt()
	return t.ContractTrigger("transferFrom", from, to, rawAmount)
}

// Approve allows spender to withdraw from your account multiple times up to the amount
func (t *TRC20Contract) Approve(spender string, amount decimal.Decimal) ([]byte, error) {
	decimals, err := t.Decimals()
	if err != nil {
		return nil, fmt.Errorf("failed to get decimals: %v", err)
	}

	// Convert decimal amount to big.Int with proper decimals
	rawAmount := amount.Shift(int32(decimals)).BigInt()
	return t.ContractTrigger("approve", spender, rawAmount)
}

// Allowance returns the amount which spender is still allowed to withdraw from owner
func (t *TRC20Contract) Allowance(owner, spender string) (decimal.Decimal, error) {
	data, err := t.ContractTrigger("allowance", owner, spender)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to create allowance call: %v", err)
	}

	ownerAddr, err := types.NewAddress(t.Address)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to parse owner address: %v", err)
	}

	result, err := t.client.TriggerConstantSmartContract(t.Contract, ownerAddr, data)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to call allowance: %v", err)
	}

	decoded, err := t.DecodeResult("allowance", result)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to decode allowance result: %v", err)
	}

	decimals, err := t.Decimals()
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to get decimals: %v", err)
	}

	// Convert big.Int to decimal and adjust for decimals
	return decimal.NewFromBigInt(decoded.(*big.Int), -int32(decimals)), nil
}
