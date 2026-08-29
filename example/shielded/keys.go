package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/client/lowlevel"
)

// shieldedKeys is the whole sapling key hierarchy for one shielded TRC-20
// address, stored as hex. The JSON shape matches the file the node-facing
// workflow produces, so an existing file can be reused as-is.
//
// Field roles (each is 32 bytes unless noted):
//
//	sk   spending key      the root secret; everything below derives from it
//	ask  authorizing key   signs spend authorizations
//	nsk  nullifier secret  derives nk, which blinds nullifiers
//	ovk  outgoing viewing  decrypts notes you sent
//	ak   authorizing pub   public half of ask
//	nk   nullifier key     public half of nsk
//	ivk  incoming viewing  decrypts notes you received; derived from ak + nk
//	d    diversifier       11 bytes, makes one ivk map to many addresses
//
// sk, ask, nsk, ovk and ivk are secrets: ak+nk derive ivk, and ivk or ovk
// reveals every balance and history. Treat all five as private and never write
// them to a shared log.
type shieldedKeys struct {
	SK             string `json:"sk"`
	ASK            string `json:"ask"`
	NSK            string `json:"nsk"`
	OVK            string `json:"ovk"`
	AK             string `json:"ak"`
	NK             string `json:"nk"`
	IVK            string `json:"ivk"`
	Diversifier    string `json:"diversifier"`
	PaymentAddress string `json:"paymentAddress"`
	CreatedAt      string `json:"createdAt"`
	// StartBlock records the chain head when the address was created, so a
	// later scan knows the earliest block that can contain its notes.
	StartBlock int64 `json:"startBlock,omitempty"`

	// path is where these keys were loaded from and should be saved back to.
	// Unexported, so it never lands in the JSON.
	path string
}

// keyBundle is the decoded form of the key material a flow needs. Passing one of
// these instead of four adjacent []byte arguments keeps ivk/ak/nk/ovk from being
// silently transposable at a call site, where only the node would notice.
type keyBundle struct {
	ivk []byte
	ak  []byte
	nk  []byte
	ovk []byte
	ask []byte
	nsk []byte
}

// bytes decodes a hex field into raw key material.
func (k *shieldedKeys) bytes(field, name string) ([]byte, error) {
	raw, err := hex.DecodeString(field)
	if err != nil {
		return nil, fmt.Errorf("%s in %s is not valid hex: %w", name, k.path, err)
	}
	return raw, nil
}

// decode validates the file and returns every key in raw form.
func (k *shieldedKeys) decode() (*keyBundle, error) {
	if err := k.validate(); err != nil {
		return nil, err
	}
	kb := &keyBundle{}
	for _, f := range []struct {
		name  string
		hex   string
		field *[]byte
	}{
		{"ivk", k.IVK, &kb.ivk},
		{"ak", k.AK, &kb.ak},
		{"nk", k.NK, &kb.nk},
		{"ovk", k.OVK, &kb.ovk},
		{"ask", k.ASK, &kb.ask},
		{"nsk", k.NSK, &kb.nsk},
	} {
		raw, err := k.bytes(f.hex, f.name)
		if err != nil {
			return nil, err
		}
		*f.field = raw
	}
	return kb, nil
}

// loadKeys reads and validates the key file.
func loadKeys(path string) (*shieldedKeys, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no key file at %s: run -mode=walletgen first", path)
		}
		return nil, fmt.Errorf("read key file %s: %w", path, err)
	}

	var k shieldedKeys
	if err := json.Unmarshal(raw, &k); err != nil {
		return nil, fmt.Errorf("parse key file %s: %w", path, err)
	}
	k.path = path
	if err := k.validate(); err != nil {
		return nil, fmt.Errorf("key file %s: %w", path, err)
	}
	return &k, nil
}

