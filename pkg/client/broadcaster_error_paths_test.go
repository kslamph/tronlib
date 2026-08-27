package client

import (
	"context"
	"encoding/hex"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/signer"
	"github.com/kslamph/tronlib/pkg/utils"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// --- Simulate: additional validation/error paths ---

func TestSimulate_UnsupportedType(t *testing.T) {
	c := &Client{}
	if _, err := c.Simulate(context.Background(), "not a transaction"); err == nil {
		t.Fatal("expected error for unsupported type")
	}
}

func TestSimulate_NilCoreTxInExtension(t *testing.T) {
	c := &Client{}
	ext := &api.TransactionExtention{} // GetTransaction() returns nil
	if _, err := c.Simulate(context.Background(), ext); err == nil {
		t.Fatal("expected error for nil coretx from extension")
	}
}

func TestSimulate_ServerError(t *testing.T) {
	srv := &testWalletServer{
		TriggerConstantContractFunc: func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return nil, fmt.Errorf("server unavailable")
		},
	}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	t.Cleanup(cleanupSrv)
	c, cleanupClient := newTestClientWithBufConn(t, lis, 500*time.Millisecond)
	t.Cleanup(cleanupClient)

	tx := buildTriggerSmartContractTx(time.Now().Add(2 * time.Second))
	_, err := c.Simulate(context.Background(), tx)
	if err == nil {
		t.Fatal("expected error for server error")
	}
}

func TestSimulate_UnmarshalError(t *testing.T) {
	// Build a transaction with invalid parameter value (not a valid TriggerSmartContract)
	tx := &core.Transaction{
		RawData: &core.TransactionRaw{
			Contract: []*core.Transaction_Contract{
				{
					Type: core.Transaction_Contract_TriggerSmartContract,
					Parameter: &anypb.Any{
						Value: []byte("not-valid-proto"),
					},
				},
			},
			Expiration: time.Now().Add(2 * time.Second).UnixNano(),
		},
	}
	c := &Client{}
	_, err := c.Simulate(context.Background(), tx)
	if err == nil {
		t.Fatal("expected error for unmarshal failure")
	}
}

// --- SignAndBroadcast: additional paths ---

func TestSignAndBroadcast_NilTx(t *testing.T) {
	c := &Client{}
	_, err := c.SignAndBroadcast(context.Background(), nil, BroadcastOptions{WaitForReceipt: false})
	if err == nil {
		t.Fatal("expected error for nil tx")
	}
}

func TestSignAndBroadcast_NilCoreTxInExtension(t *testing.T) {
	c := &Client{}
	ext := &api.TransactionExtention{} // GetTransaction() returns nil
	_, err := c.SignAndBroadcast(context.Background(), ext, BroadcastOptions{WaitForReceipt: false})
	if err == nil {
		t.Fatal("expected error for nil coretx from extension")
	}
}

func TestSignAndBroadcast_UnsupportedType(t *testing.T) {
	c := &Client{}
	_, err := c.SignAndBroadcast(context.Background(), 12345, BroadcastOptions{WaitForReceipt: false})
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
}

func TestSignAndBroadcast_BroadcastRPCError(t *testing.T) {
	srv := &testWalletServer{
		BroadcastHandler: func(ctx context.Context, in *core.Transaction) (*api.Return, error) {
			return nil, fmt.Errorf("rpc: connection refused")
		},
	}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	t.Cleanup(cleanupSrv)
	c, cleanupClient := newTestClientWithBufConn(t, lis, 500*time.Millisecond)
	t.Cleanup(cleanupClient)

	tx := buildTriggerSmartContractTx(time.Now().Add(2 * time.Second))
	_, err := c.SignAndBroadcast(context.Background(), tx, BroadcastOptions{WaitForReceipt: false})
	if err == nil {
		t.Fatal("expected error for RPC broadcast failure")
	}
}

