// package lowlevel provides 1:1 wrappers around WalletClient gRPC methods
package lowlevel

import (
	"context"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
)

// Proposal related gRPC calls

// ProposalCreate creates a governance proposal
func ProposalCreate(cp ConnProvider, ctx context.Context, req *core.ProposalCreateContract) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "proposal create", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.ProposalCreate(ctx, req)
	})
}

// ProposalApprove approves a governance proposal
func ProposalApprove(cp ConnProvider, ctx context.Context, req *core.ProposalApproveContract) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "proposal approve", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.ProposalApprove(ctx, req)
	})
}

// ProposalDelete deletes a governance proposal
func ProposalDelete(cp ConnProvider, ctx context.Context, req *core.ProposalDeleteContract) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "proposal delete", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.ProposalDelete(ctx, req)
	})
}

// ListProposals lists proposals
func ListProposals(cp ConnProvider, ctx context.Context, req *api.EmptyMessage) (*api.ProposalList, error) {
	return Call(cp, ctx, "list proposals", func(cl api.WalletClient, ctx context.Context) (*api.ProposalList, error) {
		return cl.ListProposals(ctx, req)
	})
}

// GetPaginatedProposalList gets a paginated list of proposals
func GetPaginatedProposalList(cp ConnProvider, ctx context.Context, req *api.PaginatedMessage) (*api.ProposalList, error) {
	return Call(cp, ctx, "get paginated proposal list", func(cl api.WalletClient, ctx context.Context) (*api.ProposalList, error) {
		return cl.GetPaginatedProposalList(ctx, req)
	})
}

// GetProposalById gets a proposal by ID
func GetProposalById(cp ConnProvider, ctx context.Context, req *api.BytesMessage) (*core.Proposal, error) {
	return Call(cp, ctx, "get proposal by id", func(cl api.WalletClient, ctx context.Context) (*core.Proposal, error) {
		return cl.GetProposalById(ctx, req)
	})
}
