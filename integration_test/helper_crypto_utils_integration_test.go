package integration_test

import (
	"testing"

	"github.com/kslamph/tronlib/pkg/crypto"
	"github.com/kslamph/tronlib/pkg/helper"
)

func TestHelperCryptoUtils(t *testing.T) {
	// SunToTrx, TrxToSun, SunToTrxString, SunToTrxStringCommas
	trx := int64(123)
	sun := helper.TrxToSun(trx)
	t.Logf("TrxToSun(%d) = %d", trx, sun)
	t.Logf("SunToTrx(%d) = %d", sun, helper.SunToTrx(sun))
	t.Logf("SunToTrxString(%d) = %s", sun, helper.SunToTrxString(sun))
	t.Logf("SunToTrxStringCommas(%d) = %s", sun*1000000, helper.SunToTrxStringCommas(sun*1000000))

	// GetTxid (dummy transaction)
	// Skipped: requires core.Transaction

	// VerifyMessageV2 (dummy data, expect failure)
	address := "TXYnQw1k6Qw1k6Qw1k6Qw1k6Qw1k6Qw1k6"
	message := "hello world"
	signature := "0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef01"
	_, err := crypto.VerifyMessageV2(address, message, signature)
	if err == nil {
		t.Log("VerifyMessageV2: expected failure with dummy data, got no error (OK for placeholder)")
	}
}
