package types

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kslamph/tronlib/pb/api"
)

// ---------- TronError ----------

func TestTronError_Error(t *testing.T) {
	tests := []struct {
		name     string
		code     int32
		message  string
		cause    error
		contains []string
	}{
		{
			name:     "with cause",
			code:     42,
			message:  "something failed",
			cause:    fmt.Errorf("underlying"),
			contains: []string{"TRON error 42", "something failed", "underlying"},
		},
		{
			name:     "without cause",
			code:     7,
			message:  "boom",
			cause:    nil,
			contains: []string{"TRON error 7", "boom"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := NewTronError(tc.code, tc.message, tc.cause)
			msg := e.Error()
			for _, s := range tc.contains {
				assert.Contains(t, msg, s)
			}
		})
	}
}

func TestTronError_Unwrap(t *testing.T) {
	inner := fmt.Errorf("inner")
	e := NewTronError(1, "msg", inner)
	assert.True(t, errors.Is(e, inner))

	eNil := NewTronError(1, "msg", nil)
	assert.Nil(t, eNil.Unwrap())
}

// ---------- TransactionError ----------

func TestTransactionError_Error(t *testing.T) {
	tests := []struct {
		name     string
		txID     string
		message  string
		cause    error
		contains []string
	}{
		{
			name:     "with cause",
			txID:     "abc123",
			message:  "reverted",
			cause:    fmt.Errorf("reason"),
			contains: []string{"transaction abc123 failed", "reverted", "reason"},
		},
		{
			name:     "without cause",
			txID:     "xyz",
			message:  "oops",
			cause:    nil,
			contains: []string{"transaction xyz failed", "oops"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := NewTransactionError(tc.txID, tc.message, tc.cause)
			msg := e.Error()
			for _, s := range tc.contains {
				assert.Contains(t, msg, s)
			}
		})
	}
}

func TestTransactionError_Unwrap(t *testing.T) {
	inner := fmt.Errorf("inner")
	e := NewTransactionError("tx1", "msg", inner)
	assert.True(t, errors.Is(e, inner))

	eNil := NewTransactionError("tx1", "msg", nil)
	assert.Nil(t, eNil.Unwrap())
}

// ---------- ContractError ----------

func TestContractError_Error(t *testing.T) {
	tests := []struct {
		name     string
		addr     string
		method   string
		message  string
		cause    error
		contains []string
	}{
		{
			name:     "with cause",
			addr:     "TAddr",
			method:   "transfer",
			message:  "reverted",
			cause:    fmt.Errorf("reason"),
			contains: []string{"contract TAddr", "method transfer", "reverted", "reason"},
		},
		{
			name:     "without cause",
			addr:     "TAddr2",
			method:   "mint",
			message:  "fail",
			cause:    nil,
			contains: []string{"contract TAddr2", "method mint", "fail"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := NewContractError(tc.addr, tc.method, tc.message, tc.cause)
			msg := e.Error()
			for _, s := range tc.contains {
				assert.Contains(t, msg, s)
			}
		})
	}
}

func TestContractError_Unwrap(t *testing.T) {
	inner := fmt.Errorf("inner")
	e := NewContractError("addr", "method", "msg", inner)
	assert.True(t, errors.Is(e, inner))

	eNil := NewContractError("addr", "method", "msg", nil)
	assert.Nil(t, eNil.Unwrap())
}

// ---------- WrapTransactionResult ----------

func TestWrapTransactionResult(t *testing.T) {
	// nil result
	err := WrapTransactionResult(nil, "deploy")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil result for deploy")
	var te *TronError
	require.True(t, errors.As(err, &te))
	assert.Equal(t, int32(0), te.Code)

	// success (result.Result == true)
	err = WrapTransactionResult(&api.Return{Result: true, Code: api.Return_SUCCESS}, "call")
	assert.NoError(t, err)

	// failure with message
	err = WrapTransactionResult(&api.Return{
		Result:  false,
		Code:    api.Return_OTHER_ERROR,
		Message: []byte("revert"),
	}, "invoke")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invoke failed")
	assert.Contains(t, err.Error(), "revert")

	// failure without message -> unknown error
	err = WrapTransactionResult(&api.Return{
		Result:  false,
		Code:    api.Return_OTHER_ERROR,
		Message: nil,
	}, "trigger")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown error")
}

// ---------- Constants: GetNetwork ----------

func TestGetNetwork(t *testing.T) {
	tests := []struct {
		input   string
		isNil   bool
		mainnet bool
		testnet bool
		nile    bool
	}{
		{input: TronMainNet, mainnet: true},
		{input: TronTestNet, testnet: true},
		{input: TronNileNet, nile: true},
		{input: "unknown", isNil: true},
		{input: "", isNil: true},
	}
	for _, tc := range tests {
		name := tc.input
		if name == "" {
			name = "empty"
		}
		t.Run(name, func(t *testing.T) {
			n := GetNetwork(tc.input)
			if tc.isNil {
				assert.Nil(t, n)
			} else {
				require.NotNil(t, n)
				if tc.mainnet {
					assert.Equal(t, TronMainNet, n.Name)
				}
				if tc.testnet {
					assert.Equal(t, TronTestNet, n.Name)
				}
				if tc.nile {
					assert.Equal(t, TronNileNet, n.Name)
				}
			}
		})
	}
}

// ---------- Constants: String methods ----------

func TestResourceTypeString(t *testing.T) {
	tests := []struct {
		r    ResourceType
		want string
	}{
		{ResourceBandwidth, "BANDWIDTH"},
		{ResourceEnergy, "ENERGY"},
		{ResourceType(99), "UNKNOWN"},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.r.String())
		})
	}
}

func TestContractTypeString(t *testing.T) {
	tests := []struct {
		c    ContractType
		want string
	}{
		{ContractTypeTRC20, "TRC20"},
		{ContractTypeTRC721, "TRC721"},
		{ContractTypeTRC1155, "TRC1155"},
		{ContractTypeCustom, "CUSTOM"},
		{ContractTypeUnknown, "UNKNOWN"},
		{ContractType(99), "UNKNOWN"},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.c.String())
		})
	}
}

func TestTransactionStatusString(t *testing.T) {
	tests := []struct {
		s    TransactionStatus
		want string
	}{
		{TransactionStatusPending, "PENDING"},
		{TransactionStatusConfirmed, "CONFIRMED"},
		{TransactionStatusFailed, "FAILED"},
		{TransactionStatusReverted, "REVERTED"},
		{TransactionStatusUnknown, "UNKNOWN"},
		{TransactionStatus(99), "UNKNOWN"},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.s.String())
		})
	}
}
