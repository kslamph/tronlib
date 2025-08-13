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

package trc20_test

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/trc20"
	"github.com/kslamph/tronlib/pkg/types"
)

// ExampleNewManager shows basic TRC20 reads and a transfer build using NewManager.
func ExampleNewManager() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cli, _ := client.NewClient("grpc://127.0.0.1:50051")
	defer cli.Close()

	token, _ := types.NewAddress("Ttokenxxxxxxxxxxxxxxxxxxxxxxxxxxx")
	holder, _ := types.NewAddress("Tholderxxxxxxxxxxxxxxxxxxxxxxxxx")
	recipient, _ := types.NewAddress("Trecipientxxxxxxxxxxxxxxxxxxxxx")

	t20, _ := trc20.NewManager(cli, token)
	_, _ = t20.Name(ctx)
	_, _ = t20.Symbol(ctx)
	_, _ = t20.Decimals(ctx)
	_, _ = t20.BalanceOf(ctx, holder)

	amount := decimal.NewFromFloat(1.23)
	_, _, _ = t20.Transfer(ctx, holder, recipient, amount)
}

// ExampleToWei demonstrates converting a decimal amount to on-chain integer units.
func ExampleToWei() {
	// Convert human 1.23 with 6 decimals to on-chain integer
	wei, _ := trc20.ToWei(decimal.NewFromFloat(1.23), 6)
	_ = wei
	// Output:
}

// ExampleFromWei demonstrates converting on-chain integer units to a decimal amount.
func ExampleFromWei() {
	// Convert on-chain integer to decimal with 6 decimals
	wei, _ := trc20.ToWei(decimal.NewFromFloat(1.23), 6)
	dec, _ := trc20.FromWei(wei, 6)
	_ = dec
	// Output:
}
