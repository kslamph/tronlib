package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
)

const (
	AccountResourceNodeEndpoint = "127.0.0.1:50051"
	ContractAccount             = "TXF1xDbVGdxFGbovmmmXvBGu8ZiE3Lq4mR"
	UserAccount                 = "TQrY8tryqsYVCYS3MFbtffiPp2ccyn4STm"
)

func TestGetAccountNetAndResource(t *testing.T) {
	c, err := client.NewClient(client.ClientConfig{
		NodeAddress: AccountResourceNodeEndpoint,
		Timeout:     10 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer c.Close()

	accounts := []struct {
		name       string
		addrStr    string
		isContract bool
	}{
		{"ContractAccount", ContractAccount, true},
		{"UserAccount", UserAccount, false},
	}

	for _, acc := range accounts {
		addr, err := types.NewAddress(acc.addrStr)
		if err != nil {
			t.Errorf("Failed to create address for %s: %v", acc.name, err)
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		net, errNet := c.GetAccountNet(ctx, addr)
		resource, errRes := c.GetAccountResource(ctx, addr)

		t.Logf("%s: GetAccountNet err=%v, result=%#v", acc.name, errNet, net)
		t.Logf("%s: GetAccountResource err=%v, result=%#v", acc.name, errRes, resource)

		// For both contract and user accounts, expect a valid result and no error
		if errNet != nil || net == nil {
			t.Errorf("%s: Expected valid result for GetAccountNet, got err=%v, result=%#v", acc.name, errNet, net)
		}
		if errRes != nil || resource == nil {
			t.Errorf("%s: Expected valid result for GetAccountResource, got err=%v, result=%#v", acc.name, errRes, resource)
		}
	}
}
