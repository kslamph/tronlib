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

// getContractOwnerAddress is a helper method to parse the contract address
func (t *TRC20Contract) getContractOwnerAddress() (*types.Address, error) {
	return types.NewAddress(t.Address)
}

// callConstantMethod is a helper to reduce repetitive constant method calling pattern
func (t *TRC20Contract) callConstantMethod(method string, params ...interface{}) (interface{}, error) {
	data, err := t.EncodeInput(method, params...)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s call: %v", method, err)
	}

	ownerAddr, err := t.getContractOwnerAddress()
	if err != nil {
		return nil, fmt.Errorf("failed to parse owner address: %v", err)
	}

	result, err := t.client.TriggerConstantSmartContract(t.Contract, ownerAddr, data)
	if err != nil {
		return nil, fmt.Errorf("failed to call %s: %v", method, err)
	}

	decoded, err := t.DecodeResult(method, result)
	if err != nil {
		return nil, fmt.Errorf("failed to decode %s result: %v", method, err)
	}

	return decoded, nil
}

// convertToDecimal converts a big.Int result to decimal with proper precision
func (t *TRC20Contract) convertToDecimal(value *big.Int) (decimal.Decimal, error) {
	decimals, err := t.Decimals()
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to get decimals: %v", err)
	}
	return decimal.NewFromBigInt(value, -int32(decimals)), nil
}

// Name returns the name of the token
func (t *TRC20Contract) Name() (string, error) {
	decoded, err := t.callConstantMethod("name")
	if err != nil {
		return "", err
	}
	return decoded.(string), nil
}

// Symbol returns the symbol of the token (cached)
func (t *TRC20Contract) Symbol() (string, error) {
	t.symbolOnce.Do(func() {
		decoded, err := t.callConstantMethod("symbol")
		if err != nil {
			t.symbolErr = err
			return
		}
		t.symbolCache = decoded.(string)
	})

	return t.symbolCache, t.symbolErr
}

// Decimals returns the number of decimals used to format token amounts (cached)
func (t *TRC20Contract) Decimals() (uint8, error) {
	t.decimalsOnce.Do(func() {
		decoded, err := t.callConstantMethod("decimals")
		if err != nil {
			t.decimalsErr = err
			return
		}
		t.decimalsCache = uint8(decoded.(*big.Int).Uint64())
	})

	return t.decimalsCache, t.decimalsErr
}

// TotalSupply returns the total token supply as decimal
func (t *TRC20Contract) TotalSupply() (decimal.Decimal, error) {
	decoded, err := t.callConstantMethod("totalSupply")
	if err != nil {
		return decimal.Zero, err
	}
	return t.convertToDecimal(decoded.(*big.Int))
}

// BalanceOf returns the account balance of another account with address as decimal
func (t *TRC20Contract) BalanceOf(address string) (decimal.Decimal, error) {
	decoded, err := t.callConstantMethod("balanceOf", address)
	if err != nil {
		return decimal.Zero, err
	}
	return t.convertToDecimal(decoded.(*big.Int))
}

// Allowance returns the amount which spender is still allowed to withdraw from owner
func (t *TRC20Contract) Allowance(owner, spender string) (decimal.Decimal, error) {
	decoded, err := t.callConstantMethod("allowance", owner, spender)
	if err != nil {
		return decimal.Zero, err
	}
	return t.convertToDecimal(decoded.(*big.Int))
}
