package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
)

// ContractCache manages a thread-safe cache of contracts
type ContractCache struct {
	contracts map[string]*types.Contract
	mutex     sync.RWMutex
}

// NewContractCache creates a new contract cache
func NewContractCache() *ContractCache {
	return &ContractCache{
		contracts: make(map[string]*types.Contract),
	}
}

// Get retrieves a contract from cache, returns nil if not found
func (c *ContractCache) Get(address string) *types.Contract {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.contracts[address]
}

// Set stores a contract in the cache
func (c *ContractCache) Set(address string, contract *types.Contract) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.contracts[address] = contract
}

// GetOrFetch retrieves a contract from cache, or fetches it if not cached
func (c *ContractCache) GetOrFetch(ctx context.Context, client *client.Client, addressBytes []byte) (*types.Contract, error) {
	// Convert address bytes to base58 string for caching
	addr, err := types.NewAddressFromBytes(addressBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse address bytes: %v", err)
	}
	addressString := addr.String()

	// Try to get from cache first
	if contract := c.Get(addressString); contract != nil {
		return contract, nil
	}

	// Fetch contract from network
	contract, err := client.NewContractFromAddress(ctx, addr)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch contract %s: %v", addressString, err)
	}

	// Cache the contract
	c.Set(addressString, contract)
	return contract, nil
}

// DecodedLog represents a decoded event log
type DecodedLog struct {
	BlockNumber     uint64
	BlockTimestamp  uint64
	TransactionHash string
	ContractAddress string
	EventName       string
	Parameters      []types.DecodedEventParameter
	RawTopics       [][]byte
	RawData         []byte
}

// ProcessTransactionLogs processes all logs in a transaction and returns decoded events
func ProcessTransactionLogs(
	ctx context.Context,
	client *client.Client,
	cache *ContractCache,
	blockNumber uint64,
	blockTimestamp uint64,
	txHash string,
	logs []*core.TransactionInfo_Log,
) ([]*DecodedLog, error) {
	var decodedLogs []*DecodedLog

	for _, log := range logs {
		// Get contract address from log
		contractAddrBytes := log.GetAddress()
		if len(contractAddrBytes) == 0 {
			continue
		}

		// Add TRON address prefix (0x41) to the address bytes
		fullAddrBytes := make([]byte, 21)
		fullAddrBytes[0] = 0x41 // TRON address prefix
		copy(fullAddrBytes[1:], contractAddrBytes)

		// Convert to base58 address format for display
		addr, err := types.NewAddressFromBytes(fullAddrBytes)
		if err != nil {
			continue
		}
		contractAddressBase58 := addr.String()

		// Get or fetch contract from cache using full address bytes
		contract, err := cache.GetOrFetch(ctx, client, fullAddrBytes)
		if err != nil {
			// Log error but continue processing other logs
			fmt.Printf("Warning: Failed to get contract %s: %v\n", contractAddressBase58, err)
			continue
		}

		// Get topics and data from log
		topics := log.GetTopics()
		data := log.GetData()

		// Decode the event log using the contract's DecodeEventLog method
		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("Warning: Panic while decoding event for contract %s: %v\n", contractAddressBase58, r)
				}
			}()

			decodedEvent, err := contract.DecodeEventLog(topics, data)
			if err != nil {
				fmt.Printf("Warning: Failed to decode event for contract %s: %v\n", contractAddressBase58, err)
				return
			}

			// Create decoded log entry
			decodedLog := &DecodedLog{
				BlockNumber:     blockNumber,
				BlockTimestamp:  blockTimestamp,
				TransactionHash: txHash,
				ContractAddress: contractAddressBase58,
				EventName:       decodedEvent.EventName,
				Parameters:      decodedEvent.Parameters,
				RawTopics:       topics,
				RawData:         data,
			}

			decodedLogs = append(decodedLogs, decodedLog)
		}()
	}

	return decodedLogs, nil
}

// ProcessBlockTransactions processes all transactions in a block
func ProcessBlockTransactions(
	ctx context.Context,
	client *client.Client,
	cache *ContractCache,
	blockNumber int64,
) ([]*DecodedLog, error) {
	var allDecodedLogs []*DecodedLog

	// Get transaction info for the block
	txInfoList, err := client.GetTransactionInfoByBlockNum(ctx, blockNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction info for block %d: %v", blockNumber, err)
	}

	fmt.Printf("  Processing %d transactions in block %d\n", len(txInfoList.TransactionInfo), blockNumber)

	// Process each transaction
	for _, txInfo := range txInfoList.TransactionInfo {
		// Get transaction hash
		txHash := hex.EncodeToString(txInfo.GetId())

		// Get logs from transaction info
		logs := txInfo.GetLog()
		if len(logs) == 0 {
			continue
		}

		fmt.Printf("  Found %d logs in transaction %s\n", len(logs), txHash)

		// Process logs for this transaction
		decodedLogs, err := ProcessTransactionLogs(
			ctx,
			client,
			cache,
			uint64(txInfo.GetBlockNumber()),
			uint64(txInfo.GetBlockTimeStamp()),
			txHash,
			logs,
		)
		if err != nil {
			fmt.Printf("Warning: Failed to process logs for transaction %s: %v\n", txHash, err)
			continue
		}

		allDecodedLogs = append(allDecodedLogs, decodedLogs...)
	}

	return allDecodedLogs, nil
}

