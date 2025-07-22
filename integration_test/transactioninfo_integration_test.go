package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/parser"
	"github.com/kslamph/tronlib/pkg/types"
)

func getContract(tclient *client.Client, address string) *types.Contract {
	contract, err := tclient.NewContractFromAddress(context.Background(), types.MustNewAddress(address))
	if err != nil {
		return nil
	}
	return contract
}

func TestParseTransactionInfoLogIntegration(t *testing.T) {
	tclient, err := client.NewClient(client.ClientConfig{
		NodeAddress: "127.0.0.1:50051",
		Timeout:     15 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	contract1 := getContract(tclient, "TXJgMdjVX5dKiQaUi9QobwNxtSQaFqccvd")
	if contract1 == nil {
		t.Fatalf("Failed to get contract1")
	}

	contract2 := getContract(tclient, "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	if contract2 == nil {
		t.Fatalf("Failed to get contract2")
	}

	transactionInfo, err := tclient.GetTransactionInfoById(context.Background(), "60a3be8dcf42cc13a38c35a00b89e115520756ae4106a7896d750fd9d8463f9c")
	if err != nil {
		t.Fatalf("Failed to get transaction info: %v", err)
	}

	contractsMap := parser.ContractsSliceToMap([]*types.Contract{contract1, contract2})
	decodedEvents := parser.ParseTransactionInfoLog(transactionInfo, contractsMap)
	for _, decodedEvent := range decodedEvents {
		t.Logf("Contract: %s, Event: %s", decodedEvent.ContractAddress, decodedEvent.Event.EventName)
		for _, param := range decodedEvent.Event.Parameters {
			t.Logf("  %s ( %s ) = %v ", param.Name, param.Type, param.Value)
		}
		t.Log("--------------------------------")
	}
}
