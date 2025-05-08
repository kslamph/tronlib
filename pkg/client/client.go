package client

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

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

// TronClient manages connections to multiple Tron nodes
type TronClient struct {
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
			fmt.Printf("Successfully connected to node: %s\n", node.address)
		} else {
			conn.Close()
			fmt.Printf("Failed to verify connection to node %s: %v\n", node.address, verifyErr)
		}
	} else {
		fmt.Printf("Failed to connect to node %s: %v\n", node.address, err)
	}

	return node
}

// NewTronClient creates a new TronClient with the provided configuration
func NewTronClient(config ClientConfig) (*TronClient, error) {
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

	client := &TronClient{
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

	fmt.Printf("Successfully connected to %d/%d nodes\n", connectedNodes, len(config.Nodes))
	if connectedNodes == 0 {
		fmt.Println("Warning: No nodes were initially available. Will continue trying in background.")
	}

	// Start the connection management goroutine
	go client.manageConnections()

	return client, nil
}

// manageConnections periodically checks and manages connections to all nodes
func (c *TronClient) manageConnections() {
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
				fmt.Printf("Node cooldown period ended: %s\n", node.address)
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
						fmt.Printf("Reconnected to node: %s\n", node.address)
						success = true
					} else {
						conn.Close()
						fmt.Printf("Failed to verify reconnection to node %s: %v\n", node.address, verifyErr)
					}
				} else {
					fmt.Printf("Failed to reconnect to node %s: %v\n", node.address, err)
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
func (c *TronClient) selectNode() (*TronNodeStatus, error) {
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
		log.Printf("Using best performing node: %s (Avg response time: %v)\n", bestNode.address, bestNode.avgResponseTime)
		return bestNode, nil
	}
	log.Printf("Using random node from available nodes\n")
	return availableNodes[rand.Intn(len(availableNodes))], nil
}

// getConnection returns a gRPC connection to a selected node
func (c *TronClient) getConnection() (*grpc.ClientConn, *TronNodeStatus, error) {
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
func (c *TronClient) updateNodeMetrics(node *TronNodeStatus, duration time.Duration, err error) {
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
				fmt.Printf("Node placed in cooldown due to error (%s): %s (%v)\n", errCode.String(), node.address, err)
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
func (c *TronClient) Close() {
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
func (c *TronClient) BuildTransaction(contract interface{}) (*api.TransactionExtention, error) {
	result, err := c.ExecuteWithClient(func(ctx context.Context, conn *grpc.ClientConn) (interface{}, error) {
		walletClient := api.NewWalletClient(conn)
		switch v := contract.(type) {
		case *core.TransferContract:
			return walletClient.CreateTransaction2(ctx, v)
		case *core.TriggerSmartContract:
			return walletClient.TriggerConstantContract(ctx, v)
		case *core.FreezeBalanceV2Contract:
			return walletClient.FreezeBalanceV2(ctx, v)
		case *core.UnfreezeBalanceV2Contract:
			return walletClient.UnfreezeBalanceV2(ctx, v)
		case *core.DelegateResourceContract:
			return walletClient.DelegateResource(ctx, v)
		case *core.UnDelegateResourceContract:
			return walletClient.UnDelegateResource(ctx, v)
		case *core.WithdrawExpireUnfreezeContract:
			return walletClient.WithdrawExpireUnfreeze(ctx, v)
		default:
			return nil, fmt.Errorf("unsupported contract type: %T", contract)
		}
	})

	if err != nil {
		return nil, fmt.Errorf("failed to build transaction: %v", err)
	}

	return result.(*api.TransactionExtention), nil
}

// ExecuteWithClient executes a function with a gRPC client connection
func (c *TronClient) ExecuteWithClient(fn func(ctx context.Context, conn *grpc.ClientConn) (interface{}, error)) (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	conn, node, err := c.getConnection()
	if err != nil {
		return nil, err
	}

	start := time.Now()
	result, err := fn(ctx, conn)
	duration := time.Since(start)

	go c.updateNodeMetrics(node, duration, err)

	return result, err
}
