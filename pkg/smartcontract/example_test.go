package smartcontract_test

import (
	"context"
	"encoding/hex"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/smartcontract"
	"github.com/kslamph/tronlib/pkg/types"
)

// ExampleSmartContractManager demonstrates constructing the manager and deploying a contract.
func ExampleSmartContractManager() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cli, _ := client.NewClient("grpc://127.0.0.1:50051")
	defer cli.Close()

	mgr := smartcontract.NewManager(cli)
	owner, _ := types.NewAddress("Townerxxxxxxxxxxxxxxxxxxxxxxxxxx")

	abiJSON := `{"entrys":[{"type":"constructor","inputs":[{"name":"_owner","type":"address"}]},{"type":"function","name":"setValue","inputs":[{"name":"v","type":"uint256"}]},{"type":"function","name":"getValue","inputs":[],"outputs":[{"name":"","type":"uint256"}],"constant":true}]}`
	bytecode, _ := hex.DecodeString("60806040deadbeef")

	_, _ = mgr.DeployContract(ctx, owner, "MyContract", abiJSON, bytecode, 0, 100, 30000, owner.Bytes())
	// Output:
	//
}

// ExampleContract_Encode demonstrates encoding of a method call.
func ExampleContract_Encode() {
	cli, _ := client.NewClient("grpc://127.0.0.1:50051")
	defer cli.Close()

	addr, _ := types.NewAddress("Tcontractxxxxxxxxxxxxxxxxxxxxxxxx")
	abiJSON := `{"entrys":[{"type":"function","name":"setValue","inputs":[{"name":"v","type":"uint256"}]}]}`
	c, _ := smartcontract.NewContract(cli, addr, abiJSON)

	_, _ = c.Encode("setValue", uint64(42))
}

// ExampleContract_DecodeResult demonstrates decoding of a constant method return.
func ExampleContract_DecodeResult() {
	cli, _ := client.NewClient("grpc://127.0.0.1:50051")
	defer cli.Close()

	addr, _ := types.NewAddress("Tcontractxxxxxxxxxxxxxxxxxxxxxxxx")
	abiJSON := `{"entrys":[{"type":"function","name":"getValue","inputs":[],"outputs":[{"name":"","type":"uint256"}],"constant":true}]}`
	c, _ := smartcontract.NewContract(cli, addr, abiJSON)

	// Fake return bytes for illustration
	fake := make([]byte, 32)
	_, _ = c.DecodeResult("getValue", fake)
}
