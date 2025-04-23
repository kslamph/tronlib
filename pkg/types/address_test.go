package types

import (
	"testing"
)

func TestAddressConversion(t *testing.T) {
	tests := []struct {
		name      string
		base58    string
		hex       string
		wantError bool
	}{
		{
			name:      "Valid address pair 1",
			base58:    "TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb",
			hex:       "41e28b3cfd4e0e909077821478e9fcb86b84be786e",
			wantError: false,
		},
		{
			name:      "Valid address pair 2",
			base58:    "TXNYeYdao7JL7wBtmzbk7mAie7UZsdgVjx",
			hex:       "41eac49bc766be29be1b6d36619eff8f86ed4d04df",
			wantError: false,
		},
		{
			name:      "Invalid base58 address - wrong prefix",
			base58:    "AWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb",
			hex:       "",
			wantError: true,
		},
		{
			name:      "Invalid base58 address - wrong length",
			base58:    "TWd4WrZ9wn84f5x1hZhL4DHvk738ns5",
			hex:       "",
			wantError: true,
		},
		{
			name:      "Invalid hex address - wrong prefix",
			base58:    "",
			hex:       "51e28b3cfd4e0e909077821478e9fcb86b84be786e",
			wantError: true,
		},
		{
			name:      "Invalid hex address - wrong length",
			base58:    "",
			hex:       "41e28b3cfd4e0e909077821478e9fcb86b84be78",
			wantError: true,
		},
		{
			name:      "Invalid hex address - not hex",
			base58:    "",
			hex:       "41x28b3cfd4e0e909077821478e9fcb86b84be786e",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test base58 to hex conversion
			if tt.base58 != "" {
				addr, err := NewAddress(tt.base58)
				if tt.wantError {
					if err == nil {
						t.Errorf("NewAddress() expected error for base58 address %s", tt.base58)
					}
					return
				}
				if err != nil {
					t.Errorf("NewAddress() unexpected error: %v", err)
					return
				}

				if got := addr.Hex(); got[2:] != tt.hex {
					t.Errorf("ToHex() = %v, want %v", got[2:], tt.hex)
				}
			}

			// Test hex to base58 conversion
			if tt.hex != "" {
				addr, err := NewAddressFromHex(tt.hex)
				if tt.wantError {
					if err == nil {
						t.Errorf("NewHexAddress() expected error for hex address %s", tt.hex)
					}
					return
				}
				if err != nil {
					t.Errorf("NewHexAddress() unexpected error: %v", err)
					return
				}

				if got := addr.String(); got != tt.base58 {
					t.Errorf("ToBase58() = %v, want %v", got, tt.base58)
				}
			}
		})
	}
}

func TestAddressValidation(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		inputType string // "base58" or "hex"
		wantError bool
	}{
		{
			name:      "Empty base58 address",
			input:     "",
			inputType: "base58",
			wantError: true,
		},
		{
			name:      "Empty hex address",
			input:     "",
			inputType: "hex",
			wantError: true,
		},
		{
			name:      "Invalid checksum in base58",
			input:     "TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwa", // Changed last char
			inputType: "base58",
			wantError: true,
		},
		{
			name:      "Invalid hex characters",
			input:     "41e28b3cfd4e0e909077821478e9fcb86b84be786g", // 'g' is not hex
			inputType: "hex",
			wantError: true,
		},
		{
			name:      "Too long base58 address",
			input:     "TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwbXX",
			inputType: "base58",
			wantError: true,
		},
		{
			name:      "Too long hex address",
			input:     "41e28b3cfd4e0e909077821478e9fcb86b84be786e00",
			inputType: "hex",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.inputType == "base58" {
				_, err = NewAddress(tt.input)
			} else {
				_, err = NewAddressFromHex(tt.input)
			}

			if tt.wantError && err == nil {
				t.Errorf("Expected error for input %s", tt.input)
			}
			if !tt.wantError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
