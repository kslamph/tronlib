// Package utils houses ABI encode/decode logic, type parsing, and common
// helpers shared by higher-level packages. The ABIProcessor is the central
// type for converting between Go values and TRON/EVM ABI representations.
//
// # ABI Processing
//
// The ABIProcessor handles encoding and decoding of ABI data:
//
//	abi, _ := utils.NewABIProcessor(nil).ParseABI(abiJSON)
//	proc := utils.NewABIProcessor(abi)
//	data, _ := proc.EncodeMethod("setValue", []string{"uint256"}, 42)
//
// # Encoding Functions
//
// The package provides direct encoding functions for common types:
//   - EncodeAddress - Encode an address
//   - EncodeUint256 - Encode a uint256 value
//   - EncodeString - Encode a string
//
// # Decoding Functions
//
// The package provides direct decoding functions:
//   - DecodeAddress - Decode an address
//   - DecodeUint256 - Decode a uint256 value
//   - DecodeString - Decode a string
//
// # Error Handling
//
// Common error types:
//   - ErrInvalidABI - Invalid ABI format
//   - ErrInvalidType - Invalid data type
//   - ErrInvalidData - Invalid data format
//
// Always check for errors in production code.
package utils
