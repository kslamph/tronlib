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

package lowlevel

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pkg/types"
)

// ValidateTransactionResult checks the common result pattern for transaction operations.
func ValidateTransactionResult(result *api.TransactionExtention, operation string) error {
	if result == nil {
		return fmt.Errorf("nil result for %s", operation)
	}
	if result.Result == nil {
		return fmt.Errorf("nil result field for %s", operation)
	}
	if !result.Result.Result {
		return types.WrapTransactionResult(result.Result, operation)
	}
	return nil
}

// TxCall is a specialization of Call for RPCs returning *api.TransactionExtention.
func TxCall(cp ConnProvider, ctx context.Context, operation string, call func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error)) (*api.TransactionExtention, error) {
	return Call(cp, ctx, operation, call, ValidateTransactionResult)
}
