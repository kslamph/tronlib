# Transaction Workflow Package

The workflow package provides a transaction workflow management system with action chaining pattern and controlled flow conditions for TRON blockchain transactions.

## Features

- **Action Chain Pattern**: Fluent interface for building transaction workflows
- **Controlled Flow Conditions**: State-based validation prevents invalid operations
- **Multi-Signature Support**: Support for multi-party transaction signing
- **Concurrency Safety**: Thread-safe operations with mutex protection
- **Error Propagation**: Comprehensive error handling throughout the chain

## States

The workflow tracks transactions through these states:

- `StateUnsigned`: Initial state, allows timeout/fee modifications
- `StateSigned`: Transaction has been signed, ready for broadcast
- `StateBroadcasted`: Transaction has been sent to the network
- `StateError`: An error occurred, workflow is stopped

## Basic Usage

### Simple Transfer

```go
import (
    "context"
    "time"
    "github.com/kslamph/tronlib/pkg/workflow"
)

// Create workflow and chain operations
wf := workflow.NewWorkflow(client, transaction).
    SetTimeout(time.Now().Add(10*time.Minute).UnixMilli()).
    SetFeeLimit(1000000). // 1 TRX
    Sign(signer)

if err := wf.GetError(); err != nil {
    log.Fatal(err)
}

// Broadcast with waiting for confirmation
txID, success, txInfo, err := wf.Broadcast(ctx, 30)
```

### Multi-Signature Workflow

```go
// Multi-signature workflow
wf := workflow.NewWorkflow(client, transaction).
    Sign(signer1).                    // First signature
    MultiSign(signer2, 1).            // Second signature with permission ID
    MultiSign(signer3, 2)             // Third signature

// Get signed transaction for external handling
txID, signedTx, err := wf.GetSignedTransaction()
```

### Sign-Only (No Broadcast)

```go
// Sign only for external broadcast or additional signatures
wf := workflow.NewWorkflow(client, transaction).
    SetTimeout(time.Now().Add(2*time.Hour).UnixMilli()).
    SetFeeLimit(500000).
    Sign(signer)

txID := wf.GetTxid()
_, signedTx, err := wf.GetSignedTransaction()

// Transaction can now be:
// 1. Passed to another application for additional signatures
// 2. Broadcasted by another service
// 3. Stored for later processing
```

## API Reference

### Constructor

- `NewWorkflow(client *client.Client, tx interface{}) *TransactionWorkflow`
  - Creates new workflow with transaction (accepts `*core.Transaction` or `*api.TransactionExtention`)

### Unsigned Transaction Methods

- `SetTimeout(timestamp int64) *TransactionWorkflow`
  - Sets expiration timestamp (Unix milliseconds)
  - Only works on unsigned transactions

- `SetFeeLimit(feeLimit int64) *TransactionWorkflow`
  - Sets maximum fee in SUN (1 TRX = 1,000,000 SUN)
  - Only works on unsigned transactions

### Signing Methods

- `Sign(signer *signer.PrivateKeySigner) *TransactionWorkflow`
  - Signs transaction, can be called multiple times
  - Transitions to `StateSigned`

- `MultiSign(signer *signer.PrivateKeySigner, permissionID int32) *TransactionWorkflow`
  - Multi-signature signing with permission ID
  - Can be called multiple times

### Signed Transaction Methods

- `GetTxid() string`
  - Returns transaction ID as hex string
  - Returns empty string if not signed

- `GetSignedTransaction() (string, *api.TransactionExtention, error)`
  - Returns copy of signed transaction
  - Only works on signed transactions

- `Broadcast(ctx context.Context, waitSeconds int64) (string, bool, *core.TransactionInfo, error)`
  - Broadcasts transaction to network
  - `waitSeconds`: 0 = no waiting, >0 = wait for receipt (smart contracts only)
  - Returns: txID, success, transactionInfo, error

### Utility Methods

- `GetError() error`
  - Returns any error that occurred during workflow

- `GetState() WorkflowState`
  - Returns current workflow state

- `EstimateFee() (int64, error)` *(placeholder)*
  - Fee estimation (not implemented yet)

## State Validation

The workflow enforces state-based validation:

- `SetTimeout()` and `SetFeeLimit()` only work on unsigned transactions
- `GetTxid()`, `GetSignedTransaction()`, and `Broadcast()` only work on signed transactions
- Once an error occurs, the workflow enters error state and stops processing
- All operations return the workflow instance for chaining

## Error Handling

```go
wf := workflow.NewWorkflow(client, invalidTx).
    SetTimeout(time.Now().Add(1*time.Hour).UnixMilli()).
    Sign(signer)

// Check for errors
if err := wf.GetError(); err != nil {
    fmt.Printf("Workflow error: %v\n", err)
    fmt.Printf("Current state: %s\n", wf.GetState().String())
}
```

## Broadcasting Behavior

- For regular transactions: successful broadcast = successful transaction
- For smart contract transactions: broadcast success ≠ transaction success
  - Use `waitSeconds > 0` to wait for transaction receipt
  - Check `TransactionInfo` for actual execution result

## Concurrency Safety

All workflow operations are thread-safe using read-write mutexes. Multiple goroutines can safely:
- Read workflow state
- Chain operations (each operation is atomic)
- Access transaction data

However, avoid sharing the same workflow instance across goroutines for chaining operations, as this may lead to unexpected behavior.