# GoDoc Summary

This page outlines the key entry points and examples available in package documentation.

## 📦 Core Packages

### pkg/client

**Package overview**: Connection management and RPC helpers for interacting with TRON full nodes over gRPC.

**Key features**:
- Connection pooling with configurable size
- Transport security (TLS) support
- Broadcasting and simulation helpers
- Context-based timeout management

**Examples**:
- `ExampleClient_SignAndBroadcast` - Complete example of signing and broadcasting a transaction

### pkg/smartcontract

**Package overview**: High-level helpers for deploying and interacting with TRON smart contracts.

**Key features**:
- Contract deployment with constructor parameters
- Energy estimation for contract calls
- Contract metadata and ABI retrieval
- Contract setting management

**Examples**:
- `ExampleSmartContractManager` - Deploying and interacting with a smart contract

### pkg/trc20

**Package overview**: Typed, ergonomic interface for TRC20 tokens with decimal conversion.

**Key features**:
- Caching of immutable properties (name, symbol, decimals)
- Conversion between human decimals and on-chain integer amounts
- Common actions (balance, allowance, approve, transfer)

**Examples**:
- `ExampleNewManager` - Working with a TRC20 token contract

### pkg/eventdecoder

**Package overview**: Lightweight event decoding from topics/data with built-in support for common events.

**Key features**:
- Built-in support for TRC20 events
- ABI registration for custom events
- Log decoding from transaction receipts

**Examples**:
- Package-level `Example` - Decoding a TRC20 Transfer event

## 🛠 Supporting Packages

### pkg/account

**Package overview**: Account management operations.

**Key features**:
- Balance queries
- TRX transfers
- Account creation

### pkg/resources

**Package overview**: Resource management (bandwidth, energy).

**Key features**:
- Freezing/unfreezing resources
- Resource delegation
- Resource retrieval

### pkg/network

**Package overview**: Network-related operations.

**Key features**:
- Node information retrieval
- Network parameter queries

### pkg/voting

**Package overview**: Voting operations.

**Key features**:
- Vote casting
- Vote retrieval
- Vote withdrawal

### pkg/signer

**Package overview**: Transaction signing utilities.

**Key features**:
- Private key signing
- HD wallet support

### pkg/types

**Package overview**: Core types and constants.

**Key features**:
- Address representation
- Transaction structures
- Error definitions

### pkg/utils

**Package overview**: Utility functions.

**Key features**:
- ABI encoding/decoding
- Data conversion
- Validation functions