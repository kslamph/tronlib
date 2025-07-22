package integration_test

import (
	"context"
	"testing"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/smartcontract"
	"github.com/kslamph/tronlib/pkg/types"
)

const (
	TRC20NodeEndpoint    = "127.0.0.1:50051"
	TRC20ContractAddress = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
	TRC20OwnerAddress    = "TLFCPghmgrD2GGGoL1AvSSocTtWvU6fVMe"
	TRC20SpenderAddress  = "TXF1xDbVGdxFGbovmmmXvBGu8ZiE3Lq4mR"
)

func TestTRC20ReadOnly(t *testing.T) {
	c, err := client.NewClient(client.ClientConfig{
		NodeAddress: TRC20NodeEndpoint,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer c.Close()

	ctx := context.Background()

	contract, err := smartcontract.NewTRC20Contract(TRC20ContractAddress, c)
	if err != nil {
		t.Fatalf("Failed to create TRC20 contract: %v", err)
	}

	callCtx, cancel := context.WithTimeout(ctx, c.GetTimeout())
	defer cancel()

	if symbol, err := contract.Symbol(callCtx); err != nil {
		t.Errorf("Failed to get symbol: %v", err)
	} else if symbol == "" {
		t.Error("Symbol is empty")
	}

	callCtx, cancel = context.WithTimeout(ctx, c.GetTimeout())
	defer cancel()
	if decimals, err := contract.Decimals(callCtx); err != nil {
		t.Errorf("Failed to get decimals: %v", err)
	} else if decimals <= 0 {
		t.Error("Decimals should be positive")
	}

	callCtx, cancel = context.WithTimeout(ctx, c.GetTimeout())
	defer cancel()
	if name, err := contract.Name(callCtx); err != nil {
		t.Errorf("Failed to get name: %v", err)
	} else if name == "" {
		t.Error("Name is empty")
	}

	owner, _ := types.NewAddress(TRC20OwnerAddress)
	spender, _ := types.NewAddress(TRC20SpenderAddress)

	callCtx, cancel = context.WithTimeout(ctx, c.GetTimeout())
	defer cancel()
	if allowance, err := contract.Allowance(callCtx, owner.String(), spender.String()); err != nil {
		t.Errorf("Failed to get allowance: %v", err)
	} else if allowance.IsNegative() {
		t.Error("Allowance should not be negative")
	}

	callCtx, cancel = context.WithTimeout(ctx, c.GetTimeout())
	defer cancel()
	if balance, err := contract.BalanceOf(callCtx, owner.String()); err != nil {
		t.Errorf("Failed to get balance: %v", err)
	} else if balance.IsNegative() {
		t.Error("Balance should not be negative")
	}
}
