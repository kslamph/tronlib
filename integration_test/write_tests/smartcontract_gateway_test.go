package write_tests

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/smartcontract"
	"github.com/kslamph/tronlib/pkg/types"
)

const (
	// Standard Nile Testnet address for the USDT contract
	nileUSDTAddress = "TWRvzd6FQcsyp7hwCtttjZGpU1kfvVEtNK"
	// An arbitrary address
	testAddress = "TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x"
)

func TestContractTriggerConstantContractDecodeName(t *testing.T) {
	// This is now an integration test requiring a local node.
	c, err := client.NewClient("grpc://grpc.nile.trongrid.io:50051", client.WithTimeout(5*time.Second))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	defer c.Close()

	contractAddr := types.MustNewAddressFromBase58(nileUSDTAddress)
	owner := types.MustNewAddressFromBase58(testAddress)

	// We can fetch the ABI from the network by passing no ABI to NewContract
	sc, err := smartcontract.NewInstance(c, contractAddr)
	if err != nil {
		t.Fatalf("new contract: %v", err)
	}

	res, err := sc.Call(context.Background(), owner, "name")
	if err != nil {
		t.Fatalf("TriggerConstantContract err: %v", err)
	}

	expectedName := "TronLib Test"
	if res.(string) != expectedName {
		t.Fatalf("unexpected decoded name: got '%v', want '%s'", res, expectedName)
	}
}

func TestContractTriggerSmartContractEncodesAndSends(t *testing.T) {
	// This is now an integration test requiring a local node.
	c, err := client.NewClient("grpc://grpc.nile.trongrid.io:50051", client.WithTimeout(5*time.Second))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	defer c.Close()

	contractAddr := types.MustNewAddressFromBase58(nileUSDTAddress)
	from := types.MustNewAddressFromBase58(testAddress)
	// A random recipient for encoding purposes
	to := types.MustNewAddressFromBase58("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")

	// We need the ABI to encode the method call
	sc, err := smartcontract.NewInstance(c, contractAddr)
	if err != nil {
		t.Fatalf("new contract: %v", err)
	}

	// This test now only verifies that the call can be constructed and sent to the node
	// without an immediate error. It does not sign or broadcast the transaction.
	txExt, err := sc.Invoke(context.Background(), from, 0, "transfer", to, big.NewInt(10))
	if err != nil {
		t.Fatalf("TriggerSmartContract err: %v", err)
	}

	if txExt == nil || txExt.Transaction == nil {
		t.Fatal("TriggerSmartContract returned a nil transaction extension or transaction")
	}
	if len(txExt.Transaction.RawData.Contract) == 0 {
		t.Fatal("transaction has no contract")
	}

	res, err := c.Simulate(context.Background(), txExt)
	if err != nil {
		t.Fatalf("Simulate err: %v", err)
	}

	t.Logf("Simulate result: %v", res)
}
