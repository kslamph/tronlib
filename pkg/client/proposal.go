// Copyright (c) 2025 github.com/kslamph
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

// package client provides 1:1 wrappers around WalletClient gRPC methods
package client

import (
	"context"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
)

// Proposal related gRPC calls

// ProposalCreate creates a governance proposal
func (c *Client) ProposalCreate(ctx context.Context, req *core.ProposalCreateContract) (*api.TransactionExtention, error) {
	return c.grpcTransactionCallWrapper(ctx, "proposal create", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.ProposalCreate(ctx, req)
	})
}

// ProposalApprove approves a governance proposal
func (c *Client) ProposalApprove(ctx context.Context, req *core.ProposalApproveContract) (*api.TransactionExtention, error) {
	return c.grpcTransactionCallWrapper(ctx, "proposal approve", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.ProposalApprove(ctx, req)
	})
}

// ProposalDelete deletes a governance proposal
func (c *Client) ProposalDelete(ctx context.Context, req *core.ProposalDeleteContract) (*api.TransactionExtention, error) {
	return c.grpcTransactionCallWrapper(ctx, "proposal delete", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.ProposalDelete(ctx, req)
	})
}

// ListProposals lists proposals
func (c *Client) ListProposals(ctx context.Context, req *api.EmptyMessage) (*api.ProposalList, error) {
	return grpcGenericCallWrapper(c, ctx, "list proposals", func(cl api.WalletClient, ctx context.Context) (*api.ProposalList, error) {
		return cl.ListProposals(ctx, req)
	})
}

// GetPaginatedProposalList gets a paginated list of proposals
func (c *Client) GetPaginatedProposalList(ctx context.Context, req *api.PaginatedMessage) (*api.ProposalList, error) {
	return grpcGenericCallWrapper(c, ctx, "get paginated proposal list", func(cl api.WalletClient, ctx context.Context) (*api.ProposalList, error) {
		return cl.GetPaginatedProposalList(ctx, req)
	})
}

// GetProposalById gets a proposal by ID
func (c *Client) GetProposalById(ctx context.Context, req *api.BytesMessage) (*core.Proposal, error) {
	return grpcGenericCallWrapper(c, ctx, "get proposal by id", func(cl api.WalletClient, ctx context.Context) (*core.Proposal, error) {
		return cl.GetProposalById(ctx, req)
	})
}
