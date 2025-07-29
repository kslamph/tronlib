package trc20

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/smartcontract"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/shopspring/decimal"
	// For ABIEncoder, ABIDecoder, etc.
)

// TRC20Client provides a high-level, type-safe interface for TRC20 token interactions.
type TRC20Client struct {
	contract *smartcontract.Contract // Underlying smart contract client

	// Cached properties (read-only and typically constant for a TRC20 token)
	cachedName     string
	cachedSymbol   string
	cachedDecimals uint8
	mu             sync.RWMutex // Mutex for thread-safe access to cached properties

	// Pre-parsed ABI for common TRC20 methods
	trc20ABI *abi.ABI
}

// NewTRC20Client creates a new TRC20Client instance.
// It takes the contract address and an initialized tronlib client.
// NewTRC20Client creates a new TRC20Client instance.
// It takes the contract address and an initialized tronlib client.
func NewTRC20Client(tronClient *client.Client, contractAddress *types.Address) (*TRC20Client, error) {
	// Create a generic smart contract instance
	contract, err := smartcontract.NewContract(tronClient, contractAddress, ERC20ABI)
	if err != nil {
		return nil, fmt.Errorf("failed to create smart contract instance for TRC20: %w", err)
	}

	parsedABI, err := abi.JSON(strings.NewReader(ERC20ABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse TRC20 ABI: %w", err)
	}

	c := &TRC20Client{
		contract: contract,
		trc20ABI: &parsedABI,
	}

	// Pre-fetch immutable properties
	_, err = c.Decimals() // This will populate cachedDecimals
	if err != nil {
		return nil, fmt.Errorf("failed to pre-fetch decimals: %w", err)
	}
	_, err = c.Name() // This will populate cachedName
	if err != nil {
		return nil, fmt.Errorf("failed to pre-fetch name: %w", err)
	}
	_, err = c.Symbol() // This will populate cachedSymbol
	if err != nil {
		return nil, fmt.Errorf("failed to pre-fetch symbol: %w", err)
	}

	return c, nil
}

// Name returns the name of the TRC20 token, fetching and caching it on first call.
func (t *TRC20Client) Name() (string, error) {
	t.mu.RLock()
	if t.cachedName != "" {
		defer t.mu.RUnlock()
		return t.cachedName, nil
	}
	t.mu.RUnlock()

	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cachedName != "" { // Double-check locking
		return t.cachedName, nil
	}

	result, err := t.contract.TriggerConstantContract(context.Background(), t.contract.Address, "name")
	if err != nil {
		return "", fmt.Errorf("failed to call name method: %w", err)
	}
	nameResult, ok := result.([]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected type for name result: %T", result)
	}
	if len(nameResult) != 1 {
		return "", fmt.Errorf("unexpected length for name result: %d", len(nameResult))
	}
	name, ok := nameResult[0].(string)
	if !ok {
		return "", fmt.Errorf("unexpected type for name value: %T", nameResult[0])
	}
	t.cachedName = name
	return name, nil
}

// Symbol returns the symbol of the TRC20 token, fetching and caching it on first call.
func (t *TRC20Client) Symbol() (string, error) {
	t.mu.RLock()
	if t.cachedSymbol != "" {
		defer t.mu.RUnlock()
		return t.cachedSymbol, nil
	}
	t.mu.RUnlock()

	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cachedSymbol != "" { // Double-check locking
		return t.cachedSymbol, nil
	}

	result, err := t.contract.TriggerConstantContract(context.Background(), t.contract.Address, "symbol")
	if err != nil {
		return "", fmt.Errorf("failed to call symbol method: %w", err)
	}
	symbolResult, ok := result.([]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected type for symbol result: %T", result)
	}
	if len(symbolResult) != 1 {
		return "", fmt.Errorf("unexpected length for symbol result: %d", len(symbolResult))
	}
	symbol, ok := symbolResult[0].(string)
	if !ok {
		return "", fmt.Errorf("unexpected type for symbol value: %T", symbolResult[0])
	}
	t.cachedSymbol = symbol
	return symbol, nil
}

// Decimals returns the number of decimal places of the TRC20 token, fetching and caching it on first call.
func (t *TRC20Client) Decimals() (uint8, error) {
	t.mu.RLock()
	if t.cachedDecimals != 0 { // Assuming 0 is not a valid decimal count for a TRC20 token
		defer t.mu.RUnlock()
		return t.cachedDecimals, nil
	}
	t.mu.RUnlock()

	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cachedDecimals != 0 { // Double-check locking
		return t.cachedDecimals, nil
	}

	result, err := t.contract.TriggerConstantContract(context.Background(), t.contract.Address, "decimals")
	if err != nil {
		return 0, fmt.Errorf("failed to call decimals method: %w", err)
	}

	// The `go-ethereum/abi` library decodes `uint8` from `uint256` as `*big.Int`
	// so we need to convert it.
	decimalsResult, ok := result.([]interface{})
	if !ok {
		return 0, fmt.Errorf("unexpected type for decimals result: %T", result)
	}
	if len(decimalsResult) != 1 {
		return 0, fmt.Errorf("unexpected length for decimals result: %d", len(decimalsResult))
	}
	decimalsValue, ok := decimalsResult[0].(uint8)
	if !ok {
		return 0, fmt.Errorf("unexpected type for decimals value: %T", decimalsResult[0])
	}
	// if !decimalBigInt.IsUint64() || decimalBigInt.Uint64() > 255 {
	// 	return 0, fmt.Errorf("decimals value out of range for uint8: %s", decimalBigInt.String())
	// }
	// decimals := uint8(decimalBigInt.Uint64())
	t.cachedDecimals = decimalsValue
	return decimalsValue, nil
}

// BalanceOf retrieves the balance of an owner address as a decimal.Decimal.
func (t *TRC20Client) BalanceOf(ownerAddress *types.Address) (decimal.Decimal, error) {
	decimals, err := t.Decimals()
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to get decimals for BalanceOf: %w", err)
	}

	result, err := t.contract.TriggerConstantContract(context.Background(), ownerAddress, "balanceOf", ownerAddress)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to call balanceOf method: %w", err)
	}

	balanceResult, ok := result.([]interface{})
	if !ok {
		return decimal.Zero, fmt.Errorf("unexpected type for balanceOf result: %T", result)
	}
	if len(balanceResult) != 1 {
		return decimal.Zero, fmt.Errorf("unexpected length for balanceOf result: %d", len(balanceResult))
	}
	bitIntBalance, ok := balanceResult[0].(*big.Int)
	if !ok {
		return decimal.Zero, fmt.Errorf("unexpected type for balanceOf value: %T", balanceResult[0])
	}
	// bigIntBalance, ok := new(big.Int).SetString(rawBalance, 10)
	// if !ok {
	// 	return decimal.Zero, fmt.Errorf("failed to parse balance string: %s", rawBalance)
	// }
	convertedBalance, err := fromWei(bitIntBalance, decimals)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to convert raw balance: %w", err)
	}

	return convertedBalance, nil
}

