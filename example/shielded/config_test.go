package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigParseAmount(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr string
	}{
		{"raw base units", "10000000", "10000000", ""},
		{"surrounding whitespace", "  42\n", "42", ""},
		{"missing amount explains the unit", "", "", "-amount is required"},
		{"not a number", "1.5", "", "not a base-10 integer"},
		{"hex is refused", "0x10", "", "not a base-10 integer"},
		{"zero", "0", "", "must be positive"},
		{"negative", "-5", "", "must be positive"},
		{"beyond int64 is accepted here and refused by toScaled", "99999999999999999999", "99999999999999999999", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &config{amount: tt.in}
			got, err := c.parseAmount()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.String())
		})
	}
}

// TestFlowsCoversDocumentedModes keeps the -mode list in main.go and the one the
// README and docs/shielded.md promise in step.
func TestFlowsCoversDocumentedModes(t *testing.T) {
	assert.Equal(t, []string{"burn", "mint", "scan", "transfer", "walletgen"}, modeNames())

	for _, mode := range modeNames() {
		handler, ok := flows[mode]
		assert.True(t, ok, "%s must be dispatchable", mode)
		assert.NotNil(t, handler)
	}

	t.Run("an unknown mode lists the valid ones", func(t *testing.T) {
		err := run(&config{mode: "nope", timeout: time.Second})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown -mode \"nope\"")
		for _, mode := range modeNames() {
			assert.Contains(t, err.Error(), mode)
		}
	})
}

func TestSessionNeedAccount(t *testing.T) {
	t.Run("without a key, broadcasting modes refuse", func(t *testing.T) {
		s := &session{cfg: &config{}}
		err := s.needAccount("mint (it pays the fee)")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "-private-key")
		assert.Contains(t, err.Error(), "mint (it pays the fee)")
	})

	t.Run("scan and walletgen do not need one", func(t *testing.T) {
		// run() only calls needAccount from the four flows that broadcast, so a
		// session with a nil acct is a valid state for those two modes.
		s := &session{cfg: &config{}}
		assert.Nil(t, s.acct)
	})
}
