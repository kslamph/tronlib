package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
)

const (
	DelegationNodeEndpoint = "127.0.0.1:50051"
	DelegationTestAddress  = "TYUkwxLiWrt16YLWr5tK7KcQeNya3vhyLM"
)

func TestClientDelegationQueries(t *testing.T) {
	c, err := client.NewClient(client.ClientConfig{
		NodeAddress: DelegationNodeEndpoint,
		Timeout:     15 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), c.GetTimeout())
	defer cancel()

	addr := types.MustNewAddress(DelegationTestAddress)

	// GetDelegatedResourceV2
	req := &api.DelegatedResourceMessage{FromAddress: addr.Bytes()}
	if res, err := c.GetDelegatedResourceV2(ctx, req); err != nil {
		t.Errorf("GetDelegatedResourceV2 failed: %v", err)
	} else {
		t.Logf("GetDelegatedResourceV2: %+v", res)
	}

	// GetDelegatedResourceAccountIndexV2
	if idx, err := c.GetDelegatedResourceAccountIndexV2(ctx, addr.Bytes()); err != nil {
		t.Errorf("GetDelegatedResourceAccountIndexV2 failed: %v", err)
	} else {
		t.Logf("GetDelegatedResourceAccountIndexV2: %+v", idx)
	}

	// GetCanDelegatedMaxSize
	maxReq := &api.CanDelegatedMaxSizeRequestMessage{OwnerAddress: addr.Bytes()}
	if max, err := c.GetCanDelegatedMaxSize(ctx, maxReq); err != nil {
		t.Errorf("GetCanDelegatedMaxSize failed: %v", err)
	} else {
		t.Logf("GetCanDelegatedMaxSize: %+v", max)
	}

	// GetCanWithdrawUnfreezeAmount
	withdrawReq := &api.CanWithdrawUnfreezeAmountRequestMessage{OwnerAddress: addr.Bytes()}
	if amt, err := c.GetCanWithdrawUnfreezeAmount(ctx, withdrawReq); err != nil {
		t.Errorf("GetCanWithdrawUnfreezeAmount failed: %v", err)
	} else {
		t.Logf("GetCanWithdrawUnfreezeAmount: %+v", amt)
	}
}
