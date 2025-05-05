package client

import (
	"fmt"
	"time"
)

// ClientOptions contains configuration options for the Tron client
type ClientOptions struct {
	Endpoints   []string      // List of gRPC endpoints for failover
	ApiKey      string        // For rate limiting/auth if needed
	Timeout     time.Duration // Request timeout
	RetryConfig *RetryConfig  // Retry configuration

	// Worker configuration
	PriorityWorkers int           // Number of workers dedicated to priority tasks (default: 3)
	RateLimit       time.Duration // Minimum time between requests to same endpoint (default: 500ms)
}

// RetryConfig contains configuration for retry behavior
type RetryConfig struct {
	MaxAttempts     int           // Maximum number of retry attempts (total attempts = MaxAttempts + 1)
	InitialBackoff  time.Duration // Initial backoff duration
	MaxBackoff      time.Duration // Maximum backoff duration
	BackoffFactor   float64       // Multiplier for each subsequent backoff
	RetryableErrors []ErrorCode   // Error codes that should trigger a retry
}

// DefaultOptions returns a ClientOptions with sensible defaults
func DefaultOptions() *ClientOptions {
	return &ClientOptions{
		Endpoints:       []string{"grpc.trongrid.io:50051"},
		Timeout:         10 * time.Second,
		PriorityWorkers: 3,                      // Default 3 priority workers
		RateLimit:       500 * time.Millisecond, // Default 500ms rate limit
		RetryConfig: &RetryConfig{
			MaxAttempts:    2, // Will try 3 times total (initial + 2 retries)
			InitialBackoff: time.Second,
			MaxBackoff:     10 * time.Second,
			BackoffFactor:  2.0,
			RetryableErrors: []ErrorCode{
				ErrNetwork,
				ErrResourceExhausted,
			},
		},
	}
}

// validate checks if the options are valid
func (o *ClientOptions) validate() error {
	if len(o.Endpoints) == 0 {
		return fmt.Errorf("at least one endpoint must be provided")
	}

	if o.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	if o.RetryConfig == nil {
		return fmt.Errorf("retry config must be provided")
	}

	if o.RetryConfig.MaxAttempts < 0 {
		return fmt.Errorf("max attempts cannot be negative")
	}

	if o.RetryConfig.InitialBackoff <= 0 {
		return fmt.Errorf("initial backoff must be positive")
	}

	if o.RetryConfig.MaxBackoff < o.RetryConfig.InitialBackoff {
		return fmt.Errorf("max backoff must be greater than or equal to initial backoff")
	}

	if o.RetryConfig.BackoffFactor <= 0 {
		return fmt.Errorf("backoff factor must be positive")
	}

	if o.PriorityWorkers < 0 {
		return fmt.Errorf("priority workers cannot be negative")
	}

	if o.RateLimit < 0 {
		return fmt.Errorf("rate limit duration cannot be negative")
	}

	return nil
}
