// Package lowlevel provides 1:1 wrappers around the TRON Wallet gRPC API.
//
// The types in this package are intentionally thin and free of business logic.
// They are used by higher-level packages to perform raw RPCs while keeping the
// top-level client namespace clean.
package lowlevel
