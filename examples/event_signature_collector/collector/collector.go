package collector

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"event_signature_collector/database"

	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
)

// EventSignatureCollector collects event signatures from TRON blockchain
type EventSignatureCollector struct {
	client *client.Client
	db     *database.Database
	cache  *ContractCache
	mutex  sync.RWMutex
}

// NewEventSignatureCollector creates a new event signature collector
func NewEventSignatureCollector(client *client.Client, db *database.Database) *EventSignatureCollector {
	return &EventSignatureCollector{
		client: client,
		db:     db,
		cache:  NewContractCache(),
	}
}

// Start begins the collection loop
func (c *EventSignatureCollector) Start(ctx context.Context) {
	ticker := time.NewTicker(4 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Collection stopped")
			return
		case <-ticker.C:
			if err := c.processLatestBlock(ctx); err != nil {
				log.Printf("Error processing latest block: %v", err)
			}
		}
	}
}

// StartWithInterval begins the collection loop with custom interval
func (c *EventSignatureCollector) StartWithInterval(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Collection stopped")
			return
		case <-ticker.C:
			if err := c.processLatestBlock(ctx); err != nil {
				log.Printf("Error processing latest block: %v", err)
			}
		}
	}
}

// processLatestBlock processes the latest block for event signatures
func (c *EventSignatureCollector) processLatestBlock(ctx context.Context) error {
	// Get current block
	block, err := c.client.GetNowBlock(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current block: %v", err)
	}

	blockNum := block.GetBlockHeader().GetRawData().GetNumber()
	log.Printf("Processing block %d", blockNum)

	// Get transaction info for the block
	txInfoList, err := c.client.GetTransactionInfoByBlockNum(ctx, blockNum)
	if err != nil {
		return fmt.Errorf("failed to get transaction info for block %d: %v", blockNum, err)
	}

	log.Printf("Found %d transactions in block %d", len(txInfoList.TransactionInfo), blockNum)

	// Process each transaction
	for _, txInfo := range txInfoList.TransactionInfo {
		if err := c.processTransaction(ctx, txInfo); err != nil {
			log.Printf("Warning: Failed to process transaction %s: %v", hex.EncodeToString(txInfo.GetId()), err)
			continue
		}
	}

	return nil
}

// processTransaction processes a single transaction for event signatures
func (c *EventSignatureCollector) processTransaction(ctx context.Context, txInfo *core.TransactionInfo) error {
	// Get logs from transaction info
	logs := txInfo.GetLog()
	if len(logs) == 0 {
		return nil // No logs to process
	}

	// Process each log
	for _, logEntry := range logs {
		if err := c.processLog(ctx, logEntry); err != nil {
			log.Printf("Warning: Failed to process log: %v", err)
			continue
		}
	}

	return nil
}

// processLog processes a single log for event signatures
func (c *EventSignatureCollector) processLog(ctx context.Context, logEntry *core.TransactionInfo_Log) error {
	// Get contract address from log
	contractAddrBytes := logEntry.GetAddress()
	if len(contractAddrBytes) == 0 {
		return fmt.Errorf("empty contract address in log")
	}

	// Add TRON address prefix (0x41) to the address bytes
	fullAddrBytes := make([]byte, 21)
	fullAddrBytes[0] = 0x41 // TRON address prefix
	copy(fullAddrBytes[1:], contractAddrBytes)

	// Convert to base58 address format
	addr, err := types.NewAddressFromBytes(fullAddrBytes)
	if err != nil {
		return fmt.Errorf("failed to parse contract address: %v", err)
	}
	contractAddressBase58 := addr.String()

	// Get topics and data from log
	topics := logEntry.GetTopics()
	if len(topics) == 0 {
		return fmt.Errorf("no topics in log")
	}

	// First topic is the event signature
	eventSignature := hex.EncodeToString(topics[0])

	// Get or fetch contract from cache
	contract, err := c.cache.GetOrFetch(ctx, c.client, fullAddrBytes)
	if err != nil {
		// If we can't get the contract, skip recording unknown events
		log.Printf("Skipping unknown event signature: %s from contract %s (contract not found)", eventSignature, contractAddressBase58)
		return nil
	}

	// Try to decode the event
	decodedEvent, err := contract.DecodeEventLog(topics, logEntry.GetData())
	if err != nil {
		// If decoding fails due to actual error, skip recording
		log.Printf("Skipping event signature: %s from contract %s (decoding error: %v)", eventSignature, contractAddressBase58, err)
		return nil
	}

	// Check if this is an unknown event (event name starts with "unknown_event")
	if strings.HasPrefix(decodedEvent.EventName, "unknown_event") {
		// Skip recording unknown events
		log.Printf("Skipping unknown event signature: %s from contract %s", eventSignature, contractAddressBase58)
		return nil
	}

	// Extract parameter types and names
	paramTypes := make([]string, len(decodedEvent.Parameters))
	paramNames := make([]string, len(decodedEvent.Parameters))
	for i, param := range decodedEvent.Parameters {
		paramTypes[i] = param.Type
		paramNames[i] = param.Name
	}

	// Convert to JSON
	parameterTypesJSON, err := json.Marshal(paramTypes)
	if err != nil {
		return fmt.Errorf("failed to marshal parameter types: %v", err)
	}

	parameterNamesJSON, err := json.Marshal(paramNames)
	if err != nil {
		return fmt.Errorf("failed to marshal parameter names: %v", err)
	}

	// Save the event signature
	if err := c.db.SaveEventSignature(eventSignature, decodedEvent.EventName, string(parameterTypesJSON), string(parameterNamesJSON), contractAddressBase58); err != nil {
		return fmt.Errorf("failed to save event signature: %v", err)
	}

	log.Printf("Saved event signature: %s (%s) with parameter types %s and names %s from contract %s",
		eventSignature, decodedEvent.EventName, string(parameterTypesJSON), string(parameterNamesJSON), contractAddressBase58)

	return nil
}

// ContractCache caches contracts to avoid repeated network calls
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

// Get retrieves a contract from cache
func (c *ContractCache) Get(address string) *types.Contract {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.contracts[address]
}

// Set stores a contract in cache
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
