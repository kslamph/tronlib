package utils_test

import (
	"testing"

	"github.com/kslamph/tronlib/pkg/signer"
	"github.com/kslamph/tronlib/pkg/utils"
)

func TestVerifyMessageV2(t *testing.T) {
	// Create a test signer
	privateKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	s, err := signer.NewPrivateKeySigner(privateKey)
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	testCases := []struct {
		name    string
		message string
	}{
		{"Simple text", "Hello TRON!"},
		{"Empty message", ""},
		{"Unicode message", "Hello 世界! 🌍"},
		{"Hex message", "0x48656c6c6f"},
		{"Long message", "This is a very long message that tests the verification of longer strings with various characters and numbers 12345"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Sign the message
			signature, err := s.SignMessageV2(tc.message)
			if err != nil {
				t.Fatalf("Failed to sign message: %v", err)
			}

			// Verify with correct address
			valid, err := utils.VerifyMessageV2(tc.message, signature, s.Address().String())
			if err != nil {
				t.Fatalf("Failed to verify message: %v", err)
			}

			if !valid {
				t.Fatal("Message verification should be valid")
			}

			// Test with wrong address
			wrongAddress := "TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH" // Random valid address
			valid, err = utils.VerifyMessageV2(tc.message, signature, wrongAddress)
			if err != nil {
				t.Fatalf("Failed to verify message with wrong address: %v", err)
			}

			if valid {
				t.Fatal("Message verification should be invalid with wrong address")
			}

			t.Logf("Test case '%s' passed", tc.name)
		})
	}
}

func TestVerifyMessageV2_InvalidInputs(t *testing.T) {
	validMessage := "Hello TRON!"
	validSignature := "0x2c96d0b076b1ea6a2fc3135fadeab9c95e0efd0738e54c65e8cf4f5bbfe184fe45553edceb71d1f9c548c4d90ccd6bfef4b76e7b9c2205c9ee9e76b38f13bf651b"
	validAddress := "TZ1EafTG8FRtE6ef3H2dhaucDdjv36fzPY"

	testCases := []struct {
		name      string
		message   string
		signature string
		address   string
		expectErr bool
	}{
		{"Invalid signature format", validMessage, "invalid", validAddress, true},
		{"Signature without 0x", validMessage, "2c96d0b076b1ea6a2fc3135fadeab9c95e0efd0738e54c65e8cf4f5bbfe184fe45553edceb71d1f9c548c4d90ccd6bfef4b76e7b9c2205c9ee9e76b38f13bf651b", validAddress, true},
		{"Short signature", validMessage, "0x1234", validAddress, true},
		{"Invalid address", validMessage, validSignature, "invalid_address", true},
		{"Empty address", validMessage, validSignature, "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			valid, err := utils.VerifyMessageV2(tc.message, tc.signature, tc.address)

			if tc.expectErr {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				if valid {
					t.Fatal("Expected invalid result but got valid")
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
			}

			t.Logf("Test case '%s' passed with expected error: %v", tc.name, err)
		})
	}
}

func TestVerifyMessageV2_CrossSigner(t *testing.T) {
	// Test verification across different signer implementations
	privateKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

	// Create signers
	pkSigner, err := signer.NewPrivateKeySigner(privateKey)
	if err != nil {
		t.Fatalf("Failed to create private key signer: %v", err)
	}

	hdSigner, err := signer.NewHDWalletSigner(mnemonic, "", "")
	if err != nil {
		t.Fatalf("Failed to create HD wallet signer: %v", err)
	}

	message := "Cross-signer verification test"

	// Test private key signer
	pkSignature, err := pkSigner.SignMessageV2(message)
	if err != nil {
		t.Fatalf("Failed to sign with private key signer: %v", err)
	}

	valid, err := utils.VerifyMessageV2(message, pkSignature, pkSigner.Address().String())
	if err != nil {
		t.Fatalf("Failed to verify private key signature: %v", err)
	}
	if !valid {
		t.Fatal("Private key signature should be valid")
	}

	// Test HD wallet signer
	hdSignature, err := hdSigner.SignMessageV2(message)
	if err != nil {
		t.Fatalf("Failed to sign with HD wallet signer: %v", err)
	}

	valid, err = utils.VerifyMessageV2(message, hdSignature, hdSigner.Address().String())
	if err != nil {
		t.Fatalf("Failed to verify HD wallet signature: %v", err)
	}
	if !valid {
		t.Fatal("HD wallet signature should be valid")
	}

	// Cross-verify (should fail)
	valid, err = utils.VerifyMessageV2(message, pkSignature, hdSigner.Address().String())
	if err != nil {
		t.Fatalf("Failed to cross-verify: %v", err)
	}
	if valid {
		t.Fatal("Cross-verification should fail")
	}

	t.Log("Cross-signer verification test passed")
}