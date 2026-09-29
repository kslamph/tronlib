package rpc

import (
	"context"
	"testing"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
)

// TestProposalWrappers exercises the 1:1 port of lowlevel/proposal.go through
// the bufconn fake (happy path + node-error path per wrapper).
func TestProposalWrappers(t *testing.T) {
	runWrapperGroup(t, []wrapperCase{
		{"ProposalCreate", "ProposalCreate", "proposal create", func(cp ConnProvider, ctx context.Context) error {
			_, err := ProposalCreate(cp, ctx, &core.ProposalCreateContract{})
			return err
		}},
		{"ProposalApprove", "ProposalApprove", "proposal approve", func(cp ConnProvider, ctx context.Context) error {
			_, err := ProposalApprove(cp, ctx, &core.ProposalApproveContract{})
			return err
		}},
		{"ProposalDelete", "ProposalDelete", "proposal delete", func(cp ConnProvider, ctx context.Context) error {
			_, err := ProposalDelete(cp, ctx, &core.ProposalDeleteContract{})
			return err
		}},
		{"ListProposals", "ListProposals", "list proposals", func(cp ConnProvider, ctx context.Context) error {
			_, err := ListProposals(cp, ctx, &api.EmptyMessage{})
			return err
		}},
		{"GetPaginatedProposalList", "GetPaginatedProposalList", "get paginated proposal list", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetPaginatedProposalList(cp, ctx, &api.PaginatedMessage{})
			return err
		}},
		{"GetProposalById", "GetProposalById", "get proposal by id", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetProposalById(cp, ctx, &api.BytesMessage{})
			return err
		}},
	})
}
