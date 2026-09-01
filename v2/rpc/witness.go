package rpc

import (
	"context"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/v2/tron"
)

// Voting and witness related gRPC calls (1:1 port of lowlevel/witness.go,
// plus the v4.8.2 GetPaginatedNowWitnessList addition in both Wallet and
// WalletSolidity variants — v1 has no wrapper for it).

// VoteWitnessAccount2 votes for witnesses (v2 - preferred)
func VoteWitnessAccount2(cp ConnProvider, ctx context.Context, req *core.VoteWitnessContract) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "vote witness account2", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.VoteWitnessAccount2(ctx, req)
	})
}

// WithdrawBalance2 withdraws balance (claim rewards) (v2 - preferred)
func WithdrawBalance2(cp ConnProvider, ctx context.Context, req *core.WithdrawBalanceContract) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "withdraw balance2", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.WithdrawBalance2(ctx, req)
	})
}

// CreateWitness2 creates a witness (v2 - preferred)
func CreateWitness2(cp ConnProvider, ctx context.Context, req *core.WitnessCreateContract) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "create witness2", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.CreateWitness2(ctx, req)
	})
}

// UpdateWitness2 updates witness information (v2 - preferred)
func UpdateWitness2(cp ConnProvider, ctx context.Context, req *core.WitnessUpdateContract) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "update witness2", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.UpdateWitness2(ctx, req)
	})
}

// ListWitnesses gets list of witnesses
func ListWitnesses(cp ConnProvider, ctx context.Context, req *api.EmptyMessage) (*api.WitnessList, error) {
	return Call(cp, ctx, "list witnesses", func(cl api.WalletClient, ctx context.Context) (*api.WitnessList, error) {
		return cl.ListWitnesses(ctx, req)
	})
}

// GetRewardInfo gets reward information
func GetRewardInfo(cp ConnProvider, ctx context.Context, req *api.BytesMessage) (*api.NumberMessage, error) {
	return Call(cp, ctx, "get reward info", func(cl api.WalletClient, ctx context.Context) (*api.NumberMessage, error) {
		return cl.GetRewardInfo(ctx, req)
	})
}

// GetBrokerageInfo gets brokerage information
func GetBrokerageInfo(cp ConnProvider, ctx context.Context, req *api.BytesMessage) (*api.NumberMessage, error) {
	return Call(cp, ctx, "get brokerage info", func(cl api.WalletClient, ctx context.Context) (*api.NumberMessage, error) {
		return cl.GetBrokerageInfo(ctx, req)
	})
}

// UpdateBrokerage updates brokerage
func UpdateBrokerage(cp ConnProvider, ctx context.Context, req *core.UpdateBrokerageContract) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "update brokerage", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.UpdateBrokerage(ctx, req)
	})
}

// GetPaginatedNowWitnessList gets a page of the current witness list
// (WalletFullNode; new RPC introduced in java-tron v4.8.2, absent from v1).
func GetPaginatedNowWitnessList(cp ConnProvider, ctx context.Context, req *api.PaginatedMessage) (*api.WitnessList, error) {
	return Call(cp, ctx, "get paginated now witness list", func(cl api.WalletClient, ctx context.Context) (*api.WitnessList, error) {
		return cl.GetPaginatedNowWitnessList(ctx, req)
	})
}

// GetPaginatedNowWitnessListSolidity gets a page of the current witness list
// from the solidity node (WalletSolidity variant of the v4.8.2 RPC).
func GetPaginatedNowWitnessListSolidity(cp ConnProvider, ctx context.Context, req *api.PaginatedMessage) (*api.WitnessList, error) {
	return callSolidity(cp, ctx, "get paginated now witness list", func(cl api.WalletSolidityClient, ctx context.Context) (*api.WitnessList, error) {
		return cl.GetPaginatedNowWitnessList(ctx, req)
	})
}

// Witnesses returns one page of the current witness list (the
// GetPaginatedNowWitnessList wrapper, decoded onto the value shape the
// facade's spec §10.1 Witness declares: address, vote count, isJobs).
// offset/limit pass through to the node's PaginatedMessage unchanged;
// limit 0 means the node's rpc default, never "all".
// Facade-facing convenience; added for Task 9.
func Witnesses(cp ConnProvider, ctx context.Context, offset, limit int64) ([]Witness, error) {
	wl, err := GetPaginatedNowWitnessList(cp, ctx, &api.PaginatedMessage{Offset: offset, Limit: limit})
	if err != nil {
		return nil, err
	}
	out := make([]Witness, 0, len(wl.GetWitnesses()))
	for _, w := range wl.GetWitnesses() {
		addr, err := tron.AddressFromBytes(w.GetAddress())
		if err != nil {
			return nil, &tron.Error{Code: tron.CodeAddressInvalid, Op: "rpc.Witnesses", Cause: err,
				Hint: "the node returned a witness whose address is not a 0x41-prefixed 21-byte value"}
		}
		out = append(out, Witness{Address: addr, VoteCount: w.GetVoteCount(), IsJobs: w.GetIsJobs()})
	}
	return out, nil
}

// Witness is one super-representative candidate as the facade's Witnesses
// page returns it (spec §10.1 shape; a decoded view of core.Witness).
type Witness struct {
	// Address is the witness's TRON address.
	Address tron.Address
	// VoteCount is the total votes the witness currently holds.
	VoteCount int64
	// IsJobs reports whether the witness is currently producing blocks.
	IsJobs bool
}

// callSolidity mirrors Call for the WalletSolidity service: identical
// connection lifecycle, timeout handling and error-code mapping, but backed
// by api.NewWalletSolidityClient. It lives beside the solidity wrappers that
// use it (further solidity-variant wrappers land with later tasks).
func callSolidity[T any](cp ConnProvider, ctx context.Context, operation string, call func(client api.WalletSolidityClient, ctx context.Context) (T, error)) (T, error) {
	var zero T

	conn, err := cp.GetConnection(ctx)
	if err != nil {
		return zero, classifyConnError(operation, err)
	}
	defer func() {
		if conn != nil {
			cp.ReturnConnection(conn)
		}
	}()

	cl := api.NewWalletSolidityClient(conn)

	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cp.GetTimeout())
		defer cancel()
	}

	result, err := call(cl, ctx)
	if err != nil {
		// Same mapping as Call: mid-call deadline/cancel is chain.timeout,
		// every other failure is rpc.method_failed.
		return zero, mapCallError(operation, err)
	}
	return result, nil
}
