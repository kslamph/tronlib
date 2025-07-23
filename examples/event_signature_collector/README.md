# Event Signature Collector

This example demonstrates how to collect and store event signatures from TRON blockchain transactions. It scans blocks for smart contract events and stores their signatures in a SQLite database for later analysis.

## Key Features

1. **Event Signature Collection**: Collects event signatures from smart contract transactions
2. **Backward Block Scanning**: Scans current block once, then scans backwards from current-1200 blocks continuously
3. **Contract Caching**: Implements a thread-safe contract cache to avoid repeated network calls
4. **Database Storage**: Stores event signatures in SQLite database with parameter information
5. **Continuous Processing**: Scans blocks as fast as possible without any delays
6. **Graceful Shutdown**: Handles SIGINT/SIGTERM signals for clean shutdown

## How It Works

### Scanning Strategy
1. **Startup**: Scans the current block once to capture latest events
2. **Backward Scanning**: Scans backwards from current block -1200 continuously without delay
3. **Completion**: Stops when reaching the target block number (current - 1200)

### Event Processing
1. **Block Processing**: Gets transaction info for each block
2. **Transaction Processing**: For each transaction, processes all event logs
3. **Contract Resolution**: Uses cached contracts or fetches new ones
4. **Event Decoding**: Uses `contract.DecodeEventLog()` to decode events
5. **Database Storage**: Saves event signatures with parameter types and names

### Key Components

#### EventSignatureCollector
```go
type EventSignatureCollector struct {
    client *client.Client
    db     *database.Database
    cache  *ContractCache
    mutex  sync.RWMutex
}
```

#### ContractCache
```go
type ContractCache struct {
    contracts map[string]*smartcontract.Contract
    mutex     sync.RWMutex
}
```

## Usage

### Building
```bash
cd examples/event_signature_collector
make
```

### Running the Collector
```bash
# Run with default settings
./collector

# Run with custom node and database
./collector -node 127.0.0.1:50051 -db my_signatures.db

# Run with custom timeout
./collector -timeout 60s

# Run with all custom options
./collector -node 127.0.0.1:50051 -db signatures.db -timeout 30s
```

### Querying Collected Data
```bash
# Query collected signatures
./query

# Query with custom database
./query -db my_signatures.db
```

## Command Line Options

- `-node`: TRON node address (default: 127.0.0.1:50051)
- `-db`: Path to SQLite database (default: event_signatures.db)
- `-timeout`: Client timeout duration (default: 30s)

## Example Output

```
Starting event signature collector...
Node: 127.0.0.1:50051
Database: event_signatures.db
Strategy: Scan current block once, then scan backwards from current-1200 blocks continuously
Press Ctrl+C to stop

Processing block 12345678
Found 15 transactions in block 12345678
Saved event signature: ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef (Transfer) with parameter types ["address","address","uint256"] and names ["from","to","value"] from contract TXJgMdjVX5dKiQaUi9QobwNxtSQaFqccvd

Starting backward scan from block 12345678 to block 12344478
Processing block 12345677
Found 12 transactions in block 12345677
...
```

## Database Schema

The SQLite database stores event signatures with the following information:
- Event signature (hex string)
- Event name
- Parameter types (JSON array)
- Parameter names (JSON array)
- Contract address (base58)
- Timestamp of collection

## Benefits

1. **Comprehensive Collection**: Captures event signatures from a wide range of blocks
2. **Performance**: Contract caching reduces network calls
3. **Efficiency**: Backward scanning ensures no blocks are missed
4. **Speed**: Continuous scanning without delays maximizes processing speed
5. **Scalability**: Can process large numbers of blocks and transactions
6. **Reliability**: Graceful error handling and recovery
7. **Flexibility**: Configurable database storage

## Configuration

You can modify the following parameters:
- `NodeAddress`: TRON node endpoint
- `Timeout`: Request timeout duration
- `InitConnections`: Initial connection pool size
- `MaxConnections`: Maximum connection pool size
- `Database`: SQLite database path

## Dependencies

- `github.com/kslamph/tronlib/pkg/client`
- `github.com/kslamph/tronlib/pkg/types`
- `github.com/kslamph/tronlib/pb/core`
- SQLite3 database 