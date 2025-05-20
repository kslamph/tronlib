package client

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"log"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/status"
)

var (
	// ErrAllNodesUnavailable is returned when all nodes are unavailable
	ErrAllNodesUnavailable = errors.New("all nodes are unavailable")

	// ErrNodeCoolingDown is returned when a node is in cooldown period
	ErrNodeCoolingDown = errors.New("node is in cooldown period")

	// ErrRateLimitExceeded is returned when a node's rate limit is exceeded
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
)

// RateLimit defines the rate limiting parameters
type RateLimit struct {
	Times  int64
	Window time.Duration
}

// NodeConfig represents the configuration for a single Tron node
type NodeConfig struct {
	Address   string
	RateLimit RateLimit
}

// ClientConfig represents the configuration for the TronClient
type ClientConfig struct {
	Nodes              []NodeConfig
	CooldownPeriod     time.Duration
	MetricsWindowSize  int
	BestNodePercentage int   // 0-100, percentage of requests to route to the best performing node
	TimeoutMs          int64 // Timeout in milliseconds for all RPC calls
}

// TronNodeStatus tracks the status and metrics for a single node
type TronNodeStatus struct {
	address           string
	conn              *grpc.ClientConn
	rateLimit         RateLimit
	requestsInWindow  int64
	windowStart       time.Time
	inCooldown        bool
	cooldownUntil     time.Time
	responseTimes     []time.Duration
	lastResponseTime  time.Time
	avgResponseTime   time.Duration
	reconnectInterval time.Duration
	lastReconnectTime time.Time
	mu                sync.RWMutex
}

// Client manages connections to multiple Tron nodes
type Client struct {
	nodes              []*TronNodeStatus
	cooldownPeriod     time.Duration
	metricsWindowSize  int
	bestNodePercentage int
	timeout            time.Duration
	mu                 sync.RWMutex
}

// connectToNode attempts to establish a connection to a single node
func connectToNode(ctx context.Context, address string) (*grpc.ClientConn, error) {
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			d := net.Dialer{Timeout: 5 * time.Second}
			return d.DialContext(ctx, "tcp", addr)
		}),
	}

	conn, err := grpc.DialContext(ctx, address, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %s: %v", address, err)
	}

	return conn, nil
}

// verifyConnection attempts to make a simple call to verify the connection
func verifyConnection(ctx context.Context, conn *grpc.ClientConn) error {
	client := api.NewWalletClient(conn)
	_, err := client.GetNodeInfo(ctx, &api.EmptyMessage{})
	return err
}

// initializeNode attempts to connect to a single node
func initializeNode(nodeConfig NodeConfig) *TronNodeStatus {
	node := &TronNodeStatus{
		address:           nodeConfig.Address,
		rateLimit:         nodeConfig.RateLimit,
		windowStart:       time.Now(),
		responseTimes:     make([]time.Duration, 0),
		reconnectInterval: DefaultInitialReconnectInterval,
		lastReconnectTime: time.Now(),
	}

	// Create context with timeout for this specific node
	ctx, cancel := context.WithTimeout(context.Background(), DefaultInitialConnectionTimeout)
	defer cancel()

	// Attempt connection
	conn, err := connectToNode(ctx, node.address)
	if err == nil {
		// Verify connection with a simple call
		if verifyErr := verifyConnection(ctx, conn); verifyErr == nil {
			node.conn = conn
			// fmt.Printf("Successfully connected to node: %s\n", node.address)
		} else {
			conn.Close()
			// fmt.Printf("Failed to verify connection to node %s: %v\n", node.address, verifyErr)
		}
	} else {
		log.Printf("[WARN] %v\n", err)
	}

	return node
}

