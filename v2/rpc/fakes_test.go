package rpc

// Bufconn fake WalletServer — ported from v1 pkg/client/test_fakes_test.go and
// internal/testutil. This is Phase 2's hermetic-test foundation: Tasks 4-10
// build their tests on this fake instead of a live node.

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

// okReturn is the success api.Return used by fake methods returning a Return
// directly (BroadcastTransaction) or nested in a TransactionExtention.
var okReturn = &api.Return{Result: true, Code: api.Return_SUCCESS}

// okExtention is the default success TransactionExtention returned by the
// TxCall-style fake methods (ValidateTransactionResult requires a non-nil
// Result with Result == true).
func okExtention() *api.TransactionExtention {
	return &api.TransactionExtention{Result: okReturn}
}

// testWalletServer is a fake implementing api.WalletServer for unit tests.
// It covers every RPC wrapped by the wallet-core wrappers (block, witness,
// account, asset, transaction groups) via the generic dispatch mechanism:
// a test sets Handlers[method] to customize behavior; unset methods fall back
// to the canned defaults in defaultResponses.
type testWalletServer struct {
	api.UnimplementedWalletServer

	// Handlers can be set per test keyed by RPC method name (e.g.
	// "GetNowBlock2") to override the default canned response.
	Handlers map[string]func(ctx context.Context, in any) (any, error)

	// Typed per-test handlers (Task 3 client tests and Task 6-8 contract/
	// token tests predate the generic dispatch mechanism and use these).
	BroadcastHandler            func(ctx context.Context, in *core.Transaction) (*api.Return, error)
	GetNowBlockHandler          func(ctx context.Context, in *api.EmptyMessage) (*core.Block, error)
	GetBlockByNumHandler        func(ctx context.Context, in *api.NumberMessage) (*core.Block, error)
	TriggerConstantContractFunc func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error)
	GetTxInfoByIdHandler        func(ctx context.Context, in *api.BytesMessage) (*core.TransactionInfo, error)
}

// dispatch routes a fake RPC to the per-test handler if one is registered,
// otherwise to the canned default. It panics on a method with no default
// response, which is a fake wiring bug (every wrapped RPC has one).
func (s *testWalletServer) dispatch(ctx context.Context, method string, in any) (any, error) {
	if h, ok := s.Handlers[method]; ok && h != nil {
		return h(ctx, in)
	}
	// Legacy typed handlers take precedence over the defaults.
	switch method {
	case "BroadcastTransaction":
		if s.BroadcastHandler != nil {
			return s.BroadcastHandler(ctx, in.(*core.Transaction))
		}
	case "GetNowBlock":
		if s.GetNowBlockHandler != nil {
			return s.GetNowBlockHandler(ctx, in.(*api.EmptyMessage))
		}
	case "GetBlockByNum":
		if s.GetBlockByNumHandler != nil {
			return s.GetBlockByNumHandler(ctx, in.(*api.NumberMessage))
		}
	case "TriggerConstantContract":
		if s.TriggerConstantContractFunc != nil {
			return s.TriggerConstantContractFunc(ctx, in.(*core.TriggerSmartContract))
		}
	case "GetTransactionInfoById":
		if s.GetTxInfoByIdHandler != nil {
			return s.GetTxInfoByIdHandler(ctx, in.(*api.BytesMessage))
		}
	}
	if def, ok := defaultResponses[method]; ok {
		return def, nil
	}
	panic("testWalletServer: no handler or default response for " + method)
}

