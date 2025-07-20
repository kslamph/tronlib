package parser

import (
	"context"
	"encoding/hex"
	"log"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
)

// Helper function to decode hex strings
func hexDecode(s string) []byte {
	data, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return data
}

func getContract(tclient *client.Client, address string) *types.Contract {

	contract, err := tclient.NewContractFromAddress(context.Background(), types.MustNewAddress(address))
	if err != nil {
		return nil
	}
	return contract
}

func TestParseTransactionInfoLog(t *testing.T) {
	tclient, err := client.NewClient(client.ClientConfig{
		NodeAddress: "127.0.0.1:50051",
		Timeout:     30 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	contract := getContract(tclient, "TXJgMdjVX5dKiQaUi9QobwNxtSQaFqccvd")
	if contract == nil {
		t.Fatalf("Failed to get contract: %v", err)
	}
	// contracts := []*types.Contract{contract}

	transactionInfo, err := tclient.GetTransactionInfoById(context.Background(), "60a3be8dcf42cc13a38c35a00b89e115520756ae4106a7896d750fd9d8463f9c")
	if err != nil {
		t.Fatalf("Failed to get transaction info: %v", err)
	}
	log.Printf("%x\n", contract.AddressBytes)
	contractsMap := ContractsSliceToMap([]*types.Contract{contract})

	decodedEvents := ParseTransactionInfoLog(transactionInfo, contractsMap)
	for _, decodedEvent := range decodedEvents {
		t.Log(decodedEvent.EventName)
		for _, param := range decodedEvent.Parameters {
			t.Log(param.Name, param.Value)
		}
	}
}
