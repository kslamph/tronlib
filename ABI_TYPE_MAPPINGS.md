# ABI Type Mappings

This document describes the Go types that can be used for encoding and decoding different Solidity types in the tronlib ABI processor.

## Address Types

For Solidity `address` type, the following Go types are accepted for encoding:

1. `string` - TRON address in hex format (e.g., "41598F46D7838183A664307841598F46D7838183A6")
2. `[]byte` - EVM address bytes (20 bytes)
3. `eCommon.Address` - go-ethereum common.Address type
4. `types.Address` - tronlib types.Address
5. `*types.Address` - pointer to tronlib types.Address

For decoding, addresses are returned as `*types.Address`

## Integer Types

For all integer types (uint8, uint16, uint32, uint64, uint128, uint256, int8, int16, int32, int64, int128, int256), the following Go types are accepted for encoding:

1. The corresponding Go integer type (e.g., uint8 for uint8, int64 for int64)
2. `*big.Int` - for all integer types, especially uint256/int256

For decoding, integers are returned as `*big.Int` for uint256/int256 and as the appropriate Go integer type for smaller integers (when they fit).

## Boolean Types

For Solidity `bool` type, the following Go type is accepted for encoding:

1. `bool` - Go boolean type

For decoding, booleans are returned as `bool`.

## String Types

For Solidity `string` type, the following Go type is accepted for encoding:

1. `string` - Go string type

For decoding, strings are returned as `string`.

## Bytes Types

For Solidity `bytes` type, the following Go types are accepted for encoding:

1. `[]byte` - Go byte slice
2. `string` - hex-encoded string (with or without "0x" prefix)

For decoding, bytes are returned as `[]byte`.

## Fixed-Size Bytes Types

For Solidity `bytes1`, `bytes2`, ..., `bytes32` types, the following Go types are accepted for encoding:

1. `[N]byte` - Go fixed-size byte array (e.g., [32]byte for bytes32)
2. `[]byte` - Go byte slice (must be exactly N bytes)
3. `string` - hex-encoded string (with or without "0x" prefix, must be exactly N bytes)

For decoding, fixed-size bytes are returned as `[]byte`.

## Array Types

For dynamic arrays (e.g., `uint256[]`), the following Go types are accepted for encoding:

1. `[]T` - Go slice of the appropriate type (e.g., []*big.Int for uint256[], []string for string[])

For static arrays (e.g., `uint256[3]`), the following Go types are accepted for encoding:

1. `[N]T` - Go fixed-size array of the appropriate type
2. `[]T` - Go slice of the appropriate type (must have exactly N elements)

For decoding, arrays are returned as slices of the appropriate type.

## Special Notes

1. For fixed-size byte arrays (bytes1, bytes2, ..., bytes32), the ABI processor preserves the original fixed-size array type when possible to ensure compatibility with the go-ethereum ABI package.

2. For integer arrays, the ABI processor preserves the original slice type to ensure compatibility with the go-ethereum ABI package.

3. All type conversions are handled automatically by the ABI processor, but using the appropriate Go types will result in better performance and fewer conversions.