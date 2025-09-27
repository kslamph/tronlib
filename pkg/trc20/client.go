package trc20

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pkg/client/lowlevel"
	"github.com/kslamph/tronlib/pkg/smartcontract"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/shopspring/decimal"
)

// TRC20Manager provides a high-level, type-safe interface for TRC20 token interactions.
//
// The TRC20Manager wraps a smart contract instance with convenience methods for
// common TRC20 operations. It automatically handles:
//   - Conversion between human-readable decimal amounts and on-chain integer values
//   - Caching of immutable token properties (name, symbol, decimals)
//   - Encoding and decoding of method calls and return values
//
// Use NewManager to create a new TRC20Manager instance for a specific token contract.
type TRC20Manager struct {
	contract *smartcontract.Instance // Underlying smart contract client

	// Cached properties (read-only and typically constant for a TRC20 token)
	cachedName     string
	cachedSymbol   string
	cachedDecimals uint8
	mu             sync.RWMutex // Mutex for thread-safe access to cached properties

	// Pre-parsed ABI for common TRC20 methods
	trc20ABI *abi.ABI
}

// NewManager constructs a TRC20 manager bound to the given token contract
// address using the provided TRON connection provider.
//
// This function creates a new TRC20Manager instance for interacting with a
// specific TRC20 token contract. It automatically fetches and caches the
// token's metadata (name, symbol, decimals) for efficient subsequent operations.
//
// Example:
//
//	cli, err := client.NewClient("grpc://grpc.trongrid.io:50051")
//	if err != nil {
//	    // handle error
//	}
//	defer cli.Close()
//
//	tokenAddr, err := types.NewAddress("TContractAddressHere")
//	if err != nil {
//	    // handle error
//	}
//
//	trc20Mgr, err := trc20.NewManager(cli, tokenAddr)
//	if err != nil {
//	    // handle error
//	}
func NewManager(tronClient lowlevel.ConnProvider, contractAddress *types.Address) (*TRC20Manager, error) {
	// Create a generic smart contract instance
	contract, err := smartcontract.NewInstance(tronClient, contractAddress, ERC20ABI)
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

// ToWeiWithDecimals converts a user-facing decimal amount into on-chain units
// using the provided decimals.
func ToWeiWithDecimals(amount decimal.Decimal, decimals uint8) (*big.Int, error) {
	return ToWei(amount, decimals)
}

// FromWeiWithDecimals converts raw on-chain units into a user-facing decimal
// using the provided decimals.
func FromWeiWithDecimals(value *big.Int, decimals uint8) (decimal.Decimal, error) {
	return FromWei(value, decimals)
}

// Name returns the token name, fetching and caching it on first call.
//
// This method returns the name of the TRC20 token (e.g., "TetherUSD").
// The result is cached after the first successful call for improved performance.
//
// Example:
//
//	name, err := trc20Mgr.Name(ctx)
//	if err != nil {
//	    // handle error
//	}
//	fmt.Printf("Token name: %s\n", name)
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

	result, err := t.contract.Call(ctx, t.contract.Address, "name")
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

// Symbol returns the token symbol, fetching and caching it on first call.
//
// This method returns the symbol of the TRC20 token (e.g., "USDT").
// The result is cached after the first successful call for improved performance.
//
// Example:
//
//	symbol, err := trc20Mgr.Symbol(ctx)
//	if err != nil {
//	    // handle error
//	}
//	fmt.Printf("Token symbol: %s\n", symbol)
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

	result, err := t.contract.Call(ctx, t.contract.Address, "symbol")
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

// Decimals returns the token's decimals, fetching and caching it on first call.
//
// This method returns the number of decimal places the token uses for display purposes.
// For example, USDT typically uses 6 decimals, meaning 1 USDT is represented as 1000000
// in on-chain integer values. The result is cached after the first successful call.
//
// Example:
//
//	decimals, err := trc20Mgr.Decimals(ctx)
//	if err != nil {
//	    // handle error
//	}
//	fmt.Printf("Token decimals: %d\n", decimals)
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

	result, err := t.contract.Call(ctx, t.contract.Address, "decimals")
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

func (t *TRC20Manager) TotalSupply(ctx context.Context) (decimal.Decimal, error) {
	decimals, err := t.Decimals(ctx)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to get decimals for TotalSupply: %w", err)
	}

	result, err := t.contract.Call(ctx, t.contract.Address, "totalSupply")
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to call totalSupply method: %w", err)
	}

	bitIntTotalSupply, ok := result.(*big.Int)
	if !ok {
		return decimal.Zero, fmt.Errorf("unexpected type for totalSupply value: %T", result)
	}
	convertedTotalSupply, err := fromWei(bitIntTotalSupply, decimals)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to convert raw total supply: %w", err)
	}

	return convertedTotalSupply, nil
}

