// Copyright (c) 2025 github.com/kslamph
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package smartcontract_test

import (
	"context"
	"encoding/hex"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/smartcontract"
	"github.com/kslamph/tronlib/pkg/types"
)

// ExampleManager demonstrates constructing the manager and deploying a contract.
func ExampleManager() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cli, _ := client.NewClient("grpc://127.0.0.1:50051")
	defer cli.Close()

	mgr := smartcontract.NewManager(cli)
	owner, _ := types.NewAddress("Townerxxxxxxxxxxxxxxxxxxxxxxxxxx")

	abiJSON := `{"entrys":[{"type":"constructor","inputs":[{"name":"_owner","type":"address"}]},{"type":"function","name":"setValue","inputs":[{"name":"v","type":"uint256"}]},{"type":"function","name":"getValue","inputs":[],"outputs":[{"name":"","type":"uint256"}],"constant":true}]}`
	bytecode, _ := hex.DecodeString("60806040deadbeef")

	_, _ = mgr.Deploy(ctx, owner, "MyContract", abiJSON, bytecode, 0, 100, 30000, owner.Bytes())
	// Output:
	//
}

// ExampleInstance_Encode demonstrates encoding of a method call.
func ExampleInstance_Encode() {
	cli, _ := client.NewClient("grpc://127.0.0.1:50051")
	defer cli.Close()

	addr, _ := types.NewAddress("Tcontractxxxxxxxxxxxxxxxxxxxxxxxx")
	abiJSON := `{"entrys":[{"type":"function","name":"setValue","inputs":[{"name":"v","type":"uint256"}]}]}`
	c, _ := smartcontract.NewInstance(cli, addr, abiJSON)

	_, _ = c.Encode("setValue", uint64(42))
}

// ExampleInstance_DecodeResult demonstrates decoding of a constant method return.
func ExampleInstance_DecodeResult() {
	cli, _ := client.NewClient("grpc://127.0.0.1:50051")
	defer cli.Close()

	addr, _ := types.NewAddress("Tcontractxxxxxxxxxxxxxxxxxxxxxxxx")
	abiJSON := `{"entrys":[{"type":"function","name":"getValue","inputs":[],"outputs":[{"name":"","type":"uint256"}],"constant":true}]}`
	c, _ := smartcontract.NewInstance(cli, addr, abiJSON)

	// Fake return bytes for illustration
	fake := make([]byte, 32)
	_, _ = c.DecodeResult("getValue", fake)
}
