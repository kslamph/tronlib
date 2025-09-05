// Copyright (c) 2025 github.com/kslamph
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

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
