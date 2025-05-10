# tronlib - Unofficial TRON SDK for Go

`tronlib` is a Go library designed to facilitate interaction with the TRON blockchain. It provides tools for account management, transaction creation and broadcasting, TRC20 token operations, and smart contract interactions.

## Features

*   **Client Management**: Connect to TRON nodes with configurable options like multiple node support, rate limiting, timeouts, and automatic node health checking.
*   **Account Management**: Create accounts from private keys, query account balances, and resource information (Bandwidth, Energy).
*   **Transaction Building**: A fluent builder pattern for creating various transaction types including:
    *   TRX Transfers
    *   Freezing/Unfreezing TRX for resources (Bandwidth/Energy)
    *   Delegating/Reclaiming resources
*   **TRC20 Token Support**:
    *   Query token details (name, symbol, decimals, total supply).
    *   Check account balances and allowances.
    *   Transfer TRC20 tokens.
*   **Smart Contract Interaction**:
    *   Interact with generic smart contracts using their ABI.
    *   Encode function calls and decode results.
    *   Trigger constant (read-only) contract functions.
*   **Blockchain Queries**: Fetch transaction details and other on-chain data.

## Installation

```bash
go get github.com/kslamph/tronlib
```

## Client Initialization

### Basic Client

Connect to the TRON network using default settings (connects to TronGrid mainnet).

```go
package main

import (
	"log"

	"github.com/kslamph/tronlib/pkg/client"
)

func main() {
	tronClient, err := client.NewClient(client.DefaultClientConfig())
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer tronClient.Close()

	// Use tronClient for operations
	log.Println("Successfully connected to TRON network!")
}
```

### Advanced Client Configuration

Customize client behavior by providing a `ClientConfig`. This example connects to Shasta testnet nodes with specific rate limits and timeouts.

```go
package main

import (
	"log"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
)

func main() {
	config := client.ClientConfig{
		Nodes: []client.NodeConfig{
			{
				Address:   "grpc.shasta.trongrid.io:50051", // Shasta Testnet
				RateLimit: client.RateLimit{Times: 100, Window: time.Minute},
			},
			// Add more nodes for redundancy
		},
		TimeoutMs:          5000, // 5 second timeout for RPC calls
		CooldownPeriod:     30 * time.Second,
		MetricsWindowSize:  5,
		BestNodePercentage: 80,
	}

	tronClient, err := client.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer tronClient.Close()

	log.Println("Successfully connected with custom client configuration!")
}
```
Refer to [`examples/read/read.go`](examples/read/read.go:1) for a detailed example of client configuration.

## Account Management

### Working with `Address` Type

The `types.Address` struct represents a TRON address. You can create and convert addresses in various formats.

**Creating an Address:**

*   From a Base58 string:
    ```go
    import "github.com/kslamph/tronlib/pkg/types"
    // ...
    base58Addr := "TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x"
    addr, err := types.NewAddress(base58Addr)
    if err != nil {
        log.Fatalf("Failed to create address from base58: %v", err)
    }
    ```
*   From a Hex string:
    ```go
    hexAddr := "41424b666d634537704d3863777845684154746b4d46774166314665516377593978" // Example hex starting with 41
    addrFromHex, err := types.NewAddressFromHex(hexAddr)
    if err != nil {
        log.Fatalf("Failed to create address from hex: %v", err)
    }
    ```
*   From bytes:
    ```go
    byteAddr := []byte{0x41, 0x42, 0x4b, 0x66, 0x6d, 0x63, 0x45, 0x37, 0x70, 0x4d, 0x38, 0x63, 0x77, 0x78, 0x45, 0x68, 0x41, 0x54, 0x74, 0x6b, 0x4d, 0x46, 0x77, 0x41, 0x66, 0x31, 0x46, 0x65, 0x51, 0x63, 0x77, 0x59, 0x39, 0x78} // Example byte array
    // Or get bytes from an existing address object: byteAddr := addr.Bytes()
    addrFromBytes, err := types.NewAddressFromBytes(byteAddr)
    if err != nil {
        log.Fatalf("Failed to create address from bytes: %v", err)
    }
    ```

**Converting an Address:**

Once you have an `*types.Address` object (e.g., `addr` from above):

*   Get Base58 string: `addr.String()` (recommended) or `addr.GetBase58Addr()`
    ```go
    log.Printf("Base58: %s", addr.String())
    ```
*   Get Hex string (prefixed with `41`): `addr.Hex()` (recommended) or `addr.GetHex()`
    ```go
    log.Printf("Hex: %s", addr.Hex())
    ```