// defaultResponses holds one canned success response per wrapped RPC method.
// TxCall-style methods carry a success Result so ValidateTransactionResult
// passes; a test that needs a node-level failure overrides via Handlers.
var defaultResponses = map[string]any{
	// Task 3 client-test RPCs
	"GetNowBlock":             &core.Block{},
	"GetBlockByNum":           &core.Block{},
	"TriggerConstantContract": okExtention(),
	// account
	"GetAccount":              &core.Account{},
	"GetAccountById":          &core.Account{},
	"GetAccountBalance":       &core.AccountBalanceResponse{},
	"GetBlockBalanceTrace":    &core.BlockBalanceTrace{},
	"CreateAccount2":          okExtention(),
	"UpdateAccount2":          okExtention(),
	"SetAccountId":            &core.Transaction{},
	"AccountPermissionUpdate": okExtention(),
	"GetAccountNet":           &api.AccountNetMessage{},
	"GetAccountResource":      &api.AccountResourceMessage{},
	// asset
	"CreateAssetIssue2":          okExtention(),
	"UpdateAsset2":               okExtention(),
	"TransferAsset2":             okExtention(),
	"ParticipateAssetIssue2":     okExtention(),
	"UnfreezeAsset2":             okExtention(),
	"GetAssetIssueByAccount":     &api.AssetIssueList{},
	"GetAssetIssueByName":        &core.AssetIssueContract{},
	"GetAssetIssueListByName":    &api.AssetIssueList{},
	"GetAssetIssueById":          &core.AssetIssueContract{},
	"GetAssetIssueList":          &api.AssetIssueList{},
	"GetPaginatedAssetIssueList": &api.AssetIssueList{},
	// block
	"GetNowBlock2":                 &api.BlockExtention{},
	"GetBlockByNum2":               &api.BlockExtention{},
	"GetBlockById":                 &core.Block{},
	"GetBlockByLimitNext2":         &api.BlockListExtention{},
	"GetBlockByLatestNum2":         &api.BlockListExtention{},
	"GetTransactionInfoByBlockNum": &api.TransactionInfoList{},
	"ListNodes":                    &api.NodeList{},
	"GetNodeInfo":                  &core.NodeInfo{},
	"GetChainParameters":           &core.ChainParameters{},
	"GetBandwidthPrices":           &api.PricesResponseMessage{},
	"GetEnergyPrices":              &api.PricesResponseMessage{},
	"GetMemoFee":                   &api.PricesResponseMessage{},
	"GetNextMaintenanceTime":       &api.NumberMessage{},
	"TotalTransaction":             &api.NumberMessage{},
	"GetBurnTrx":                   &api.NumberMessage{},
	"GetBlock":                     &api.BlockExtention{},
	// transaction
	"CreateTransaction2":            okExtention(),
	"BroadcastTransaction":          okReturn,
	"GetTransactionById":            &core.Transaction{},
	"GetTransactionInfoById":        &core.TransactionInfo{},
	"GetTransactionCountByBlockNum": &api.NumberMessage{},
	"GetTransactionSignWeight":      &api.TransactionSignWeight{},
	"GetTransactionApprovedList":    &api.TransactionApprovedList{},
	"CreateCommonTransaction":       okExtention(),
	"GetTransactionFromPending":     &core.Transaction{},
	"GetTransactionListFromPending": &api.TransactionIdList{},
	"GetPendingSize":                &api.NumberMessage{},
	// witness
	"VoteWitnessAccount2":        okExtention(),
	"WithdrawBalance2":           okExtention(),
	"CreateWitness2":             okExtention(),
	"UpdateWitness2":             okExtention(),
	"ListWitnesses":              &api.WitnessList{},
	"GetRewardInfo":              &api.NumberMessage{},
	"GetBrokerageInfo":           &api.NumberMessage{},
	"UpdateBrokerage":            okExtention(),
	"GetPaginatedNowWitnessList": &api.WitnessList{},
}

func (s *testWalletServer) BroadcastTransaction(ctx context.Context, in *core.Transaction) (*api.Return, error) {
	v, err := s.dispatch(ctx, "BroadcastTransaction", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.Return), nil
}

func (s *testWalletServer) GetTransactionInfoById(ctx context.Context, in *api.BytesMessage) (*core.TransactionInfo, error) {
	v, err := s.dispatch(ctx, "GetTransactionInfoById", in)
	if err != nil {
		return nil, err
	}
	return v.(*core.TransactionInfo), nil
}

func (s *testWalletServer) GetNowBlock(ctx context.Context, in *api.EmptyMessage) (*core.Block, error) {
	v, err := s.dispatch(ctx, "GetNowBlock", in)
	if err != nil {
		return nil, err
	}
	return v.(*core.Block), nil
}

func (s *testWalletServer) GetBlockByNum(ctx context.Context, in *api.NumberMessage) (*core.Block, error) {
	v, err := s.dispatch(ctx, "GetBlockByNum", in)
	if err != nil {
		return nil, err
	}
	return v.(*core.Block), nil
}

func (s *testWalletServer) TriggerConstantContract(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	if s.TriggerConstantContractFunc != nil {
		return s.TriggerConstantContractFunc(ctx, in)
	}
	return okExtention(), nil
}

