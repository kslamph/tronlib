# Event Signature Collector

A comprehensive tool for collecting and analyzing event signatures from the TRON blockchain. This program scans the latest blocks every 4 seconds, extracts event signatures from transaction logs, and stores them in a SQLite database for analysis.

## Features

- **Real-time Collection**: Scans the latest block every 4 seconds
- **Event Signature Extraction**: Extracts event signatures from transaction logs
- **Parameter Analysis**: Records event parameters and their types
- **Contract Tracking**: Tracks which contracts use each event signature (up to 10 contracts)
- **Duplicate Detection**: Handles different parameter types for the same signature
- **SQLite Storage**: Efficient local storage with indexing
- **Export Capabilities**: Export data to JSON format
- **Statistics**: Comprehensive statistics and analytics

## Architecture

The program is organized into several modules:

- **`main.go`**: Main program entry point and orchestration
- **`database.go`**: SQLite database operations and schema
- **`collector.go`**: Event signature collection logic
- **`query.go`**: Data querying and export utilities

## Database Schema

### Event Signatures Table
```sql
CREATE TABLE event_signatures (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    signature TEXT UNIQUE NOT NULL,           -- Event signature hash (32 bytes hex)
    event_name TEXT NOT NULL,                 -- Decoded event name
    parameters TEXT NOT NULL,                 -- JSON array of parameter types
    first_seen DATETIME NOT NULL,             -- First time signature was seen
    last_seen DATETIME NOT NULL,              -- Last time signature was seen
    usage_count INTEGER DEFAULT 1,            -- Total usage count
    contract_list TEXT NOT NULL               -- JSON array of contract addresses
);
```

### Contract Usage Table
```sql
CREATE TABLE contract_usage (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    signature_id INTEGER NOT NULL,            -- Foreign key to event_signatures
    contract_addr TEXT NOT NULL,              -- Contract address
    first_seen DATETIME NOT NULL,             -- First time contract used signature
    last_seen DATETIME NOT NULL,              -- Last time contract used signature
    usage_count INTEGER DEFAULT 1,            -- Usage count for this contract
    FOREIGN KEY (signature_id) REFERENCES event_signatures (id),
    UNIQUE(signature_id, contract_addr)
);
```

## Installation

1. **Install Dependencies**:
   ```bash
   go mod tidy
   ```

2. **Install SQLite Driver** (if not already installed):
   ```bash
   go get github.com/mattn/go-sqlite3
   ```

## Usage

### Running the Collector

```bash
go run .
```

The program will:
- Connect to the TRON network
- Create/initialize the SQLite database
- Start scanning blocks every 4 seconds
- Display real-time collection progress
- Handle graceful shutdown with Ctrl+C

### Configuration

Edit `main.go` to modify:
- **Node Address**: Change `NodeAddress` in the client configuration
- **Scan Interval**: Modify the ticker duration in `collector.go`
- **Database Path**: Change the database filename in `main.go`

## Data Analysis

### Querying Event Signatures

```go
// Get all signatures
signatures, err := db.QueryEventSignatures(QueryOptions{})

// Get recent signatures with limit
signatures, err := db.QueryEventSignatures(QueryOptions{
    Limit: 10,
    SortBy: "last_seen",
    SortOrder: "desc",
})

// Search by event name
signatures, err := db.QueryEventSignatures(QueryOptions{
    EventName: "Transfer",
})

// Search by contract
signatures, err := db.QueryEventSignatures(QueryOptions{
    Contract: "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
})
```

### Getting Statistics

```go
stats, err := db.GetStatistics()
PrintStatistics(stats)
```

### Exporting Data

```go
// Export all signatures to JSON
err := db.ExportToJSON("event_signatures.json", QueryOptions{})

// Export recent signatures
err := db.ExportToJSON("recent_signatures.json", QueryOptions{
    Limit: 100,
    SortBy: "last_seen",
    SortOrder: "desc",
})
```

## Event Signature Handling

### Known Events
When a contract has an ABI and the event can be decoded:
- Event name is extracted from the ABI
- Parameter types are recorded
- Full event information is stored

### Unknown Events
When a contract has no ABI or decoding fails:
- Event name is recorded as `unknown_event(0x<signature>)`
- Parameters are recorded as an empty array
- Signature is still tracked for future analysis

### Parameter Type Variations
If the same event signature is used with different parameter types:
- A new entry is created with a modified signature
- Original signature gets a timestamp suffix
- Both variants are tracked separately

## Example Output

```
Processing block 74123097
Found 400 transactions in block 74123097
Saved event signature: ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef (Transfer) with parameters ["address","address","uint256"] from contract TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t
Saved unknown event signature: c42079f94a6350d7e6235f29174924f928cc2ac818eb64fed8004e115fbcca67 from contract TSUUVjysXV8YqHytSNjfkNXnnB49QDvZpx
```

## Database Queries

### Most Used Event Signatures
```sql
SELECT signature, event_name, usage_count 
FROM event_signatures 
ORDER BY usage_count DESC 
LIMIT 10;
```

### Recent Unknown Events
```sql
SELECT signature, contract_list, first_seen 
FROM event_signatures 
WHERE event_name LIKE 'unknown_event%' 
ORDER BY first_seen DESC 
LIMIT 20;
```

### Contract Usage Analysis
```sql
SELECT es.event_name, cu.contract_addr, cu.usage_count
FROM event_signatures es
JOIN contract_usage cu ON es.id = cu.signature_id
ORDER BY cu.usage_count DESC;
```

## Performance Considerations

- **Caching**: Contracts are cached to avoid repeated network calls
- **Indexing**: Database indexes on signature, event_name, and contract addresses
- **Batch Processing**: Transactions are processed in batches
- **Memory Management**: Graceful handling of large datasets

## Error Handling

The program includes comprehensive error handling:
- Network connection failures
- Database errors
- Contract ABI parsing errors
- Event decoding failures
- Graceful shutdown

## Future Enhancements

- Web interface for data visualization
- Real-time alerts for new event signatures
- Integration with external ABI sources
- Machine learning for event signature classification
- Support for other blockchain networks

## License

This project is part of the tronlib repository.