package client

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	// ErrConnectionFailed is returned when connection to the node fails
	ErrConnectionFailed = errors.New("connection to node failed")
	// ErrClientClosed is returned when trying to use a closed client
	ErrClientClosed = errors.New("client is closed")
	// ErrContextCancelled is returned when context is cancelled
	ErrContextCancelled = errors.New("context cancelled")
)

// ClientConfig represents the configuration for the TronClient
type ClientConfig struct {
	NodeAddress     string        // Single node address
	Timeout         time.Duration // Universal timeout for all operations (connection + RPC calls)
	InitConnections int           // Initial number of connections in pool
	MaxConnections  int           // Maximum number of connections in pool
	IdleTimeout     time.Duration // How long connections can be idle
}

// Client manages connection to a single Tron node with connection pooling
type Client struct {
	pool        *ConnPool
	timeout     time.Duration
	nodeAddress string
	closed      int32
}

// NewClient creates a new TronClient with the provided configuration (lazy connection)
func NewClient(config ClientConfig) (*Client, error) {
	if config.NodeAddress == "" {
		return nil, errors.New("node address must be provided")
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second // Default timeout for all operations
	}

	maxConnections := config.MaxConnections
	if maxConnections <= 0 {
		maxConnections = 5 // Default pool size
	}

	initConnections := config.InitConnections
	if initConnections <= 0 {
		initConnections = 1 // Default initial pool size
	}

	factory := func(ctx context.Context) (*grpc.ClientConn, error) {
		return grpc.NewClient(config.NodeAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Use the same timeout for connection pool
	pool, err := NewConnPool(factory, initConnections, maxConnections)
	if err != nil {
		return nil, err
	}

	return &Client{
		pool:        pool,
		timeout:     timeout,
		nodeAddress: config.NodeAddress,
	}, nil
}

// GetConnection safely gets a connection from the pool
func (c *Client) GetConnection(ctx context.Context) (*grpc.ClientConn, error) {
	if atomic.LoadInt32(&c.closed) == 1 {
		return nil, ErrClientClosed
	}

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return nil, ErrContextCancelled
	default:
	}

	// Apply client timeout if context doesn't have a deadline
	// This ensures the entire operation (connection + RPC) respects the timeout
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	return c.pool.Get(ctx)
}

// ReturnConnection safely returns a connection to the pool
func (c *Client) ReturnConnection(conn *grpc.ClientConn) {
	if atomic.LoadInt32(&c.closed) == 1 {
		return
	}
	c.pool.Put(conn)
}

// Close closes the client and all connections in the pool
func (c *Client) Close() {
	if !atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		return // Already closed
	}
	c.pool.Close()
}

// GetTimeout returns the client's configured timeout
func (c *Client) GetTimeout() time.Duration {
	return c.timeout
}

// Transaction creation methods

// CreateTransferTransaction creates a TRX transfer transaction
func (c *Client) CreateTransferTransaction(ctx context.Context, ownerAddress, toAddress []byte, amount int64) (*api.TransactionExtention, error) {
	return c.grpcCallWrapper(ctx, "transfer", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.CreateTransaction2(ctx, &core.TransferContract{
			OwnerAddress: ownerAddress,
			ToAddress:    toAddress,
			Amount:       amount,
		})
	})
}

// CreateTriggerSmartContractTransaction creates a smart contract trigger transaction
func (c *Client) CreateTriggerSmartContractTransaction(ctx context.Context, ownerAddress, contractAddress []byte, data []byte, callValue int64) (*api.TransactionExtention, error) {
	return c.grpcCallWrapper(ctx, "smart contract", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.TriggerContract(ctx, &core.TriggerSmartContract{
			OwnerAddress:    ownerAddress,
			ContractAddress: contractAddress,
			Data:            data,
			CallValue:       callValue,
		})
	})
}

// CreateFreezeTransaction creates a freeze balance transaction
func (c *Client) CreateFreezeTransaction(ctx context.Context, ownerAddress []byte, amount int64, resource core.ResourceCode) (*api.TransactionExtention, error) {
	return c.grpcCallWrapper(ctx, "freeze", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.FreezeBalanceV2(ctx, &core.FreezeBalanceV2Contract{
			OwnerAddress:  ownerAddress,
			FrozenBalance: amount,
			Resource:      resource,
		})
	})
}

// CreateUnfreezeTransaction creates an unfreeze balance transaction
func (c *Client) CreateUnfreezeTransaction(ctx context.Context, ownerAddress []byte, amount int64, resource core.ResourceCode) (*api.TransactionExtention, error) {
	return c.grpcCallWrapper(ctx, "unfreeze", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.UnfreezeBalanceV2(ctx, &core.UnfreezeBalanceV2Contract{
			OwnerAddress:    ownerAddress,
			UnfreezeBalance: amount,
			Resource:        resource,
		})
	})
}

// CreateDelegateResourceTransaction creates a delegate resource transaction
func (c *Client) CreateDelegateResourceTransaction(ctx context.Context, ownerAddress, receiverAddress []byte, amount int64, resource core.ResourceCode, lock bool, lockPeriod int64) (*api.TransactionExtention, error) {
	return c.grpcCallWrapper(ctx, "delegate resource", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.DelegateResource(ctx, &core.DelegateResourceContract{
			OwnerAddress:    ownerAddress,
			ReceiverAddress: receiverAddress,
			Balance:         amount,
			Resource:        resource,
			Lock:            lock,
			LockPeriod:      lockPeriod,
		})
	})
}

// CreateUndelegateResourceTransaction creates an undelegate resource transaction
func (c *Client) CreateUndelegateResourceTransaction(ctx context.Context, ownerAddress, receiverAddress []byte, amount int64, resource core.ResourceCode) (*api.TransactionExtention, error) {
	return c.grpcCallWrapper(ctx, "undelegate resource", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.UnDelegateResource(ctx, &core.UnDelegateResourceContract{
			OwnerAddress:    ownerAddress,
			ReceiverAddress: receiverAddress,
			Balance:         amount,
			Resource:        resource,
		})
	})
}

// CreateWithdrawExpireUnfreezeTransaction creates a withdraw expire unfreeze transaction
func (c *Client) CreateWithdrawExpireUnfreezeTransaction(ctx context.Context, ownerAddress []byte) (*api.TransactionExtention, error) {
	return c.grpcCallWrapper(ctx, "withdraw expire unfreeze", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.WithdrawExpireUnfreeze(ctx, &core.WithdrawExpireUnfreezeContract{
			OwnerAddress: ownerAddress,
		})
	})
}

// CreateWithdrawBalanceTransaction creates a withdraw balance transaction (claim rewards)
func (c *Client) CreateWithdrawBalanceTransaction(ctx context.Context, ownerAddress []byte) (*api.TransactionExtention, error) {
	return c.grpcCallWrapper(ctx, "withdraw balance", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.WithdrawBalance2(ctx, &core.WithdrawBalanceContract{
			OwnerAddress: ownerAddress,
		})
	})
}

// CreateDeployContractTransaction creates a deploy contract transaction
func (c *Client) CreateDeployContractTransaction(ctx context.Context, contract *core.CreateSmartContract) (*api.TransactionExtention, error) {
	return c.grpcCallWrapper(ctx, "deploy contract", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.DeployContract(ctx, contract)
	})
}
