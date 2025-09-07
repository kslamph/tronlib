// Package smartcontract provides high-level helpers to deploy, query, and interact with
// TRON smart contracts. It includes a package-level Manager for deployment/admin tasks
// and a per-address Instance for bound interaction.
//
// # Manager Operations
//
// The Manager encapsulates common operations such as:
//   - Deploy: Deploy a contract with optional constructor parameters
//   - EstimateEnergy: Estimate energy usage for a transaction
//   - GetContract: Retrieve on-chain contract metadata
//   - GetContractInfo: Retrieve detailed contract info
//   - UpdateSetting / UpdateEnergyLimit / ClearContractABI: Administrative tasks
//
// # Quick Start
//
// The Manager provides a high-level interface for common contract operations:
//
//	mgr := client.SmartContract()
//	txExt, err := mgr.Deploy(context.Background(), owner, "MyContract", abiJSON, bytecode, 0, 100, 30000, owner.Bytes())
//	if err != nil { /* handle */ }
//
// For interacting with a deployed contract, create an Instance bound to an address:
//
//	c, err := smartcontract.NewInstance(cli, contractAddr, abiJSON)
//	if err != nil { /* handle */ }
//	txExt, err := c.Invoke(ctx, owner, 0, "setValue", uint64(42))
//	if err != nil { /* handle */ }
//
// # Error Handling
//
// Common error types:
//   - ErrContractNotFound - Contract not found on chain
//   - ErrInvalidABI - Invalid ABI format
//   - ErrInsufficientEnergy - Insufficient energy for contract execution
//
// Always check for errors in production code.
package smartcontract