// validate checks that every field the flows need is present and the right
// length.
//
// Without this, a truncated file hex-decodes an absent field to empty bytes and
// the node then rejects the request with an opaque error, or worse, a spend is
// built with a zero-length ask.
func (k *shieldedKeys) validate() error {
	if k.PaymentAddress == "" {
		return fmt.Errorf("missing paymentAddress")
	}
	for _, f := range []struct {
		name string
		hex  string
		size int
	}{
		{"sk", k.SK, 32},
		{"ask", k.ASK, 32},
		{"nsk", k.NSK, 32},
		{"ovk", k.OVK, 32},
		{"ak", k.AK, 32},
		{"nk", k.NK, 32},
		{"ivk", k.IVK, 32},
		{"diversifier", k.Diversifier, 11},
	} {
		raw, err := hex.DecodeString(f.hex)
		if err != nil {
			return fmt.Errorf("%s is not valid hex: %w", f.name, err)
		}
		if len(raw) != f.size {
			return fmt.Errorf("%s is %d bytes, want %d", f.name, len(raw), f.size)
		}
	}
	return nil
}

// saveKeys writes the key file with owner-only permissions.
func (k *shieldedKeys) saveKeys() error {
	k.CreatedAt = time.Now().Format(time.RFC3339)
	data, err := json.MarshalIndent(k, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal keys: %w", err)
	}
	if err := os.WriteFile(k.path, data, 0o600); err != nil {
		return fmt.Errorf("write key file %s: %w", k.path, err)
	}
	fmt.Printf("keys written to %s\n", k.path)
	return nil
}

// fingerprint is how key material is reported by default: enough to tell two
// values apart, never enough to spend or de-anonymise anything. CODING_STANDARDS
// §8 forbids echoing private keys, and a terminal scrollback is a log.
func fingerprint(raw []byte) string {
	if len(raw) < 4 {
		return hex.EncodeToString(raw)
	}
	return hex.EncodeToString(raw[:4]) + "..."
}

