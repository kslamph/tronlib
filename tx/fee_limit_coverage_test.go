package tx

// Which builders stamp the DefaultFeeLimit default, and which leave the
// node's value alone. architecture.md §6.4 documents this split, and this
// test is what keeps the document honest: the four core builders floor a
// zero fee_limit (they buy energy, and fee_limit = 0 cannot purchase any),
// while the twelve native-operation builders make no energy purchase and
// never touch the field.
//
// A new builder that falls on the other side of this line fails here, and the
// failure asks for a decision rather than passing silently.

import (
	"context"
	"testing"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/rpc"
	"github.com/kslamph/tronlib/v2/tron"
)

// stampedBuilders are the builders that must carry DefaultFeeLimit even when
// the node answers with fee_limit 0.
var stampedBuilders = map[string]func(t *testing.T, cp rpc.ConnProvider) Tx{
	"BuildTransfer": func(t *testing.T, cp rpc.ConnProvider) Tx {
		tx, err := BuildTransfer(t.Context(), cp, testFrom, testTo, 1_000_000)
		if err != nil {
			t.Fatalf("BuildTransfer: %v", err)
		}
		return tx
	},
	"BuildTriggerSmartContract": func(t *testing.T, cp rpc.ConnProvider) Tx {
		tx, err := BuildTriggerSmartContract(t.Context(), cp, testFrom, testTo, []byte{0x01}, 0)
		if err != nil {
			t.Fatalf("BuildTriggerSmartContract: %v", err)
		}
		return tx
	},
	"BuildDeploy": func(t *testing.T, cp rpc.ConnProvider) Tx {
		tx, err := BuildDeploy(t.Context(), cp, testFrom, DeployParams{Bytecode: []byte{0x60}})
		if err != nil {
			t.Fatalf("BuildDeploy: %v", err)
		}
		return tx
	},
	"BuildAssetTransfer": func(t *testing.T, cp rpc.ConnProvider) Tx {
		tx, err := BuildAssetTransfer(t.Context(), cp, testFrom, testTo, "1000001", 1)
		if err != nil {
			t.Fatalf("BuildAssetTransfer: %v", err)
		}
		return tx
	},
}

// untouchedBuilders make no energy purchase, so a fee-limit ceiling has
// nothing to bound: they pass the node's value through, and the fake reports
// 0 — the value a real CreateTransaction2-family RPC commonly answers with.
var untouchedBuilders = map[string]func(t *testing.T, cp rpc.ConnProvider) Tx{
	"BuildVoteWitness": func(t *testing.T, cp rpc.ConnProvider) Tx {
		tx, err := BuildVoteWitness(t.Context(), cp, testFrom, []Vote{{Witness: testTo, Count: 10}})
		if err != nil {
			t.Fatalf("BuildVoteWitness: %v", err)
		}
		return tx
	},
	"BuildWithdrawRewards": func(t *testing.T, cp rpc.ConnProvider) Tx {
		tx, err := BuildWithdrawRewards(t.Context(), cp, testFrom)
		if err != nil {
			t.Fatalf("BuildWithdrawRewards: %v", err)
		}
		return tx
	},
	"BuildFreezeBalanceV2": func(t *testing.T, cp rpc.ConnProvider) Tx {
		tx, err := BuildFreezeBalanceV2(t.Context(), cp, testFrom, ResourceEnergy, 1_000_000)
		if err != nil {
			t.Fatalf("BuildFreezeBalanceV2: %v", err)
		}
		return tx
	},
	"BuildUnfreezeBalanceV2": func(t *testing.T, cp rpc.ConnProvider) Tx {
		tx, err := BuildUnfreezeBalanceV2(t.Context(), cp, testFrom, ResourceEnergy, 1_000_000)
		if err != nil {
			t.Fatalf("BuildUnfreezeBalanceV2: %v", err)
		}
		return tx
	},
	"BuildWithdrawExpireUnfreeze": func(t *testing.T, cp rpc.ConnProvider) Tx {
		tx, err := BuildWithdrawExpireUnfreeze(t.Context(), cp, testFrom)
		if err != nil {
			t.Fatalf("BuildWithdrawExpireUnfreeze: %v", err)
		}
		return tx
	},
	"BuildCancelAllUnfreezeV2": func(t *testing.T, cp rpc.ConnProvider) Tx {
		tx, err := BuildCancelAllUnfreezeV2(t.Context(), cp, testFrom)
		if err != nil {
			t.Fatalf("BuildCancelAllUnfreezeV2: %v", err)
		}
		return tx
	},
	"BuildDelegateResource": func(t *testing.T, cp rpc.ConnProvider) Tx {
		tx, err := BuildDelegateResource(t.Context(), cp, testFrom, testTo, ResourceEnergy, 1_000_000, DelegateOptions{})
		if err != nil {
			t.Fatalf("BuildDelegateResource: %v", err)
		}
		return tx
	},
	"BuildUnDelegateResource": func(t *testing.T, cp rpc.ConnProvider) Tx {
		tx, err := BuildUnDelegateResource(t.Context(), cp, testFrom, testTo, ResourceEnergy, 1_000_000)
		if err != nil {
			t.Fatalf("BuildUnDelegateResource: %v", err)
		}
		return tx
	},
	"BuildUpdateSetting": func(t *testing.T, cp rpc.ConnProvider) Tx {
		tx, err := BuildUpdateSetting(t.Context(), cp, testFrom, testTo, 100)
		if err != nil {
			t.Fatalf("BuildUpdateSetting: %v", err)
		}
		return tx
	},
	"BuildUpdateEnergyLimit": func(t *testing.T, cp rpc.ConnProvider) Tx {
		tx, err := BuildUpdateEnergyLimit(t.Context(), cp, testFrom, testTo, 1_000_000)
		if err != nil {
			t.Fatalf("BuildUpdateEnergyLimit: %v", err)
		}
		return tx
	},
	"BuildClearABI": func(t *testing.T, cp rpc.ConnProvider) Tx {
		tx, err := BuildClearABI(t.Context(), cp, testFrom, testTo)
		if err != nil {
			t.Fatalf("BuildClearABI: %v", err)
		}
		return tx
	},
	"BuildAccountPermissionUpdate": func(t *testing.T, cp rpc.ConnProvider) Tx {
		bitmap, err := OperationsBitmap(TypeTransfer)
		if err != nil {
			t.Fatalf("OperationsBitmap: %v", err)
		}
		tx, err := BuildAccountPermissionUpdate(t.Context(), cp, testFrom, twoOfThree(t, bitmap))
		if err != nil {
			t.Fatalf("BuildAccountPermissionUpdate: %v", err)
		}
		return tx
	},
}

