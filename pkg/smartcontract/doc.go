// Package smartcontract exposes high-level helpers for deploying and
// interacting with TRON smart contracts.
//
// The SmartContractManager encapsulates common operations such as:
//   - DeployContract: deploy a contract with optional constructor parameters
//   - EstimateEnergy: simulate a call to estimate resource usage
//   - GetContract / GetContractInfo: fetch metadata and ABI
//   - UpdateSetting / UpdateEnergyLimit / ClearContractABI: manage contract settings
//
// This package focuses on safety and clarity of error messages, returning
// sentinel errors from pkg/types where relevant and wrapping validation
// failures with precise context.
package smartcontract
