package client

import (
	"github.com/kslamph/tronlib/pkg/account"
	"github.com/kslamph/tronlib/pkg/network"
	"github.com/kslamph/tronlib/pkg/resources"
	"github.com/kslamph/tronlib/pkg/smartcontract"
	"github.com/kslamph/tronlib/pkg/trc10"
	"github.com/kslamph/tronlib/pkg/trc20"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/kslamph/tronlib/pkg/voting"
)

// Account is the gateway method to access the AccountManager.
// It returns an *account.AccountManager, satisfying the high-level API need.
func (c *Client) Account() *account.AccountManager {
	// gRPCClient interface in the account package, so it can be passed directly.
	return account.NewManager(c)
}

// SmartContract is the gateway method to access the Manager.
func (c *Client) SmartContract() *smartcontract.Manager {
	return smartcontract.NewManager(c)
}

// ContractInstance constructs a contract instance for the given address using the
// provided TRON client. The ABI can be omitted to fetch from the network, or
// supplied as either a JSON string or a *core.SmartContract_ABI.
func (c *Client) ContractInstance(contractAddress *types.Address, abi any) (*smartcontract.Instance, error) {
	return smartcontract.NewInstance(c, contractAddress, abi)
}

// TRC20 returns a TRC20 manager for a given token address.
func (c *Client) TRC20(addr *types.Address) *trc20.TRC20Manager {
	trc20mgr, err := trc20.NewManager(c, addr)
	if err != nil {
		return nil
	}
	return trc20mgr
}

// Network returns the high-level NetworkManager.
func (c *Client) Network() *network.NetworkManager {
	return network.NewManager(c)
}

// Resources returns the high-level ResourcesManager.
func (c *Client) Resources() *resources.ResourcesManager {
	return resources.NewManager(c)
}

// TRC10 returns the high-level TRC10Manager.
func (c *Client) TRC10() *trc10.TRC10Manager {
	return trc10.NewManager(c)
}

// Voting returns the high-level VotingManager.
func (c *Client) Voting() *voting.VotingManager {
	return voting.NewManager(c)
}
