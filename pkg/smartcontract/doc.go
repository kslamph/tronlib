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

// Package smartcontract exposes high-level helpers for deploying and
// interacting with TRON smart contracts.
//
// The SmartContractManager encapsulates common operations such as:
//   - DeployContract: deploy a contract with optional constructor parameters
//   - EstimateEnergy: simulate a call to estimate resource usage
//   - GetContract / GetContractInfo: fetch metadata and ABI
//   - UpdateSetting / UpdateEnergyLimit / ClearContractABI: manage contract settings
//
// # Contract Manager
//
// The SmartContractManager provides a high-level interface for common contract operations:
//
//	cli, _ := client.NewClient("grpc://127.0.0.1:50051")
//	defer cli.Close()
//
//	mgr := smartcontract.NewManager(cli)
//	owner, _ := types.NewAddress("Townerxxxxxxxxxxxxxxxxxxxxxxxxxx")
//
//	abiJSON := `{"entrys":[{"type":"constructor","inputs":[{"name":"_owner","type":"address"}]},{"type":"function","name":"setValue","inputs":[{"name":"v","type":"uint256"}]},{"type":"function","name":"getValue","inputs":[],"outputs":[{"name":"","type":"uint256"}],"constant":true}]}`
//	bytecode := []byte{0x60,0x80,0x60,0x40} // truncated
//
//	_, _ = mgr.DeployContract(context.Background(), owner, "MyContract", abiJSON, bytecode, 0, 100, 30000, owner.Bytes())
//
// # Typed Contract Client
//
// You can also work with a typed Contract client to build transactions and
// decode results using its ABI-aware helpers:
//
//	contractAddr, _ := types.NewAddress("Tcontractxxxxxxxxxxxxxxxxxxxxxxxx")
//	c, err := smartcontract.NewContract(cli, contractAddr, abiJSON)
//	if err != nil { /* handle */ }
//
//	// State-changing call (build tx only)
//	txExt, err := c.TriggerSmartContract(ctx, owner, 0, "setValue", uint64(42))
//	if err != nil { /* handle */ }
//
// # Error Handling
//
// This package focuses on safety and clarity of error messages, returning
// sentinel errors from pkg/types where relevant and wrapping validation
// failures with precise context.
//
// Common error types:
//   - ErrInvalidABI - Invalid ABI format
//   - ErrInvalidBytecode - Invalid contract bytecode
//   - ErrContractNotFound - Contract not found on chain
//
// Always check for errors in production code.
package smartcontract
