package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/client/lowlevel"
)

// ShieldedKeys holds all the necessary keys for shielded operations
type ShieldedKeys struct {
	SK             string `json:"sk"`             // spending key
	ASK            string `json:"ask"`            // ask key
	NSK            string `json:"nsk"`            // nsk key
	OVK            string `json:"ovk"`            // outgoing viewing key
	AK             string `json:"ak"`             // ak key
	NK             string `json:"nk"`             // nk key
	IVK            string `json:"ivk"`            // incoming viewing key
	Diversifier    string `json:"diversifier"`    // diversifier
	PaymentAddress string `json:"paymentAddress"` // shielded payment address
	CreatedAt      string `json:"createdAt"`      // timestamp when keys were created
}

// saveKeys saves the shielded keys to a JSON file
func saveKeys(keys *ShieldedKeys) error {
	keys.CreatedAt = time.Now().Format(time.RFC3339)
	data, err := json.MarshalIndent(keys, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal keys: %v", err)
	}

	err = os.WriteFile(KeyFile, data, 0600) // Read/write for owner only
	if err != nil {
		return fmt.Errorf("failed to write key file: %v", err)
	}

	fmt.Printf("✅ Saved shielded keys to %s\n", KeyFile)
	return nil
}

// loadKeys loads the shielded keys from a JSON file
func loadKeys() (*ShieldedKeys, error) {
	data, err := os.ReadFile(KeyFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("key file does not exist")
		}
		return nil, fmt.Errorf("failed to read key file: %v", err)
	}

	var keys ShieldedKeys
	err = json.Unmarshal(data, &keys)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal keys: %v", err)
	}

	fmt.Printf("✅ Loaded existing shielded keys from %s (created: %s)\n", KeyFile, keys.CreatedAt)
	return &keys, nil
}

// keysExist checks if the key file exists
func keysExist() bool {
	_, err := os.Stat(KeyFile)
	return !os.IsNotExist(err)
}

// clearKeys removes the saved key file (useful for testing with new keys)
func clearKeys() error {
	if !keysExist() {
		return fmt.Errorf("key file does not exist")
	}

	err := os.Remove(KeyFile)
	if err != nil {
		return fmt.Errorf("failed to remove key file: %v", err)
	}

	fmt.Printf("🗑️  Cleared shielded keys from %s\n", KeyFile)
	return nil
}

