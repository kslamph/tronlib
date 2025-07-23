package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/smartcontract"
	"github.com/kslamph/tronlib/pkg/types"
)

const (
	ContractNodeEndpoint = "127.0.0.1:50051"
	ContractAddress      = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
	OwnerAddress         = "TYUkwxLiWrt16YLWr5tK7KcQeNya3vhyLM"
)

func TestClientContractQueries(t *testing.T) {
	c, err := client.NewClient(client.ClientConfig{
		NodeAddress: ContractNodeEndpoint,
		Timeout:     15 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), c.GetTimeout())
	defer cancel()

	contractAddr := types.MustNewAddress(ContractAddress)
	ownerAddr := types.MustNewAddress(OwnerAddress)

	// NewContractFromAddress
	contract, err := c.NewContractFromAddress(ctx, contractAddr)
	if err != nil {
		t.Errorf("NewContractFromAddress failed: %v", err)
	} else {
		t.Logf("NewContractFromAddress: %+v", contract)
	}

	// TriggerConstantSmartContract (example: call name() method)
	contractABI := `[{"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"type":"function"}]`
	sc, err := smartcontract.NewContract(contractABI, ContractAddress)
	if err != nil {
		t.Errorf("Failed to create contract: %v", err)
	} else {
		nameData, _ := sc.EncodeInput("name")
		if result, err := c.TriggerConstantSmartContract(ctx, contract, ownerAddr, nameData); err != nil {
			t.Errorf("TriggerConstantSmartContract failed: %v", err)
		} else {
			t.Logf("TriggerConstantSmartContract result: %+v", result)
		}

		// EstimateEnergy (example: call name() method)
		if energy, err := c.EstimateEnergy(ctx, contract, ownerAddr, nameData); err != nil {
			t.Errorf("EstimateEnergy failed: %v", err)
		} else {
			t.Logf("EstimateEnergy: %d", energy)
		}
	}

	// GetContractInfo
	if info, err := c.GetContractInfo(ctx, contractAddr.Bytes()); err != nil {
		t.Errorf("GetContractInfo failed: %v", err)
	} else {
		t.Logf("GetContractInfo: %+v", info)
	}
}