// NewClient creates a new TronClient with the provided configuration
func NewClient(config ClientConfig) (*Client, error) {
	if len(config.Nodes) == 0 {
		return nil, errors.New("at least one node must be configured")
	}

	cooldownPeriod := config.CooldownPeriod
	if cooldownPeriod == 0 {
		cooldownPeriod = DefaultCooldownPeriod
	}

	metricsWindowSize := config.MetricsWindowSize
	if metricsWindowSize == 0 {
		metricsWindowSize = DefaultMetricsWindowSize
	}

	bestNodePercentage := config.BestNodePercentage
	if bestNodePercentage == 0 {
		bestNodePercentage = 90 // Default to 90% routing to best node
	}

	timeoutMs := config.TimeoutMs
	if timeoutMs == 0 {
		timeoutMs = DefaultTimeoutMs
	}

	client := &Client{
		nodes:              make([]*TronNodeStatus, 0, len(config.Nodes)),
		cooldownPeriod:     cooldownPeriod,
		metricsWindowSize:  metricsWindowSize,
		bestNodePercentage: bestNodePercentage,
		timeout:            time.Duration(timeoutMs) * time.Millisecond,
	}

	// Initialize nodes concurrently
	var wg sync.WaitGroup
	nodesChan := make(chan *TronNodeStatus, len(config.Nodes))

	for _, nodeConfig := range config.Nodes {
		wg.Add(1)
		go func(config NodeConfig) {
			defer wg.Done()
			nodesChan <- initializeNode(config)
		}(nodeConfig)
	}

	// Wait for all connection attempts to complete
	wg.Wait()
	close(nodesChan)

	// Collect results
	var connectedNodes int
	for node := range nodesChan {
		if node.conn != nil {
			connectedNodes++
		}
		client.nodes = append(client.nodes, node)
	}

	log.Printf("[INFO] Successfully Connected to %d/%d nodes\n", connectedNodes, len(config.Nodes)) // fmt.Printf("Successfully connected to %d/%d nodes\n", connectedNodes, len(config.Nodes))
	if connectedNodes == 0 {
		log.Println("[WARN] No nodes were initially available. Will continue trying in background.")
	}

	// Start the connection management goroutine
	go client.manageConnections()

	return client, nil
}

// manageConnections periodically checks and manages connections to all nodes
func (c *Client) manageConnections() {
	ticker := time.NewTicker(DefaultInitialReconnectInterval)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		c.mu.RLock()
		nodes := c.nodes
		c.mu.RUnlock()

		for _, node := range nodes {
			if !node.mu.TryLock() {
				continue // Skip if we can't get the lock immediately
			}

			// Check if cooldown period is over
			if node.inCooldown && now.After(node.cooldownUntil) {
				node.inCooldown = false
				// log.Printf("[INFO] Node cooldown period ended: %s\n", node.address)
			}

			// Reset rate limiting window if needed
			if node.rateLimit.Window > 0 && time.Since(node.windowStart) > node.rateLimit.Window {
				node.requestsInWindow = 0
				node.windowStart = now
			}

			// Initialize reconnection interval if not set
			if node.reconnectInterval == 0 {
				node.reconnectInterval = DefaultInitialReconnectInterval
			}

			// Check if it's time to attempt reconnection
			if (node.conn == nil || node.conn.GetState() == connectivity.TransientFailure ||
				node.conn.GetState() == connectivity.Shutdown) &&
				now.Sub(node.lastReconnectTime) >= node.reconnectInterval {

				// Close existing connection if any
				if node.conn != nil {
					node.conn.Close()
				}

				// Try to establish new connection
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				conn, err := connectToNode(ctx, node.address)
				success := false
				if err == nil {
					if verifyErr := verifyConnection(ctx, conn); verifyErr == nil {
						node.conn = conn
						node.reconnectInterval = DefaultInitialReconnectInterval // Reset interval on success
						log.Printf("Reconnected to node: %s\n", node.address)
						success = true
					} else {
						conn.Close()
						log.Printf("[WARN] Failed to verify reconnection to node %s: %v\n", node.address, verifyErr)
					}
				} else {
					log.Printf("[WARN] Failed to reconnect to node %s: %v\n", node.address, err)
				}
				cancel()

				// Update reconnection state
				node.lastReconnectTime = now
				if !success {
					// Exponential backoff: double the interval up to max
					node.reconnectInterval *= 2
					if node.reconnectInterval > DefaultMaxReconnectInterval {
						node.reconnectInterval = DefaultMaxReconnectInterval
					}
				}
			}

			node.mu.Unlock()
		}
	}
}

