package crypto

import (
	"strings"
	"testing"
)

func TestVerifyMessageV2(t *testing.T) {
	tests := []struct {
		name        string
		address     string
		message     string
		signature   string
		wantValid   bool
		wantErr     bool
		errContains string
	}{
		{
			name:      "Valid signature",
			address:   "TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x",
			message:   "sign message testing",
			signature: "0x88bacb8549cbe7c3e26d922b05e88757197b77410fb0db1fabb9f30480202c84691b7025e928d36be962cfd7b4a8d2353b97f36d64bdc14398e9568091b701201b",
			wantValid: true,
			wantErr:   false,
		},
		{
			name:        "Invalid signature length",
			address:     "TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x",
			message:     "test message",
			signature:   "0x1234",
			wantValid:   false,
			wantErr:     true,
			errContains: "invalid signature length",
		},
		{
			name:        "Invalid hex signature",
			address:     "TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x",
			message:     "test message",
			signature:   "0xZZ",
			wantValid:   false,
			wantErr:     true,
			errContains: "invalid",
		},
		{
			name:      "Invalid signature for message",
			address:   "TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x",
			message:   "wrong message",
			signature: "0x88bacb8549cbe7c3e26d922b05e88757197b77410fb0db1fabb9f30480202c84691b7025e928d36be962cfd7b4a8d2353b97f36d64bdc14398e9568091b701201b",
			wantValid: false,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			valid, err := VerifyMessageV2(tt.address, tt.message, tt.signature)

			// Check error expectations
			if tt.wantErr {
				if err == nil {
					t.Errorf("VerifyMessageV2() error = nil, wantErr = true")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("VerifyMessageV2() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("VerifyMessageV2() unexpected error: %v", err)
				return
			}

			// Check validity expectations
			if valid != tt.wantValid {
				t.Errorf("VerifyMessageV2() = %v, want %v", valid, tt.wantValid)
			}
		})
	}
}
