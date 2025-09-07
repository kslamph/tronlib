package write_tests

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/kslamph/tronlib/pkg/client"
)

// loadEnv loads environment variables from the given path.
func loadEnv(path string) {
	if err := godotenv.Load(path); err != nil {
		log.Fatalf("Error loading .env file from %s: %v", path, err)
	}
}

// newTestNileClient creates a new gRPC client for the Nile testnet.
func newTestNileClient() (*client.Client, error) {
	nileNodeURL := os.Getenv("NILE_NODE_URL")
	if nileNodeURL == "" {
		return nil, fmt.Errorf("NILE_NODE_URL not set")
	}
	return client.NewClient(nileNodeURL)
}

// newCtx creates a new context with a timeout for tests.
func NewCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 60*time.Second)
}
