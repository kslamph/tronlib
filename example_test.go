package tronlib_test

// Compile-only examples for the root facade (architecture §10's program shape).
//
// These examples deliberately carry NO // Output: comment: go test compiles
// them but never executes them, so the happy path is proven to build
// against the real surface without dialing a real node. docgen sync-docs
// DOES extract them: the CI drift gate passes this directory as
// -example-pkg ., docs/examples.md carries the tronlib.* markers, and every
// Example here must have one — so an edit to a body must be followed by a
// docgen sync of that file or the gate fails.

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kslamph/tronlib/v2"
)

// Example is the architecture §10 happy path: one import, dial, sign, broadcast.
func Example() {
	ctx := context.Background()

	cli, err := tronlib.Dial(ctx, "grpc://grpc.nile.trongrid.io:50051")
	if err != nil {
		fmt.Println("dial:", err)
		return
	}
	defer cli.Close()

	signer, err := tronlib.KeyFromHex(os.Getenv("TRON_PRIVATE_KEY"))
	if err != nil {
		fmt.Println("key:", err)
		return
	}
	to, err := tronlib.ParseAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
	if err != nil {
		fmt.Println("address:", err)
		return
	}

	transfer, err := cli.TransferTRX(ctx, signer.Address(), to, tronlib.TRX(1))
	if err != nil {
		fmt.Println("build:", err)
		return
	}
	signed, err := transfer.Sign(signer)
	if err != nil {
		fmt.Println("sign:", err)
		return
	}
	rec, err := cli.Broadcast(ctx, signed)
	if err != nil {
		fmt.Println("broadcast:", err)
		return
	}
	if !rec.OK() {
		fmt.Println("node rejected:", rec.NodeCode)
		return
	}
	// Inclusion is not finality; custody waits for solidification.
	if _, err := cli.WaitForSolid(ctx, rec.TxID); err != nil {
		fmt.Println("wait:", err)
	}
}

// ExampleClient_trx reads the account and chain state a transfer needs
// before it is built: tip, balance and energy price.
func ExampleClient_trx() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cli, err := tronlib.Dial(ctx, "grpc://grpc.nile.trongrid.io:50051",
		tronlib.WithTimeout(5*time.Second), tronlib.WithPool(1, 2))
	if err != nil {
		fmt.Println("dial:", err)
		return
	}
	defer cli.Close()

	from, err := tronlib.ParseAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
	if err != nil {
		fmt.Println("address:", err)
		return
	}
	bal, err := cli.TronBalance(ctx, from)
	if err != nil {
		fmt.Println("balance:", err)
		return
	}
	if _, err := cli.ChainTip(ctx); err != nil {
		fmt.Println("tip:", err)
		return
	}
	price, err := cli.EnergyPrice(ctx)
	if err != nil {
		fmt.Println("price:", err)
		return
	}
	fmt.Println(bal.Formatted(), "TRX at", price.SunPerEnergy, "sun/energy")
}

// ExampleClient_token reads a TRC-20 balance through the facade's Token
// handle; amounts minted by the Handle carry the token's decimals. The token
// address must be a contract: Client.Token builds a token.Handle, which
// calls decimals() eagerly, so an EOA address fails in that call instead of
// returning a handle. `owner` is the account whose balance is read.
func ExampleClient_token() {
	ctx := context.Background()

	cli, err := tronlib.Dial(ctx, "grpc://grpc.nile.trongrid.io:50051")
	if err != nil {
		fmt.Println("dial:", err)
		return
	}
	defer cli.Close()

	owner, err := tronlib.ParseAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
	if err != nil {
		fmt.Println("address:", err)
		return
	}
	// Nile's official USDT contract (docs/verification.md, address appendix).
	usdt, err := tronlib.ParseAddress("TXLAQ63Xg1NAzckPwKHvzw7CSEmLMEqcdj")
	if err != nil {
		fmt.Println("address:", err)
		return
	}
	handle, err := cli.Token(ctx, usdt)
	if err != nil {
		fmt.Println("token:", err)
		return
	}
	bal, err := handle.BalanceOf(ctx, owner)
	if err != nil {
		fmt.Println("balanceOf:", err)
		return
	}
	fmt.Println(bal.String())
}
