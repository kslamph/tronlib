package tron

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
)

// Address is a TRON address: 21 bytes, 0x41-prefixed.
// Value semantics; the zero value is the unset address (IsZero).
type Address struct {
	base58 string
	b      [21]byte
}

// ParseAddress parses a base58check-encoded TRON address ("T...", 34 chars).
// It is the primary constructor: base58 is the form users see and paste.
func ParseAddress(s string) (Address, error) {
	dec, err := base58Decode(s)
	if err != nil {
		return Address{}, &Error{Code: CodeAddressInvalid, Op: "ParseAddress", Hint: "pass a 34-character base58 address starting with T", Cause: err}
	}
	if len(dec) != 25 {
		return Address{}, &Error{Code: CodeAddressInvalid, Op: "ParseAddress", Hint: fmt.Sprintf("decoded length %d, want 25", len(dec))}
	}
	body, sum := dec[:21], dec[21:]
	h1 := sha256.Sum256(body)
	h2 := sha256.Sum256(h1[:])
	if h2[0] != sum[0] || h2[1] != sum[1] || h2[2] != sum[2] || h2[3] != sum[3] {
		return Address{}, &Error{Code: CodeAddressInvalid, Op: "ParseAddress", Hint: "checksum mismatch"}
	}
	if body[0] != 0x41 {
		return Address{}, &Error{Code: CodeAddressWrongPrefix, Op: "ParseAddress", Hint: fmt.Sprintf("prefix 0x%02x, want 0x41", body[0])}
	}
	var a Address
	a.base58 = s
	copy(a.b[:], body)
	return a, nil
}

// MustAddress parses s and panics on error. For package-level address
// literals in tests and examples; anything that handles user input must
// use ParseAddress.
func MustAddress(s string) Address {
	a, err := ParseAddress(s)
	if err != nil {
		panic(err)
	}
	return a
}

// AddressFromBytes builds an Address from its 21-byte wire form,
// 0x41-prefixed. It copies b; later mutation of the caller's slice does not
// affect the returned value.
func AddressFromBytes(b []byte) (Address, error) {
	if len(b) != 21 {
		return Address{}, &Error{Code: CodeAddressInvalid, Op: "AddressFromBytes", Hint: fmt.Sprintf("got %d bytes, want 21 (0x41 prefix + 20-byte payload)", len(b))}
	}
	if b[0] != 0x41 {
		return Address{}, &Error{Code: CodeAddressWrongPrefix, Op: "AddressFromBytes", Hint: fmt.Sprintf("prefix 0x%02x, want 0x41", b[0])}
	}
	var a Address
	copy(a.b[:], b)
	a.base58 = base58CheckEncode(b)
	return a, nil
}

// AddressFromHex builds an Address from hex, accepting either the 42-character
// 41-prefixed form or the bare 40-character payload; both are the same Address.
func AddressFromHex(hex21 string) (Address, error) {
	const op = "AddressFromHex"
	hint := "pass 40 hex characters, optionally prefixed with 41"
	var payload []byte
	switch len(hex21) {
	case 42:
		b, err := hex.DecodeString(hex21)
		if err != nil {
			return Address{}, &Error{Code: CodeAddressInvalid, Op: op, Hint: hint, Cause: err}
		}
		if b[0] != 0x41 {
			return Address{}, &Error{Code: CodeAddressWrongPrefix, Op: op, Hint: fmt.Sprintf("prefix 0x%02x, want 0x41", b[0])}
		}
		payload = b[1:]
	case 40:
		b, err := hex.DecodeString(hex21)
		if err != nil {
			return Address{}, &Error{Code: CodeAddressInvalid, Op: op, Hint: hint, Cause: err}
		}
		payload = b
	default:
		return Address{}, &Error{Code: CodeAddressInvalid, Op: op, Hint: fmt.Sprintf("got %d hex characters; %s", len(hex21), hint)}
	}
	var a Address
	a.b[0] = 0x41
	copy(a.b[1:], payload)
	a.base58 = base58CheckEncode(a.b[:])
	return a, nil
}

// String returns the base58check form.
func (a Address) String() string { return a.base58 }

// Bytes returns a copy of the 21-byte wire form, 0x41-prefixed.
// Mutating the result does not affect the Address.
func (a Address) Bytes() []byte { return append([]byte(nil), a.b[:]...) }

// Hex returns the 0x41-prefixed hex form (42 characters).
func (a Address) Hex() string { return hex.EncodeToString(a.b[:]) }

// IsZero reports whether a is the unset address.
func (a Address) IsZero() bool { return a == Address{} }

// b58Alphabet is the base58 alphabet: digits and letters minus 0, O, I, l —
// the visually ambiguous characters.
const b58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

// b58Index maps a byte to its b58Alphabet position, or -1 when the byte is
// not a base58 character. Built once in init so validation is a table lookup.
var b58Index [256]int8

func init() {
	for i := range b58Index {
		b58Index[i] = -1
	}
	for i := 0; i < len(b58Alphabet); i++ {
		b58Index[b58Alphabet[i]] = int8(i)
	}
}

// isValidB58 reports whether c is a base58 character.
func isValidB58(c byte) bool { return b58Index[c] >= 0 }

// base58Decode decodes s strictly: every byte must be in the alphabet.
// Leading '1's decode to leading zero bytes.
func base58Decode(s string) ([]byte, error) {
	if len(s) == 0 {
		return nil, errors.New("empty input")
	}
	// Strict alphabet check first, so "not base58" is a distinct error from "too short".
	for i := 0; i < len(s); i++ {
		if !isValidB58(s[i]) {
			return nil, fmt.Errorf("invalid base58 character %q", s[i])
		}
	}
	num := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		carry := int64(b58Index[s[i]])
		for j := len(num) - 1; j >= 0; j-- {
			carry += int64(num[j]) * 58
			num[j] = byte(carry % 256)
			carry /= 256
		}
		for carry > 0 {
			num = append([]byte{byte(carry % 256)}, num...)
			carry /= 256
		}
	}
	// Leading '1's are leading zero bytes.
	leading := 0
	for leading < len(s) && s[leading] == '1' {
		leading++
	}
	out := make([]byte, leading+len(num))
	copy(out[leading:], num)
	return out, nil
}

// base58CheckEncode appends the 4-byte sha256d checksum of body and base58-encodes
// the result — the standard text form of a TRON address (34 chars, "T...").
func base58CheckEncode(body []byte) string {
	h1 := sha256.Sum256(body)
	h2 := sha256.Sum256(h1[:])
	return base58Encode(append(body, h2[0], h2[1], h2[2], h2[3]))
}

// base58Encode encodes b; leading zero bytes become leading '1's.
// It is the raw codec primitive; callers wanting the checksummed text form
// use base58CheckEncode.
func base58Encode(b []byte) string {
	zeros := 0
	for zeros < len(b) && b[zeros] == 0 {
		zeros++
	}
	num := b[zeros:]
	var digits []byte
	for len(num) > 0 {
		rem := 0
		out := make([]byte, 0, len(num))
		for _, c := range num {
			acc := rem*256 + int(c)
			d := acc / 58
			rem = acc % 58
			if len(out) > 0 || d != 0 {
				out = append(out, byte(d))
			}
		}
		digits = append(digits, byte(rem))
		num = out
	}
	buf := make([]byte, 0, zeros+len(digits))
	for i := 0; i < zeros; i++ {
		buf = append(buf, '1')
	}
	for i := len(digits) - 1; i >= 0; i-- {
		buf = append(buf, b58Alphabet[digits[i]])
	}
	return string(buf)
}