// selectNode chooses the appropriate node for the next request
func (c *Client) selectNode() (*TronNodeStatus, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	useBestNode := rand.Intn(100) < c.bestNodePercentage
	var availableNodes []*TronNodeStatus
	var bestNode *TronNodeStatus
	var bestAvgTime time.Duration

	now := time.Now()

	for _, node := range c.nodes {
		// Try to acquire read lock without blocking
		if !node.mu.TryRLock() {
			// Skip this node if it's locked (likely being reconnected)
			continue
		}

		// Guard against lock release in deferred function
		func() {
			defer node.mu.RUnlock()

			// Skip nodes in cooldown
			if node.inCooldown && now.Before(node.cooldownUntil) {
				return
			}

			// Skip nodes that have hit their rate limit
			if node.rateLimit.Times > 0 && node.rateLimit.Window > 0 &&
				node.requestsInWindow >= node.rateLimit.Times &&
				now.Sub(node.windowStart) < node.rateLimit.Window {
				return
			}

			// Skip nodes with no active connection
			if node.conn == nil {
				return
			}

			// Check connection state
			state := node.conn.GetState()
			if state != connectivity.Ready && state != connectivity.Idle {
				return
			}

			availableNodes = append(availableNodes, node)

			// Track the best performing node if needed
			if useBestNode && len(node.responseTimes) > 0 {
				if bestNode == nil || node.avgResponseTime < bestAvgTime {
					bestNode = node
					bestAvgTime = node.avgResponseTime
				}
			}
		}()
	}

	if len(availableNodes) == 0 {
		return nil, ErrAllNodesUnavailable
	}

	if useBestNode && bestNode != nil {
		return bestNode, nil
	}
	return availableNodes[rand.Intn(len(availableNodes))], nil
}

// getConnection returns a gRPC connection to a selected node
func (c *Client) getConnection() (*grpc.ClientConn, *TronNodeStatus, error) {
	node, err := c.selectNode()
	if err != nil {
		return nil, nil, err
	}

	node.mu.Lock()
	defer node.mu.Unlock()

	// Double-check rate limit now that we have the lock
	now := time.Now()
	if node.rateLimit.Window > 0 && now.Sub(node.windowStart) > node.rateLimit.Window {
		node.requestsInWindow = 0
		node.windowStart = now
	}

	if node.rateLimit.Times > 0 && node.requestsInWindow >= node.rateLimit.Times {
		return nil, nil, ErrRateLimitExceeded
	}

	node.requestsInWindow++

	return node.conn, node, nil
}

// updateNodeMetrics updates the response time metrics for a node
func (c *Client) updateNodeMetrics(node *TronNodeStatus, duration time.Duration, err error) {
	node.mu.Lock()
	defer node.mu.Unlock()

	metricDuration := duration

	if err != nil {
		metricDuration = 400 * time.Millisecond // Penalty for any failure

		s, ok := status.FromError(err)
		if ok {
			// Check for codes: DeadlineExceeded (4), ResourceExhausted (8), Unavailable (14)
			errCode := s.Code()
			if errCode == 4 || errCode == 8 || errCode == 14 {
				node.inCooldown = true
				node.cooldownUntil = time.Now().Add(c.cooldownPeriod)
				// log.Printf("[WARN] Node placed in cooldown due to error (%s): %s (%v)\n", errCode.String(), node.address, err)
			}
		}
		// For other errors (non-gRPC or gRPC errors not triggering cooldown), we still apply the 400ms penalty
		// but don't put the node in cooldown unless specified above.
	}

	node.responseTimes = append(node.responseTimes, metricDuration)
	if len(node.responseTimes) > c.metricsWindowSize {
		node.responseTimes = node.responseTimes[1:]
	}

	var total time.Duration
	for _, t := range node.responseTimes {
		total += t
	}
	// Ensure len(node.responseTimes) is not zero to prevent panic,
	// though it should be at least 1 due to the append operation.
	if len(node.responseTimes) > 0 {
		node.avgResponseTime = total / time.Duration(len(node.responseTimes))
	}
	node.lastResponseTime = time.Now()
}

// Close closes all connections to Tron nodes
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, node := range c.nodes {
		node.mu.Lock()
		if node.conn != nil {
			node.conn.Close()
			node.conn = nil
		}
		node.mu.Unlock()
	}
}