// --- Wallet method impls exercised by the wallet-core wrappers ---
// Each delegates to dispatch so Handlers[method] overrides the canned default.

func (s *testWalletServer) GetAccount(ctx context.Context, in *core.Account) (*core.Account, error) {
	v, err := s.dispatch(ctx, "GetAccount", in)
	if err != nil {
		return nil, err
	}
	return v.(*core.Account), nil
}

func (s *testWalletServer) GetAccountById(ctx context.Context, in *core.Account) (*core.Account, error) {
	v, err := s.dispatch(ctx, "GetAccountById", in)
	if err != nil {
		return nil, err
	}
	return v.(*core.Account), nil
}

func (s *testWalletServer) GetAccountBalance(ctx context.Context, in *core.AccountBalanceRequest) (*core.AccountBalanceResponse, error) {
	v, err := s.dispatch(ctx, "GetAccountBalance", in)
	if err != nil {
		return nil, err
	}
	return v.(*core.AccountBalanceResponse), nil
}

func (s *testWalletServer) GetBlockBalanceTrace(ctx context.Context, in *core.BlockBalanceTrace_BlockIdentifier) (*core.BlockBalanceTrace, error) {
	v, err := s.dispatch(ctx, "GetBlockBalanceTrace", in)
	if err != nil {
		return nil, err
	}
	return v.(*core.BlockBalanceTrace), nil
}

func (s *testWalletServer) CreateTransaction2(ctx context.Context, in *core.TransferContract) (*api.TransactionExtention, error) {
	v, err := s.dispatch(ctx, "CreateTransaction2", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.TransactionExtention), nil
}

func (s *testWalletServer) CreateAccount2(ctx context.Context, in *core.AccountCreateContract) (*api.TransactionExtention, error) {
	v, err := s.dispatch(ctx, "CreateAccount2", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.TransactionExtention), nil
}

func (s *testWalletServer) UpdateAccount2(ctx context.Context, in *core.AccountUpdateContract) (*api.TransactionExtention, error) {
	v, err := s.dispatch(ctx, "UpdateAccount2", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.TransactionExtention), nil
}

func (s *testWalletServer) SetAccountId(ctx context.Context, in *core.SetAccountIdContract) (*core.Transaction, error) {
	v, err := s.dispatch(ctx, "SetAccountId", in)
	if err != nil {
		return nil, err
	}
	return v.(*core.Transaction), nil
}

func (s *testWalletServer) AccountPermissionUpdate(ctx context.Context, in *core.AccountPermissionUpdateContract) (*api.TransactionExtention, error) {
	v, err := s.dispatch(ctx, "AccountPermissionUpdate", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.TransactionExtention), nil
}

func (s *testWalletServer) GetAccountNet(ctx context.Context, in *core.Account) (*api.AccountNetMessage, error) {
	v, err := s.dispatch(ctx, "GetAccountNet", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.AccountNetMessage), nil
}

func (s *testWalletServer) GetAccountResource(ctx context.Context, in *core.Account) (*api.AccountResourceMessage, error) {
	v, err := s.dispatch(ctx, "GetAccountResource", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.AccountResourceMessage), nil
}

func (s *testWalletServer) CreateAssetIssue2(ctx context.Context, in *core.AssetIssueContract) (*api.TransactionExtention, error) {
	v, err := s.dispatch(ctx, "CreateAssetIssue2", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.TransactionExtention), nil
}

func (s *testWalletServer) UpdateAsset2(ctx context.Context, in *core.UpdateAssetContract) (*api.TransactionExtention, error) {
	v, err := s.dispatch(ctx, "UpdateAsset2", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.TransactionExtention), nil
}

func (s *testWalletServer) TransferAsset2(ctx context.Context, in *core.TransferAssetContract) (*api.TransactionExtention, error) {
	v, err := s.dispatch(ctx, "TransferAsset2", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.TransactionExtention), nil
}

func (s *testWalletServer) ParticipateAssetIssue2(ctx context.Context, in *core.ParticipateAssetIssueContract) (*api.TransactionExtention, error) {
	v, err := s.dispatch(ctx, "ParticipateAssetIssue2", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.TransactionExtention), nil
}

func (s *testWalletServer) UnfreezeAsset2(ctx context.Context, in *core.UnfreezeAssetContract) (*api.TransactionExtention, error) {
	v, err := s.dispatch(ctx, "UnfreezeAsset2", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.TransactionExtention), nil
}