*   Get byte slice: `addr.Bytes()` (recommended) or `addr.GetBytes()`
    ```go
    log.Printf("Bytes: %x", addr.Bytes())
    ```

### Create Account from Private Key

```go
import (
	"log"
	"github.com/kslamph/tronlib/pkg/types"
)

func main() {
	privateKey := "YOUR_PRIVATE_KEY_HEX_STRING" // Replace with actual private key
	account, err := types.NewAccountFromPrivateKey(privateKey)
	if err != nil {
		log.Fatalf("Failed to create account from private key: %v", err)
	}

	log.Printf("Account Address: %s\n", account.Address().String())
	log.Printf("Public Key: %s\n", account.PublicKeyHex())
}
```

### Get Account Information

Fetch details like TRX balance and frozen resources.

```go
// Assuming tronClient is an initialized *client.Client
// Assuming types is "github.com/kslamph/tronlib/pkg/types"

addrString := "TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x" // Example address
addr, err := types.NewAddress(addrString)
if err != nil {
	log.Fatalf("Failed to create address: %v", err)
}

accountInfo, err := tronClient.GetAccount(addr)
if err != nil {
	log.Fatalf("Failed to get account info: %v", err)
}

log.Printf("Address: %s\n", addr.String())
log.Printf("TRX Balance: %d SUN (%f TRX)\n", accountInfo.GetBalance(), float64(accountInfo.GetBalance())/1_000_000.0)

for _, frozen := range accountInfo.GetFrozenV2() {
	log.Printf("Frozen Type: %v, Amount: %d SUN\n", frozen.GetType(), frozen.GetAmount())
}
```
See [`examples/read/read.go`](examples/read/read.go:70) for more.

### Working with `Account` Type

The `types.Account` struct holds the private key, public key, and the corresponding TRON address.

**Creating an Account:**

*   The primary way is from a hexadecimal private key string (as shown in the "Create Account from Private Key" section).
    ```go
    privateKey := "YOUR_PRIVATE_KEY_HEX_STRING"
    account, err := types.NewAccountFromPrivateKey(privateKey)
    if err != nil {
        log.Fatalf("Failed to create account: %v", err)
    }
    ```
*   From a Mnemonic (HD Wallet) - for advanced use cases:
    ```go
    import "github.com/kslamph/tronlib/pkg/types"
    // ...
    mnemonic := "your twelve word mnemonic phrase goes here replace this text" // Replace with actual mnemonic
    hdPath := "m/44'/195'/0'/0/0" // Standard TRON HD path for the first account
    accountFromHD, err := types.NewAccountFromHDWallet(mnemonic, hdPath)
    if err != nil {
        log.Fatalf("Failed to create account from HD Wallet: %v", err)
    }
    log.Printf("Account from HD Wallet: %s", accountFromHD.Address().String())
    ```

**Accessing Account Details:**

Once you have an `*types.Account` object (e.g., `account` from above):

*   Get the `*types.Address` object: `account.Address()`
    ```go
    tronAddress := account.Address()
    log.Printf("Account's TRON Address (Base58): %s", tronAddress.String())
    ```
*   Get Public Key (Hex, uncompressed format): `account.PublicKeyHex()`
    ```go
    log.Printf("Public Key (Hex): %s", account.PublicKeyHex())
    ```
*   Get Private Key (Hex): `account.PrivateKeyHex()`
    ```go
    // Be very careful when handling or displaying private keys.
    log.Printf("Private Key (Hex): %s", account.PrivateKeyHex())
    ```

**Signing Messages (Off-Chain):**

Sign an arbitrary string message. This is useful for proving ownership of an address without needing an on-chain transaction. The signature format is V2 (Tron specific).

```go
messageToSign := "Hello, tronlib! This is a test message."
signatureHex, err := account.SignMessageV2(messageToSign)
if err != nil {
    log.Fatalf("Failed to sign message: %v", err)
}
log.Printf("Message: \"%s\"", messageToSign)
log.Printf("Signature (Hex): %s", signatureHex)

// Verification of this signature would typically be done by a verifier
// (e.g., a backend service or another TRON tool) using the message,
// the signature, and the expected signer's address.
// tronlib itself does not currently provide a VerifyMessageV2 function.
```
The `Sign()` and `MultiSign()` methods on the `Account` type are primarily used internally by the transaction builder when you call `.Sign(account)` on a transaction.

## Sending Transactions

`tronlib` uses a fluent builder pattern for creating and sending transactions.

### Transfer TRX

