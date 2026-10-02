// Package rpc provides the v2 transport core: dialing a TRON node over
// grpc:// or grpcs://, a bounded connection pool, and the generic Call
// lifecycle that every 1:1 gRPC wrapper is built on.
//
// # Transport core
//
// Dial validates the endpoint scheme (grpc://host:port for plaintext,
// grpcs://host:port for TLS) and returns a *Client. Dial is lazy:
// it constructs the gRPC connection factory without network I/O,
// so a dead node is discovered on the first call (as chain.connection), not
// at dial time. (The root facade's Dial performs the reachability round trip;
// that is the facade's concern, not this package's.)
//
// # Connection pool
//
// The Client maintains a bounded pool of gRPC connections (default 1..5,
// tunable with WithPool). GetConnection hands out a healthy connection;
// ReturnConnection returns it. Call does both for you. Close is idempotent;
// a closed client answers GetConnection with chain.closed.
//
// # Call lifecycle
//
// Call[T] is the single place wrapper functions meet the wire:
//
//	got, err := rpc.Call(cp, ctx, "rpc.GetNowBlock",
//	    func(cl api.WalletClient, ctx context.Context) (*core.Block, error) {
//	        return cl.GetNowBlock(ctx, &api.EmptyMessage{})
//	    }, validateBlock)
//
// It acquires a connection (chain.connection on failure, chain.closed if the
// client is closed), returns it on the way out, applies the provider's
// timeout when the context carries no deadline (chain.timeout on expiry,
// with the context error preserved so errors.Is(err, context.DeadlineExceeded)
// also holds), invokes the call, and runs the optional validator. The node
// reporting an RPC failure surfaces as rpc.method_failed with the operation
// name in Op. Validation failures pass through: a *tron.Error from the
// validator is returned unchanged; any other error is wrapped the same way.
//
// # The ConnProvider seam
//
// ConnProvider is the narrow interface (GetConnection / ReturnConnection /
// GetTimeout) that every free-function wrapper in this package takes, so the
// ~106 gRPC wrappers stay testable against a fake provider: *Client
// implements it, and tests substitute a provider backed by a bufconn server.
// Network identity (Network()/VerifyNetwork genesis fingerprint) lives in
// the root facade, which owns WithNetwork; this package stays a mechanical
// projection of the node's gRPC surface and never infers a network.
package rpc
