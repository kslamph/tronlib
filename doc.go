// Package tronlib is a Go SDK for the TRON blockchain.
//
// v2 is a clean-room redesign. See docs/architecture.md for the design and
// the reasoning behind each breaking change.
//
// # The root facade
//
// This package is the facade: ONE import for the happy path.
// The facade is an on-ramp — every data type is a type alias (zero
// conversion tax between facade and subpackage code), every amount and
// address constructor is a one-line re-export, and every Client method is a
// one-line delegation to the subpackage owner. The facade never
// reimplements; anything beyond the happy path is one Raw() call away in
// the subpackages (rpc for the full 1:1 gRPC surface, tx for
// builders/options, contract for ABI-driven calls, key for message
// signing).
//
// The happy path:
//
//	cli, err := tronlib.Dial(ctx, "grpc://grpc.nile.trongrid.io:50051")
//	defer cli.Close()
//
//	signer, err := tronlib.KeyFromHex(os.Getenv("TRON_PRIVATE_KEY"))
//	to, err := tronlib.ParseAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
//
//	transfer, err := cli.Account(signer.Address()).TransferTRX(ctx, to, tronlib.TRX(1))
//	signed, err := transfer.Sign(signer)
//	rec, err := cli.Broadcast(ctx, signed)
//
// One import; v1 needed four.
//
// Network identity is explicit configuration, not a derivation. TRON has no
// chain ID, and the 21-byte address prefix is 0x41 on Mainnet, Shasta and
// Nile alike, so an address byte cannot discriminate a network. Declare it
// with WithNetwork; Client.Network reports the declaration with no I/O, and
// Client.VerifyNetwork compares the endpoint's genesis block id against a
// recorded table (heuristic: a redeployed testnet changes its genesis and a
// private chain matches nothing). Dial does not verify automatically — Dial
// is lazy by design, so verification is an explicit call.
package tronlib
