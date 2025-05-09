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
- Client initialization with robust connection management (multiple nodes, health checks, failover, cooldowns) and timeout configuration.
- Account and address management
- Reading account information from the blockchain
- Transferring TRX between addresses
- Managing resources (Bandwidth/Energy) through freeze/unfreeze/delegate operations

```go
// Initialize client
// The client manages connections to multiple Tron nodes, handles health checks,
// failover, rate limiting, and node selection based on performance.

// Option 1: Default Configuration (connects to a list of Tron MainNet nodes)
// This uses predefined MainNet nodes and default settings for timeout, cooldown, etc.
// Ensure "github.com/kslamph/tronlib/pkg/client" and "log" are imported.
tronClient, err := client.NewClient(client.DefaultClientConfig()) // Renamed 'client' to 'tronClient'
if err != nil {
	log.Fatalf("Failed to create client: %v", err)
}
defer tronClient.Close() // Use tronClient

// Option 2: Custom Configuration (Example shown commented out)
// You can customize nodes, timeouts, rate limits, and other parameters.
// Import "time" for duration values if customizing.
/*
// Example for connecting to Shasta TestNet nodes:
shastaConfig := client.ClientConfig{
	Nodes: client.ShastaNodes(), // Helper function for default Shasta node(s)
	// To use specific nodes (e.g., a single Shasta node or your private node):
	// Nodes: []client.NodeConfig{
	// 	{
	// 		Address: "grpc.shasta.trongrid.io:50051",
	// 		// Optional: configure rate limit for this node
	// 		RateLimit: client.RateLimit{Times: 5, Window: 1 * time.Second},
	// 	},
	// 	// {Address: "another.node.example.com:50051", RateLimit: client.DefaultRateLimit()},
	// },
	TimeoutMs:          10000,                  // e.g., 10 seconds for gRPC calls
	CooldownPeriod:     30 * time.Second,       // Time a node stays in cooldown after certain errors
	BestNodePercentage: 80,                     // % of requests to route to the best performing node
	// MetricsWindowSize:  5,                   // Default is 3 (requests for avg response time)
}
tronClientCustom, err := client.NewClient(shastaConfig) // or your customConfig
if err != nil {
	log.Fatalf("Failed to create custom client: %v", err)
}
defer tronClientCustom.Close()

// Use tronClient (from Option 1) or tronClientCustom (from Option 2) for subsequent operations.
*/

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
account, err := tronClient.GetAccount(receiverAddr)
if err != nil {
    log.Fatalf("Failed to get account: %v", err)
}

// Transfer TRX
tx := transaction.NewTransaction(tronClient, senderAccount)
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