// deriveKeys walks the seven RPC calls that build a shielded address.
//
// Each call is a pure key derivation, but they are chained through the node,
// which means the node sees sk, ask, nsk, ovk, ak, nk and ivk in sequence and
// can reconstruct the whole hierarchy. That is the cost of not doing sapling
// locally; see the trust section in docs/shielded.md.
//
// GetNewShieldedAddress performs all seven in a single RPC and returns a
// ShieldedAddressInfo with every field populated. The long form is spelled out
// here because the hierarchy is what you need to understand to use the rest of
// the API.
//
// Key material is printed as a fingerprint unless showKeys is set; the payment
// address is the only value here that is safe to paste anywhere.
func deriveKeys(ctx context.Context, cli *client.Client, showKeys bool) (*shieldedKeys, error) {
	k := &shieldedKeys{}
	report := func(step, label, field string) {
		raw, err := hex.DecodeString(field)
		if err != nil {
			fmt.Printf("%s %s  <undecodable>\n", step, label)
			return
		}
		value := fingerprint(raw)
		if showKeys {
			value = field
		}
		fmt.Printf("%s %s  %s\n", step, label, value)
	}

	// 1. sk: the root spending key.
	skResp, err := lowlevel.GetSpendingKey(cli, ctx, &api.EmptyMessage{})
	if err != nil {
		return nil, fmt.Errorf("getspendingkey: %w", err)
	}
	k.SK = hex.EncodeToString(skResp.GetValue())
	report("1. ", "sk   (spending key)     ", k.SK)

	// 2. expand sk into ask + nsk + ovk.
	expanded, err := lowlevel.GetExpandedSpendingKey(cli, ctx, &api.BytesMessage{Value: skResp.GetValue()})
	if err != nil {
		return nil, fmt.Errorf("getexpandedspendingkey: %w", err)
	}
	k.ASK = hex.EncodeToString(expanded.GetAsk())
	k.NSK = hex.EncodeToString(expanded.GetNsk())
	k.OVK = hex.EncodeToString(expanded.GetOvk())
	report("2. ", "ask  (authorizing key)  ", k.ASK)
	report("   ", "nsk  (nullifier secret) ", k.NSK)
	report("   ", "ovk  (outgoing viewing) ", k.OVK)

	// 3. ak: public half of ask, needed to verify spend authorizations.
	akResp, err := lowlevel.GetAkFromAsk(cli, ctx, &api.BytesMessage{Value: expanded.GetAsk()})
	if err != nil {
		return nil, fmt.Errorf("getakfromask: %w", err)
	}
	k.AK = hex.EncodeToString(akResp.GetValue())
	report("3. ", "ak   (authorizing pub)  ", k.AK)

	// 4. nk: public half of nsk, needed to compute nullifiers.
	nkResp, err := lowlevel.GetNkFromNsk(cli, ctx, &api.BytesMessage{Value: expanded.GetNsk()})
	if err != nil {
		return nil, fmt.Errorf("getnkfromnsk: %w", err)
	}
	k.NK = hex.EncodeToString(nkResp.GetValue())
	report("4. ", "nk   (nullifier key)    ", k.NK)

	// 5. ivk: from ak + nk. This is the key you scan the chain with.
	ivkResp, err := lowlevel.GetIncomingViewingKey(cli, ctx, &api.ViewingKeyMessage{
		Ak: akResp.GetValue(),
		Nk: nkResp.GetValue(),
	})
	if err != nil {
		return nil, fmt.Errorf("getincomingviewingkey: %w", err)
	}
	k.IVK = hex.EncodeToString(ivkResp.GetIvk())
	report("5. ", "ivk  (incoming viewing) ", k.IVK)

	// 6. d: an 11 byte diversifier, so one ivk can own many distinct addresses.
	divResp, err := lowlevel.GetDiversifier(cli, ctx, &api.EmptyMessage{})
	if err != nil {
		return nil, fmt.Errorf("getdiversifier: %w", err)
	}
	k.Diversifier = hex.EncodeToString(divResp.GetD())
	report("6. ", "d    (diversifier)      ", k.Diversifier)

	// 7. the ztron payment address, what you hand to a sender.
	payResp, err := lowlevel.GetZenPaymentAddress(cli, ctx, &api.IncomingViewingKeyDiversifierMessage{
		Ivk: &api.IncomingViewingKeyMessage{Ivk: ivkResp.GetIvk()},
		D:   &api.DiversifierMessage{D: divResp.GetD()},
	})
	if err != nil {
		return nil, fmt.Errorf("getzenpaymentaddress: %w", err)
	}
	k.PaymentAddress = payResp.GetPaymentAddress()
	fmt.Printf("7. payment address        %s\n", k.PaymentAddress)

	if k.PaymentAddress == "" {
		return nil, fmt.Errorf("getzenpaymentaddress returned an empty payment address")
	}
	return k, nil
}

// runWalletGen derives a fresh key hierarchy and stores it.
//
// An existing key file is left alone unless -force is passed: replacing it
// orphans every note owned by the previous address.
func runWalletGen(ctx context.Context, s *session) error {
	c := s.cfg
	if _, err := os.Stat(c.keyFile); err == nil {
		if !c.force {
			stored, err := loadKeys(c.keyFile)
			if err != nil {
				return err
			}
			fmt.Printf("keys already exist at %s\n", c.keyFile)
			fmt.Printf("payment address: %s\n", stored.PaymentAddress)
			fmt.Printf("pass -force to derive a new address instead\n")
			return nil
		}
		fmt.Printf("-force: replacing %s; notes owned by the old address become unreachable\n", c.keyFile)
	}

	head, err := currentBlock(ctx, s)
	if err != nil {
		return fmt.Errorf("resolve scan start height: %w", err)
	}

	derived, err := deriveKeys(ctx, s.cli, c.showKeys)
	if err != nil {
		return err
	}
	if err := derived.validate(); err != nil {
		return fmt.Errorf("node returned an unusable key set: %w", err)
	}
	derived.path = c.keyFile
	derived.StartBlock = head
	fmt.Printf("scan start block: %d\n", head)
	return derived.saveKeys()
}
