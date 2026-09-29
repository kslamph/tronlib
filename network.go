package tronlib

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/rpc"
	"github.com/kslamph/tronlib/v2/tron"
)

// Network is a declared TRON network identity. It is explicit configuration,
// never inferred: TRON has no chain ID, and the 21-byte address prefix is
// 0x41 on Mainnet, Shasta and Nile alike, so an address byte cannot
// discriminate a network (spec §10).
type Network string

const (
	Mainnet Network = "mainnet"
	Shasta  Network = "shasta"
	Nile    Network = "nile"
	Private Network = "private"
)

// genesisID is the recorded block-0 id of each public network, fetched from
// the network's own getblockbynum (spec §2.1). It is data, not a derivation.
var genesisID = map[Network]string{
	Mainnet: "00000000000000001ebf88508a03865c71d452e25f4d51194196a1d22b6653dc",
	Nile:    "0000000000000000d698d4192c56cb6be724a558448e2684802de4d6cd8690dc",
	Shasta:  "0000000000000000de1aa88295e1fcf982742f773e0419c5a9c134c994a9059e",
}

// WithNetwork declares the endpoint's network. The undeclared zero value and
// Private skip verification; a public declaration opts into the
// genesis-fingerprint check (spec §10).
func WithNetwork(n Network) DialOption { return DialOption{network: n} }

// Network returns the network declared at Dial time. It is a pure accessor:
// no I/O, and no inference from the endpoint.
func (c *Client) Network() Network { return c.network }

// VerifyNetwork compares the endpoint's genesis block id against the recorded
// table for the declared network and returns chain.network_mismatch when they
// disagree. It is heuristic: a redeployed testnet changes its genesis and a
// private chain matches nothing, so WithNetwork is the source of truth — this
// only detects a mismatch, it never infers identity (spec §10, risk R7).
//
// The undeclared zero value and Private return nil without a read — there is
// no declaration to contradict. A declared network absent from the table
// fails closed as a mismatch. Dial does not call this automatically: Dial is
// lazy by design, so verification is an explicit call.
func (c *Client) VerifyNetwork(ctx context.Context) error {
	const op = "tronlib.Client.VerifyNetwork"
	n := c.network
	if n == "" || n == Private {
		return nil
	}
	want, ok := genesisID[n]
	if !ok {
		return &tron.Error{
			Code: tron.CodeChainNetworkMismatch,
			Op:   op,
			Hint: fmt.Sprintf("network %q has no recorded genesis fingerprint; declare Mainnet, Shasta, Nile, or Private", n),
		}
	}
	blk, err := rpc.GetBlockByNum2(c.inner, ctx, &api.NumberMessage{Num: 0})
	if err != nil {
		return err
	}
	got := hex.EncodeToString(blk.GetBlockid())
	if got != want {
		return &tron.Error{
			Code: tron.CodeChainNetworkMismatch,
			Op:   op,
			Hint: fmt.Sprintf("configured %s (genesis %s) but the endpoint's genesis is %s; the endpoint is on a different network", n, want, got),
		}
	}
	return nil
}
