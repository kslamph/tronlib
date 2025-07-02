package client

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"google.golang.org/grpc"
)

var (
	// ErrConnectionFailed is returned when connection to the node fails
	ErrConnectionFailed = errors.New("connection to node failed")
)

// ClientConfig represents the configuration for the TronClient
type ClientConfig struct {
	NodeAddress string        // Single node address
	Timeout     time.Duration // Timeout for RPC calls
}

// Client manages connection to a single Tron node
type Client struct {
	conn        *grpc.ClientConn
	timeout     time.Duration
	nodeAddress string
	mu          sync.Mutex
}

// NewClient creates a new TronClient with the provided configuration (lazy connection)
func NewClient(config ClientConfig) (*Client, error) {
	if config.NodeAddress == "" {
		return nil, errors.New("node address must be provided")
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second // Default timeout
	}

	return &Client{
		conn:        nil,
		timeout:     timeout,
		nodeAddress: config.NodeAddress,
	}, nil
}

// ensureConnection establishes the connection if needed
func (c *Client) ensureConnection() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If already connected, return
	if c.conn != nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			d := net.Dialer{Timeout: 5 * time.Second}
			return d.DialContext(ctx, "tcp", addr)
		}),
	}

	conn, err := grpc.DialContext(ctx, c.nodeAddress, opts...)
	if err != nil {
		return err
	}

	c.conn = conn
	return nil
}

// GetConnection returns the gRPC connection, establishing it if needed
func (c *Client) GetConnection() (*grpc.ClientConn, error) {
	if err := c.ensureConnection(); err != nil {
		return nil, err
	}
	return c.conn, nil
}

// Close closes the connection to the Tron node
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}

// Transaction creation methods

// CreateTransferTransaction creates a TRX transfer transaction
func (c *Client) CreateTransferTransaction(ownerAddress, toAddress []byte, amount int64) (*api.TransactionExtention, error) {
	return c.grpcCallWrapper("transfer", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.CreateTransaction2(ctx, &core.TransferContract{
			OwnerAddress: ownerAddress,
			ToAddress:    toAddress,
			Amount:       amount,
		})
	})
}

// CreateTriggerSmartContractTransaction creates a smart contract trigger transaction
func (c *Client) CreateTriggerSmartContractTransaction(ownerAddress, contractAddress []byte, data []byte, callValue int64) (*api.TransactionExtention, error) {
	return c.grpcCallWrapper("smart contract", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.TriggerContract(ctx, &core.TriggerSmartContract{
			OwnerAddress:    ownerAddress,
			ContractAddress: contractAddress,
			Data:            data,
			CallValue:       callValue,
		})
	})
}

// CreateFreezeTransaction creates a freeze balance transaction
func (c *Client) CreateFreezeTransaction(ownerAddress []byte, amount int64, resource core.ResourceCode) (*api.TransactionExtention, error) {
	return c.grpcCallWrapper("freeze", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.FreezeBalanceV2(ctx, &core.FreezeBalanceV2Contract{
			OwnerAddress:  ownerAddress,
			FrozenBalance: amount,
			Resource:      resource,
		})
	})
}

// CreateUnfreezeTransaction creates an unfreeze balance transaction
func (c *Client) CreateUnfreezeTransaction(ownerAddress []byte, amount int64, resource core.ResourceCode) (*api.TransactionExtention, error) {
	return c.grpcCallWrapper("unfreeze", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.UnfreezeBalanceV2(ctx, &core.UnfreezeBalanceV2Contract{
			OwnerAddress:    ownerAddress,
			UnfreezeBalance: amount,
			Resource:        resource,
		})
	})
}

// CreateDelegateResourceTransaction creates a delegate resource transaction
func (c *Client) CreateDelegateResourceTransaction(ownerAddress, receiverAddress []byte, amount int64, resource core.ResourceCode, lock bool, lockPeriod int64) (*api.TransactionExtention, error) {
	if err := c.ensureConnection(); err != nil {
		return nil, fmt.Errorf("connection error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	client := api.NewWalletClient(c.conn)
	result, err := client.DelegateResource(ctx, &core.DelegateResourceContract{
		OwnerAddress:    ownerAddress,
		ReceiverAddress: receiverAddress,
		Balance:         amount,
		Resource:        resource,
		Lock:            lock,
		LockPeriod:      lockPeriod,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create delegate resource transaction: %v", err)
	}

	if !result.Result.Result {
		return nil, fmt.Errorf("failed to create delegate resource transaction: %v", result.Result)
	}

	return result, nil
}

// CreateUndelegateResourceTransaction creates an undelegate resource transaction
func (c *Client) CreateUndelegateResourceTransaction(ownerAddress, receiverAddress []byte, amount int64, resource core.ResourceCode) (*api.TransactionExtention, error) {
	if err := c.ensureConnection(); err != nil {
		return nil, fmt.Errorf("connection error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	client := api.NewWalletClient(c.conn)
	result, err := client.UnDelegateResource(ctx, &core.UnDelegateResourceContract{
		OwnerAddress:    ownerAddress,
		ReceiverAddress: receiverAddress,
		Balance:         amount,
		Resource:        resource,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create undelegate resource transaction: %v", err)
	}

	if !result.Result.Result {
		return nil, fmt.Errorf("failed to create undelegate resource transaction: %v", result.Result)
	}

	return result, nil
}

// CreateWithdrawExpireUnfreezeTransaction creates a withdraw expire unfreeze transaction
func (c *Client) CreateWithdrawExpireUnfreezeTransaction(ownerAddress []byte) (*api.TransactionExtention, error) {
	if err := c.ensureConnection(); err != nil {
		return nil, fmt.Errorf("connection error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	client := api.NewWalletClient(c.conn)
	result, err := client.WithdrawExpireUnfreeze(ctx, &core.WithdrawExpireUnfreezeContract{
		OwnerAddress: ownerAddress,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create withdraw expire unfreeze transaction: %v", err)
	}

	if !result.Result.Result {
		return nil, fmt.Errorf("failed to create withdraw expire unfreeze transaction: %v", result.Result)
	}

	return result, nil
}

// CreateWithdrawBalanceTransaction creates a withdraw balance transaction (claim rewards)
func (c *Client) CreateWithdrawBalanceTransaction(ownerAddress []byte) (*api.TransactionExtention, error) {
	if err := c.ensureConnection(); err != nil {
		return nil, fmt.Errorf("connection error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	client := api.NewWalletClient(c.conn)
	result, err := client.WithdrawBalance2(ctx, &core.WithdrawBalanceContract{
		OwnerAddress: ownerAddress,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create withdraw balance transaction: %v", err)
	}

	if !result.Result.Result {
		return nil, fmt.Errorf("failed to create withdraw balance transaction: %v", result.Result)
	}

	return result, nil
}
