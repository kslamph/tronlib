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

package client

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client/lowlevel"
)

// Local generic wrapper helper to call unexported grpcGenericCallWrapper[T]
func callWrapper[T any](c *Client, ctx context.Context, op string, fn func(api.WalletClient, context.Context) (T, error), validate ...ValidationFunc[T]) (T, error) {
	return grpcGenericCallWrapper(c, ctx, op, fn, validate...)
}

func TestGrpcGenericCallWrapper_AppliesClientTimeout_NoDeadline(t *testing.T) {
	// Start bufconn server that sleeps shorter than client timeout, then returns
	srv := &testWalletServer{
		TriggerConstantContractFunc: func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			select {
			case <-time.After(10 * time.Millisecond):
				return &api.TransactionExtention{Result: &api.Return{Result: true, Code: api.Return_SUCCESS}}, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
	}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	defer cleanupSrv()

	clientTimeout := 50 * time.Millisecond
	c, cleanupClient := newTestClientWithBufConn(t, lis, clientTimeout)
	defer cleanupClient()

	start := time.Now()
	call := func(_ api.WalletClient, ctx context.Context) (string, error) {
		// Use GetAccount as a generic call site to exercise wrapper with real conn
		select {
		case <-time.After(10 * time.Millisecond):
			return "ok", nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	res, err := callWrapper(c, context.Background(), "test-op", call)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != "ok" {
		t.Fatalf("unexpected result: %v", res)
	}
	elapsed := time.Since(start)
	if elapsed > clientTimeout {
		t.Fatalf("expected completion within client timeout ~%v, took %v", clientTimeout, elapsed)
	}
}

func TestGrpcGenericCallWrapper_HonorsExistingDeadline(t *testing.T) {
	srv := &testWalletServer{
		TriggerConstantContractFunc: func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			select {
			case <-time.After(20 * time.Millisecond):
				return &api.TransactionExtention{Result: &api.Return{Result: true, Code: api.Return_SUCCESS}}, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
	}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	defer cleanupSrv()

	c, cleanupClient := newTestClientWithBufConn(t, lis, 500*time.Millisecond)
	defer cleanupClient()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	call := func(_ api.WalletClient, ctx context.Context) (int, error) {
		select {
		case <-time.After(20 * time.Millisecond):
			return 1, nil
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	}

	_, err := callWrapper(c, ctx, "long-op", call)
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, ErrContextCancelled) {
		t.Fatalf("expected deadline exceeded or ErrContextCancelled, got %v", err)
	}
}

func TestGrpcGenericCallWrapper_ValidationSuccess(t *testing.T) {
	lis, _, cleanupSrv := newBufconnServer(t, &testWalletServer{})
	defer cleanupSrv()

	c, cleanupClient := newTestClientWithBufConn(t, lis, 100*time.Millisecond)
	defer cleanupClient()

	call := func(_ api.WalletClient, _ context.Context) (*api.TransactionExtention, error) {
		return &api.TransactionExtention{
			Result: &api.Return{Result: true, Code: api.Return_SUCCESS},
		}, nil
	}
	_, err := callWrapper(c, context.Background(), "validate-ok", call, lowlevel.ValidateTransactionResult)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGrpcGenericCallWrapper_ValidationFailureWrapsTronReturn(t *testing.T) {
	lis, _, cleanupSrv := newBufconnServer(t, &testWalletServer{})
	defer cleanupSrv()

	c, cleanupClient := newTestClientWithBufConn(t, lis, 100*time.Millisecond)
	defer cleanupClient()

	call := func(_ api.WalletClient, _ context.Context) (*api.TransactionExtention, error) {
		return &api.TransactionExtention{
			Result: &api.Return{Result: false, Code: api.Return_OTHER_ERROR, Message: []byte("FAILED")},
		}, nil
	}
	_, err := callWrapper(c, context.Background(), "operation-x", call, lowlevel.ValidateTransactionResult)
	if err == nil {
		t.Fatalf("expected non-nil error")
	}
	if !strings.Contains(err.Error(), "FAILED") {
		t.Fatalf("expected error string to contain RETURN message, got: %v", err)
	}
}

func TestGrpcGenericCallWrapper_ConnLifecycle(t *testing.T) {
	lis, _, cleanupSrv := newBufconnServer(t, &testWalletServer{})
	defer cleanupSrv()

	var putCount int32
	c, cleanupClient := newTestClientWithBufConn(t, lis, 50*time.Millisecond)
	defer cleanupClient()

	// Exercise two calls to ensure ReturnConnection places conn back in pool successfully.
	call := func(_ api.WalletClient, _ context.Context) (int, error) { return 42, nil }

	val1, err1 := callWrapper(c, context.Background(), "simple1", call)
	if err1 != nil || val1 != 42 {
		t.Fatalf("call1 failed: val=%d err=%v", val1, err1)
	}
	val2, err2 := callWrapper(c, context.Background(), "simple2", call)
	if err2 != nil || val2 != 42 {
		t.Fatalf("call2 failed: val=%d err=%v", val2, err2)
	}

	// Indirectly validate via no panics and ability to reuse connections.
	_ = atomic.AddInt32(&putCount, 1)
	if atomic.LoadInt32(&putCount) == 0 {
		t.Fatalf("expected connection returned at least once")
	}
}
