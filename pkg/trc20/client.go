package trc20

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/smartcontract"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/shopspring/decimal"
)

// TRC20Manager provides a high-level, type-safe interface for TRC20 token interactions.
type TRC20Manager struct {
	contract *smartcontract.Contract // Underlying smart contract client

	// Cached properties (read-only and typically constant for a TRC20 token)
	cachedName     string
	cachedSymbol   string
	cachedDecimals uint8
	mu             sync.RWMutex // Mutex for thread-safe access to cached properties

	// Pre-parsed ABI for common TRC20 methods
	trc20ABI *abi.ABI
}

// NewManager creates a new TRC20 instance.
// It takes the contract address and an initialized tronlib client.
func NewManager(tronClient *client.Client, contractAddress *types.Address) (*TRC20Manager, error) {
	// Create a generic smart contract instance
	contract, err := smartcontract.NewContract(tronClient, contractAddress, ERC20ABI)
	if err != nil {
		return nil, fmt.Errorf("failed to create smart contract instance for TRC20: %w", err)
	}

	parsedABI, err := abi.JSON(strings.NewReader(ERC20ABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse TRC20 ABI: %w", err)
	}

	c := &TRC20Manager{
		contract: contract,
		trc20ABI: &parsedABI,
	}

	// Pre-fetch immutable properties using a non-cancellable context
	prefetchCtx, cancel := context.WithTimeout(context.Background(), tronClient.GetTimeout())
	defer cancel()

	_, err = c.Decimals(prefetchCtx) // This will populate cachedDecimals
	if err != nil {
		return nil, fmt.Errorf("failed to pre-fetch decimals: %w", err)
	}
	_, err = c.Name(prefetchCtx) // This will populate cachedName
	if err != nil {
		return nil, fmt.Errorf("failed to pre-fetch name: %w", err)
	}
	_, err = c.Symbol(prefetchCtx) // This will populate cachedSymbol
	if err != nil {
		return nil, fmt.Errorf("failed to pre-fetch symbol: %w", err)
	}

	return c, nil
}

// ToWeiWithDecimals is a convenience wrapper that converts the given amount using explicit decimals.
func ToWeiWithDecimals(amount decimal.Decimal, decimals uint8) (*big.Int, error) {
	return ToWei(amount, decimals)
}

// FromWeiWithDecimals is a convenience wrapper that converts the given value using explicit decimals.
func FromWeiWithDecimals(value *big.Int, decimals uint8) (decimal.Decimal, error) {
	return FromWei(value, decimals)
}

// Name returns the name of the TRC20 token, fetching and caching it on first call.
func (t *TRC20Manager) Name(ctx context.Context) (string, error) {
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

	result, err := t.contract.TriggerConstantContract(ctx, t.contract.Address, "name")
	if err != nil {
		return "", fmt.Errorf("failed to call name method: %w", err)
	}

	name, ok := result.(string)
	if !ok {
		return "", fmt.Errorf("unexpected type for name value: %T", result)
	}
	t.cachedName = name
	return name, nil
}

// Symbol returns the symbol of the TRC20 token, fetching and caching it on first call.
func (t *TRC20Manager) Symbol(ctx context.Context) (string, error) {
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

	result, err := t.contract.TriggerConstantContract(ctx, t.contract.Address, "symbol")
	if err != nil {
		return "", fmt.Errorf("failed to call symbol method: %w", err)
	}

	symbol, ok := result.(string)
	if !ok {
		return "", fmt.Errorf("unexpected type for symbol value: %T", result)
	}
	t.cachedSymbol = symbol
	return symbol, nil
}

// Decimals returns the number of decimal places of the TRC20 token, fetching and caching it on first call.
func (t *TRC20Manager) Decimals(ctx context.Context) (uint8, error) {
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

	result, err := t.contract.TriggerConstantContract(ctx, t.contract.Address, "decimals")
	if err != nil {
		return 0, fmt.Errorf("failed to call decimals method: %w", err)
	}

	// The `go-ethereum/abi` library decodes `uint8` from `uint256` as `*big.Int`
	// so we need to convert it.
	decimalsResult, ok := result.(uint8)
	if !ok {
		return 0, fmt.Errorf("unexpected type for uint8 result: %T", result)
	}
	t.cachedDecimals = decimalsResult
	return decimalsResult, nil
}

// BalanceOf retrieves the balance of an owner address as a decimal.Decimal.
func (t *TRC20Manager) BalanceOf(ctx context.Context, ownerAddress *types.Address) (decimal.Decimal, error) {
	decimals, err := t.Decimals(ctx)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to get decimals for BalanceOf: %w", err)
	}

	result, err := t.contract.TriggerConstantContract(ctx, ownerAddress, "balanceOf", ownerAddress)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to call balanceOf method: %w", err)
	}

	bitIntBalance, ok := result.(*big.Int)
	if !ok {
		return decimal.Zero, fmt.Errorf("unexpected type for balanceOf value: %T", result)
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
func (t *TRC20Manager) Transfer(ctx context.Context, fromAddress *types.Address, toAddress *types.Address, amount decimal.Decimal) (string, *api.TransactionExtention, error) {
	decimals, err := t.Decimals(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("%w: failed to get decimals for Transfer", err)
	}

	rawAmount, err := toWei(amount, decimals)
	if err != nil {
		return "", nil, fmt.Errorf("%w: invalid amount", err)
	}

	txExt, err := t.contract.TriggerSmartContract(ctx, fromAddress, 0, "transfer", toAddress, rawAmount)
	if err != nil {
		return "", nil, fmt.Errorf("%w: failed to call transfer method", err)
	}

	txidHex := fmt.Sprintf("%x", txExt.GetTxid())
	return txidHex, txExt, nil
}

// Approve approves a spender to spend tokens on behalf of the caller.
func (t *TRC20Manager) Approve(ctx context.Context, ownerAddress *types.Address, spenderAddress *types.Address, amount decimal.Decimal) (string, *api.TransactionExtention, error) {
	decimals, err := t.Decimals(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("%w: failed to get decimals for Approve", err)
	}

	rawAmount, err := toWei(amount, decimals)
	if err != nil {
		return "", nil, fmt.Errorf("%w: invalid amount", err)
	}

	txExt, err := t.contract.TriggerSmartContract(ctx, ownerAddress, 0, "approve", spenderAddress, rawAmount)
	if err != nil {
		return "", nil, fmt.Errorf("%w: failed to call approve method", err)
	}

	txidHex := fmt.Sprintf("%x", txExt.GetTxid())
	return txidHex, txExt, nil
}

// Allowance retrieves the allowance amount a spender has over an owner's tokens.
func (t *TRC20Manager) Allowance(ctx context.Context, ownerAddress *types.Address, spenderAddress *types.Address) (decimal.Decimal, error) {
	decimals, err := t.Decimals(ctx)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to get decimals for Allowance: %w", err)
	}

	result, err := t.contract.TriggerConstantContract(ctx, ownerAddress, "allowance", ownerAddress, spenderAddress)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to call allowance method: %w", err)
	}

	rawAllowance, ok := result.(*big.Int)
	if !ok {
		return decimal.Zero, fmt.Errorf("unexpected type for allowance value: %T", result)
	}

	convertedAllowance, err := fromWei(rawAllowance, decimals)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to convert raw allowance: %w", err)

	}

	return convertedAllowance, nil
}