// generateShieldedKeys generates new shielded keys using the TRON node
func generateShieldedKeys(cli *client.Client, ctx context.Context) (*ShieldedKeys, error) {
	fmt.Println("\n🔑 Generating new shielded keys...")

	// Step 1: Generate spending key (sk)
	fmt.Println("Step 1: Generating spending key...")
	spendingKeyResp, err := lowlevel.GetSpendingKey(cli, ctx, &api.EmptyMessage{})
	if err != nil {
		return nil, fmt.Errorf("failed to generate spending key: %v", err)
	}
	sk := spendingKeyResp.GetValue()
	fmt.Printf("Generated spending key (sk): %x\n", sk)

	// Step 2: Generate expanded spending key (ask, nsk, ovk)
	fmt.Println("Step 2: Generating expanded spending key...")
	expandedKeyResp, err := lowlevel.GetExpandedSpendingKey(cli, ctx, &api.BytesMessage{Value: sk})
	if err != nil {
		return nil, fmt.Errorf("failed to generate expanded spending key: %v", err)
	}
	ask := expandedKeyResp.GetAsk()
	nsk := expandedKeyResp.GetNsk()
	ovk := expandedKeyResp.GetOvk()
	fmt.Printf("Generated ask: %x\n", ask)
	fmt.Printf("Generated nsk: %x\n", nsk)
	fmt.Printf("Generated ovk: %x\n", ovk)

	// Step 3: Generate ak from ask
	fmt.Println("Step 3: Generating ak from ask...")
	akResp, err := lowlevel.GetAkFromAsk(cli, ctx, &api.BytesMessage{Value: ask})
	if err != nil {
		return nil, fmt.Errorf("failed to generate ak: %v", err)
	}
	ak := akResp.GetValue()
	fmt.Printf("Generated ak: %x\n", ak)

	// Step 4: Generate nk from nsk
	fmt.Println("Step 4: Generating nk from nsk...")
	nkResp, err := lowlevel.GetNkFromNsk(cli, ctx, &api.BytesMessage{Value: nsk})
	if err != nil {
		return nil, fmt.Errorf("failed to generate nk: %v", err)
	}
	nk := nkResp.GetValue()
	fmt.Printf("Generated nk: %x\n", nk)

	// Step 5: Generate incoming viewing key (ivk)
	fmt.Println("Step 5: Generating incoming viewing key...")
	ivkResp, err := lowlevel.GetIncomingViewingKey(cli, ctx, &api.ViewingKeyMessage{
		Ak: ak,
		Nk: nk,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate ivk: %v", err)
	}
	ivk := ivkResp.GetIvk()
	fmt.Printf("Generated ivk: %x\n", ivk)

	// Step 6: Generate diversifier (d)
	fmt.Println("Step 6: Generating diversifier...")
	diversifierResp, err := lowlevel.GetDiversifier(cli, ctx, &api.EmptyMessage{})
	if err != nil {
		return nil, fmt.Errorf("failed to generate diversifier: %v", err)
	}
	d := diversifierResp.GetD()
	fmt.Printf("Generated diversifier (d): %x\n", d)

	// Step 7: Generate payment address
	fmt.Println("Step 7: Generating shielded payment address...")
	paymentAddrResp, err := lowlevel.GetZenPaymentAddress(cli, ctx, &api.IncomingViewingKeyDiversifierMessage{
		Ivk: &api.IncomingViewingKeyMessage{Ivk: ivk},
		D:   &api.DiversifierMessage{D: d},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate payment address: %v", err)
	}
	paymentAddress := paymentAddrResp.GetPaymentAddress()
	fmt.Printf("Generated shielded payment address: %s\n", paymentAddress)

	// Create the keys structure
	keys := &ShieldedKeys{
		SK:             hex.EncodeToString(sk),
		ASK:            hex.EncodeToString(ask),
		NSK:            hex.EncodeToString(nsk),
		OVK:            hex.EncodeToString(ovk),
		AK:             hex.EncodeToString(ak),
		NK:             hex.EncodeToString(nk),
		IVK:            hex.EncodeToString(ivk),
		Diversifier:    hex.EncodeToString(d),
		PaymentAddress: paymentAddress,
	}

	return keys, nil
}

// loadOrGenerateKeys loads existing keys or generates new ones if they don't exist
func loadOrGenerateKeys(cli *client.Client, ctx context.Context) (*ShieldedKeys, []byte, []byte, []byte, []byte, []byte, []byte, []byte, []byte, string, error) {
	var keys *ShieldedKeys
	var err error

	if keysExist() {
		fmt.Println("\n🔑 Loading existing shielded keys...")
		keys, err = loadKeys()
		if err != nil {
			log.Printf("Failed to load existing keys: %v", err)
			fmt.Println("Generating new keys instead...")
			keys = nil
		}
	}

	// Generate new keys if we don't have existing ones
	if keys == nil {
		keys, err = generateShieldedKeys(cli, ctx)
		if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, nil, "", fmt.Errorf("failed to generate keys: %v", err)
		}

		// Save the new keys
		err = saveKeys(keys)
		if err != nil {
			log.Printf("Failed to save keys: %v", err)
		}
	}

	// Convert hex strings back to bytes
	sk, _ := hex.DecodeString(keys.SK)
	ask, _ := hex.DecodeString(keys.ASK)
	nsk, _ := hex.DecodeString(keys.NSK)
	ovk, _ := hex.DecodeString(keys.OVK)
	ak, _ := hex.DecodeString(keys.AK)
	nk, _ := hex.DecodeString(keys.NK)
	ivk, _ := hex.DecodeString(keys.IVK)
	d, _ := hex.DecodeString(keys.Diversifier)

	return keys, sk, ask, nsk, ovk, ak, nk, ivk, d, keys.PaymentAddress, nil
}

// validateAndVerifyKeys validates key lengths and verifies IVK derivation
func validateAndVerifyKeys(cli *client.Client, ctx context.Context, ivk, ak, nk []byte) error {
	// Validate key lengths
	if len(ivk) != 32 {
		return fmt.Errorf("invalid IVK length: expected 32 bytes, got %d", len(ivk))
	}
	if len(ak) != 32 {
		return fmt.Errorf("invalid AK length: expected 32 bytes, got %d", len(ak))
	}
	if len(nk) != 32 {
		return fmt.Errorf("invalid NK length: expected 32 bytes, got %d", len(nk))
	}

	// Verify IVK is correctly derived from AK and NK
	fmt.Println("\nVerifying IVK derivation...")
	verifyIvkResp, err := lowlevel.GetIncomingViewingKey(cli, ctx, &api.ViewingKeyMessage{
		Ak: ak,
		Nk: nk,
	})
	if err != nil {
		log.Printf("Warning: Could not verify IVK derivation: %v", err)
		return nil
	}

	expectedIvk := verifyIvkResp.GetIvk()
	if !bytes.Equal(ivk, expectedIvk) {
		return fmt.Errorf("IVK mismatch! Stored: %x, Expected: %x", ivk, expectedIvk)
	}

	fmt.Println("✅ IVK derivation verified correctly")
	return nil
}
