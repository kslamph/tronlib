// Package testutil provides shared helpers for offline gRPC-based tests.
//
// The helpers here centralize the bufconn (in-process gRPC) scaffolding that
// was previously copy-pasted across every package's test files, so that tests
// only need to register their fake WalletServer implementation and dial it.
package testutil

import (
	"context"
	"net"
	"testing"

	"github.com/kslamph/tronlib/pb/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

// NewBufconnServer starts an in-process gRPC server on a bufconn listener and
// registers impl as the WalletServer. The server and listener are torn down
// automatically via t.Cleanup.
func NewBufconnServer(t *testing.T, impl api.WalletServer) *bufconn.Listener {
	t.Helper()
	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer()
	api.RegisterWalletServer(srv, impl)
	go func() { _ = srv.Serve(lis) }()

	t.Cleanup(func() {
		srv.Stop()
		_ = lis.Close()
	})
	return lis
}

// DialBufconn opens a *grpc.ClientConn to a bufconn listener. The connection is
// closed automatically via t.Cleanup.
func DialBufconn(t *testing.T, lis *bufconn.Listener) *grpc.ClientConn {
	t.Helper()
	conn, err := grpc.Dial("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithInsecure(),
	)
	if err != nil {
		t.Fatalf("failed to dial bufnet: %v", err)
	}

	t.Cleanup(func() {
		_ = conn.Close()
	})
	return conn
}