func (s *testWalletServer) GetAssetIssueByAccount(ctx context.Context, in *core.Account) (*api.AssetIssueList, error) {
	v, err := s.dispatch(ctx, "GetAssetIssueByAccount", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.AssetIssueList), nil
}

func (s *testWalletServer) GetAssetIssueByName(ctx context.Context, in *api.BytesMessage) (*core.AssetIssueContract, error) {
	v, err := s.dispatch(ctx, "GetAssetIssueByName", in)
	if err != nil {
		return nil, err
	}
	return v.(*core.AssetIssueContract), nil
}

func (s *testWalletServer) GetAssetIssueListByName(ctx context.Context, in *api.BytesMessage) (*api.AssetIssueList, error) {
	v, err := s.dispatch(ctx, "GetAssetIssueListByName", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.AssetIssueList), nil
}

func (s *testWalletServer) GetAssetIssueById(ctx context.Context, in *api.BytesMessage) (*core.AssetIssueContract, error) {
	v, err := s.dispatch(ctx, "GetAssetIssueById", in)
	if err != nil {
		return nil, err
	}
	return v.(*core.AssetIssueContract), nil
}

func (s *testWalletServer) GetAssetIssueList(ctx context.Context, in *api.EmptyMessage) (*api.AssetIssueList, error) {
	v, err := s.dispatch(ctx, "GetAssetIssueList", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.AssetIssueList), nil
}

func (s *testWalletServer) GetPaginatedAssetIssueList(ctx context.Context, in *api.PaginatedMessage) (*api.AssetIssueList, error) {
	v, err := s.dispatch(ctx, "GetPaginatedAssetIssueList", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.AssetIssueList), nil
}

func (s *testWalletServer) GetNowBlock2(ctx context.Context, in *api.EmptyMessage) (*api.BlockExtention, error) {
	v, err := s.dispatch(ctx, "GetNowBlock2", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.BlockExtention), nil
}

func (s *testWalletServer) GetBlockByNum2(ctx context.Context, in *api.NumberMessage) (*api.BlockExtention, error) {
	v, err := s.dispatch(ctx, "GetBlockByNum2", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.BlockExtention), nil
}

func (s *testWalletServer) GetBlockById(ctx context.Context, in *api.BytesMessage) (*core.Block, error) {
	v, err := s.dispatch(ctx, "GetBlockById", in)
	if err != nil {
		return nil, err
	}
	return v.(*core.Block), nil
}

func (s *testWalletServer) GetBlockByLimitNext2(ctx context.Context, in *api.BlockLimit) (*api.BlockListExtention, error) {
	v, err := s.dispatch(ctx, "GetBlockByLimitNext2", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.BlockListExtention), nil
}

func (s *testWalletServer) GetBlockByLatestNum2(ctx context.Context, in *api.NumberMessage) (*api.BlockListExtention, error) {
	v, err := s.dispatch(ctx, "GetBlockByLatestNum2", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.BlockListExtention), nil
}

func (s *testWalletServer) GetTransactionInfoByBlockNum(ctx context.Context, in *api.NumberMessage) (*api.TransactionInfoList, error) {
	v, err := s.dispatch(ctx, "GetTransactionInfoByBlockNum", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.TransactionInfoList), nil
}

func (s *testWalletServer) ListNodes(ctx context.Context, in *api.EmptyMessage) (*api.NodeList, error) {
	v, err := s.dispatch(ctx, "ListNodes", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.NodeList), nil
}

func (s *testWalletServer) GetNodeInfo(ctx context.Context, in *api.EmptyMessage) (*core.NodeInfo, error) {
	v, err := s.dispatch(ctx, "GetNodeInfo", in)
	if err != nil {
		return nil, err
	}
	return v.(*core.NodeInfo), nil
}

func (s *testWalletServer) GetChainParameters(ctx context.Context, in *api.EmptyMessage) (*core.ChainParameters, error) {
	v, err := s.dispatch(ctx, "GetChainParameters", in)
	if err != nil {
		return nil, err
	}
	return v.(*core.ChainParameters), nil
}

func (s *testWalletServer) GetBandwidthPrices(ctx context.Context, in *api.EmptyMessage) (*api.PricesResponseMessage, error) {
	v, err := s.dispatch(ctx, "GetBandwidthPrices", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.PricesResponseMessage), nil
}