// TestFeeLimitDefaultCoverage pins the §6.4 split. The fake reports
// fee_limit 0 (see baseExt), which is exactly the value that cannot purchase
// energy — the case the default exists for.
func TestFeeLimitDefaultCoverage(t *testing.T) {
	cp := newTxTestClient(t, &fakeWalletServer{})
	for name, build := range stampedBuilders {
		t.Run(name+"/stamps the default", func(t *testing.T) {
			if got := build(t, cp).FeeLimit(); got != DefaultFeeLimit {
				t.Errorf("FeeLimit = %d, want the %d default (a fee_limit of 0 cannot purchase energy)", got, DefaultFeeLimit)
			}
		})
	}
	for name, build := range untouchedBuilders {
		t.Run(name+"/leaves the node's value", func(t *testing.T) {
			if got := build(t, cp).FeeLimit(); got != 0 {
				t.Errorf("FeeLimit = %d, want the node's 0 passed through: this builder buys no energy, so it must not stamp a default (see architecture.md §6.4)", got)
			}
		})
	}
}

// TestEveryBuilderIsClassified is the completeness half: a builder added to
// the package but to neither table would go untested, and the §6.4 claim
// would silently become false again — which is how the previous version of
// that claim ("stamped by every builder") came to be wrong.
func TestEveryBuilderIsClassified(t *testing.T) {
	// The builders this file knows about, by their RPC-facing constructor
	// name. Reflection cannot see them (they are package functions, not
	// methods), so the check is against the source of truth in §6.4: a
	// name that appears in neither table is a builder whose fee_limit
	// behaviour is unspecified.
	known := map[string]bool{}
	for name := range stampedBuilders {
		known[name] = true
	}
	for name := range untouchedBuilders {
		if known[name] {
			t.Errorf("%s is listed in both tables; a builder stamps the default or it does not", name)
		}
		known[name] = true
	}
	for _, name := range allBuilderNames {
		if !known[name] {
			t.Errorf("builder %s is in neither table: add it to stampedBuilders or untouchedBuilders and record the decision in architecture.md §6.4", name)
		}
	}
	// The reverse: a table entry that is not a builder at all.
	for name := range known {
		if !contains(allBuilderNames, name) {
			t.Errorf("table entry %s is not a builder in this package", name)
		}
	}
}

// allBuilderNames is the package's exported Build* constructors. It is the
// list the completeness check walks; adding a builder means adding it here,
// which is the point — the check then refuses to pass until the new builder
// has been classified.
var allBuilderNames = []string{
	"BuildTransfer", "BuildTriggerSmartContract", "BuildDeploy", "BuildAssetTransfer",
	"BuildVoteWitness", "BuildWithdrawRewards",
	"BuildFreezeBalanceV2", "BuildUnfreezeBalanceV2", "BuildWithdrawExpireUnfreeze",
	"BuildCancelAllUnfreezeV2", "BuildDelegateResource", "BuildUnDelegateResource",
	"BuildUpdateSetting", "BuildUpdateEnergyLimit", "BuildClearABI",
	"BuildAccountPermissionUpdate",
}

func contains(list []string, want string) bool {
	for _, got := range list {
		if got == want {
			return true
		}
	}
	return false
}

// TestStampedBuildersFloorOnlyZero pins the flooring rule itself: a non-zero
// fee_limit from the node is passed through, not overwritten. A builder that
// overwrote a node-supplied cap would silently raise a user's chosen ceiling
// back to 150 TRX.
func TestStampedBuildersFloorOnlyZero(t *testing.T) {
	const nodeValue = 42_000_000
	cp := newTxTestClient(t, &fakeWalletServer{
		CreateTx2: func(_ context.Context, _ *core.TransferContract) (*api.TransactionExtention, error) {
			ext := transferExt()
			ext.GetTransaction().GetRawData().FeeLimit = nodeValue
			return ext, nil
		},
	})
	tx, err := BuildTransfer(t.Context(), cp, testFrom, testTo, 1_000_000)
	if err != nil {
		t.Fatalf("BuildTransfer: %v", err)
	}
	if got := tx.FeeLimit(); got != tron.SUN(nodeValue) {
		t.Errorf("FeeLimit = %d, want the node's %d passed through; the default is a floor, not an overwrite", got, nodeValue)
	}
}
