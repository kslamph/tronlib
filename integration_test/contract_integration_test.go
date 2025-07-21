package integration_test

import (
	"testing"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
)

const (
	NodeEndpoint        = "127.0.0.1:50051"
	USDTContractAddress = "TXDk8mbtRbXeYuMNS83CfKPaYYT8XWv9Hz"
	USDT_ABI            = `[
		{"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"type":"function"},
		{"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"type":"function"},
		{"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"type":"function"},
		{"constant":true,"inputs":[{"name":"owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"type":"function"},
		{"constant":true,"inputs":[{"name":"owner","type":"address"},{"name":"spender","type":"address"}],"name":"allowance","outputs":[{"name":"","type":"uint256"}],"type":"function"}
	]`
)

func TestUSDTContractReadOnly(t *testing.T) {
	c, err := client.NewClient(client.ClientConfig{
		NodeAddress: NodeEndpoint,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer c.Close()

	contract, err := types.NewContract(USDT_ABI, USDTContractAddress)
	if err != nil {
		t.Fatalf("Failed to create contract: %v", err)
	}

	// Test symbol
	data, err := contract.EncodeInput("symbol")
	if err != nil {
		t.Errorf("Failed to encode symbol input: %v", err)
	}
	_ = data // In a real test, would call the contract and decode result

	// Test event signature cache (simulate concurrent access)
	sig := []byte{0xa9, 0x05, 0x9c, 0xbb} // Example: Transfer(address,address,uint256) 4-byte signature
	for i := 0; i < 20; i++ {
		go func() {
			_, _ = contract.DecodeEventSignature(sig)
		}()
	}
}
