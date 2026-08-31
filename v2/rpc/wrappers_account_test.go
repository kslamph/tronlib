package rpc

import (
	"context"
	"testing"

	"github.com/kslamph/tronlib/pb/core"
)

func TestAccountWrappers(t *testing.T) {
	runWrapperGroup(t, []wrapperCase{
		{"GetAccount", "GetAccount", "get account", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetAccount(cp, ctx, &core.Account{})
			return err
		}},
		{"GetAccountById", "GetAccountById", "get account by id", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetAccountById(cp, ctx, &core.Account{})
			return err
		}},
		{"GetAccountNet", "GetAccountNet", "get account net", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetAccountNet(cp, ctx, &core.Account{})
			return err
		}},
		{"GetAccountResource", "GetAccountResource", "get account resource", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetAccountResource(cp, ctx, &core.Account{})
			return err
		}},
		{"CreateAccount2", "CreateAccount2", "create account2", func(cp ConnProvider, ctx context.Context) error {
			_, err := CreateAccount2(cp, ctx, &core.AccountCreateContract{})
			return err
		}},
		{"UpdateAccount2", "UpdateAccount2", "update account2", func(cp ConnProvider, ctx context.Context) error {
			_, err := UpdateAccount2(cp, ctx, &core.AccountUpdateContract{})
			return err
		}},
		{"SetAccountId", "SetAccountId", "set account id", func(cp ConnProvider, ctx context.Context) error {
			_, err := SetAccountId(cp, ctx, &core.SetAccountIdContract{})
			return err
		}},
		{"AccountPermissionUpdate", "AccountPermissionUpdate", "account permission update", func(cp ConnProvider, ctx context.Context) error {
			_, err := AccountPermissionUpdate(cp, ctx, &core.AccountPermissionUpdateContract{})
			return err
		}},
		{"GetAccountBalance", "GetAccountBalance", "get account balance", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetAccountBalance(cp, ctx, &core.AccountBalanceRequest{})
			return err
		}},
		{"GetBlockBalanceTrace", "GetBlockBalanceTrace", "get block balance trace", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetBlockBalanceTrace(cp, ctx, &core.BlockBalanceTrace_BlockIdentifier{})
			return err
		}},
	})
}