// Transfer transfers tokens from the caller to a recipient, taking a decimal.Decimal amount.
func (t *TRC20Client) Transfer(fromAddress *types.Address, toAddress *types.Address, amount decimal.Decimal) (string, error) {
	decimals, err := t.Decimals()
	if err != nil {
		return "", fmt.Errorf("failed to get decimals for Transfer: %w", err)
	}

	rawAmount, err := toWei(amount, decimals)
	if err != nil {
		return "", fmt.Errorf("invalid amount: %w", err)
	}

	txExt, err := t.contract.TriggerSmartContract(context.Background(), fromAddress, 0, "transfer", toAddress.String(), rawAmount)
	if err != nil {
		return "", fmt.Errorf("failed to call transfer method: %w", err)
	}

	return fmt.Sprintf("%x", txExt.GetTxid()), nil
}

// Approve approves a spender to spend tokens on behalf of the caller.
func (t *TRC20Client) Approve(ownerAddress *types.Address, spenderAddress *types.Address, amount decimal.Decimal) (string, error) {
	decimals, err := t.Decimals()
	if err != nil {
		return "", fmt.Errorf("failed to get decimals for Approve: %w", err)
	}

	rawAmount, err := toWei(amount, decimals)
	if err != nil {
		return "", fmt.Errorf("invalid amount: %w", err)
	}

	txExt, err := t.contract.TriggerSmartContract(context.Background(), ownerAddress, 0, "approve", spenderAddress.String(), rawAmount)
	if err != nil {
		return "", fmt.Errorf("failed to call approve method: %w", err)
	}

	return fmt.Sprintf("%x", txExt.GetTxid()), nil
}

// Allowance retrieves the allowance amount a spender has over an owner's tokens.
func (t *TRC20Client) Allowance(ownerAddress *types.Address, spenderAddress *types.Address) (decimal.Decimal, error) {
	decimals, err := t.Decimals()
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to get decimals for Allowance: %w", err)
	}

	result, err := t.contract.TriggerConstantContract(context.Background(), ownerAddress, "allowance", ownerAddress, spenderAddress)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to call allowance method: %w", err)
	}

	allowanceResult, ok := result.([]interface{})
	if !ok {
		return decimal.Zero, fmt.Errorf("unexpected type for allowance result: %T", result)
	}
	if len(allowanceResult) != 1 {
		return decimal.Zero, fmt.Errorf("unexpected length for allowance result: %d", len(allowanceResult))
	}
	rawAllowance, ok := allowanceResult[0].(*big.Int)
	if !ok {
		return decimal.Zero, fmt.Errorf("unexpected type for allowance value: %T", allowanceResult[0])
	}

	convertedAllowance, err := fromWei(rawAllowance, decimals)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to convert raw allowance: %w", err)

	}

	return convertedAllowance, nil
}
