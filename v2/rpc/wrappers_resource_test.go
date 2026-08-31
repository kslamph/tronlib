package rpc

import (
	"context"
	"testing"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
)

// TestResourceWrappers exercises the 1:1 port of lowlevel/resource.go through
// the bufconn fake (happy path + node-error path per wrapper).
func TestResourceWrappers(t *testing.T) {
	runWrapperGroup(t, []wrapperCase{
		{"FreezeBalanceV2", "FreezeBalanceV2", "freeze balance v2", func(cp ConnProvider, ctx context.Context) error {
			_, err := FreezeBalanceV2(cp, ctx, &core.FreezeBalanceV2Contract{})
			return err
		}},
		{"UnfreezeBalanceV2", "UnfreezeBalanceV2", "unfreeze balance v2", func(cp ConnProvider, ctx context.Context) error {
			_, err := UnfreezeBalanceV2(cp, ctx, &core.UnfreezeBalanceV2Contract{})
			return err
		}},
		{"DelegateResource", "DelegateResource", "delegate resource", func(cp ConnProvider, ctx context.Context) error {
			_, err := DelegateResource(cp, ctx, &core.DelegateResourceContract{})
			return err
		}},
		{"UnDelegateResource", "UnDelegateResource", "undelegate resource", func(cp ConnProvider, ctx context.Context) error {
			_, err := UnDelegateResource(cp, ctx, &core.UnDelegateResourceContract{})
			return err
		}},
		{"CancelAllUnfreezeV2", "CancelAllUnfreezeV2", "cancel all unfreeze v2", func(cp ConnProvider, ctx context.Context) error {
			_, err := CancelAllUnfreezeV2(cp, ctx, &core.CancelAllUnfreezeV2Contract{})
			return err
		}},
		{"WithdrawExpireUnfreeze", "WithdrawExpireUnfreeze", "withdraw expire unfreeze", func(cp ConnProvider, ctx context.Context) error {
			_, err := WithdrawExpireUnfreeze(cp, ctx, &core.WithdrawExpireUnfreezeContract{})
			return err
		}},
		{"GetDelegatedResourceV2", "GetDelegatedResourceV2", "get delegated resource v2", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetDelegatedResourceV2(cp, ctx, &api.DelegatedResourceMessage{})
			return err
		}},
		{"GetDelegatedResourceAccountIndexV2", "GetDelegatedResourceAccountIndexV2", "get delegated resource account index v2", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetDelegatedResourceAccountIndexV2(cp, ctx, &api.BytesMessage{})
			return err
		}},
		{"GetCanDelegatedMaxSize", "GetCanDelegatedMaxSize", "get can delegated max size", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetCanDelegatedMaxSize(cp, ctx, &api.CanDelegatedMaxSizeRequestMessage{})
			return err
		}},
		{"GetAvailableUnfreezeCount", "GetAvailableUnfreezeCount", "get available unfreeze count", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetAvailableUnfreezeCount(cp, ctx, &api.GetAvailableUnfreezeCountRequestMessage{})
			return err
		}},
		{"GetCanWithdrawUnfreezeAmount", "GetCanWithdrawUnfreezeAmount", "get can withdraw unfreeze amount", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetCanWithdrawUnfreezeAmount(cp, ctx, &api.CanWithdrawUnfreezeAmountRequestMessage{})
			return err
		}},
		{"FreezeBalance2", "FreezeBalance2", "freeze balance2", func(cp ConnProvider, ctx context.Context) error {
			_, err := FreezeBalance2(cp, ctx, &core.FreezeBalanceContract{})
			return err
		}},
		{"UnfreezeBalance2", "UnfreezeBalance2", "unfreeze balance2", func(cp ConnProvider, ctx context.Context) error {
			_, err := UnfreezeBalance2(cp, ctx, &core.UnfreezeBalanceContract{})
			return err
		}},
	})
}
