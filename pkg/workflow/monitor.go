package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/client/lowlevel"
)

// Monitor provides transaction monitoring utilities
type Monitor struct {
	client *client.Client
}

// NewMonitor creates a new monitor instance
func NewMonitor(client *client.Client) *Monitor {
	return &Monitor{
		client: client,
	}
}

// MonitorConfig configures transaction monitoring
type MonitorConfig struct {
	Confirmations int           // Number of confirmations to wait for
	Timeout       time.Duration // Maximum time to wait
	PollInterval  time.Duration // How often to check for updates
}

// DefaultMonitorConfig returns default monitoring configuration
func DefaultMonitorConfig() *MonitorConfig {
	return &MonitorConfig{
		Confirmations: 1,
		Timeout:       60 * time.Second,
		PollInterval:  3 * time.Second,
	}
}

// WaitForConfirmation waits for a transaction to be confirmed
func (m *Monitor) WaitForConfirmation(ctx context.Context, txID []byte, config *MonitorConfig) (*core.TransactionInfo, error) {
	if config == nil {
		config = DefaultMonitorConfig()
	}

	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()

	// Convert txID to BytesMessage
	req := &api.BytesMessage{Value: txID}

	// Poll for transaction confirmation
	ticker := time.NewTicker(config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			return nil, fmt.Errorf("timeout waiting for transaction confirmation after %v", config.Timeout)

		case <-ticker.C:
			// Get transaction info
			txInfo, err := lowlevel.GetTransactionInfoById(m.client, timeoutCtx, req)
			if err != nil {
				// Transaction might not be found yet, continue polling
				continue
			}

			// Check transaction result
			if txInfo.Result == core.TransactionInfo_FAILED {
				return txInfo, fmt.Errorf("transaction failed: %s", txInfo.ResMessage)
			}

			// Check if transaction is confirmed
			if txInfo.BlockNumber > 0 {
				// For basic confirmation, any block number > 0 means confirmed
				// In production, you might want to check against current block height
				// to ensure sufficient confirmations
				if config.Confirmations <= 1 {
					return txInfo, nil
				}

				// For multiple confirmations, we'd need to check current block height
				// This is a simplified implementation
				return txInfo, nil
			}
		}
	}
}

// GetTransactionStatus gets the current status of a transaction
func (m *Monitor) GetTransactionStatus(ctx context.Context, txID []byte) (*TransactionStatus, error) {
	req := &api.BytesMessage{Value: txID}

	// Try to get transaction info first
	txInfo, err := lowlevel.GetTransactionInfoById(m.client, ctx, req)
	if err != nil {
		// If transaction info not found, try to get the transaction itself
		tx, txErr := lowlevel.GetTransactionById(m.client, ctx, req)
		if txErr != nil {
			return &TransactionStatus{
				Status:  StatusNotFound,
				Message: "Transaction not found",
			}, nil
		}

		// Transaction exists but no info yet (pending)
		return &TransactionStatus{
			Status:      StatusPending,
			Message:     "Transaction pending",
			Transaction: tx,
		}, nil
	}

	// Determine status based on transaction info
	status := &TransactionStatus{
		TransactionInfo: txInfo,
		Transaction:     nil, // Could fetch if needed
	}

	switch txInfo.Result {
	case core.TransactionInfo_SUCESS:
		if txInfo.BlockNumber > 0 {
			status.Status = StatusConfirmed
			status.Message = "Transaction confirmed"
		} else {
			status.Status = StatusPending
			status.Message = "Transaction pending confirmation"
		}
	case core.TransactionInfo_FAILED:
		status.Status = StatusFailed
		status.Message = fmt.Sprintf("Transaction failed: %s", txInfo.ResMessage)
	default:
		status.Status = StatusUnknown
		status.Message = "Unknown transaction status"
	}

	return status, nil
}

// TransactionStatus represents the status of a transaction
type TransactionStatus struct {
	Status          Status                 `json:"status"`
	Message         string                 `json:"message"`
	TransactionInfo *core.TransactionInfo  `json:"transaction_info,omitempty"`
	Transaction     *core.Transaction      `json:"transaction,omitempty"`
}

// Status represents transaction status
type Status string

const (
	StatusPending   Status = "pending"
	StatusConfirmed Status = "confirmed"
	StatusFailed    Status = "failed"
	StatusNotFound  Status = "not_found"
	StatusUnknown   Status = "unknown"
)

// IsSuccess returns true if the transaction was successful
func (s *TransactionStatus) IsSuccess() bool {
	return s.Status == StatusConfirmed
}

// IsFailed returns true if the transaction failed
func (s *TransactionStatus) IsFailed() bool {
	return s.Status == StatusFailed
}

// IsPending returns true if the transaction is still pending
func (s *TransactionStatus) IsPending() bool {
	return s.Status == StatusPending
}

// MonitorMultiple monitors multiple transactions concurrently
func (m *Monitor) MonitorMultiple(ctx context.Context, txIDs [][]byte, config *MonitorConfig) (map[string]*TransactionStatus, error) {
	if config == nil {
		config = DefaultMonitorConfig()
	}

	results := make(map[string]*TransactionStatus)
	resultChan := make(chan struct {
		txID   string
		status *TransactionStatus
		err    error
	}, len(txIDs))

	// Start monitoring each transaction in a goroutine
	for _, txID := range txIDs {
		go func(id []byte) {
			status, err := m.GetTransactionStatus(ctx, id)
			resultChan <- struct {
				txID   string
				status *TransactionStatus
				err    error
			}{
				txID:   fmt.Sprintf("%x", id),
				status: status,
				err:    err,
			}
		}(txID)
	}

	// Collect results
	for i := 0; i < len(txIDs); i++ {
		result := <-resultChan
		if result.err != nil {
			return nil, fmt.Errorf("failed to monitor transaction %s: %w", result.txID, result.err)
		}
		results[result.txID] = result.status
	}

	return results, nil
}