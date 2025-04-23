package types

import (
	"testing"
)

func TestAccountFromPrivateKey(t *testing.T) {
	tests := []struct {
		name      string
		privKey   string
		wantAddr  string
		wantError bool
	}{
		{
			name:      "Valid account 1",
			privKey:   "cfae06d915cf9784272fa99d4db961b8cbafd59c8b2f77ab7422be5424d3e49c",
			wantAddr:  "TQJ6R9SPvD5SyqgYqTBq3yc6mFtEgatPDu",
			wantError: false,
		},
		{
			name:      "Valid account 2",
			privKey:   "dccf423c8dce5c744f72ebc62bd59797bd8fff10953f4f90af6cbb1121f96415",
			wantAddr:  "TT3G7td4FPhirwPC44BxGhfHGeD4uj6r7j",
			wantError: false,
		},
		{
			name:      "Invalid private key - too short",
			privKey:   "cfae06d915cf9784272fa99d4db961b8",
			wantAddr:  "",
			wantError: true,
		},
		{
			name:      "Invalid private key - not hex",
			privKey:   "cfae06d915cf9784272fa99d4db961b8cbafd59c8b2f77ab7422be5424d3e49g",
			wantAddr:  "",
			wantError: true,
		},
		{
			name:      "Empty private key",
			privKey:   "",
			wantAddr:  "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account, err := NewAccountFromPrivateKey(tt.privKey)
			if tt.wantError {
				if err == nil {
					t.Errorf("NewAccountFromPrivateKey() expected error for private key %s", tt.privKey)
				}
				return
			}
			if err != nil {
				t.Errorf("NewAccountFromPrivateKey() unexpected error: %v", err)
				return
			}

			if got := account.Address().String(); got != tt.wantAddr {
				t.Errorf("Account address = %v, want %v", got, tt.wantAddr)
			}

			// Verify private key is preserved correctly
			if got := account.PrivateKeyHex(); got != tt.privKey {
				t.Errorf("Account private key = %v, want %v", got, tt.privKey)
			}
		})
	}
}

func TestAccountFromHDWallet(t *testing.T) {
	const testMnemonic = "fat social problem enable number gain parrot balance reduce bunker beach image marriage motion friend system dolphin bind leaf spin eye slogan rack track"

	tests := []struct {
		name      string
		mnemonic  string
		path      string
		wantAddr  string
		wantPKey  string
		wantError bool
	}{
		{
			name:      "Default path",
			mnemonic:  testMnemonic,
			path:      "",
			wantAddr:  "TNPjabV8z8y7DRCsG9txgTwtz4ixDzCRzs",
			wantPKey:  "2da7c0d5a25677bea36ba96a71e9683eb9b9b7b0188e4229964c9ba87ee0f020",
			wantError: false,
		},
		{
			name:      "Custom path",
			mnemonic:  testMnemonic,
			path:      "m/44'/195'/10'/0/5",
			wantAddr:  "TRucjoUVF6MkHUxm61epieK1SxPbC4wbPP",
			wantPKey:  "87753ab3def8420afd1ca1c31acb2edcf33ac437bd837b1e39b7d57448ae53c5",
			wantError: false,
		},
		{
			name:      "Invalid mnemonic",
			mnemonic:  "invalid mnemonic phrase",
			path:      "",
			wantAddr:  "",
			wantPKey:  "",
			wantError: true,
		},
		{
			name:      "Invalid derivation path",
			mnemonic:  testMnemonic,
			path:      "m/44'/195'/x'/0/0",
			wantAddr:  "",
			wantPKey:  "",
			wantError: true,
		},
		{
			name:      "Empty mnemonic",
			mnemonic:  "",
			path:      "",
			wantAddr:  "",
			wantPKey:  "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account, err := NewAccountFromHDWallet(tt.mnemonic, tt.path)
			if tt.wantError {
				if err == nil {
					t.Errorf("NewAccountFromHDWallet() expected error for mnemonic %s and path %s", tt.mnemonic, tt.path)
				}
				return
			}
			if err != nil {
				t.Errorf("NewAccountFromHDWallet() unexpected error: %v", err)
				return
			}

			if got := account.Address().String(); got != tt.wantAddr {
				t.Errorf("Account address = %v, want %v", got, tt.wantAddr)
			}

			if got := account.PrivateKeyHex(); got != tt.wantPKey {
				t.Errorf("Account private key = %v, want %v", got, tt.wantPKey)
			}
		})
	}
}

func TestAccountSigningWithKnownResult(t *testing.T) {
	const (
		privateKey    = "f8c6f45b2aa8b68ab5f3910bdeb5239428b731618113e2881f46e374bf796b02"
		messageToSign = "sign message testing"
		wantSignature = "0x88bacb8549cbe7c3e26d922b05e88757197b77410fb0db1fabb9f30480202c84691b7025e928d36be962cfd7b4a8d2353b97f36d64bdc14398e9568091b701201b"
	)

	account, err := NewAccountFromPrivateKey(privateKey)
	if err != nil {
		t.Fatalf("Failed to create test account: %v", err)
	}

	signature, err := account.SignMessageV2(messageToSign)
	if err != nil {
		t.Fatalf("SignMessageV2() unexpected error: %v", err)
	}

	if signature != wantSignature {
		t.Errorf("SignMessageV2() signature = %v, want %v", signature, wantSignature)
	}
}
