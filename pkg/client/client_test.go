package client_test

import (
	"context"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
)

func TestNewClient(t *testing.T) {
	// Test with default configuration
	config := client.ClientConfig{
		NodeAddress: "127.0.0.1:50051",
	}

	clientInstance, err := client.NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer clientInstance.Close()

	if clientInstance == nil {
		t.Fatal("Client should not be nil")
	}

	// Note: nodeAddress is not exposed via public method, so we can't test it directly
	// The node address is validated through the config passed to NewClient

	if clientInstance.GetTimeout() != 30*time.Second {
		t.Errorf("Expected timeout %v, got %v", 30*time.Second, clientInstance.GetTimeout())
	}
}

func TestClientWithCustomConfig(t *testing.T) {
	config := client.ClientConfig{
		NodeAddress:    "127.0.0.1:50051",
		Timeout:        60 * time.Second,
		MaxConnections: 10,
	}

	clientInstance, err := client.NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer clientInstance.Close()

	if clientInstance.GetTimeout() != 60*time.Second {
		t.Errorf("Expected timeout %v, got %v", 60*time.Second, clientInstance.GetTimeout())
	}
}

func TestClientClosed(t *testing.T) {
	config := client.ClientConfig{
		NodeAddress: "127.0.0.1:50051",
	}

	clientInstance, err := client.NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Close the client
	clientInstance.Close()

	// Test that client is closed by attempting to get a connection
	// Since isClosed() is unexported, we test the behavior indirectly

	// Test that operations fail on closed client
	ctx := context.Background()
	_, err = clientInstance.GetConnection(ctx)
	if err == nil {
		t.Error("Expected error when getting connection from closed client")
	}
}

func TestContextCancellation(t *testing.T) {
	config := client.ClientConfig{
		NodeAddress: "127.0.0.1:50051",
	}

	clientInstance, err := client.NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer clientInstance.Close()

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Test that operations fail with cancelled context
	_, err = clientInstance.GetConnection(ctx)
	if err == nil {
		t.Error("Expected error when context is cancelled")
	}
}

func TestConnectionPool(t *testing.T) {
	config := client.ClientConfig{
		NodeAddress:    "127.0.0.1:50051",
		MaxConnections: 3,
	}

	clientInstance, err := client.NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer clientInstance.Close()

	// Test that the client was created successfully
	// Note: nodeAddress is not exposed via public method, so we can't test it directly
}

func TestErrorTypes(t *testing.T) {
	// Test error type definitions
	if client.ErrConnectionFailed == nil {
		t.Error("ErrConnectionFailed should not be nil")
	}

	if client.ErrClientClosed == nil {
		t.Error("ErrClientClosed should not be nil")
	}

	if client.ErrContextCancelled == nil {
		t.Error("ErrContextCancelled should not be nil")
	}
}

// func TestClientTimeoutPropagation(t *testing.T) {
// 	// Test that client timeout is properly applied
// 	config := ClientConfig{
// 		NodeAddress: "localhost:99999", // Use a non-existent port to force connection failure
// 		Timeout:     5 * time.Second,
// 	}

// 	client, err := NewClient(config)
// 	if err != nil {
// 		t.Fatalf("Failed to create client: %v", err)
// 	}
// 	defer client.Close()

// 	// Verify timeout is set correctly
// 	if client.GetTimeout() != 5*time.Second {
// 		t.Errorf("Expected timeout 5s, got %v", client.GetTimeout())
// 	}

// 	// Test that timeout is applied to context when no deadline exists
// 	ctx := context.Background()
// 	start := time.Now()

// 	// This should fail quickly due to connection timeout, but the context should have the right timeout
// 	_, err = client.GetConnection(ctx)
// 	elapsed := time.Since(start)

// 	// Should fail quickly (within 6 seconds) due to connection timeout
// 	if elapsed > 6*time.Second {
// 		t.Errorf("Connection attempt took too long: %v, expected to fail quickly", elapsed)
// 	}

// 	// Should be a connection error (connection failed to non-existent port)
// 	if err == nil {
// 		t.Error("Expected connection error, got nil")
// 	} else if !errors.Is(err, ErrConnectionFailed) {
// 		t.Errorf("Expected connection failed error, got: %v", err)
// 	}
// }

// func TestClientTimeoutWithExistingDeadline(t *testing.T) {
// 	// Test that existing context deadline takes precedence
// 	config := ClientConfig{
// 		NodeAddress: "localhost:99999", // Use a non-existent port to force connection failure
// 		Timeout:     30 * time.Second,  // Long timeout
// 	}

// 	client, err := NewClient(config)
// 	if err != nil {
// 		t.Fatalf("Failed to create client: %v", err)
// 	}
// 	defer client.Close()

// 	// Create context with short deadline
// 	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
// 	defer cancel()

// 	start := time.Now()
// 	_, err = client.GetConnection(ctx)
// 	elapsed := time.Since(start)

// 	// Should fail quickly due to context deadline, not client timeout
// 	if elapsed > 200*time.Millisecond {
// 		t.Errorf("Connection attempt took too long: %v, expected to fail quickly due to context deadline", elapsed)
// 	}

// 	// Should be a context cancellation error or connection failed error
// 	if err == nil {
// 		t.Error("Expected error, got nil")
// 	} else if !errors.Is(err, ErrContextCancelled) && !errors.Is(err, ErrConnectionFailed) {
// 		t.Errorf("Expected context cancelled or connection failed error, got: %v", err)
// 	}
// }
