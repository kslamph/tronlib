# tronlib

A Go client library for interacting with the TRON blockchain, providing high-level managers for accounts, resources, network, voting, smart contracts, and TRC20 helpers.

## Quickstart

Minimal example showing client setup, context usage, and broadcast options.

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
)

func main() {
	// Always use a context; prefer per-operation timeouts.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a client using DefaultClientConfig and remember to Close.
	cfg := client.DefaultClientConfig()
	cli, err := client.NewClient(cfg)
	if err != nil {
		log.Fatalf("new client: %v", err)
	}
	defer cli.Close()

	// Default broadcast options (copy/paste friendly).
	opts := client.DefaultBroadcastOptions()
	// Override any fields as needed (examples):
	opts.WaitTimeout = 45 * time.Second
	opts.FeeLimit = 50_000_000 // 50 TRX in SUN

	// Example address usage
	addr, err := types.NewAddressFromBase58("TXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX") // replace with real address
	if err != nil {
		log.Fatalf("parse addr: %v", err)
	}

	_ = ctx
	_ = addr
	_ = opts
	_ = cli
}
```

Notes on context:
- Pass context to every call. Use short per-operation timeouts.
- For polling operations, ensure WaitTimeout and PollInterval align with your context deadline.

## Accounts and Resources

Demonstrates explicit manager aliases for discoverability.

```go
package accounts_resources_example

import (
	"context"
	"log"
	"time"

	"github.com/kslamph/tronlib/pkg/account"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/resources"
	"github.com/kslamph/tronlib/pkg/types"
)

func Example() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cli, err := client.NewClient(client.DefaultClientConfig())
	if err != nil {
		log.Fatalf("client: %v", err)
	}
	defer cli.Close()

	// Aliases for discoverability:
	//   type AccountManager = account.Manager
	//   type ResourceManager = resources.Manager
	var am account.AccountManager = account.NewManager(cli)
	var rm resources.ResourceManager = resources.NewManager(cli)

	owner, _ := types.NewAddressFromBase58("Txxxxxxxxxxxxxxxxxxxxxxxxxxxxxx1")
	peer, _ := types.NewAddressFromBase58("Txxxxxxxxxxxxxxxxxxxxxxxxxxxxxx2")

	// Get balance
	bal, err := am.GetBalance(ctx, owner)
	if err != nil {
		log.Fatalf("get balance: %v", err)
	}
	_ = bal

	// Freeze/Unfreeze with resource type constants
	// Bandwidth = ResourceTypeBandwidth, Energy = ResourceTypeEnergy
	_, err = rm.FreezeBalanceV2(ctx, owner, 1_000_000, resources.ResourceTypeEnergy) // 1 TRX in SUN
	if err != nil {
		log.Fatalf("freeze: %v", err)
	}

	_, err = rm.UnfreezeBalanceV2(ctx, owner, 1_000_000, resources.ResourceTypeEnergy)
	if err != nil {
		log.Fatalf("unfreeze: %v", err)
	}

	// Delegate/Undelegate example
	_, _ = rm.DelegateResource(ctx, owner, peer, 2_000_000, resources.ResourceTypeBandwidth, false)
	_, _ = rm.UnDelegateResource(ctx, owner, peer, 2_000_000, resources.ResourceTypeBandwidth)
}
```

## Smart Contract

Deploy and trigger examples using the SmartContractManager alias.

```go
package smartcontract_example

import (
	"context"
	"encoding/hex"
	"log"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/smartcontract"
	"github.com/kslamph/tronlib/pkg/types"
)

func Deploy() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cli, err := client.NewClient(client.DefaultClientConfig())
	if err != nil {
		log.Fatalf("client: %v", err)
	}
	defer cli.Close()

	var scm smartcontract.SmartContractManager = smartcontract.NewManager(cli)

	owner, _ := types.NewAddressFromBase58("Txxxxxxxxxxxxxxxxxxxxxxxxxxxxxx1")

	// Prepare ABI JSON and bytecode (dummy ABI and hex for illustration)
	abiJSON := `{"entrys":[{"name":"constructor","inputs":[{"name":"_owner","type":"address"}],"type":"constructor"},{"name":"setValue","inputs":[{"name":"v","type":"uint256"}],"type":"function"},{"name":"getValue","inputs":[],"type":"function","constant":true,"outputs":[{"name":"","type":"uint256"}]}]}`
	bytecode, _ := hex.DecodeString("6080604052348015600f57600080fd5b...deadbeef") // dummy hex

	// Deploy with constructor params (e.g., owner address as bytes)
	_, err = scm.DeployContract(
		ctx,
		owner,
		"MyContract",
		abiJSON,
		bytecode,
		0,      // callValue
		100,    // consumeUserResourcePercent
		3_0000, // originEnergyLimit
		owner.Bytes(), // constructor param example
	)
	if err != nil {
		log.Fatalf("deploy: %v", err)
	}
}