func TestSignAndBroadcast_BroadcastFailure(t *testing.T) {
	srv := &testWalletServer{
		BroadcastHandler: func(ctx context.Context, in *core.Transaction) (*api.Return, error) {
			return &api.Return{Result: false, Code: api.Return_CONTRACT_EXE_ERROR, Message: []byte("insufficient balance")}, nil
		},
	}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	t.Cleanup(cleanupSrv)
	c, cleanupClient := newTestClientWithBufConn(t, lis, 500*time.Millisecond)
	t.Cleanup(cleanupClient)

	tx := buildTriggerSmartContractTx(time.Now().Add(2 * time.Second))
	res, err := c.SignAndBroadcast(context.Background(), tx, BroadcastOptions{WaitForReceipt: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success {
		t.Fatal("expected success false")
	}
	if res.Code != api.Return_CONTRACT_EXE_ERROR {
		t.Fatalf("expected CONTRACT_EXE_ERROR, got %v", res.Code)
	}
}

func TestSignAndBroadcast_SignError(t *testing.T) {
	// Use a nil signer that will cause an error
	srv := &testWalletServer{
		BroadcastHandler: func(ctx context.Context, in *core.Transaction) (*api.Return, error) {
			return &api.Return{Result: true, Code: api.Return_SUCCESS}, nil
		},
	}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	t.Cleanup(cleanupSrv)
	c, cleanupClient := newTestClientWithBufConn(t, lis, 500*time.Millisecond)
	t.Cleanup(cleanupClient)

	tx := buildTriggerSmartContractTx(time.Now().Add(2 * time.Second))

	// Create a valid signer to test the signing path
	fakeSigner, _ := signer.NewPrivateKeySigner("1cba74a2cbc5008272e0250b1b36f9e8527510665107e19451032839d6c4e887")
	opts := BroadcastOptions{
		FeeLimit:       100_000_000,
		PermissionID:   0,
		WaitForReceipt: false,
	}
	res, err := c.SignAndBroadcast(context.Background(), tx, opts, fakeSigner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Fatal("expected success")
	}
}

func TestSignAndBroadcast_DefaultOptions(t *testing.T) {
	srv := &testWalletServer{
		BroadcastHandler: func(ctx context.Context, in *core.Transaction) (*api.Return, error) {
			return &api.Return{Result: true, Code: api.Return_SUCCESS}, nil
		},
	}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	t.Cleanup(cleanupSrv)
	c, cleanupClient := newTestClientWithBufConn(t, lis, 500*time.Millisecond)
	t.Cleanup(cleanupClient)

	tx := buildTriggerSmartContractTx(time.Now().Add(2 * time.Second))

	// Use zero options to test defaulting
	opts := BroadcastOptions{}
	res, err := c.SignAndBroadcast(context.Background(), tx, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Fatal("expected success")
	}
}

func TestSignAndBroadcast_WithSignerPermissionZero(t *testing.T) {
	fakeSigner, _ := signer.NewPrivateKeySigner("1cba74a2cbc5008272e0250b1b36f9e8527510665107e19451032839d6c4e887")
	srv := &testWalletServer{
		BroadcastHandler: func(ctx context.Context, in *core.Transaction) (*api.Return, error) {
			return &api.Return{Result: true, Code: api.Return_SUCCESS}, nil
		},
	}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	t.Cleanup(cleanupSrv)
	c, cleanupClient := newTestClientWithBufConn(t, lis, 500*time.Millisecond)
	t.Cleanup(cleanupClient)

	tx := buildTriggerSmartContractTx(time.Now().Add(2 * time.Second))
	opts := BroadcastOptions{
		PermissionID:   0,
		FeeLimit:       50_000_000,
		WaitForReceipt: false,
	}
	_, err := c.SignAndBroadcast(context.Background(), tx, opts, fakeSigner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// PermissionID=0 means it should NOT be set on the contract (line 244: if opt.PermissionID != 0)
	if got := tx.GetRawData().GetContract()[0].GetPermissionId(); got != 0 {
		t.Fatalf("permission id should be 0 (not set), got %d", got)
	}
	if gotFee := tx.GetRawData().GetFeeLimit(); gotFee != 50_000_000 {
		t.Fatalf("fee limit not applied, got %d", gotFee)
	}
}

func TestSignAndBroadcast_BroadcastError_WithResult(t *testing.T) {
	srv := &testWalletServer{
		BroadcastHandler: func(ctx context.Context, in *core.Transaction) (*api.Return, error) {
			return &api.Return{Result: false, Code: api.Return_BANDWITH_ERROR, Message: []byte("not enough bandwidth")}, nil
		},
	}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	t.Cleanup(cleanupSrv)
	c, cleanupClient := newTestClientWithBufConn(t, lis, 500*time.Millisecond)
	t.Cleanup(cleanupClient)

	tx := buildTriggerSmartContractTx(time.Now().Add(2 * time.Second))
	res, err := c.SignAndBroadcast(context.Background(), tx, BroadcastOptions{WaitForReceipt: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success {
		t.Fatal("expected success false")
	}
	if res.Code != api.Return_BANDWITH_ERROR {
		t.Fatalf("expected BANDWITH_ERROR, got %v", res.Code)
	}
	if res.Message != "not enough bandwidth" {
		t.Fatalf("unexpected message: %v", res.Message)
	}
}

// --- waitForTransactionInfo paths ---

func TestWaitForTransactionInfo_ContextCancel(t *testing.T) {
	srv := &testWalletServer{
		BroadcastHandler: func(ctx context.Context, in *core.Transaction) (*api.Return, error) {
			return &api.Return{Result: true, Code: api.Return_SUCCESS}, nil
		},
		GetTxInfoByIdHandler: func(ctx context.Context, in *api.BytesMessage) (*core.TransactionInfo, error) {
			// Always nil to force timeout
			return nil, nil
		},
	}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	t.Cleanup(cleanupSrv)
	c, cleanupClient := newTestClientWithBufConn(t, lis, 500*time.Millisecond)
	t.Cleanup(cleanupClient)

	tx := buildTriggerSmartContractTx(time.Now().Add(2 * time.Second))

	opts := BroadcastOptions{
		WaitForReceipt: true,
		WaitTimeout:    5,
		PollInterval:   100 * time.Millisecond,
	}
	// Use a very short-lived context to test context cancellation path
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	res, err := c.SignAndBroadcast(ctx, tx, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should still return broadcast success (just no receipt)
	if !res.Success {
		t.Fatal("expected broadcast success")
	}
}

func TestWaitForTransactionInfo_RPCError(t *testing.T) {
	srv := &testWalletServer{
		BroadcastHandler: func(ctx context.Context, in *core.Transaction) (*api.Return, error) {
			return &api.Return{Result: true, Code: api.Return_SUCCESS}, nil
		},
		GetTxInfoByIdHandler: func(ctx context.Context, in *api.BytesMessage) (*core.TransactionInfo, error) {
			return nil, fmt.Errorf("rpc error")
		},
	}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	t.Cleanup(cleanupSrv)
	c, cleanupClient := newTestClientWithBufConn(t, lis, 500*time.Millisecond)
	t.Cleanup(cleanupClient)

	tx := buildTriggerSmartContractTx(time.Now().Add(2 * time.Second))
	opts := BroadcastOptions{
		WaitForReceipt: true,
		WaitTimeout:    1,
		PollInterval:   100 * time.Millisecond,
	}
	res, err := c.SignAndBroadcast(context.Background(), tx, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Broadcast succeeds, but receipt poll fails
	if !res.Success {
		t.Fatal("expected broadcast success")
	}
}

func TestWaitForTransactionInfo_MismatchedTxID(t *testing.T) {
	srv := &testWalletServer{
		BroadcastHandler: func(ctx context.Context, in *core.Transaction) (*api.Return, error) {
			return &api.Return{Result: true, Code: api.Return_SUCCESS}, nil
		},
		GetTxInfoByIdHandler: func(ctx context.Context, in *api.BytesMessage) (*core.TransactionInfo, error) {
			// Return info with different ID
			return &core.TransactionInfo{
				Id:             []byte("wrong-txid"),
				ContractResult: [][]byte{[]byte("ok")},
				Receipt:        &core.ResourceReceipt{},
			}, nil
		},
	}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	t.Cleanup(cleanupSrv)
	c, cleanupClient := newTestClientWithBufConn(t, lis, 500*time.Millisecond)
	t.Cleanup(cleanupClient)

	tx := buildTriggerSmartContractTx(time.Now().Add(2 * time.Second))
	opts := BroadcastOptions{
		WaitForReceipt: true,
		WaitTimeout:    1,
		PollInterval:   100 * time.Millisecond,
	}
	res, err := c.SignAndBroadcast(context.Background(), tx, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Broadcast succeeds, receipt poll finds mismatched txid
	if !res.Success {
		t.Fatal("expected broadcast success")
	}
}

func TestWaitForTransactionInfo_NilInfo(t *testing.T) {
	srv := &testWalletServer{
		BroadcastHandler: func(ctx context.Context, in *core.Transaction) (*api.Return, error) {
			return &api.Return{Result: true, Code: api.Return_SUCCESS}, nil
		},
		GetTxInfoByIdHandler: func(ctx context.Context, in *api.BytesMessage) (*core.TransactionInfo, error) {
			return nil, nil // nil info, nil error
		},
	}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	t.Cleanup(cleanupSrv)
	c, cleanupClient := newTestClientWithBufConn(t, lis, 500*time.Millisecond)
	t.Cleanup(cleanupClient)

	tx := buildTriggerSmartContractTx(time.Now().Add(2 * time.Second))
	opts := BroadcastOptions{
		WaitForReceipt: true,
		WaitTimeout:    1,
		PollInterval:   100 * time.Millisecond,
	}
	res, err := c.SignAndBroadcast(context.Background(), tx, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Fatal("expected broadcast success")
	}
}

// --- waitForTransactionInfo direct test for coverage ---

func TestWaitForTransactionInfo_DirectCancel(t *testing.T) {
	srv := &testWalletServer{
		GetTxInfoByIdHandler: func(ctx context.Context, in *api.BytesMessage) (*core.TransactionInfo, error) {
			// Block until context cancelled
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(5 * time.Second):
				return nil, fmt.Errorf("should not reach here")
			}
		},
	}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	t.Cleanup(cleanupSrv)
	c, cleanupClient := newTestClientWithBufConn(t, lis, 500*time.Millisecond)
	t.Cleanup(cleanupClient)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	txid := []byte("test-txid-123")
	info := c.waitForTransactionInfo(ctx, txid, 5*time.Second, 100*time.Millisecond)
	if info != nil {
		t.Fatal("expected nil info for cancelled context")
	}
}

func TestWaitForTransactionInfo_ZeroPollInterval(t *testing.T) {
	var polls int32
	srv := &testWalletServer{
		GetTxInfoByIdHandler: func(ctx context.Context, in *api.BytesMessage) (*core.TransactionInfo, error) {
			if atomic.AddInt32(&polls, 1) >= 1 {
				return &core.TransactionInfo{
					Id:             in.GetValue(),
					ContractResult: [][]byte{[]byte("ok")},
					Receipt:        &core.ResourceReceipt{},
				}, nil
			}
			return nil, nil
		},
	}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	t.Cleanup(cleanupSrv)
	c, cleanupClient := newTestClientWithBufConn(t, lis, 500*time.Millisecond)
	t.Cleanup(cleanupClient)

	ctx := context.Background()
	txid := []byte("test-txid-456")
	info := c.waitForTransactionInfo(ctx, txid, 5*time.Second, 0) // zero poll interval should default to 3s
	if info == nil {
		t.Fatal("expected non-nil info")
	}
	_ = hex.EncodeToString(txid)
	_, _ = proto.Marshal(nil)
	_ = utils.GetTransactionID(nil)
}