```go
import (
	"log"
	"os" // For environment variables

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/transaction"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/joho/godotenv" // For .env file (optional)
)

func main() {
	_ = godotenv.Load() // Load .env file if present
	privateKey := os.Getenv("TRON_PRIVATE_KEY")
	if privateKey == "" {
		log.Fatal("TRON_PRIVATE_KEY environment variable not set")
	}

	tronClient, _ := client.NewClient(client.DefaultClientConfig()) // Simplified client init
	defer tronClient.Close()

	ownerAccount, err := types.NewAccountFromPrivateKey(privateKey)
	if err != nil {
		log.Fatalf("Failed to create owner account: %v", err)
	}

	receiverAddrStr := "TQJ6R9SPvD5SyqgYqTBq3yc6mFtEgatPDu" // Example receiver
	receiverAddr, err := types.NewAddress(receiverAddrStr)
	if err != nil {
		log.Fatalf("Failed to create receiver address: %v", err)
	}

	amountSun := int64(1_000_000) // 1 TRX = 1,000,000 SUN

	tx := transaction.NewTransaction(tronClient).SetOwner(ownerAccount.Address())
	tx.TransferTRX(receiverAddr, amountSun)

	receipt := tx.Sign(ownerAccount).Broadcast().GetReceipt()
	if receipt.Err != nil {
		log.Fatalf("TRX Transfer failed: %v\nMessage: %s", receipt.Err, receipt.Message)
	}

	log.Printf("TRX Transfer successful! Transaction ID: %s\n", receipt.TxID)

	// Optionally wait for confirmation
	confirmation, err := tronClient.WaitForTransactionInfo(receipt.TxID, 10) // Wait up to 10 blocks
	if err != nil {
		log.Printf("Failed to get transaction confirmation: %v", err)
	} else {
		log.Printf("Transaction confirmed. Result: %v\n", confirmation.GetResult())
	}
}
```

### Freeze TRX for Resources (e.g., Bandwidth)

```go
// ... (setup client, ownerAccount as above) ...
tx := transaction.NewTransaction(tronClient).SetOwner(ownerAccount.Address())
amountToFreezeSun := int64(1_000_000) // 1 TRX
resourceCode := types.ResourceCode_BANDWIDTH // Or types.ResourceCode_ENERGY

tx.Freeze(amountToFreezeSun, resourceCode)

receipt := tx.Sign(ownerAccount).Broadcast().GetReceipt()
if receipt.Err != nil {
	log.Fatalf("Freeze failed: %v\nMessage: %s", receipt.Err, receipt.Message)
}
log.Printf("Freeze successful! Transaction ID: %s\n", receipt.TxID)
```

### Unfreeze TRX

```go
// ... (setup client, ownerAccount as above) ...
tx := transaction.NewTransaction(tronClient).SetOwner(ownerAccount.Address())
amountToUnfreezeSun := int64(1_000_000) // 1 TRX
resourceCode := types.ResourceCode_BANDWIDTH // Or types.ResourceCode_ENERGY

tx.Unfreeze(amountToUnfreezeSun, resourceCode)

receipt := tx.Sign(ownerAccount).Broadcast().GetReceipt()
if receipt.Err != nil {
	log.Fatalf("Unfreeze failed: %v\nMessage: %s", receipt.Err, receipt.Message)
}
log.Printf("Unfreeze successful! Transaction ID: %s\n", receipt.TxID)
```

### Delegate Resources (e.g., Energy)

```go
// ... (setup client, ownerAccount, receiverAddr as above) ...
tx := transaction.NewTransaction(tronClient).SetOwner(ownerAccount.Address())
amountToDelegateSun := int64(1_000_000) // 1 TRX
resourceCode := types.ResourceCode_ENERGY

tx.Delegate(receiverAddr, amountToDelegateSun, resourceCode)

receipt := tx.Sign(ownerAccount).Broadcast().GetReceipt()
if receipt.Err != nil {
	log.Fatalf("Delegate failed: %v\nMessage: %s", receipt.Err, receipt.Message)
}
log.Printf("Delegate successful! Transaction ID: %s\n", receipt.TxID)
```

### Reclaim Delegated Resources

```go
// ... (setup client, ownerAccount, receiverAddr as above) ...
tx := transaction.NewTransaction(tronClient).SetOwner(ownerAccount.Address())
amountToReclaimSun := int64(1_000_000) // 1 TRX
resourceCode := types.ResourceCode_ENERGY

// Note: For reclaim, the owner is the one who initially delegated.
// The first address parameter in Reclaim is the delegator (owner),
// the second is the recipient of the delegation.
tx.Reclaim(ownerAccount.Address(), receiverAddr, amountToReclaimSun, resourceCode)

receipt := tx.Sign(ownerAccount).Broadcast().GetReceipt()
if receipt.Err != nil {
	log.Fatalf("Reclaim failed: %v\nMessage: %s", receipt.Err, receipt.Message)
}
log.Printf("Reclaim successful! Transaction ID: %s\n", receipt.TxID)
```
For more transaction examples, see [`examples/write/send.go`](examples/write/send.go:1).

