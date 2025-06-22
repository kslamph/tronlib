package client

import "time"

const (
	// DefaultCooldownPeriod defines the default cooldown period for nodes that encounter errors
	DefaultCooldownPeriod = 1 * time.Minute

	// DefaultRateLimitWindow defines the window for rate limiting calculations
	DefaultRateLimitWindow = 1 * time.Second

	// DefaultInitialReconnectInterval defines the initial interval for reconnection attempts
	DefaultInitialReconnectInterval = 5 * time.Second

	// DefaultMaxReconnectInterval defines the maximum interval between reconnection attempts
	DefaultMaxReconnectInterval = 2 * time.Minute

	// DefaultMetricsWindowSize defines how many recent requests to consider for response time metrics
	DefaultMetricsWindowSize = 3

	// DefaultTimeoutMs defines the default timeout in milliseconds for RPC calls
	DefaultTimeoutMs = 5000 // 5 seconds

	// DefaultInitialConnectionTimeout is the timeout for each individual node's initial connection
	DefaultInitialConnectionTimeout = 5 * time.Second
)

func MainNodes() []NodeConfig {
	return []NodeConfig{
		// {Address: "3.225.171.164:50051", RateLimit: DefaultRateLimit()},
		{Address: "52.53.189.99:50051", RateLimit: DefaultRateLimit()},
		{Address: "18.196.99.16:50051", RateLimit: DefaultRateLimit()},
		{Address: "34.253.187.192:50051", RateLimit: DefaultRateLimit()},
		{Address: "18.133.82.227:50051", RateLimit: DefaultRateLimit()},
		{Address: "35.180.51.163:50051", RateLimit: DefaultRateLimit()},
		{Address: "54.252.224.209:50051", RateLimit: DefaultRateLimit()},
		{Address: "52.15.93.92:50051", RateLimit: DefaultRateLimit()},
		{Address: "34.220.77.106:50051", RateLimit: DefaultRateLimit()},
		{Address: "15.207.144.3:50051", RateLimit: DefaultRateLimit()},
		{Address: "13.124.62.58:50051", RateLimit: DefaultRateLimit()},
		{Address: "15.222.19.181:50051", RateLimit: DefaultRateLimit()},
		{Address: "18.209.42.127:50051", RateLimit: DefaultRateLimit()},
		{Address: "3.218.137.187:50051", RateLimit: DefaultRateLimit()},
		{Address: "34.237.210.82:50051", RateLimit: DefaultRateLimit()},
		{Address: "13.228.119.63:50051", RateLimit: DefaultRateLimit()},
		{Address: "18.139.193.235:50051", RateLimit: DefaultRateLimit()},
		{Address: "18.141.79.38:50051", RateLimit: DefaultRateLimit()},
		{Address: "18.139.248.26:50051", RateLimit: DefaultRateLimit()},
		{Address: "grpc.trongrid.io:50051", RateLimit: DefaultRateLimit()},
	}
}

func ShastaNodes() []NodeConfig {
	return []NodeConfig{
		{Address: "grpc.shasta.trongrid.io:50051", RateLimit: DefaultRateLimit()},
	}
}

func NileNodes() []NodeConfig {
	return []NodeConfig{
		{Address: "grpc.nile.trongrid.io:50051", RateLimit: DefaultRateLimit()},
	}
}

func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		Nodes:              MainNodes(),
		CooldownPeriod:     DefaultCooldownPeriod,
		MetricsWindowSize:  DefaultMetricsWindowSize,
		BestNodePercentage: 90, // Default to 90% routing to best node
		TimeoutMs:          DefaultTimeoutMs,
	}
}
func DefaultRateLimit() RateLimit {
	return RateLimit{
		Times:  5,
		Window: DefaultRateLimitWindow,
	}
}
