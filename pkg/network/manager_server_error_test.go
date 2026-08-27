package network

import (
	"context"
	"testing"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestNetworkManager_ServerError verifies every RPC surfaces an error when the
// node returns a transport error (here: gRPC Unavailable).
func TestNetworkManager_ServerError(t *testing.T) {
	nodeErr := status.Error(codes.Unavailable, "node unavailable")
	ctx := context.Background()

	tests := []struct {
		name string
		fake *fakeWalletServer
		call func(m *NetworkManager) error
	}{
		{"GetNodeInfo", &fakeWalletServer{GetNodeInfoFunc: func(context.Context, *api.EmptyMessage) (*core.NodeInfo, error) { return nil, nodeErr }}, func(m *NetworkManager) error {
			_, err := m.GetNodeInfo(ctx)
			return err
		}},
		{"GetChainParameters", &fakeWalletServer{GetChainParametersFunc: func(context.Context, *api.EmptyMessage) (*core.ChainParameters, error) { return nil, nodeErr }}, func(m *NetworkManager) error {
			_, err := m.GetChainParameters(ctx)
			return err
		}},
		{"ListNodes", &fakeWalletServer{ListNodesFunc: func(context.Context, *api.EmptyMessage) (*api.NodeList, error) { return nil, nodeErr }}, func(m *NetworkManager) error {
			_, err := m.ListNodes(ctx)
			return err
		}},
		{"GetBlockByNumber", &fakeWalletServer{GetBlockByNum2Func: func(context.Context, *api.NumberMessage) (*api.BlockExtention, error) { return nil, nodeErr }}, func(m *NetworkManager) error {
			_, err := m.GetBlockByNumber(ctx, 1)
			return err
		}},
		{"GetTransactionInfoByBlockNum", &fakeWalletServer{GetTransactionInfoByBlockNumFunc: func(context.Context, *api.NumberMessage) (*api.TransactionInfoList, error) { return nil, nodeErr }}, func(m *NetworkManager) error {
			_, err := m.GetTransactionInfoByBlockNum(ctx, 1)
			return err
		}},
		{"GetBlockById", &fakeWalletServer{GetBlockByIdFunc: func(context.Context, *api.BytesMessage) (*core.Block, error) { return nil, nodeErr }}, func(m *NetworkManager) error {
			_, err := m.GetBlockById(ctx, []byte("blockid"))
			return err
		}},
		{"GetBlocksByLimit", &fakeWalletServer{GetBlockByLimitNext2Func: func(context.Context, *api.BlockLimit) (*api.BlockListExtention, error) { return nil, nodeErr }}, func(m *NetworkManager) error {
			_, err := m.GetBlocksByLimit(ctx, 1, 2)
			return err
		}},
		{"GetLatestBlocks", &fakeWalletServer{GetBlockByLatestNum2Func: func(context.Context, *api.NumberMessage) (*api.BlockListExtention, error) { return nil, nodeErr }}, func(m *NetworkManager) error {
			_, err := m.GetLatestBlocks(ctx, 2)
			return err
		}},
		{"GetNowBlock", &fakeWalletServer{GetNowBlock2Func: func(context.Context, *api.EmptyMessage) (*api.BlockExtention, error) { return nil, nodeErr }}, func(m *NetworkManager) error {
			_, err := m.GetNowBlock(ctx)
			return err
		}},
		{"GetTransactionInfoById", &fakeWalletServer{GetTransactionInfoByIdFunc: func(context.Context, *api.BytesMessage) (*core.TransactionInfo, error) { return nil, nodeErr }}, func(m *NetworkManager) error {
			_, err := m.GetTransactionInfoById(ctx, "a1b2c3d4")
			return err
		}},
		{"GetTransactionById", &fakeWalletServer{GetTransactionByIdFunc: func(context.Context, *api.BytesMessage) (*core.Transaction, error) { return nil, nodeErr }}, func(m *NetworkManager) error {
			_, err := m.GetTransactionById(ctx, "a1b2c3d4")
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr, cleanup := setupTestServer(t, tt.fake)
			defer cleanup()
			if err := tt.call(mgr); err == nil {
				t.Fatal("expected error from server")
			}
		})
	}
}