func (s *testWalletServer) GetEnergyPrices(ctx context.Context, in *api.EmptyMessage) (*api.PricesResponseMessage, error) {
	v, err := s.dispatch(ctx, "GetEnergyPrices", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.PricesResponseMessage), nil
}

func (s *testWalletServer) GetMemoFee(ctx context.Context, in *api.EmptyMessage) (*api.PricesResponseMessage, error) {
	v, err := s.dispatch(ctx, "GetMemoFee", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.PricesResponseMessage), nil
}

func (s *testWalletServer) GetNextMaintenanceTime(ctx context.Context, in *api.EmptyMessage) (*api.NumberMessage, error) {
	v, err := s.dispatch(ctx, "GetNextMaintenanceTime", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.NumberMessage), nil
}

func (s *testWalletServer) TotalTransaction(ctx context.Context, in *api.EmptyMessage) (*api.NumberMessage, error) {
	v, err := s.dispatch(ctx, "TotalTransaction", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.NumberMessage), nil
}

func (s *testWalletServer) GetBurnTrx(ctx context.Context, in *api.EmptyMessage) (*api.NumberMessage, error) {
	v, err := s.dispatch(ctx, "GetBurnTrx", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.NumberMessage), nil
}

func (s *testWalletServer) GetBlock(ctx context.Context, in *api.BlockReq) (*api.BlockExtention, error) {
	v, err := s.dispatch(ctx, "GetBlock", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.BlockExtention), nil
}

func (s *testWalletServer) GetTransactionById(ctx context.Context, in *api.BytesMessage) (*core.Transaction, error) {
	v, err := s.dispatch(ctx, "GetTransactionById", in)
	if err != nil {
		return nil, err
	}
	return v.(*core.Transaction), nil
}

func (s *testWalletServer) GetTransactionCountByBlockNum(ctx context.Context, in *api.NumberMessage) (*api.NumberMessage, error) {
	v, err := s.dispatch(ctx, "GetTransactionCountByBlockNum", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.NumberMessage), nil
}

func (s *testWalletServer) GetTransactionSignWeight(ctx context.Context, in *core.Transaction) (*api.TransactionSignWeight, error) {
	v, err := s.dispatch(ctx, "GetTransactionSignWeight", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.TransactionSignWeight), nil
}

func (s *testWalletServer) GetTransactionApprovedList(ctx context.Context, in *core.Transaction) (*api.TransactionApprovedList, error) {
	v, err := s.dispatch(ctx, "GetTransactionApprovedList", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.TransactionApprovedList), nil
}

func (s *testWalletServer) CreateCommonTransaction(ctx context.Context, in *core.Transaction) (*api.TransactionExtention, error) {
	v, err := s.dispatch(ctx, "CreateCommonTransaction", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.TransactionExtention), nil
}

func (s *testWalletServer) GetTransactionFromPending(ctx context.Context, in *api.BytesMessage) (*core.Transaction, error) {
	v, err := s.dispatch(ctx, "GetTransactionFromPending", in)
	if err != nil {
		return nil, err
	}
	return v.(*core.Transaction), nil
}

func (s *testWalletServer) GetTransactionListFromPending(ctx context.Context, in *api.EmptyMessage) (*api.TransactionIdList, error) {
	v, err := s.dispatch(ctx, "GetTransactionListFromPending", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.TransactionIdList), nil
}

func (s *testWalletServer) GetPendingSize(ctx context.Context, in *api.EmptyMessage) (*api.NumberMessage, error) {
	v, err := s.dispatch(ctx, "GetPendingSize", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.NumberMessage), nil
}

func (s *testWalletServer) VoteWitnessAccount2(ctx context.Context, in *core.VoteWitnessContract) (*api.TransactionExtention, error) {
	v, err := s.dispatch(ctx, "VoteWitnessAccount2", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.TransactionExtention), nil
}

func (s *testWalletServer) WithdrawBalance2(ctx context.Context, in *core.WithdrawBalanceContract) (*api.TransactionExtention, error) {
	v, err := s.dispatch(ctx, "WithdrawBalance2", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.TransactionExtention), nil
}

func (s *testWalletServer) CreateWitness2(ctx context.Context, in *core.WitnessCreateContract) (*api.TransactionExtention, error) {
	v, err := s.dispatch(ctx, "CreateWitness2", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.TransactionExtention), nil
}