## TRC20 Token Interaction

### Initialize TRC20 Contract Client

```go
import (
	"log"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/smartcontract"
)

// Assuming tronClient is an initialized *client.Client
const usdtContractAddress = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t" // USDT on Mainnet, use appropriate for testnet

func main() {
    tronClient, _ := client.NewClient(client.DefaultClientConfig())
    defer tronClient.Close()

	trc20Contract, err := smartcontract.NewTRC20Contract(usdtContractAddress, tronClient)
	if err != nil {
		log.Fatalf("Failed to create TRC20 contract client: %v", err)
	}
	log.Println("TRC20 Contract client initialized.")
    // Use trc20Contract for operations
}
```

### Query TRC20 Token Information

```go
// Assuming trc20Contract is an initialized *smartcontract.TRC20Contract
// Assuming types is "github.com/kslamph/tronlib/pkg/types"

symbol, err := trc20Contract.Symbol()
if err != nil { log.Fatalf("Failed to get symbol: %v", err) }
log.Printf("Symbol: %s\n", symbol)

name, err := trc20Contract.Name()
if err != nil { log.Fatalf("Failed to get name: %v", err) }
log.Printf("Name: %s\n", name)

decimals, err := trc20Contract.Decimals()
if err != nil { log.Fatalf("Failed to get decimals: %v", err) }
log.Printf("Decimals: %d\n", decimals)

ownerAddrStr := "TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x" // Example owner
balance, err := trc20Contract.BalanceOf(ownerAddrStr)
if err != nil { log.Fatalf("Failed to get balance: %v", err) }
log.Printf("Balance of %s: %s\n", ownerAddrStr, balance.String())

spenderAddrStr := "TQJ6R9SPvD5SyqgYqTBq3yc6mFtEgatPDu" // Example spender
allowance, err := trc20Contract.Allowance(ownerAddrStr, spenderAddrStr)
if err != nil { log.Fatalf("Failed to get allowance: %v", err) }
log.Printf("Allowance for %s by %s: %s\n", spenderAddrStr, ownerAddrStr, allowance.String())
```

### Transfer TRC20 Tokens

```go
import (
	"log"
	"os"
	"github.com/shopspring/decimal" // For precise amounts
	// ... other necessary imports: client, smartcontract, types, transaction ...
)
// Assuming tronClient and trc20Contract are initialized
// Assuming ownerAccount is created from private key

func main() {
    // ... (client, trc20Contract, ownerAccount setup) ...
	transferToAddrStr := "TSGkU4jYbYCosYFtrVSYMWGhatFjgSRfnq" // Example recipient
	amount := decimal.NewFromInt(1234) // Amount in token's smallest unit (considering decimals)

	// The TRC20Contract's Transfer method returns a transaction builder
	receipt := trc20Contract.Transfer(ownerAccount.Address().String(), transferToAddrStr, amount).
		Sign(ownerAccount).      // Sign with the owner's account
		Broadcast().             // Broadcast to the network
		GetReceipt()             // Get the transaction receipt

	if receipt.Err != nil {
		log.Fatalf("TRC20 Transfer failed: %v\nMessage: %s", receipt.Err, receipt.Message)
	}
	log.Printf("TRC20 Transfer successful! Transaction ID: %s\n", receipt.TxID)

    // Optionally wait for confirmation
	confirmation, err := tronClient.WaitForTransactionInfo(receipt.TxID, 9)
	if err != nil {
		log.Printf("Failed to get TRC20 transfer confirmation: %v", err)
	} else {
		log.Printf("TRC20 Transfer confirmed. Result: %v\n", confirmation.GetResult())
	}
}
```
See [`examples/TRC20/trc20.go`](examples/TRC20/trc20.go:1) for a complete TRC20 example.

## Generic Smart Contract Interaction

### Initialize Generic Contract Client

You need the contract's ABI and address.

```go
import (
	"log"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
)

// Assuming tronClient is an initialized *client.Client
const contractAddress = "YOUR_CONTRACT_ADDRESS"
const contractABI = `[{"constant":true,"inputs":[],"name":"myReadOnlyFunction","outputs":[{"name":"","type":"string"}],"type":"function"}]` // Simplified ABI

func main() {
    tronClient, _ := client.NewClient(client.DefaultClientConfig())
    defer tronClient.Close()

	genericContract, err := types.NewContract(contractABI, contractAddress)
	if err != nil {
		log.Fatalf("Failed to create generic contract client: %v", err)
	}
	log.Println("Generic contract client initialized.")
    // Use genericContract for operations
}
```

