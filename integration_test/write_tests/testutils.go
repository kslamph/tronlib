//go:build integration

package write_tests

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/kslamph/tronlib/pkg/client"
)

// loadEnv loads environment variables from the given path. It is best-effort:
// variables already present in the environment take precedence, and a missing
// file is not an error — the per-test guards below skip instead.
func loadEnv(path string) {
	_ = godotenv.Load(path)
}

// newTestNileClient creates a new gRPC client for the Nile testnet. The test
// is skipped when NILE_NODE_URL is not configured, so the integration suite
// is strictly opt-in (see integration_test/TESTING_GUIDE.md).
func newTestNileClient(t *testing.T) (*client.Client, error) {
	t.Helper()
	nileNodeURL := os.Getenv("NILE_NODE_URL")
	if nileNodeURL == "" {
		t.Skip("NILE_NODE_URL not set; skipping live-network test (see integration_test/TESTING_GUIDE.md)")
	}
	return client.NewClient(nileNodeURL)
}

// NewCtx creates a new context with a timeout for tests.
func NewCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 60*time.Second)
}
