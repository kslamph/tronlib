package signer_test

import (
	"testing"

	"github.com/kslamph/tronlib/pkg/signer"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/kslamph/tronlib/pkg/utils"
)

func TestPrivateKeySigner(t *testing.T) {
	// Test private key (example - do not use in production)
	privateKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	
	// Create signer
	s, err := signer.NewPrivateKeySigner(privateKey)
	if err != nil {
		t.Fatalf("Failed to create private key signer: %v", err)
	}
	
	// Test address generation
	addr := s.Address()
	if addr == nil {
		t.Fatal("Address should not be nil")
	}
	
	// Test public key
	pubKey := s.PublicKey()
	if pubKey == nil {
		t.Fatal("Public key should not be nil")
	}
	
	// Test message signing
	message := "Hello TRON!"
	signature, err := s.SignMessageV2(message)
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}
	
	// Test message verification
	valid, err := utils.VerifyMessageV2(message, signature, addr.String())
	if err != nil {
		t.Fatalf("Failed to verify message: %v", err)
	}
	
	if !valid {
		t.Fatal("Message verification should be valid")
	}
	
	t.Logf("Private key signer test passed!")
	t.Logf("Address: %s", addr.String())
	t.Logf("Signature: %s", signature)
}

func TestHDWalletSigner(t *testing.T) {
	// Test mnemonic (example - do not use in production)
	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	
	// Create HD wallet signer
	s, err := signer.NewHDWalletSigner(mnemonic, "", "")
	if err != nil {
		t.Fatalf("Failed to create HD wallet signer: %v", err)
	}
	
	// Test address generation
	addr := s.Address()
	if addr == nil {
		t.Fatal("Address should not be nil")
	}
	
	// Test derivation path
	path := s.DerivationPath()
	if path != "m/44'/195'/0'/0/0" {
		t.Fatalf("Expected default derivation path, got: %s", path)
	}
	
	// Test account derivation
	account1, err := s.DeriveAccount(1)
	if err != nil {
		t.Fatalf("Failed to derive account 1: %v", err)
	}
	
	// Addresses should be different
	if account1.Address().String() == addr.String() {
		t.Fatal("Derived account should have different address")
	}
	
	// Test message signing with derived account
	message := "Hello from HD Wallet!"
	signature, err := account1.SignMessageV2(message)
	if err != nil {
		t.Fatalf("Failed to sign message with derived account: %v", err)
	}
	
	// Test message verification
	valid, err := utils.VerifyMessageV2(message, signature, account1.Address().String())
	if err != nil {
		t.Fatalf("Failed to verify message: %v", err)
	}
	
	if !valid {
		t.Fatal("Message verification should be valid")
	}
	
	t.Logf("HD wallet signer test passed!")
	t.Logf("Main address: %s", addr.String())
	t.Logf("Account 1 address: %s", account1.Address().String())
	t.Logf("Account 1 signature: %s", signature)
}

func TestSignerInterface(t *testing.T) {
	// Test that both implementations satisfy the Signer interface
	var signers []types.Signer
	
	// Private key signer
	privateKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	pkSigner, err := signer.NewPrivateKeySigner(privateKey)
	if err != nil {
		t.Fatalf("Failed to create private key signer: %v", err)
	}
	signers = append(signers, pkSigner)
	
	// HD wallet signer
	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	hdSigner, err := signer.NewHDWalletSigner(mnemonic, "", "")
	if err != nil {
		t.Fatalf("Failed to create HD wallet signer: %v", err)
	}
	signers = append(signers, hdSigner)
	
	// Test interface methods on both
	for i, s := range signers {
		t.Logf("Testing signer %d", i)
		
		// Test Address method
		addr := s.Address()
		if addr == nil {
			t.Fatalf("Signer %d: Address should not be nil", i)
		}
		
		// Test PublicKey method
		pubKey := s.PublicKey()
		if pubKey == nil {
			t.Fatalf("Signer %d: Public key should not be nil", i)
		}
		
		// Test SignMessageV2 method
		message := "Interface test message"
		signature, err := s.SignMessageV2(message)
		if err != nil {
			t.Fatalf("Signer %d: Failed to sign message: %v", i, err)
		}
		
		// Verify signature
		valid, err := utils.VerifyMessageV2(message, signature, addr.String())
		if err != nil {
			t.Fatalf("Signer %d: Failed to verify message: %v", i, err)
		}
		
		if !valid {
			t.Fatalf("Signer %d: Message verification should be valid", i)
		}
		
		t.Logf("Signer %d passed interface test", i)
	}
}