// BuildTransaction creates a new transaction using the provided contract
// BuildTransaction creates a new transaction using the provided contract
func (c *Client) BuildTransaction(contract interface{}) (*api.TransactionExtention, error) {
	var execFunc func(ctx context.Context, conn *grpc.ClientConn) (interface{}, error)

	walletClientFunc := func(ctx context.Context, conn *grpc.ClientConn, specificCall func(ctx context.Context, client api.WalletClient) (interface{}, error)) (interface{}, error) {
		client := api.NewWalletClient(conn)
		return specificCall(ctx, client)
	}

	switch v := contract.(type) {
	case *core.TransferContract:
		execFunc = func(ctx context.Context, conn *grpc.ClientConn) (interface{}, error) {
			return walletClientFunc(ctx, conn, func(ctx context.Context, client api.WalletClient) (interface{}, error) {
				return client.CreateTransaction2(ctx, v)
			})
		}
	case *core.TriggerSmartContract:
		execFunc = func(ctx context.Context, conn *grpc.ClientConn) (interface{}, error) {
			return walletClientFunc(ctx, conn, func(ctx context.Context, client api.WalletClient) (interface{}, error) {
				return client.TriggerContract(ctx, v)
			})
		}
	case *core.FreezeBalanceV2Contract:

		execFunc = func(ctx context.Context, conn *grpc.ClientConn) (interface{}, error) {
			return walletClientFunc(ctx, conn, func(ctx context.Context, client api.WalletClient) (interface{}, error) {
				return client.FreezeBalanceV2(ctx, v)
			})
		}
	case *core.UnfreezeBalanceV2Contract:

		execFunc = func(ctx context.Context, conn *grpc.ClientConn) (interface{}, error) {
			return walletClientFunc(ctx, conn, func(ctx context.Context, client api.WalletClient) (interface{}, error) {
				return client.UnfreezeBalanceV2(ctx, v)
			})
		}
	case *core.DelegateResourceContract:

		execFunc = func(ctx context.Context, conn *grpc.ClientConn) (interface{}, error) {
			return walletClientFunc(ctx, conn, func(ctx context.Context, client api.WalletClient) (interface{}, error) {
				return client.DelegateResource(ctx, v)
			})
		}
	case *core.UnDelegateResourceContract:

		execFunc = func(ctx context.Context, conn *grpc.ClientConn) (interface{}, error) {
			return walletClientFunc(ctx, conn, func(ctx context.Context, client api.WalletClient) (interface{}, error) {
				return client.UnDelegateResource(ctx, v)
			})
		}
	case *core.WithdrawExpireUnfreezeContract:

		execFunc = func(ctx context.Context, conn *grpc.ClientConn) (interface{}, error) {
			return walletClientFunc(ctx, conn, func(ctx context.Context, client api.WalletClient) (interface{}, error) {
				return client.WithdrawExpireUnfreeze(ctx, v)
			})
		}
	case *core.WithdrawBalanceContract:
		execFunc = func(ctx context.Context, conn *grpc.ClientConn) (interface{}, error) {
			return walletClientFunc(ctx, conn, func(ctx context.Context, client api.WalletClient) (interface{}, error) {
				return client.WithdrawBalance2(ctx, v)
			})
		}
	default:
		return nil, fmt.Errorf("unsupported contract type: %T", contract)
	}

	result, err := c.ExecuteWithClient(execFunc)
	if err != nil {
		return nil, fmt.Errorf("failed to build transaction: %v", err)
	}
	if result.(*api.TransactionExtention).Result.Result {
		return result.(*api.TransactionExtention), nil
	} else {
		return nil, fmt.Errorf("failed to build transaction: %v", result.(*api.TransactionExtention).Result)
	}

	// return result.(*api.TransactionExtention), nil
}

// ExecuteWithClient executes a function with a gRPC client connection
func (c *Client) ExecuteWithClient(fn func(ctx context.Context, conn *grpc.ClientConn) (any, error)) (any, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	var lastErr error
	retryDelay := 100 * time.Millisecond // Initial retry delay
	maxRetryDelay := 1 * time.Second     // Maximum retry delay

	for {
		select {
		case <-ctx.Done():
			if lastErr != nil {
				return nil, lastErr
			}
			return nil, ctx.Err()
		default:
			conn, node, err := c.getConnection()
			if err != nil {
				if err == ErrAllNodesUnavailable {
					lastErr = err
					// Exponential backoff with jitter
					jitter := time.Duration(rand.Int63n(int64(retryDelay) / 2))
					time.Sleep(retryDelay + jitter)

					// Increase retry delay for next attempt
					retryDelay *= 2
					if retryDelay > maxRetryDelay {
						retryDelay = maxRetryDelay
					}
					continue
				}
				return nil, err
			}

			start := time.Now()
			result, err := fn(ctx, conn)
			duration := time.Since(start)
			go c.updateNodeMetrics(node, duration, err)

			if err != nil {
				lastErr = err
				// If it's a rate limit error, retry
				if err == ErrRateLimitExceeded {
					time.Sleep(retryDelay)
					continue
				}
				return nil, err
			}
			return result, nil
		}
	}
}
