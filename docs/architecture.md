# Architecture Overview

This document provides a high-level view of tronlib's structure and how packages interact.

## Package Structure

```
tronlib/
├── cmd/                 # Command-line tools
│   ├── event_abi_generator/
│   ├── event_abi_loader/
│   ├── generate_event_builtins/
│   └── setup_nile_testnet/
├── docs/                # Documentation
├── example/             # Example applications
├── integration_test/    # Integration tests
├── pb/                  # Generated protobuf code
├── pkg/                 # Main library packages
│   ├── account/         # Account management
│   ├── client/          # Core gRPC client with connection pooling
│   ├── eventdecoder/    # Event decoding from logs
│   ├── network/         # Network-related operations
│   ├── resources/       # Resource management (bandwidth, energy)
│   ├── signer/          # Transaction signing utilities
│   ├── simulation/      # Transaction simulation
│   ├── smartcontract/   # Smart contract deployment and interaction
│   ├── trc10/           # TRC10 token support
│   ├── trc20/           # TRC20 token support
│   ├── types/           # Core types and constants
│   ├── utils/           # Utility functions
│   └── voting/          # Voting operations
├── protos/              # Protocol buffer definitions
└── scripts/             # Build and development scripts
```

## Core Components

### Client

The `client` package is the foundation of tronlib. It provides:

- Connection management with pooling and timeouts
- gRPC transport for communicating with TRON full nodes
- Broadcasting and simulation helpers

### Types

The `types` package contains core data structures:

- `Address` - TRON address representation
- `Transaction` - Transaction representation
- Error types and constants

### Utilities

The `utils` package provides helper functions for:

- ABI encoding/decoding
- Data conversion and formatting
- Validation functions

### High-Level Managers

The following packages provide high-level interfaces:

- `account` - Account operations (balance, transfers)
- `resources` - Resource management (freezing/unfreezing)
- `network` - Network-related operations
- `voting` - Voting operations
- `smartcontract` - Smart contract deployment and interaction
- `trc10` - TRC10 token operations
- `trc20` - TRC20 token operations with decimal conversion

### Event Decoder

The `eventdecoder` package provides lightweight event decoding from topics/data with built-in support for common events.

## Workflow Diagram

The following diagram shows how different packages work together in the four major workflows:

```
┌─────────────────┐    ┌──────────────────┐    ┌──────────────────┐
│   High-level    │    │   Transaction    │    │   Low-level      │
│   Managers      │    │   Processing     │    │   Components     │
│                 │    │                  │    │                  │
│  account        │    │                  │    │  types.Address   │
│  resources      │───▶│  client.Client   │───▶│  signer          │
│  network        │    │                  │    │  broadcaster     │
│  smartcontract  │    │  ABI Processor   │    │  ABI Encoder/    │
│  trc10          │    │                  │    │  Decoder         │
│  trc20          │    │  Event Decoder   │    │  Event Builtins  │
└─────────────────┘    └──────────────────┘    └──────────────────┘
                              │
                              ▼
                     ┌──────────────────┐
                     │    TRON Node     │
                     │  (gRPC endpoint) │
                     └──────────────────┘

Workflow 1: Transaction Read  ───▶  Workflow 2: Transaction Write
Workflow 3: Smart Contract Read  ───▶  Workflow 4: Smart Contract Write
```

## Key Components

### types.Address 📍
Unified address representation supporting multiple formats:
- Base58Check string (T-prefixed)
- TRON bytes (0x41-prefixed 21 bytes)
- EVM bytes (20 bytes)
- Hex forms

### signer 🔐
Private key and HD wallet management:
- Raw private key signing
- HD wallet derivation
- Mnemonic phrase support

### broadcaster 📡
Transaction broadcasting with receipt waiting:
- Sign and broadcast in one operation
- Configurable receipt waiting
- Polling interval control

### ABI Processor 🧬
Encoding/decoding of ABI data:
- Method encoding with parameter packing
- Event log decoding
- Type conversion between Go and ABI types

### Event Decoder 📊
Log decoding with built-in TRC20 events:
- Built-in support for common events
- Custom ABI registration
- Typed value extraction

## Four Major Workflows

### 1. Transaction Read
Reading transaction data from the blockchain:
```
client.GetTransactionInfoByID() ───▶ types.TransactionInfo
                              ───▶ eventdecoder.DecodeLog()
```

### 2. Transaction Write
Creating and broadcasting transactions:
```
account.TransferTRX() ───▶ client.SignAndBroadcast() ───▶ TRON Node
resources.FreezeBalanceV2() ───▶ client.SignAndBroadcast() ───▶ TRON Node
```

### 3. Smart Contract Read
Reading data from smart contracts:
```
smartcontract.Contract.TriggerSmartContract() (constant=true) ───▶ client.TriggerContract()
```

### 4. Smart Contract Write
Deploying and interacting with smart contracts:
```
smartcontract.Manager.DeployContract() ───▶ client.SignAndBroadcast() ───▶ TRON Node
smartcontract.Contract.TriggerSmartContract() (state-changing) ───▶ client.SignAndBroadcast() ───▶ TRON Node
```

## Data Flow

1. Build transactions via managers (`account`, `smartcontract`, `trc20`)
2. Sign and broadcast with `client`; optionally wait for receipt
3. Inspect receipts and logs; decode with `eventdecoder` if needed

## Concurrency & Timeouts

- Always pass context with deadlines to calls
- `client.WithTimeout` provides default when no deadline is set
- Each operation should have its own context with appropriate timeout

## Error Handling

The library follows Go's idiomatic error handling patterns:

- Functions return errors as the last return value
- Sentinel errors are defined in the `types` package
- Errors are wrapped with context where appropriate