package event

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

// TestCanonicalSignature pins the exact join rule the registry hashes: the
// event name, "(", the comma-joined declared types, ")" — indexed flags and
// parameter names are not part of it.
func TestCanonicalSignature(t *testing.T) {
	cases := []struct {
		name  string
		types []string
		want  string
	}{
		{"Transfer", []string{"address", "address", "uint256"}, "Transfer(address,address,uint256)"},
		{"SubmitTransaction", []string{"uint256", "address", "uint256", "bytes"}, "SubmitTransaction(uint256,address,uint256,bytes)"},
		{"Ping", nil, "Ping()"},
	}
	for _, c := range cases {
		if got := CanonicalSignature(c.name, c.types); got != c.want {
			t.Errorf("CanonicalSignature(%q, %v) = %q, want %q", c.name, c.types, got, c.want)
		}
	}
}

// TestSignatureKeyMatchesKeccak locks the derivation to legacy keccak-256 and
// to sigKeyOf: the exported helper and the registry's internal key function
// must be the same value, or the tool and the registry drift.
func TestSignatureKeyMatchesKeccak(t *testing.T) {
	types := []string{"uint256", "address", "uint256", "bytes"}
	key := SignatureKey("SubmitTransaction", types)
	want := crypto.Keccak256([]byte("SubmitTransaction(uint256,address,uint256,bytes)"))
	if !bytes.Equal(key[:], want) {
		t.Fatalf("SignatureKey = %x, want %x", key, want)
	}
	if sigKey(key) != sigKeyOf("SubmitTransaction(uint256,address,uint256,bytes)") {
		t.Fatal("SignatureKey and sigKeyOf disagree")
	}
}

// TestDefinitionSignatureUsesCanonical guards the delegation: a Definition's
// signature string is the canonical form, not a second formatting rule.
func TestDefinitionSignatureUsesCanonical(t *testing.T) {
	def := &Definition{Name: "Transfer", Inputs: []ParamDef{
		{Type: "address", Indexed: true, Name: "from"},
		{Type: "address", Indexed: true, Name: "to"},
		{Type: "uint256", Name: "value"},
	}}
	want := CanonicalSignature("Transfer", []string{"address", "address", "uint256"})
	if got := def.signature(); got != want {
		t.Fatalf("Definition.signature() = %q, want %q", got, want)
	}
}
