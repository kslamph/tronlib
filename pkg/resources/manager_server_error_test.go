package resources

import (
	"context"
	"testing"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestResourcesManager_ServerError verifies every RPC surfaces an error when
// the node returns a transport error (here: gRPC Unavailable).
func TestResourcesManager_ServerError(t *testing.T) {
	nodeErr := status.Error(codes.Unavailable, "node unavailable")
	owner := testAddr
	receiver := testAddr2
	ctx := context.Background()

	tests := []struct {
		name string
		fake *fakeWalletServer
		call func(m *ResourcesManager) error
	}{
		{"FreezeBalanceV2", &fakeWalletServer{FreezeBalanceV2Func: func(context.Context, *core.FreezeBalanceV2Contract) (*api.TransactionExtention, error) {
			return nil, nodeErr
		}}, func(m *ResourcesManager) error {
			_, err := m.FreezeBalanceV2(ctx, owner, 1_000_000, ResourceTypeBandwidth)
			return err
		}},
		{"UnfreezeBalanceV2", &fakeWalletServer{UnfreezeBalanceV2Func: func(context.Context, *core.UnfreezeBalanceV2Contract) (*api.TransactionExtention, error) {
			return nil, nodeErr
		}}, func(m *ResourcesManager) error {
			_, err := m.UnfreezeBalanceV2(ctx, owner, 1_000_000, ResourceTypeBandwidth)
			return err
		}},
		{"DelegateResource", &fakeWalletServer{DelegateResourceFunc: func(context.Context, *core.DelegateResourceContract) (*api.TransactionExtention, error) {
			return nil, nodeErr
		}}, func(m *ResourcesManager) error {
			_, err := m.DelegateResource(ctx, owner, receiver, 1_000_000, ResourceTypeEnergy, false)
			return err
		}},
		{"UnDelegateResource", &fakeWalletServer{UnDelegateResourceFunc: func(context.Context, *core.UnDelegateResourceContract) (*api.TransactionExtention, error) {
			return nil, nodeErr
		}}, func(m *ResourcesManager) error {
			_, err := m.UnDelegateResource(ctx, owner, receiver, 1_000_000, ResourceTypeEnergy)
			return err
		}},
		{"CancelAllUnfreezeV2", &fakeWalletServer{CancelAllUnfreezeV2Func: func(context.Context, *core.CancelAllUnfreezeV2Contract) (*api.TransactionExtention, error) {
			return nil, nodeErr
		}}, func(m *ResourcesManager) error {
			_, err := m.CancelAllUnfreezeV2(ctx, owner)
			return err
		}},
		{"WithdrawExpireUnfreeze", &fakeWalletServer{WithdrawExpireUnfreezeFunc: func(context.Context, *core.WithdrawExpireUnfreezeContract) (*api.TransactionExtention, error) {
			return nil, nodeErr
		}}, func(m *ResourcesManager) error {
			_, err := m.WithdrawExpireUnfreeze(ctx, owner)
			return err
		}},
		{"GetDelegatedResourceV2", &fakeWalletServer{GetDelegatedResourceV2Func: func(context.Context, *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
			return nil, nodeErr
		}}, func(m *ResourcesManager) error {
			_, err := m.GetDelegatedResourceV2(ctx, owner, receiver)
			return err
		}},
		{"GetDelegatedResourceAccountIndexV2", &fakeWalletServer{GetDelegatedResourceAccountIndexV2Func: func(context.Context, *api.BytesMessage) (*core.DelegatedResourceAccountIndex, error) {
			return nil, nodeErr
		}}, func(m *ResourcesManager) error {
			_, err := m.GetDelegatedResourceAccountIndexV2(ctx, owner)
			return err
		}},
		{"GetCanDelegatedMaxSize", &fakeWalletServer{GetCanDelegatedMaxSizeFunc: func(context.Context, *api.CanDelegatedMaxSizeRequestMessage) (*api.CanDelegatedMaxSizeResponseMessage, error) {
			return nil, nodeErr
		}}, func(m *ResourcesManager) error {
			_, err := m.GetCanDelegatedMaxSize(ctx, owner, 0)
			return err
		}},
		{"GetAvailableUnfreezeCount", &fakeWalletServer{GetAvailableUnfreezeCountFunc: func(context.Context, *api.GetAvailableUnfreezeCountRequestMessage) (*api.GetAvailableUnfreezeCountResponseMessage, error) {
			return nil, nodeErr
		}}, func(m *ResourcesManager) error {
			_, err := m.GetAvailableUnfreezeCount(ctx, owner)
			return err
		}},
		{"GetCanWithdrawUnfreezeAmount", &fakeWalletServer{GetCanWithdrawUnfreezeAmountFunc: func(context.Context, *api.CanWithdrawUnfreezeAmountRequestMessage) (*api.CanWithdrawUnfreezeAmountResponseMessage, error) {
			return nil, nodeErr
		}}, func(m *ResourcesManager) error {
			_, err := m.GetCanWithdrawUnfreezeAmount(ctx, owner, 0)
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
