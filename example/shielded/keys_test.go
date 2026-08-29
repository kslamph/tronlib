package main

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validKeysFile is a test-only, no-value key set: these are hand-written
// fixtures for deterministic validation, not credentials for anything.
func validKeysFile() shieldedKeys {
	return shieldedKeys{
		SK:             strings.Repeat("0a", 32),
		ASK:            strings.Repeat("0b", 32),
		NSK:            strings.Repeat("0c", 32),
		OVK:            strings.Repeat("0d", 32),
		AK:             strings.Repeat("0e", 32),
		NK:             strings.Repeat("0f", 32),
		IVK:            strings.Repeat("1a", 32),
		Diversifier:    strings.Repeat("1b", 11),
		PaymentAddress: "ztron1test",
		StartBlock:     59_808_727,
	}
}

func TestShieldedKeysValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*shieldedKeys)
		wantErr string
	}{
		{"a complete set is accepted", func(*shieldedKeys) {}, ""},
		{"missing payment address", func(k *shieldedKeys) { k.PaymentAddress = "" }, "missing paymentAddress"},
		{"short sk", func(k *shieldedKeys) { k.SK = strings.Repeat("0a", 31) }, "sk is 31 bytes, want 32"},
		{"oversized diversifier", func(k *shieldedKeys) { k.Diversifier = strings.Repeat("1b", 12) }, "diversifier is 12 bytes, want 11"},
		{"non-hex ivk", func(k *shieldedKeys) { k.IVK = "zz" }, "ivk is not valid hex"},
		{"empty ovk is caught, not sent as zero bytes", func(k *shieldedKeys) { k.OVK = "" }, "ovk is 0 bytes, want 32"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := validKeysFile()
			tt.mutate(&k)
			err := k.validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestShieldedKeysDecode(t *testing.T) {
	k := validKeysFile()
	kb, err := k.decode()
	require.NoError(t, err)

	assert.Equal(t, mustHex(t, k.IVK), kb.ivk)
	assert.Equal(t, mustHex(t, k.AK), kb.ak)
	assert.Equal(t, mustHex(t, k.NK), kb.nk)
	assert.Equal(t, mustHex(t, k.OVK), kb.ovk)
	assert.Equal(t, mustHex(t, k.ASK), kb.ask)
	assert.Equal(t, mustHex(t, k.NSK), kb.nsk)

	t.Run("a broken file never decodes", func(t *testing.T) {
		bad := validKeysFile()
		bad.ASK = strings.Repeat("0b", 8)
		_, err := bad.decode()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ask is 8 bytes, want 32")
	})
}

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	raw, err := hex.DecodeString(s)
	require.NoError(t, err)
	return raw
}

func TestLoadKeys(t *testing.T) {
	t.Run("missing file points at walletgen", func(t *testing.T) {
		_, err := loadKeys(filepath.Join(t.TempDir(), "absent.json"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "run -mode=walletgen first")
	})

	t.Run("malformed json names the file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "broken.json")
		require.NoError(t, os.WriteFile(path, []byte("{not json"), 0o600))
		_, err := loadKeys(path)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parse key file "+path)
	})

	t.Run("truncated key material is rejected before any RPC", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "short.json")
		k := validKeysFile()
		k.NSK = "0c0c"
		data, err := json.Marshal(k)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(path, data, 0o600))

		_, err = loadKeys(path)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nsk is 2 bytes, want 32")
		assert.Contains(t, err.Error(), path, "the error should say which file is broken")
	})

	t.Run("a round-tripped file loads", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "keys.json")
		k := validKeysFile()
		k.path = path
		require.NoError(t, k.saveKeys())

		loaded, err := loadKeys(path)
		require.NoError(t, err)
		assert.Equal(t, k.SK, loaded.SK)
		assert.Equal(t, k.PaymentAddress, loaded.PaymentAddress)
		assert.Equal(t, k.StartBlock, loaded.StartBlock)
		assert.NotEmpty(t, loaded.CreatedAt)
	})

	t.Run("the saved file is owner-only and carries no path field", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "keys.json")
		k := validKeysFile()
		k.path = path
		require.NoError(t, k.saveKeys())

		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

		var raw map[string]any
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(data, &raw))
		assert.NotContains(t, raw, "path", "the local path must never be serialised")
	})
}

func TestFingerprint(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want string
	}{
		{"empty", nil, ""},
		{"shorter than four bytes is shown whole", []byte{0x01, 0x02}, "0102"},
		{"four bytes", []byte{0xde, 0xad, 0xbe, 0xef}, "deadbeef..."},
		{"a full key shows only the head", []byte(strings.Repeat("\xaa", 32)), "aaaaaaaa..."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fingerprint(tt.in)
			assert.Equal(t, tt.want, got)
			assert.Less(t, len(got), 70, "a fingerprint must not be able to carry a whole key")
		})
	}
}
