package client

import (
	"context"
	"encoding/hex"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/signer"
	"github.com/kslamph/tronlib/pkg/types"
)

func TestSimulate_HappyPath(t *testing.T) {
	srv := &testWalletServer{
		TriggerConstantContractFunc: func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{
				EnergyUsed: 12345,
				Result:     &api.Return{Result: true, Message: []byte("OK")},
			}, nil
		},
	}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	t.Cleanup(cleanupSrv)

	c, cleanupClient := newTestClientWithBufConn(t, lis, 500*time.Millisecond)
	t.Cleanup(cleanupClient)

	// Build a minimal trigger smart contract transaction with future expiration
	tx := buildTriggerSmartContractTx(nil, nil, nil, time.Now().Add(2*time.Second))

	ext, err := c.Simulate(context.Background(), tx)
	if err != nil {
		t.Fatalf("Simulate error: %v", err)
	}
	if ext == nil {
		t.Fatalf("expected non-nil result")
	}
	if ext.GetEnergyUsed() <= 0 {
		t.Fatalf("expected energy used > 0, got %d", ext.GetEnergyUsed())
	}
}

func TestSignAndBroadcast_NoSigners(t *testing.T) {
	srv := &testWalletServer{
		BroadcastHandler: func(ctx context.Context, in *core.Transaction) (*api.Return, error) {
			return &api.Return{Result: true, Code: api.Return_SUCCESS}, nil
		},
	}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	t.Cleanup(cleanupSrv)

	c, cleanupClient := newTestClientWithBufConn(t, lis, 500*time.Millisecond)
	t.Cleanup(cleanupClient)

	tx := buildTriggerSmartContractTx(nil, nil, nil, time.Now().Add(2*time.Second))

	res, err := c.SignAndBroadcast(context.Background(), tx, BroadcastOptions{WaitForReceipt: false})
	if err != nil {
		t.Fatalf("SignAndBroadcast error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success true")
	}
	wantTxID := hex.EncodeToString(types.GetTransactionID(tx))
	if res.TxID != wantTxID {
		t.Fatalf("unexpected txid: got %s want %s", res.TxID, wantTxID)
	}
}

func TestSignAndBroadcast_WithSignerPermissionAndFee(t *testing.T) {
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

	tx := buildTriggerSmartContractTx(nil, nil, nil, time.Now().Add(2*time.Second))

	opts := BroadcastOptions{
		PermissionID:   2,
		FeeLimit:       1_000_000,
		WaitForReceipt: false,
	}
	res, err := c.SignAndBroadcast(context.Background(), tx, opts, fakeSigner)
	if err != nil {
		t.Fatalf("SignAndBroadcast error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success true")
	}
	if got := tx.GetRawData().GetContract()[0].GetPermissionId(); got != 2 {
		t.Fatalf("permission id not applied, got %d", got)
	}
	if gotFee := tx.GetRawData().GetFeeLimit(); gotFee != 1_000_000 {
		t.Fatalf("fee limit not applied, got %d", gotFee)
	}
}

func TestSignAndBroadcast_WaitForReceipt_Success(t *testing.T) {
	var polls int32
	var txidSeen []byte

	srv := &testWalletServer{
		BroadcastHandler: func(ctx context.Context, in *core.Transaction) (*api.Return, error) {
			txidSeen = types.GetTransactionID(in)
			return &api.Return{Result: true, Code: api.Return_SUCCESS}, nil
		},
		GetTxInfoByIdHandler: func(ctx context.Context, in *api.BytesMessage) (*core.TransactionInfo, error) {
			// Return nil a couple times, then a real receipt
			if atomic.AddInt32(&polls, 1) <= 2 {
				return nil, nil
			}
			return &core.TransactionInfo{
				Id:             in.GetValue(),
				ContractResult: [][]byte{[]byte("ok")},
				Receipt:        &core.ResourceReceipt{},
			}, nil
		},
	}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	t.Cleanup(cleanupSrv)

	c, cleanupClient := newTestClientWithBufConn(t, lis, 500*time.Millisecond)
	t.Cleanup(cleanupClient)

	tx := buildTriggerSmartContractTx(nil, nil, nil, time.Now().Add(2*time.Second))

	// WaitForReceipt=true. Use short timeout to keep tests fast; set small timeout and small poll interval to make test deterministic.
	opts := BroadcastOptions{
		WaitForReceipt: true,
		WaitTimeout:    4,                 // seconds
		PollInterval:   150 * time.Millisecond,
	}
	res, err := c.SignAndBroadcast(context.Background(), tx, opts)
	if err != nil {
		t.Fatalf("SignAndBroadcast error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected broadcast success")
	}
	if res.ContractReceipt == nil {
		t.Fatalf("expected non-nil contract receipt")
	}
	if txidSeen == nil || res.TxID != hex.EncodeToString(txidSeen) {
		t.Fatalf("txid mismatch or not captured")
	}
}

func TestSignAndBroadcast_WaitForReceipt_Timeout(t *testing.T) {
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

	tx := buildTriggerSmartContractTx(nil, nil, nil, time.Now().Add(2*time.Second))

	opts := BroadcastOptions{
		WaitForReceipt: true,
		WaitTimeout:    1, // seconds, short to avoid flakiness
		PollInterval:   100 * time.Millisecond,
	}
	res, err := c.SignAndBroadcast(context.Background(), tx, opts)
	if err != nil {
		t.Fatalf("SignAndBroadcast error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected broadcast success")
	}
	if res.ContractReceipt != nil {
		t.Fatalf("expected nil contract receipt due to timeout")
	}
}

func TestSimulate_ValidationErrors(t *testing.T) {
	t.Run("nil tx", func(t *testing.T) {
		c := &Client{}
		if _, err := c.Simulate(context.Background(), nil); err == nil {
			t.Fatalf("expected error for nil tx")
		}
	})
	t.Run("nil raw data", func(t *testing.T) {
		c := &Client{}
		tx := &core.Transaction{} // RawData nil
		if _, err := c.Simulate(context.Background(), tx); err == nil {
			t.Fatalf("expected error for nil raw data")
		}
	})
	t.Run("zero contracts", func(t *testing.T) {
		c := &Client{}
		tx := &core.Transaction{RawData: &core.TransactionRaw{
			Contract:   nil,
			Expiration: time.Now().Add(2 * time.Second).UnixNano(),
		}}
		if _, err := c.Simulate(context.Background(), tx); err == nil {
			t.Fatalf("expected error for 0 contracts")
		}
	})
	t.Run(">1 contracts", func(t *testing.T) {
		c := &Client{}
		tx := &core.Transaction{RawData: &core.TransactionRaw{
			Contract: []*core.Transaction_Contract{
				{Type: core.Transaction_Contract_TriggerSmartContract},
				{Type: core.Transaction_Contract_TriggerSmartContract},
			},
			Expiration: time.Now().Add(2 * time.Second).UnixNano(),
		}}
		if _, err := c.Simulate(context.Background(), tx); err == nil {
			t.Fatalf("expected error for >1 contracts")
		}
	})
	t.Run("expiration in the past", func(t *testing.T) {
		c := &Client{}
		tx := &core.Transaction{RawData: &core.TransactionRaw{
			Contract: []*core.Transaction_Contract{
				{Type: core.Transaction_Contract_TriggerSmartContract},
			},
			Expiration: time.Now().Add(-1 * time.Second).UnixNano(),
		}}
		if _, err := c.Simulate(context.Background(), tx); err == nil {
			t.Fatalf("expected error for past expiration")
		}
	})
}
