// Package tronlib is a Go SDK for the TRON blockchain.
//
// v2 is a clean-room redesign. See docs/superpowers/specs/2026-08-31-tronlib-v2-design.md
// for the design and the reasoning behind each breaking change.
//
// # The root facade
//
// This package is the facade: ONE import for the happy path (spec §10).
// The facade is an on-ramp — every data type is a type alias (zero
// conversion tax between facade and subpackage code), every amount and
// address constructor is a one-line re-export, and every Client method is a
// one-line delegation to the subpackage owner. The facade never
// reimplements; anything beyond the happy path is one Raw() call away in
// the subpackages (rpc for the full 1:1 gRPC surface, tx for
// builders/options, contract for ABI-driven calls, key for message
// signing).
//
// The happy path (spec §10):
//
//	cli, err := tronlib.Dial(ctx, "grpc://grpc.nile.trongrid.io:50051")
//	defer cli.Close()
//
//	signer, err := tronlib.KeyFromHex(os.Getenv("TRON_PRIVATE_KEY"))
//	to, err := tronlib.ParseAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
//
//	transfer, err := cli.TransferTRX(ctx, signer.Address(), to, tronlib.TRX(1))
//	signed, err := transfer.Sign(signer)
//	rec, err := cli.Broadcast(ctx, signed)
//
// One import; v1 needed four.
//
// Deferred from the v2.0 facade surface (Task 9 report, D2): Network and
// VerifyNetwork. TRON has no chain ID and the genesis-fingerprint heuristic
// cannot be validated offline; a wrong guess is worse than an absent
// method. ChainTip alone is the network surface until Phase 2.1.
package tronlib