// runSingleTransaction processes a single transaction by its ID
func runSingleTransaction(txID string) {
	// Validate transaction ID format
	if len(txID) != 64 {
		fmt.Printf("Error: Transaction ID must be exactly 64 characters (32 bytes)\n")
		fmt.Printf("Provided: %s (length: %d)\n", txID, len(txID))
		fmt.Printf("Example format: 60a3be8dcf42cc13a38c35a00b89e115520756ae4106a7896d750fd9d8463f9c\n")
		os.Exit(1)
	}

	// Validate hex format
	if _, err := hex.DecodeString(txID); err != nil {
		fmt.Printf("Error: Invalid hex format for transaction ID: %v\n", err)
		fmt.Printf("Transaction ID must be a valid hexadecimal string\n")
		os.Exit(1)
	}

	// Create client configuration
	config := client.ClientConfig{
		NodeAddress:     "127.0.0.1:50051", // Mainnet
		Timeout:         30 * time.Second,
		InitConnections: 1,
		MaxConnections:  5,
		IdleTimeout:     60 * time.Second,
	}

	// Create client
	client, err := client.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Create contract cache
	cache := NewContractCache()

	// Create context
	ctx := context.Background()

	fmt.Printf("Processing transaction: %s\n", txID)

	// Get transaction info by ID
	txInfo, err := client.GetTransactionInfoById(ctx, txID)
	if err != nil {
		log.Fatalf("Failed to get transaction info: %v", err)
	}

	// Get logs from transaction info
	logs := txInfo.GetLog()
	if len(logs) == 0 {
		fmt.Printf("No logs found in transaction %s\n", txID)
		return
	}

	fmt.Printf("Found %d logs in transaction %s\n", len(logs), txID)

	// Process logs for this transaction
	decodedLogs, err := ProcessTransactionLogs(
		ctx,
		client,
		cache,
		uint64(txInfo.GetBlockNumber()),
		uint64(txInfo.GetBlockTimeStamp()),
		txID,
		logs,
	)
	if err != nil {
		log.Fatalf("Failed to process logs: %v", err)
	}

	// Display decoded logs
	DisplayDecodedLogs(decodedLogs)

	// Display cache statistics
	fmt.Printf("\n=== Cache Statistics ===\n")
	fmt.Printf("Total contracts cached: %d\n", len(cache.contracts))
	fmt.Printf("Cached contract addresses:\n")
	for addr := range cache.contracts {
		fmt.Printf("  %s\n", addr)
	}
}

// DisplayDecodedLogs prints the decoded logs in a formatted way
func DisplayDecodedLogs(logs []*DecodedLog) {
	fmt.Printf("\n=== Decoded Event Logs (%d total) ===\n\n", len(logs))

	for i, log := range logs {
		fmt.Printf("Log #%d:\n", i+1)
		fmt.Printf("  Block: %d (timestamp: %d)\n", log.BlockNumber, log.BlockTimestamp)
		fmt.Printf("  Transaction: %s\n", log.TransactionHash)
		fmt.Printf("  Contract: %s\n", log.ContractAddress)
		fmt.Printf("  Event: %s\n", log.EventName)

		if len(log.Parameters) > 0 {
			fmt.Printf("  Parameters:\n")
			for _, param := range log.Parameters {
				indexed := ""
				if param.Indexed {
					indexed = " (indexed)"
				}
				fmt.Printf("    %s (%s): %s%s\n", param.Name, param.Type, param.Value, indexed)
			}
		}

		fmt.Printf("  Raw Topics: %d topics\n", len(log.RawTopics))
		for j, topic := range log.RawTopics {
			fmt.Printf("    Topic[%d]: 0x%s\n", j, hex.EncodeToString(topic))
		}

		fmt.Printf("  Raw Data: 0x%s\n", hex.EncodeToString(log.RawData))
		fmt.Println()
	}
}

func main() {
	// Check if user wants to run single transaction example
	if len(os.Args) > 1 && os.Args[1] == "--single" {
		if len(os.Args) > 2 {
			// Run with specific transaction ID
			txID := os.Args[2]
			runSingleTransaction(txID)
		} else {
			// Run with default example transaction
			ExampleSingleTransaction()
		}
		return
	}

	// Create client configuration
	config := client.ClientConfig{
		NodeAddress:     "127.0.0.1:50051", // Mainnet
		Timeout:         30 * time.Second,
		InitConnections: 1,
		MaxConnections:  5,
		IdleTimeout:     60 * time.Second,
	}

	// Create client
	client, err := client.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Create contract cache
	cache := NewContractCache()

	// Create context
	ctx := context.Background()

	// Get current block number
	block, err := client.GetNowBlock(ctx)
	if err != nil {
		log.Fatalf("Failed to get current block: %v", err)
	}

	currentBlockNum := block.GetBlockHeader().GetRawData().GetNumber()
	fmt.Printf("Current block number: %d\n", currentBlockNum)

	// Process a range of blocks (e.g., last 10 blocks)
	startBlock := currentBlockNum - 10
	if startBlock < 0 {
		startBlock = 0
	}

	fmt.Printf("Processing blocks %d to %d...\n", startBlock, currentBlockNum)

	var allDecodedLogs []*DecodedLog

	// Process each block
	for blockNum := startBlock; blockNum <= currentBlockNum; blockNum++ {
		fmt.Printf("Processing block %d...\n", blockNum)

		decodedLogs, err := ProcessBlockTransactions(ctx, client, cache, blockNum)
		if err != nil {
			fmt.Printf("Warning: Failed to process block %d: %v\n", blockNum, err)
			continue
		}

		allDecodedLogs = append(allDecodedLogs, decodedLogs...)
	}

	// Display all decoded logs
	DisplayDecodedLogs(allDecodedLogs)

	// Display cache statistics
	fmt.Printf("\n=== Cache Statistics ===\n")
	fmt.Printf("Total contracts cached: %d\n", len(cache.contracts))
	fmt.Printf("Cached contract addresses:\n")
	for addr := range cache.contracts {
		fmt.Printf("  %s\n", addr)
	}
}
