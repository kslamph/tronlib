package client

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/smartcontract"
	"github.com/kslamph/tronlib/pkg/types"
	"google.golang.org/grpc"
)

func (c *Client) NewContractFromAddress(address *types.Address) (*smartcontract.Contract, error) {
	if address == nil {
		return nil, fmt.Errorf("failed to get contract: contract address is nil")
	}

	result, err := c.ExecuteWithClient(func(ctx context.Context, conn *grpc.ClientConn) (interface{}, error) {
		walletClient := api.NewWalletClient(conn)
		return walletClient.GetContract(ctx, &api.BytesMessage{
			Value: address.Bytes(),
		})
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get contract: %v", err)
	}

	contract := result.(*core.SmartContract)

	if contract.Abi == nil {
		return nil, fmt.Errorf("failed to get contract: contract ABI is nil")
	}

	return smartcontract.NewContractFromABI(contract.Abi, address.String())
}

func (c *Client) TriggerConstantSmartContract(contract *smartcontract.Contract, ownerAddress *types.Address, data []byte) ([][]byte, error) {
	fmt.Printf("TriggerConstantSmartContract: %v\n", contract)
	// Create trigger smart contract message
	trigger := &core.TriggerSmartContract{
		OwnerAddress:    ownerAddress.Bytes(),
		ContractAddress: contract.AddressBytes,
		Data:            data,
	}

	fmt.Printf("TriggerSmartContract: %v\n", trigger)

	result, err := c.ExecuteWithClient(func(ctx context.Context, conn *grpc.ClientConn) (interface{}, error) {
		walletClient := api.NewWalletClient(conn)
		return walletClient.TriggerConstantContract(ctx, trigger)
	})
	// Call BuildTransaction to get TransactionExtention
	if err != nil {
		return nil, err
	}
	txExt := result.(*api.TransactionExtention)

	return txExt.GetConstantResult(), nil

}
