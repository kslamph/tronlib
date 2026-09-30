package eventtool

import (
	"context"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/rpc"
	"github.com/kslamph/tronlib/v2/tron"
)

// RPCFetcher resolves a contract's ABI through a node's GetContract call. It is
// the production ABIFetcher; tests use a stub instead of a network.
type RPCFetcher struct {
	cp rpc.ConnProvider
}

// NewRPCFetcher returns an ABIFetcher backed by cp (a *rpc.Client satisfies
// rpc.ConnProvider).
func NewRPCFetcher(cp rpc.ConnProvider) *RPCFetcher {
	return &RPCFetcher{cp: cp}
}

// ABI fetches addr's ABI. A nil ABI is returned as nil, which Capture treats as
// "no usable events" rather than an error.
func (r *RPCFetcher) ABI(ctx context.Context, addr tron.Address) (*core.SmartContract_ABI, error) {
	sc, err := rpc.GetContract(r.cp, ctx, &api.BytesMessage{Value: addr.Bytes()})
	if err != nil {
		return nil, err
	}
	return sc.GetAbi(), nil
}
