package smartcontract

import (
	"fmt"
	"math/big"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
)

// TRC20Contract represents a TRC20 token contract
type TRC20Contract struct {
	*types.Contract
	client *client.Client
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

	// Use contract address as caller since it's a view function
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

// Symbol returns the symbol of the token
func (t *TRC20Contract) Symbol() (string, error) {
	data, err := t.ContractTrigger("symbol")
	if err != nil {
		return "", fmt.Errorf("failed to create symbol call: %v", err)
	}

	ownerAddr, err := types.NewAddress(t.Address)
	if err != nil {
		return "", fmt.Errorf("failed to parse owner address: %v", err)
	}

	result, err := t.client.TriggerConstantSmartContract(t.Contract, ownerAddr, data)
	if err != nil {
		return "", fmt.Errorf("failed to call symbol: %v", err)
	}

	decoded, err := t.DecodeResult("symbol", result)
	if err != nil {
		return "", fmt.Errorf("failed to decode symbol result: %v", err)
	}

	return decoded.(string), nil
}

// Decimals returns the number of decimals used to format token amounts
func (t *TRC20Contract) Decimals() (uint8, error) {
	data, err := t.ContractTrigger("decimals")
	if err != nil {
		return 0, fmt.Errorf("failed to create decimals call: %v", err)
	}

	ownerAddr, err := types.NewAddress(t.Address)
	if err != nil {
		return 0, fmt.Errorf("failed to parse owner address: %v", err)
	}

	result, err := t.client.TriggerConstantSmartContract(t.Contract, ownerAddr, data)
	if err != nil {
		return 0, fmt.Errorf("failed to call decimals: %v", err)
	}

	decoded, err := t.DecodeResult("decimals", result)
	if err != nil {
		return 0, fmt.Errorf("failed to decode decimals result: %v", err)
	}

	return uint8(decoded.(*big.Int).Uint64()), nil
}

// TotalSupply returns the total token supply
func (t *TRC20Contract) TotalSupply() (*big.Int, error) {
	data, err := t.ContractTrigger("totalSupply")
	if err != nil {
		return nil, fmt.Errorf("failed to create totalSupply call: %v", err)
	}

	ownerAddr, err := types.NewAddress(t.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to parse owner address: %v", err)
	}

	result, err := t.client.TriggerConstantSmartContract(t.Contract, ownerAddr, data)
	if err != nil {
		return nil, fmt.Errorf("failed to call totalSupply: %v", err)
	}

	decoded, err := t.DecodeResult("totalSupply", result)
	if err != nil {
		return nil, fmt.Errorf("failed to decode totalSupply result: %v", err)
	}

	return decoded.(*big.Int), nil
}

// BalanceOf returns the account balance of another account with address
func (t *TRC20Contract) BalanceOf(address string) (*big.Int, error) {
	data, err := t.ContractTrigger("balanceOf", address)
	if err != nil {
		return nil, fmt.Errorf("failed to create balanceOf call: %v", err)
	}

	ownerAddr, err := types.NewAddress(t.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to parse owner address: %v", err)
	}

	result, err := t.client.TriggerConstantSmartContract(t.Contract, ownerAddr, data)
	if err != nil {
		return nil, fmt.Errorf("failed to call balanceOf: %v", err)
	}

	decoded, err := t.DecodeResult("balanceOf", result)
	if err != nil {
		return nil, fmt.Errorf("failed to decode balanceOf result: %v", err)
	}

	return decoded.(*big.Int), nil
}

// Transfer transfers tokens to a specified address
func (t *TRC20Contract) Transfer(to string, amount *big.Int) ([]byte, error) {
	return t.ContractTrigger("transfer", to, amount)
}

// TransferFrom transfers tokens from one address to another
func (t *TRC20Contract) TransferFrom(from, to string, amount *big.Int) ([]byte, error) {
	return t.ContractTrigger("transferFrom", from, to, amount)
}

// Approve allows spender to withdraw from your account multiple times up to the amount
func (t *TRC20Contract) Approve(spender string, amount *big.Int) ([]byte, error) {
	return t.ContractTrigger("approve", spender, amount)
}

// Allowance returns the amount which spender is still allowed to withdraw from owner
func (t *TRC20Contract) Allowance(owner, spender string) (*big.Int, error) {
	data, err := t.ContractTrigger("allowance", owner, spender)
	if err != nil {
		return nil, fmt.Errorf("failed to create allowance call: %v", err)
	}

	ownerAddr, err := types.NewAddress(t.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to parse owner address: %v", err)
	}

	result, err := t.client.TriggerConstantSmartContract(t.Contract, ownerAddr, data)
	if err != nil {
		return nil, fmt.Errorf("failed to call allowance: %v", err)
	}

	decoded, err := t.DecodeResult("allowance", result)
	if err != nil {
		return nil, fmt.Errorf("failed to decode allowance result: %v", err)
	}

	return decoded.(*big.Int), nil
}