func Trigger() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cli, _ := client.NewClient(client.DefaultClientConfig())
	defer cli.Close()

	var scm smartcontract.SmartContractManager = smartcontract.NewManager(cli)

	owner, _ := types.NewAddressFromBase58("Txxxxxxxxxxxxxxxxxxxxxxxxxxxxxx1")
	contractAddr, _ := types.NewAddressFromBase58("Tyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy")

	// Construct Contract with NewContract (constant calls and triggers typically use ABI)
	contract, err := smartcontract.NewContract(
		contractAddr,
		`{"entrys":[{"name":"setValue","inputs":[{"name":"v","type":"uint256"}],"type":"function"},{"name":"getValue","inputs":[],"type":"function","constant":true,"outputs":[{"name":"","type":"uint256"}]}]}`,
	)
	if err != nil {
		log.Fatalf("new contract: %v", err)
	}

	// Trigger a state-changing method (requires signing/broadcasting)
	_, err = contract.TriggerSmartContract(ctx, cli, owner, "setValue", uint64(42))
	if err != nil {
		log.Fatalf("trigger: %v", err)
	}

	// Constant (read-only) call
	_, err = contract.TriggerConstantContract(ctx, cli, owner, "getValue")
	if err != nil {
		log.Fatalf("constant: %v", err)
	}
}
```

## TRC20

Read and write paths plus unit conversion helpers.

```go
package trc20_example

import (
	"context"
	"log"
	"math/big"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/trc20"
	"github.com/kslamph/tronlib/pkg/types"
)

func Example() {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	cli, _ := client.NewClient(client.DefaultClientConfig())
	defer cli.Close()

	tokenAddr, _ := types.NewAddressFromBase58("Tzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz")
	holder, _ := types.NewAddressFromBase58("Tholderxxxxxxxxxxxxxxxxxxxxxxxxx")
	spender, _ := types.NewAddressFromBase58("Tspenderxxxxxxxxxxxxxxxxxxxxxxxx")

	erc20 := trc20.NewClient(cli, tokenAddr)

	// Read paths
	name, _ := erc20.Name(ctx)
	symbol, _ := erc20.Symbol(ctx)
	decimals, _ := erc20.Decimals(ctx)
	bal, _ := erc20.BalanceOf(ctx, holder)
	allowance, _ := erc20.Allowance(ctx, holder, spender)

	_ = name
	_ = symbol
	_ = decimals
	_ = bal
	_ = allowance

	// Write paths (sign/broadcast separately)
	// Convert human amount -> wei using token decimals; FromWei for reverse.
	amount := trc20.ToWei(big.NewFloat(12.34), int32(decimals))
	// Ensure exact scale (no fractional part beyond decimals) to avoid precision loss.
	_, _ = erc20.Transfer(ctx, holder, tokenAddr, amount)
	_, _ = erc20.Approve(ctx, holder, spender, amount)
}
```

## Transaction Broadcast

DefaultBroadcastOptions control signing and broadcast behaviors.

Fields:
- FeeLimit: Max fee in SUN.
- WaitForReceipt: Whether to wait until transaction confirmed.
- WaitTimeout: Max time to wait for receipt.
- PollInterval: Interval for polling receipt status.
- PermissionID: Optional permission ID for multi-signature/permissioned accounts.

Example:

```go
package broadcast_example

import (
	"context"
	"log"
	"time"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pkg/client"
)

func Example(txExt *api.TransactionExtention) {
	cli, _ := client.NewClient(client.DefaultClientConfig())
	defer cli.Close()

	opts := client.DefaultBroadcastOptions()

	// Use defaults to sign and broadcast
	if err := cli.SignAndBroadcast(context.Background(), txExt, opts); err != nil {
		log.Fatalf("broadcast (defaults): %v", err)
	}

	// Override wait timeout and fee limit
	opts.WaitTimeout = 90 * time.Second
	opts.FeeLimit = 100_000_000 // 100 TRX (SUN)
	if err := cli.SignAndBroadcast(context.Background(), txExt, opts); err != nil {
		log.Fatalf("broadcast (override): %v", err)
	}
}
```

## Testing Notes (for contributors)

- Tests use hermetic bufconn gRPC servers to avoid external dependencies.
- Prefer fakes for client and deterministic timeouts/polling to ensure reliability.
- Keep test timeouts small; structure code to accept contexts and injected intervals.

## Versioning/Compatibility

- Additive changes like explicit type aliases and helper functions are non-breaking and maintain backward compatibility.
