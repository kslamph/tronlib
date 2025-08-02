// package client provides 1:1 wrappers around WalletClient gRPC methods
package client

import (
	"context"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
)

// Proposal related gRPC calls

// ProposalCreate creates a new proposal
func ProposalCreate(c *client.Client, ctx context.Context, req *core.ProposalCreateContract) (*api.TransactionExtention, error) {
	return grpcTransactionCallWrapper(c, ctx, "proposal create", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.ProposalCreate(ctx, req)
	})
}

// ProposalApprove approves a proposal
func ProposalApprove(c *client.Client, ctx context.Context, req *core.ProposalApproveContract) (*api.TransactionExtention, error) {
	return grpcTransactionCallWrapper(c, ctx, "proposal approve", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.ProposalApprove(ctx, req)
	})
}

// ProposalDelete deletes a proposal
func ProposalDelete(c *client.Client, ctx context.Context, req *core.ProposalDeleteContract) (*api.TransactionExtention, error) {
	return grpcTransactionCallWrapper(c, ctx, "proposal delete", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.ProposalDelete(ctx, req)
	})
}

// ListProposals gets all proposals
func ListProposals(c *client.Client, ctx context.Context, req *api.EmptyMessage) (*api.ProposalList, error) {
	return grpcGenericCallWrapper(c, ctx, "list proposals", func(client api.WalletClient, ctx context.Context) (*api.ProposalList, error) {
		return client.ListProposals(ctx, req)
	})
}

// GetPaginatedProposalList gets paginated proposal list
func GetPaginatedProposalList(c *client.Client, ctx context.Context, req *api.PaginatedMessage) (*api.ProposalList, error) {
	return grpcGenericCallWrapper(c, ctx, "get paginated proposal list", func(client api.WalletClient, ctx context.Context) (*api.ProposalList, error) {
		return client.GetPaginatedProposalList(ctx, req)
	})
}

// GetProposalById gets proposal by ID
func GetProposalById(c *client.Client, ctx context.Context, req *api.BytesMessage) (*core.Proposal, error) {
	return grpcGenericCallWrapper(c, ctx, "get proposal by id", func(client api.WalletClient, ctx context.Context) (*core.Proposal, error) {
		return client.GetProposalById(ctx, req)
	})
}
