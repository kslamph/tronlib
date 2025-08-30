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

// Package client provides high-level client functionality. Low-level 1:1 gRPC
// helpers are implemented in package lowlevel.
package client

import (
	"context"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pkg/client/lowlevel"
)

// ValidationFunc is a function type for validating gRPC call results
// T represents the return type of the gRPC call
// Re-exported from lowlevel for backward compatibility.
type ValidationFunc[T any] = lowlevel.ValidationFunc[T]

// grpcGenericCallWrapper wraps common gRPC call patterns using the lowlevel.Call
// while keeping the high-level client signature unchanged.
func grpcGenericCallWrapper[T any](c *Client, ctx context.Context, operation string, call func(client api.WalletClient, ctx context.Context) (T, error), validateFunc ...ValidationFunc[T]) (T, error) {
	return lowlevel.Call(c, ctx, operation, call, validateFunc...)
}