func (s *testWalletServer) UpdateWitness2(ctx context.Context, in *core.WitnessUpdateContract) (*api.TransactionExtention, error) {
	v, err := s.dispatch(ctx, "UpdateWitness2", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.TransactionExtention), nil
}

func (s *testWalletServer) ListWitnesses(ctx context.Context, in *api.EmptyMessage) (*api.WitnessList, error) {
	v, err := s.dispatch(ctx, "ListWitnesses", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.WitnessList), nil
}

func (s *testWalletServer) GetRewardInfo(ctx context.Context, in *api.BytesMessage) (*api.NumberMessage, error) {
	v, err := s.dispatch(ctx, "GetRewardInfo", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.NumberMessage), nil
}

func (s *testWalletServer) GetBrokerageInfo(ctx context.Context, in *api.BytesMessage) (*api.NumberMessage, error) {
	v, err := s.dispatch(ctx, "GetBrokerageInfo", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.NumberMessage), nil
}

func (s *testWalletServer) UpdateBrokerage(ctx context.Context, in *core.UpdateBrokerageContract) (*api.TransactionExtention, error) {
	v, err := s.dispatch(ctx, "UpdateBrokerage", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.TransactionExtention), nil
}

func (s *testWalletServer) GetPaginatedNowWitnessList(ctx context.Context, in *api.PaginatedMessage) (*api.WitnessList, error) {
	v, err := s.dispatch(ctx, "GetPaginatedNowWitnessList", in)
	if err != nil {
		return nil, err
	}
	return v.(*api.WitnessList), nil
}

// testWalletSolidityServer serves the WalletSolidity service by delegating to
// the same *testWalletServer handlers, so both services share canned behavior.
type testWalletSolidityServer struct {
	api.UnimplementedWalletSolidityServer
	ws *testWalletServer
}

func (s *testWalletSolidityServer) GetTransactionInfoById(ctx context.Context, in *api.BytesMessage) (*core.TransactionInfo, error) {
	return s.ws.GetTransactionInfoById(ctx, in)
}

func (s *testWalletSolidityServer) GetNowBlock(ctx context.Context, in *api.EmptyMessage) (*core.Block, error) {
	return s.ws.GetNowBlock(ctx, in)
}

func (s *testWalletSolidityServer) GetBlockByNum(ctx context.Context, in *api.NumberMessage) (*core.Block, error) {
	return s.ws.GetBlockByNum(ctx, in)
}

func (s *testWalletSolidityServer) TriggerConstantContract(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	return s.ws.TriggerConstantContract(ctx, in)
}

func (s *testWalletSolidityServer) GetPaginatedNowWitnessList(ctx context.Context, in *api.PaginatedMessage) (*api.WitnessList, error) {
	return s.ws.GetPaginatedNowWitnessList(ctx, in)
}

// newBufconnServer spins up a bufconn-backed gRPC server serving impl and
// registers automatic teardown. When impl is a *testWalletServer, the
// WalletSolidity service is registered on the same server, backed by the same
// handlers (Solidity-side wrappers and WaitForSolid tests hit this).
func newBufconnServer(t *testing.T, impl api.WalletServer) *bufconn.Listener {
	t.Helper()
	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer()
	api.RegisterWalletServer(srv, impl)
	if ws, ok := impl.(*testWalletServer); ok {
		api.RegisterWalletSolidityServer(srv, &testWalletSolidityServer{ws: ws})
	}
	go func() { _ = srv.Serve(lis) }()

	t.Cleanup(func() {
		srv.Stop()
		_ = lis.Close()
	})
	return lis
}

// newBufconnClient creates a *Client whose pool dials the bufconn listener via
// the test-only constructor (bypassing scheme validation for passthrough:///
// addresses). Cleanup closes the client after the test.
func newBufconnClient(t *testing.T, impl api.WalletServer, timeout time.Duration) *Client {
	t.Helper()
	lis := newBufconnServer(t, impl)
	dialer := func(ctx context.Context, _ string) (net.Conn, error) {
		// DialContext honors context cancellation/timeouts.
		return lis.DialContext(ctx)
	}
	c, err := NewClientWithDialer("passthrough:///bufnet", dialer, WithTimeout(timeout), WithPool(1, 2))
	if err != nil {
		t.Fatalf("NewClientWithDialer error: %v", err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}
