# Event Logs Direct Example

This example demonstrates how to display all logs for all transactions for all contracts using the `types.DecodeEventLog` method directly, with contract caching.

## Key Features

1. **Direct Event Log Decoding**: Uses `types.DecodeEventLog` method directly without relying on `transactioninfo.go`
2. **Contract Caching**: Implements a thread-safe contract cache to avoid repeated network calls
3. **Batch Processing**: Processes multiple blocks and transactions efficiently
4. **Error Handling**: Gracefully handles errors and continues processing

## How It Works

### Contract Cache
The `ContractCache` struct provides:
- Thread-safe storage of contracts using `sync.RWMutex`
- `GetOrFetch` method that checks cache first, then fetches from network if needed
- Automatic caching of contracts for reuse

### Event Log Processing
1. **Block Processing**: Iterates through a range of blocks
2. **Transaction Processing**: For each block, gets all transaction info
3. **Log Processing**: For each transaction, processes all event logs
4. **Contract Resolution**: Uses cached contracts or fetches new ones
5. **Event Decoding**: Uses `contract.DecodeEventLog()` to decode events
6. **Result Display**: Shows decoded event information

### Key Components

#### ContractCache
```go
type ContractCache struct {
    contracts map[string]*smartcontract.Contract
    mutex     sync.RWMutex
}
```

#### DecodedLog
```go
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
```

## Usage

### Multi-Block Processing
```bash
go run main.go single_transaction_example.go
```

### Single Transaction Processing
```bash
# Run with default example transaction
go run main.go single_transaction_example.go --single

# Run with specific transaction ID
go run main.go single_transaction_example.go --single 60a3be8dcf42cc13a38c35a00b89e115520756ae4106a7896d750fd9d8463f9c
```

**Note**: Transaction ID must be exactly 64 characters (32 bytes) in hexadecimal format.

### Steps
1. **Configure Client**: Set up TRON client with appropriate node address
2. **Create Cache**: Initialize the contract cache
3. **Process Blocks**: Specify the block range to process
4. **View Results**: See decoded events with parameters

## Example Output

```
Current block number: 12345678
Processing blocks 12345668 to 12345678...
Processing block 12345668...
Processing block 12345669...

=== Decoded Event Logs (15 total) ===

Log #1:
  Block: 12345668 (timestamp: 1703123456789)
  Transaction: 60a3be8dcf42cc13a38c35a00b89e115520756ae4106a7896d750fd9d8463f9c
  Contract: TXJgMdjVX5dKiQaUi9QobwNxtSQaFqccvd
  Event: Transfer
  Parameters:
    from (address): TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t (indexed)
    to (address): TXJgMdjVX5dKiQaUi9QobwNxtSQaFqccvd (indexed)
    value (uint256): 1000000000000000000
  Raw Topics: 2 topics
    Topic[0]: 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef
    Topic[1]: 0x000000000000000000000000a614f803b6fd780986a42c78ec9c7f77e6ded13c
  Raw Data: 0x0000000000000000000000000000000000000000000000000de0b6b3a7640000

=== Cache Statistics ===
Total contracts cached: 3
Cached contract addresses:
  TXJgMdjVX5dKiQaUi9QobwNxtSQaFqccvd
  TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t
  TQn9Y2khDD95J42FQtQTdwVVRZJmJk8ZkQ
```

## Benefits

1. **Performance**: Contract caching reduces network calls
2. **Efficiency**: Direct event decoding without intermediate parsing
3. **Scalability**: Can process large numbers of blocks and transactions
4. **Reliability**: Graceful error handling and recovery
5. **Flexibility**: Easy to modify for different use cases

## Configuration

You can modify the following parameters:
- `NodeAddress`: TRON node endpoint
- `Timeout`: Request timeout duration
- `InitConnections`: Initial connection pool size
- `MaxConnections`: Maximum connection pool size
- Block range: Number of blocks to process

## Dependencies

- `github.com/kslamph/tronlib/pkg/client`
- `github.com/kslamph/tronlib/pkg/types`
- `github.com/kslamph/tronlib/pb/core` 