tronlib is a golang SDK for Tron.

Aiming to be a simple and easy-to-use high performance SDK for Tron.
high level api for Tron provided
chainable transaction builder for easy to use.
simple wallet management tools
first class smartcontract supports
first class trc-20 contracts interaction






## Features

- [x] Transaction Builder
- [x] Transaction Signer
- [x] Network Client
- [x] Crypto Utilities (Signing, Verification)
- [x] TRC-20 Smart Contract Interaction
- [x] Account Management
- [x] Address Handling
- [x] Resource Handling (Bandwidth, Energy)
- [x] Transfer Transactions


## Usage

This SDK provides a simple and intuitive way to interact with the Tron blockchain. Here are the main features demonstrated below:
- Client initialization with retry and timeout configurations
- Account and address management
- Reading account information from the blockchain
- Transferring TRX between addresses
- Managing resources (Bandwidth/Energy) through freeze/unfreeze/delegate operations

```go
// Initialize client with options
opts := &client.ClientOptions{
    Endpoints: []string{"grpc.shasta.trongrid.io:50051"},
    Timeout:   10 * time.Second,
    RetryConfig: &client.RetryConfig{
        MaxAttempts:    2,
        InitialBackoff: time.Second,
        MaxBackoff:     10 * time.Second,
        BackoffFactor:  2.0,
    },
}

// Create client
ctx := context.Background()
client, err := client.NewClient(ctx, opts)
if err != nil {
    log.Fatalf("Failed to create client: %v", err)
}
defer client.Close()

// Create accounts and addresses
senderAccount, err := types.NewAccountFromPrivateKey(privateKey)
if err != nil {
    log.Fatalf("Failed to create sender account: %v", err)
}

receiverAddr, err := types.NewAddress("TQJ6R9SPvD5SyqgYqTBq3yc6mFtEgatPDu")
if err != nil {
    log.Fatalf("Failed to create receiver address: %v", err)
}

// Read account information
account, err := client.GetAccount(receiverAddr)
if err != nil {
    log.Fatalf("Failed to get account: %v", err)
}

// Transfer TRX
tx := transaction.NewTransaction(client, senderAccount)
err = tx.TransferTRX(senderAccount.Address(), receiverAddr, 1_000_000) // 1 TRX = 1_000_000 SUN
if err != nil {
    log.Fatalf("Failed to create transfer transaction: %v", err)
}

// Set transaction parameters
tx.SetFeelimit(10_000_000) // 10 TRX
tx.SetExpiration(30)       // 30 seconds

// Sign and broadcast transaction
err = tx.Sign(senderAccount)
if err != nil {
    log.Fatalf("Failed to sign transaction: %v", err)
}

err = tx.Broadcast()
if err != nil {
    log.Fatalf("Failed to broadcast transaction: %v", err)
}

// Get transaction receipt
receipt := tx.GetReceipt()
fmt.Printf("Transaction ID: %s\n", receipt.TxID)

// Resource management
// Freeze TRX for Bandwidth (ResourceCode 0)
err = tx.Freeze(senderAccount.Address(), 1_000_000, 0)

// Freeze TRX for Energy (ResourceCode 1)
err = tx.Freeze(senderAccount.Address(), 1_000_000, 1)

// Delegate Bandwidth to another address
err = tx.Delegate(senderAccount.Address(), receiverAddr, 1_000_000, 0)

// Reclaim delegated resources
err = tx.Reclaim(senderAccount.Address(), receiverAddr, 1_000_000, 0)
```

to generate pb go files, run scripts/proto-gen.sh from any folder