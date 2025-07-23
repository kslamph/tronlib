package integration_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/trc20"
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

	contract, err := trc20.NewTRC20Contract(c, TRC20ContractAddress)
	if err != nil {
		t.Fatalf("Failed to create TRC20 contract: %v", err)
	}

	callCtx, cancel := context.WithTimeout(ctx, c.GetTimeout())
	defer cancel()

	if symbol, err := contract.Symbol(callCtx); err != nil {
		t.Errorf("Failed to get symbol: %v", err)
	} else if symbol != "USDT" {
		t.Errorf("Symbol is:[%s], want USDT", symbol)
	}

	callCtx, cancel = context.WithTimeout(ctx, c.GetTimeout())
	defer cancel()
	if decimals, err := contract.Decimals(callCtx); err != nil {
		t.Errorf("Failed to get decimals: %v", err)
	} else if decimals != 6 {
		t.Errorf("Decimals is %d, want 6", decimals)
	}

	callCtx, cancel = context.WithTimeout(ctx, c.GetTimeout())
	defer cancel()
	if name, err := contract.Name(callCtx); err != nil {
		t.Errorf("Failed to get name: %v", err)
	} else if name != "Tether USD" {
		t.Errorf("Name is[%s], want Tether USD", name)
		t.Errorf("Name bytes: %v", []byte(name))
		runes := []rune(name)
		runeHex := make([]string, len(runes))
		for i, r := range runes {
			runeHex[i] = fmt.Sprintf("U+%04X", r)
		}
		t.Errorf("Name runes: %v", runeHex)
	}

	owner := types.MustNewAddress(TRC20OwnerAddress)
	spender := types.MustNewAddress(TRC20SpenderAddress)

	callCtx, cancel = context.WithTimeout(ctx, c.GetTimeout())
	defer cancel()
	if allowance, err := contract.Allowance(callCtx, owner.String(), spender.String()); err != nil {
		t.Errorf("Failed to get allowance: %v", err)
	} else if !allowance.IsPositive() {
		t.Errorf("Allowance is %s, want positive", allowance.String())
	}

	callCtx, cancel = context.WithTimeout(ctx, c.GetTimeout())
	defer cancel()
	if balance, err := contract.BalanceOf(callCtx, owner.String()); err != nil {
		t.Errorf("Failed to get balance: %v", err)
	} else if !balance.IsPositive() {
		t.Errorf("Balance is %s, want positive", balance.String())
	} else {
		t.Logf("Balance is %s", balance.String())
	}

}