// BalanceOf retrieves the owner's balance as a decimal.Decimal.
//
// This method returns the token balance of the specified address. The balance is
// automatically converted from the on-chain integer representation to a human-readable
// decimal value using the token's decimals.
//
// Example:
//
//	balance, err := trc20Mgr.BalanceOf(ctx, address)
//	if err != nil {
//	    // handle error
//	}
//	fmt.Printf("Token balance: %s\n", balance.String())
func (t *TRC20Manager) BalanceOf(ctx context.Context, ownerAddress *types.Address) (decimal.Decimal, error) {
	decimals, err := t.Decimals(ctx)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to get decimals for BalanceOf: %w", err)
	}

	result, err := t.contract.Call(ctx, ownerAddress, "balanceOf", ownerAddress)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to call balanceOf method: %w", err)
	}

	bitIntBalance, ok := result.(*big.Int)
	if !ok {
		return decimal.Zero, fmt.Errorf("unexpected type for balanceOf value: %T", result)
	}
	convertedBalance, err := fromWei(bitIntBalance, decimals)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to convert raw balance: %w", err)
	}

	return convertedBalance, nil
}

// Transfer transfers tokens from the caller to a recipient using a
// decimal.Decimal amount. Returns txid (hex) and the raw transaction extension.
//
// This method creates a TRC20 token transfer transaction from one address to another.
// The transaction is not signed or broadcast - use client.SignAndBroadcast to complete
// the transfer. The amount should be specified as a decimal value (not in the smallest
// token units).
//
// Example:
//
//	amount := decimal.NewFromFloat(10.5) // 10.5 tokens
//	txExt, err := trc20Mgr.Transfer(ctx, from, to, amount)
//	if err != nil {
//	    // handle error
//	}
//
//	// Sign and broadcast the transaction
//	opts := client.DefaultBroadcastOptions()
//	opts.FeeLimit = 50_000_000 // 50 TRX max fee for TRC20 operations
//	result, err := cli.SignAndBroadcast(ctx, txExt, opts, signer)
func (t *TRC20Manager) Transfer(ctx context.Context, fromAddress *types.Address, toAddress *types.Address, amount decimal.Decimal) (*api.TransactionExtention, error) {
	decimals, err := t.Decimals(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get decimals for Transfer", err)
	}

	rawAmount, err := toWei(amount, decimals)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid amount", err)
	}

	txExt, err := t.contract.Invoke(ctx, fromAddress, 0, "transfer", toAddress, rawAmount)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to call transfer method", err)
	}

	return txExt, nil
}

// Approve authorizes a spender for a given amount using decimal.Decimal.
//
// This method creates an approve transaction that allows a spender address to
// spend a specified amount of tokens on behalf of the owner. The transaction
// is not signed or broadcast - use client.SignAndBroadcast to complete the approval.
//
// Example:
//
//	amount := decimal.NewFromFloat(100.0) // Allow spending 100 tokens
//	txExt, err := trc20Mgr.Approve(ctx, owner, spender, amount)
//	if err != nil {
//	    // handle error
//	}
//
//	// Sign and broadcast the transaction
//	opts := client.DefaultBroadcastOptions()
//	result, err := cli.SignAndBroadcast(ctx, txExt, opts, signer)
func (t *TRC20Manager) Approve(ctx context.Context, ownerAddress *types.Address, spenderAddress *types.Address, amount decimal.Decimal) (*api.TransactionExtention, error) {
	decimals, err := t.Decimals(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get decimals for Approve", err)
	}

	rawAmount, err := toWei(amount, decimals)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid amount", err)
	}

	txExt, err := t.contract.Invoke(ctx, ownerAddress, 0, "approve", spenderAddress, rawAmount)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to call approve method", err)
	}

	return txExt, nil
}

// Allowance retrieves the spender's allowance over the owner's tokens as a
// decimal.Decimal.
//
// This method returns the amount of tokens that the spender is allowed to spend
// on behalf of the owner. The allowance is automatically converted from the
// on-chain integer representation to a human-readable decimal value.
//
// Example:
//
//	allowance, err := trc20Mgr.Allowance(ctx, owner, spender)
//	if err != nil {
//	    // handle error
//	}
//	fmt.Printf("Allowance: %s\n", allowance.String())
func (t *TRC20Manager) Allowance(ctx context.Context, ownerAddress *types.Address, spenderAddress *types.Address) (decimal.Decimal, error) {
	decimals, err := t.Decimals(ctx)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to get decimals for Allowance: %w", err)
	}

	result, err := t.contract.Call(ctx, ownerAddress, "allowance", ownerAddress, spenderAddress)
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
