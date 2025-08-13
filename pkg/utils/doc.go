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
