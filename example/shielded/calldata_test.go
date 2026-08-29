package main

import (
	"encoding/hex"
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kslamph/tronlib/pkg/types"
)

// TestCalldataSelectors pins the three contract signatures the flows use. A
// typo in a signature string produces a valid-length but meaningless selector,
// so these are checked against the selectors the deployed ShieldedTRC20 bytecode
// actually exposes: each expected value below also appears in
// cmd/setup_nile_testnet/test_contract/build/ShieldedTRC20.bin.
func TestCalldataSelectors(t *testing.T) {
	tests := []struct {
		signature string
		want      string
	}{
		{sigMint, "855d175e"},
		{sigTransfer, "9110a55b"},
		{sigBurn, "cc105875"},
		{"scalingFactor()", "ed3437f8"},
		{"getPath(uint256)", "e1765073"},
	}
	for _, tt := range tests {
		t.Run(tt.signature, func(t *testing.T) {
			data := calldata(tt.signature)
			assert.Len(t, data, 4)
			assert.Equal(t, tt.want, hex.EncodeToString(data))
		})
	}
}

func TestCalldataAppendsArguments(t *testing.T) {
	arg := make([]byte, 32)
	arg[31] = 7
	data := calldata("approve(address,uint256)", arg, arg)
	assert.Len(t, data, 4+64)
	assert.Equal(t, arg, data[len(data)-32:])
}

func TestAbiUint256(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantHex string
		wantErr bool
	}{
		{"zero", "0", strings.Repeat("00", 31) + "00", false},
		{"one is right aligned", "1", strings.Repeat("00", 31) + "01", false},
		{"a shielded amount", "10000000", strings.Repeat("00", 28) + "00989680", false},
		{"max uint256 fits", new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1)).String(),
			strings.Repeat("ff", 32), false},
		{"one over uint256 is an error", new(big.Int).Lsh(big.NewInt(1), 256).String(), "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := abiUint256("amount", bi(tt.value))
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "does not fit in 32 bytes")
				return
			}
			require.NoError(t, err)
			assert.Len(t, got, 32)
			assert.Equal(t, tt.wantHex, hex.EncodeToString(got))
		})
	}
}

// TestAbiAddressWord guards the 0x41 trap: TRON addresses are 21 bytes with a
// 0x41 prefix, Solidity wants the 20-byte EVM form right-aligned. Using the
// prefixed form does not revert, it silently addresses something else.
func TestAbiAddressWord(t *testing.T) {
	addr, err := types.NewAddressFromBase58("TWRvzd6FQcsyp7hwCtttjZGpU1kfvVEtNK")
	require.NoError(t, err)

	word := abiAddressWord(addr)
	require.Len(t, word, 32)
	assert.Equal(t, make([]byte, 12), word[:12], "left padding must be zero")
	assert.Equal(t, addr.BytesEVM(), word[12:], "the EVM form must be right-aligned")
	assert.NotEqual(t, byte(0x41), word[12], "the 0x41 TRON prefix must not appear in the word")
}

func TestDecodeHex(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr string
	}{
		{"plain", "00ff", "00ff", ""},
		{"0x prefix tolerated", "0x00ff", "00ff", ""},
		{"whitespace tolerated", " 00ff\n", "00ff", ""},
		{"odd length", "00f", "", "not valid hex"},
		{"empty is refused", "", "", "is empty"},
		{"only a prefix is refused", "0x", "", "is empty"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeHex(tt.in, "trigger_contract_input")
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, hex.EncodeToString(got))
		})
	}
}

// errorBlob assembles a Solidity Error(string) return value.
func errorBlob(msg string) []byte {
	out := append([]byte{}, errorSelector...)
	out = append(out, make([]byte, 32)...)
	out[len(out)-1] = 0x20 // offset
	lenWord, _ := abiUint256("len", big.NewInt(int64(len(msg))))
	out = append(out, lenWord...)
	padded := make([]byte, (len(msg)+31)/32*32)
	copy(padded, msg)
	return append(out, padded...)
}

func TestRevertReason(t *testing.T) {
	panicBlob := append([]byte{}, panicSelector...)
	panicBlob = append(panicBlob, make([]byte, 32)...)
	panicBlob[len(panicBlob)-1] = 0x11

	tests := []struct {
		name         string
		data         []byte
		wantReason   string
		wantIsRevert bool
	}{
		{"short data is not a revert", []byte{0x08, 0xc3}, "", false},
		{"plain return data is not a revert", make([]byte, 32), "", false},
		{"error string", errorBlob("Position should be smaller than leafCount!"),
			"Position should be smaller than leafCount!", true},
		{"error string with no payload", errorBlob(""), "unspecified contract error", true},
		{"truncated error string header", errorSelector, "unspecified contract error", true},
		{"panic code", panicBlob, "panic code 17", true},
		{"truncated panic", panicBlob[:20], "contract panic", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason, ok := revertReason(tt.data)
			assert.Equal(t, tt.wantIsRevert, ok)
			assert.Equal(t, tt.wantReason, reason)
		})
	}
}

// TestRevertReason_HugeLengthWordDoesNotPanic is the regression guard for the
// untrusted uint256 length in an Error(string) blob: converting it to int
// without bounding it wrapped on 64 bits and sliced past the end of the buffer.
func TestRevertReason_HugeLengthWordDoesNotPanic(t *testing.T) {
	blob := errorBlob("short")
	// Overwrite the length word with 2^64 + 5, which truncates to 5 if the
	// conversion is careless.
	for i := 36; i < 68; i++ {
		blob[i] = 0xff
	}
	require.NotPanics(t, func() {
		reason, ok := revertReason(blob)
		assert.True(t, ok)
		assert.Equal(t, "unspecified contract error", reason)
	})

	// A length that claims more than the buffer holds is also refused.
	blob = errorBlob("short")
	lenWord, _ := abiUint256("len", big.NewInt(int64(len(blob)*10)))
	copy(blob[36:68], lenWord)
	reason, ok := revertReason(blob)
	assert.True(t, ok)
	assert.Equal(t, "unspecified contract error", reason)
}