### Call Constant (Read-Only) Contract Functions

This involves encoding the function call and then decoding the result. `tronlib` uses the `TriggerConstantContract` gRPC endpoint for this, which is handled internally by methods like `trc20Contract.Symbol()` or by manually using `EncodeInput` and then calling an appropriate client method if one exists for generic constant calls (the example uses `DecodeResult` on data that seems to be prepared for a constant call but doesn't show the actual call to the node for generic contracts).

The `examples/contract/contract.go` shows encoding inputs and decoding results, implying these would be used with a client call like `TriggerConstantContract`.

```go
// Assuming tronClient and genericContract are initialized

// 1. Encode the function call
// For a function "myReadOnlyFunction()" with no arguments:
encodedCallData, err := genericContract.EncodeInput("myReadOnlyFunction")
if err != nil {
	log.Fatalf("Failed to encode input for myReadOnlyFunction: %v", err)
}

// To call a constant function on the node (example structure, actual client method might vary):
// This part is not explicitly shown for generic contracts in examples/contract/contract.go,
// but TRC20 examples (e.g., contract.Symbol()) handle this internally.
// For a generic contract, you might need to use a method on the `tronClient` like `TriggerConstantContract`
// passing `contractAddress`, `encodedCallData`, and `ownerAddress` (usually "T9yD14Nj9j7xAB4dbGeiX9h8unkKHxuWwb" for constant calls).

// Let's assume you have received `resultBytes` from such a call.
// For demonstration, we'll use the encodedCallData as if it were the result for decoding structure.
// In a real scenario, resultBytes would come from the node.
var resultBytes [][]byte // This would be populated by the actual node call result

// 2. Decode the result
// Assuming resultBytes is populated from the node response
decodedResult, err := genericContract.DecodeResult("myReadOnlyFunction", resultBytes)
if err != nil {
	// This will likely fail if resultBytes is empty or not correctly formatted
	log.Printf("Failed to decode result (or resultBytes not populated from node): %v", err)
} else {
	if len(decodedResult) > 0 {
		log.Printf("Result of myReadOnlyFunction: %v\n", decodedResult[0])
	} else {
		log.Println("No result decoded for myReadOnlyFunction")
	}
}
```

**Note on State-Changing Generic Contract Calls:**
To send a transaction that modifies a generic smart contract's state, you would typically:
1.  Encode the function call and its parameters using `genericContract.EncodeInput("myMutableFunction", arg1, arg2)`.
2.  Use the `transaction.NewTransaction` builder.
3.  Call a method like `tx.TriggerSmartContract(contractAddress, callDataBytes, callValue)` (method name might vary).
4.  Then `Sign()`, `Broadcast()`, and `GetReceipt()` as with other transactions.
The TRC20 transfer example (`trc20Contract.Transfer(...)`) is a good illustration of this pattern for a specific contract type.

Refer to [`examples/contract/contract.go`](examples/contract/contract.go:1) for details on ABI usage, encoding, and decoding.

## Querying Blockchain Data

### Get Transaction Information by ID

```go
// Assuming tronClient is an initialized *client.Client
txID := "YOUR_TRANSACTION_ID"

txInfo, err := tronClient.GetTransactionInfoById(txID)
if err != nil {
	log.Fatalf("Failed to get transaction info for %s: %v", txID, err)
}

log.Printf("Transaction Info for %s:\n", txID)
log.Printf("  Block Number: %d\n", txInfo.GetBlockNumber())
log.Printf("  Result: %v\n", txInfo.GetResult())
if txInfo.GetResMessage() != nil {
    log.Printf("  Message: %s\n", string(txInfo.GetResMessage()))
}
```
See [`examples/read/read.go`](examples/read/read.go:106) for this example.

## Examples

Detailed examples can be found in the [`examples/`](examples/) directory of this repository. They cover:
*   TRC20 token interactions: [`examples/TRC20/trc20.go`](examples/TRC20/trc20.go:1)
*   Generic smart contract calls (read-only focus): [`examples/contract/contract.go`](examples/contract/contract.go:1)
*   Sending various transaction types: [`examples/write/send.go`](examples/write/send.go:1)
*   Querying account and transaction data: [`examples/read/read.go`](examples/read/read.go:1)

## Contributing

Contributions are welcome! Please feel free to submit pull requests or open issues.

## License

This project is licensed under the [Your License Here - e.g., MIT License or Apache 2.0].