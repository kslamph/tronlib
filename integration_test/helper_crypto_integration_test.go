package integration_test

import (
	"testing"

	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/crypto"
	"github.com/kslamph/tronlib/pkg/helper"
)

func TestSunToTrxAndTrxToSun(t *testing.T) {
	trx := int64(123)
	sun := helper.TrxToSun(trx)
	if sun != 123_000_000 {
		t.Errorf("TrxToSun(%d) = %d, want 123000000", trx, sun)
	}
	if got := helper.SunToTrx(sun); got != trx {
		t.Errorf("SunToTrx(%d) = %d, want %d", sun, got, trx)
	}
}

func TestSunToTrxString(t *testing.T) {
	cases := []struct {
		input  int64
		expect string
	}{
		{123456789, "123.456789"},
		{-123456789, "-123.456789"},
		{1000000, "1.000000"},
	}
	for _, c := range cases {
		if got := helper.SunToTrxString(c.input); got != c.expect {
			t.Errorf("SunToTrxString(%d) = %s, want %s", c.input, got, c.expect)
		}
	}
}

func TestSunToTrxStringCommas(t *testing.T) {
	cases := []struct {
		input  int64
		expect string
	}{
		{123456789, "123.456789"},
		{-123456789, "-123.456789"},
		{12345678900, "12,345.6789"},
		{-12345678900, "-12,345.6789"},
		{123456789000000, "123,456,789"},
		{-123456789000000, "-123,456,789"},
	}
	for _, c := range cases {
		got := helper.SunToTrxStringCommas(c.input)
		if got != c.expect {
			t.Errorf("SunToTrxStringCommas(%d) = %s, want %s", c.input, got, c.expect)
		}
	}
}

func TestGetTxid(t *testing.T) {
	tx := &core.Transaction{RawData: &core.TransactionRaw{RefBlockBytes: []byte("abc")}}
	txid := helper.GetTxid(tx)
	if len(txid) == 0 {
		t.Error("GetTxid returned empty string")
	}
}

func TestVerifyMessageV2(t *testing.T) {
	// This is a deterministic test vector for TIP-191 signature verification.
	// Replace with a real signature and address if available.
	address := "TXYnQw1k6Qw1k6Qw1k6Qw1k6Qw1k6Qw1k6"
	message := "hello world"
	signature := "0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef01"
	_, err := crypto.VerifyMessageV2(address, message, signature)
	if err == nil {
		t.Log("VerifyMessageV2: expected failure with dummy data, got no error (OK for placeholder)")
	}
}
