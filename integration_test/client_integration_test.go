package integration_test

import (
	"context"
	"sync"
	"testing"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
)

const (
	ClientNodeEndpoint = "127.0.0.1:50051"
	ClientTestAddress  = "TYUkwxLiWrt16YLWr5tK7KcQeNya3vhyLM"
)

func TestClientConnectionPoolConcurrency(t *testing.T) {
	c, err := client.NewClient(client.ClientConfig{
		NodeAddress: ClientNodeEndpoint,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer c.Close()

	addr, err := types.NewAddress(ClientTestAddress)
	if err != nil {
		t.Fatalf("Failed to create address: %v", err)
	}

	var wg sync.WaitGroup
	concurrency := 3 // Reduced for debugging
	errs := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), c.GetTimeout())
			defer cancel()
			_, err := c.GetAccount(ctx, addr)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)

	failCount := 0
	for err := range errs {
		if err != nil {
			failCount++
		}
	}
	if failCount > 0 {
		t.Errorf("%d out of %d concurrent GetAccount calls failed", failCount, concurrency)
	}
}
