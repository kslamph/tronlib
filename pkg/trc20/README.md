# TRC20 Package

The TRC20 package provides comprehensive functionality for interacting with TRC20 smart contracts on the TRON blockchain.

## Features

### Contract Management
- **NewContract**: Create a TRC20 contract instance for interaction
- **Name**: Get the token name
- **Symbol**: Get the token symbol  
- **Decimals**: Get the number of decimal places
- **TotalSupply**: Get the total token supply

### Token Operations
- **Transfer**: Transfer tokens between addresses
- **TransferFrom**: Transfer tokens on behalf of another address (requires approval)
- **Approve**: Approve another address to spend tokens on your behalf

### Query Operations
- **BalanceOf**: Get token balance for an address
- **Allowance**: Get the amount one address is allowed to spend on behalf of another

## Usage Example

```go
package main

import (
    "context"
    "log"
    
    "github.com/kslamph/tronlib/pkg/client"
    "github.com/kslamph/tronlib/pkg/trc20"
    "github.com/shopspring/decimal"
)

func main() {
    // Initialize client
    client := client.NewClient("grpc://127.0.0.1:50051")
    
    // Create TRC20 manager
    manager := trc20.NewManager(client)
    
    // Create contract instance for USDT (example address)
    contract, err := manager.NewContract("TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb")
    if err != nil {
        log.Fatal(err)
    }
    
    ctx := context.Background()
    
    // Get token information
    name, err := contract.Name(ctx)
    if err != nil {
        log.Fatal(err)
    }
    
    symbol, err := contract.Symbol(ctx)
    if err != nil {
        log.Fatal(err)
    }
    
    decimals, err := contract.Decimals(ctx)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Token: %s (%s), Decimals: %d", name, symbol, decimals)
    
    // Check balance
    balance, err := contract.BalanceOf(ctx, "TXNYeYdao7JL7wBtmzbk7mAie7UZsdgVjx")
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Balance: %s %s", balance.String(), symbol)
    
    // Transfer tokens
    amount := decimal.NewFromFloat(10.5)
    tx, err := contract.Transfer(
        ctx,
        "TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb", // from
        "TXNYeYdao7JL7wBtmzbk7mAie7UZsdgVjx", // to
        amount,
    )
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Transfer transaction created: %x", tx.Txid)
    
    // Approve spending
    approveAmount := decimal.NewFromFloat(100)
    approveTx, err := contract.Approve(
        ctx,
        "TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb", // owner
        "TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x", // spender
        approveAmount,
    )
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Approve transaction created: %x", approveTx.Txid)
    
    // Check allowance
    allowance, err := contract.Allowance(
        ctx,
        "TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb", // owner
        "TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x", // spender
    )
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Allowance: %s %s", allowance.String(), symbol)
}
```

## Validation

All functions include comprehensive input validation:
- Address format validation using the utils package
- Amount validation (positive values for transfers, non-negative for approvals)
- Required field validation
- Address comparison validation (preventing self-transfers/approvals)

## Error Handling

The package provides detailed error messages for:
- Invalid addresses (including checksum validation)
- Invalid amounts
- Network communication errors
- Smart contract execution failures
- ABI encoding/decoding errors

## Design Patterns

The TRC20 package follows the established patterns from other packages:
- One gRPC call per function
- Proper input validation using the utils package
- Clean separation between transaction creation and signing
- Integration with the client and smartcontract packages
- Consistent error handling

## Decimal Precision

The package uses the `shopspring/decimal` library for precise decimal arithmetic:
- Supports arbitrary precision decimal numbers
- Automatic conversion between human-readable amounts and contract uint256 values
- Proper handling of token decimals for accurate calculations

## Method Signatures

The package includes hardcoded method signatures for standard TRC20 functions:
- `name()`: 0x06fdde03
- `symbol()`: 0x95d89b41  
- `decimals()`: 0x313ce567
- `totalSupply()`: 0x18160ddd
- `balanceOf(address)`: 0x70a08231
- `allowance(address,address)`: 0xdd62ed3e
- `transfer(address,uint256)`: 0xa9059cbb
- `transferFrom(address,address,uint256)`: 0x23b872dd
- `approve(address,uint256)`: 0x095ea7b3

## Address Encoding

The package properly encodes TRON addresses for smart contract calls:
- Validates address format and checksum
- Pads addresses to 32 bytes for ABI compliance
- Handles the conversion from base58 TRON addresses to contract-compatible format