package key

import (
	"reflect"
	"strings"
	"testing"

	"github.com/kslamph/tronlib/v2/tron"
)

// Fixture pair ported from v1 pkg/signer/signer_test.go:58-59, hex address
// confirmed by running v1's NewPrivateKeySigner against the same key.
const (
	fixtureHexKey  = "cfae06d915cf9784272fa99d4db961b8cbafd59c8b2f77ab7422be5424d3e49c"
	fixtureHexAddr = "419d242ae8cb425d78ff1f51a808af38bac5e07ef3" // TQJ6R9SPvD5SyqgYqTBq3yc6mFtEgatPDu
)

func TestPrivateKeyFromHexDerivesFixtureAddress(t *testing.T) {
	s, err := PrivateKeyFromHex(fixtureHexKey)
	if err != nil {
		t.Fatalf("PrivateKeyFromHex: %v", err)
	}
	want, err := tron.AddressFromHex(fixtureHexAddr)
	if err != nil {
		t.Fatalf("fixture address: %v", err)
	}
	if got := s.Address(); got != want {
		t.Fatalf("Address() = %s, want %s", got, want)
	}
}

func TestPrivateKeyFromHexZeroXPrefix(t *testing.T) {
	s1, err := PrivateKeyFromHex(fixtureHexKey)
	if err != nil {
		t.Fatalf("PrivateKeyFromHex: %v", err)
	}
	s2, err := PrivateKeyFromHex("0x" + fixtureHexKey)
	if err != nil {
		t.Fatalf("PrivateKeyFromHex(0x): %v", err)
	}
	if s1.Address() != s2.Address() {
		t.Fatalf("0x prefix changed address: %s vs %s", s1.Address(), s2.Address())
	}
}

func TestPrivateKeyFromHexBadHex(t *testing.T) {
	cases := map[string]string{
		"odd length":   "abc",
		"non-hex":      "zzzz",
		"empty":        "",
		"too short":    "abcd",
		"out of range": strings.Repeat("ff", 33),
	}
	for name, in := range cases {
		_, err := PrivateKeyFromHex(in)
		if err == nil {
			t.Fatalf("%s: expected error", name)
		}
		if !tron.HasCode(err, tron.CodeKeyInvalid) {
			t.Fatalf("%s: HasCode(CodeKeyInvalid) = false, err = %v", name, err)
		}
	}
}

// TestMnemonicNotRetained: the signer must not keep the mnemonic phrase in
// memory — a heap/core dump would otherwise leak the backup phrase (every
// derived account), not just one key. Only the derived key + address are kept.
func TestMnemonicNotRetained(t *testing.T) {
	const m = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	s, err := PrivateKeyFromMnemonic(m, "", "m/44'/195'/0'/0/0")
	if err != nil {
		t.Fatalf("PrivateKeyFromMnemonic: %v", err)
	}
	v := reflect.ValueOf(s).Elem()
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if f.Kind() == reflect.String && strings.Contains(f.String(), "abandon") {
			t.Errorf("signer field %s retains the mnemonic phrase", v.Type().Field(i).Name)
		}
	}
}

// TestZeroBytes pins the scrub helper used on intermediate secret buffers.
func TestZeroBytes(t *testing.T) {
	b := []byte{1, 2, 3, 4}
	zero(b)
	for i, v := range b {
		if v != 0 {
			t.Fatalf("byte %d = %d after zero, want 0", i, v)
		}
	}
}

func TestPrivateKeyFromMnemonic(t *testing.T) {
	m := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	s, err := PrivateKeyFromMnemonic(m, "", "m/44'/195'/0'/0/0")
	if err != nil {
		t.Fatalf("PrivateKeyFromMnemonic: %v", err)
	}
	a := s.Address()
	if a.IsZero() {
		t.Fatal("zero address")
	}
	if len(a.String()) != 34 || a.String()[0] != 'T' {
		t.Fatalf("not a TRON base58 address: %q", a.String())
	}
}

func TestPrivateKeyFromMnemonicSameInputSameAddress(t *testing.T) {
	m := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	s1, err := PrivateKeyFromMnemonic(m, "pw", "m/44'/195'/0'/0/0")
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	s2, err := PrivateKeyFromMnemonic(m, "pw", "m/44'/195'/0'/0/0")
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if s1.Address() != s2.Address() {
		t.Fatalf("same mnemonic+passphrase+path gave different addresses: %s vs %s", s1.Address(), s2.Address())
	}
}

func TestPrivateKeyFromMnemonicBad(t *testing.T) {
	m := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	cases := []struct {
		name               string
		mnemonic, pass, ph string
	}{
		{"bad words", "not a real mnemonic phrase at all", "", "m/44'/195'/0'/0/0"},
		{"empty mnemonic", "", "", "m/44'/195'/0'/0/0"},
		{"bad checksum", "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon", "", "m/44'/195'/0'/0/0"},
		{"bad path", m, "", "m/44'/195'/%'/0/0"},
	}
	for _, tc := range cases {
		_, err := PrivateKeyFromMnemonic(tc.mnemonic, tc.pass, tc.ph)
		if err == nil {
			t.Fatalf("%s: expected error", tc.name)
		}
		if !tron.HasCode(err, tron.CodeKeyMnemonicInvalid) {
			t.Fatalf("%s: HasCode(CodeKeyMnemonicInvalid) = false, err = %v", tc.name, err)
		}
	}
}
