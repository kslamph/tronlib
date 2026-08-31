package tron

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Pairs taken from v1 pkg/types/address_test.go — same fixtures, new type.
var validPairs = []struct{ base58, hex20 string }{
	{"TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb", "e28b3cfd4e0e909077821478e9fcb86b84be786e"},
	{"TXNYeYdao7JL7wBtmzbk7mAie7UZsdgVjx", "eac49bc766be29be1b6d36619eff8f86ed4d04df"},
}

func TestParseAddressRoundTrip(t *testing.T) {
	for _, tc := range validPairs {
		a, err := ParseAddress(tc.base58)
		require.NoError(t, err, tc.base58)
		assert.Equal(t, tc.base58, a.String())

		got20 := hex.EncodeToString(a.Bytes()[1:]) // strip 0x41
		assert.Equal(t, tc.hex20, got20)
	}
}

func TestParseAddressRejects(t *testing.T) {
	cases := []struct {
		name, in string
		errCode  Code
	}{
		{"empty", "", CodeAddressInvalid},
		{"wrong length", "TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jw", CodeAddressInvalid},
		{"not base58", "TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb0OIl", CodeAddressInvalid}, // 0,O,I,l excluded from alphabet
		{"bad checksum", "TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwc", CodeAddressInvalid},
		{"wrong prefix byte", "1BvBMSEYstWetqTFn5Au4m4GFg7xJaNVN2", CodeAddressWrongPrefix}, // bitcoin-style, decodes but no 0x41
	}
	for _, tc := range cases {
		_, err := ParseAddress(tc.in)
		if assert.Error(t, err, tc.name) {
			assert.True(t, HasCode(err, tc.errCode), "%s: want %v, got %v", tc.name, tc.errCode, err)
		}
	}
}

func TestParseAddressErrorCode(t *testing.T) {
	// invalid base58 / bad checksum -> address.invalid
	_, err := ParseAddress("TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwc")
	require.Error(t, err)
	assert.True(t, HasCode(err, CodeAddressInvalid), "bad checksum must be CodeAddressInvalid, got %v", err)

	// decodes fine but prefix is not 0x41 -> address.wrong_prefix
	_, err = ParseAddress("1BvBMSEYstWetqTFn5Au4m4GFg7xJaNVN2")
	require.Error(t, err)
	assert.True(t, HasCode(err, CodeAddressWrongPrefix), "non-0x41 prefix must be CodeAddressWrongPrefix, got %v", err)
}

func TestAddressBytesIsCopy(t *testing.T) {
	a, _ := ParseAddress("TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb")
	b := a.Bytes()
	b[0] = 0x00 // mutate the returned slice
	assert.Equal(t, byte(0x41), a.Bytes()[0], "Bytes() must return a copy")
}

func TestAddressFromHexForms(t *testing.T) {
	a1, err := AddressFromHex("41e28b3cfd4e0e909077821478e9fcb86b84be786e")
	require.NoError(t, err)
	a2, err := AddressFromHex("e28b3cfd4e0e909077821478e9fcb86b84be786e")
	require.NoError(t, err)
	assert.Equal(t, a1, a2, "0x41-prefixed and bare forms are the same address")
}

func TestAddressHex(t *testing.T) {
	a, err := ParseAddress("TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb")
	require.NoError(t, err)
	assert.Equal(t, "41e28b3cfd4e0e909077821478e9fcb86b84be786e", a.Hex())
}

func TestAddressFromBytes(t *testing.T) {
	raw, _ := hex.DecodeString("41e28b3cfd4e0e909077821478e9fcb86b84be786e")
	a, err := AddressFromBytes(raw)
	require.NoError(t, err)
	assert.Equal(t, "TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb", a.String())

	// caller mutation of the input slice must not affect the address
	raw[0] = 0x00
	assert.Equal(t, byte(0x41), a.Bytes()[0], "AddressFromBytes must copy the input")

	// wrong length
	_, err = AddressFromBytes(raw[:20])
	assert.Error(t, err)
	// wrong prefix (right length, wrong first byte)
	_, err = AddressFromBytes(make([]byte, 21))
	assert.True(t, HasCode(err, CodeAddressWrongPrefix), "non-0x41 prefix must be CodeAddressWrongPrefix, got %v", err)
}

func TestMustAddress(t *testing.T) {
	assert.NotPanics(t, func() { MustAddress("TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb") })
	assert.Panics(t, func() { MustAddress("not an address") })
}

func TestAddressZeroValue(t *testing.T) {
	var a Address
	assert.True(t, a.IsZero())
	assert.Error(t, func() error { _, err := ParseAddress(""); return err }())
	_, err := ParseAddress("")
	assert.Error(t, err)
}

func TestAddressFromBytesDoesNotAliasCallerBuffer(t *testing.T) {
	body, _ := hex.DecodeString("41e28b3cfd4e0e909077821478e9fcb86b84be786e")
	big := make([]byte, 0, 64)                   // cap > len everywhere downstream
	big = append(big, body...)                   // body has len 21, cap 43+
	big = append(big, make([]byte, 22)...)       // spare capacity the old append could scribble into
	sentinel := append([]byte(nil), big[21:]...) // bytes after the 21-byte body

	a, err := AddressFromBytes(big[:21])
	require.NoError(t, err)
	require.Equal(t, "TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb", a.String())

	assert.Equal(t, sentinel, big[21:], "AddressFromBytes must not write past the 21-byte body into the caller's backing array")
}
