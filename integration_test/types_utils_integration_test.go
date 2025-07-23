package integration_test

import (
	"testing"

	"github.com/kslamph/tronlib/pkg/types"
)

func TestTypesUtils(t *testing.T) {
	// NewAccountFromPrivateKey (dummy key, just for API coverage)
	priv := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	acct, err := types.NewAccountFromPrivateKey(priv)
	if err != nil {
		t.Logf("NewAccountFromPrivateKey failed (expected with dummy key): %v", err)
	} else {
		t.Logf("NewAccountFromPrivateKey: %+v", acct)
	}

	// NewAccountFromHDWallet (dummy mnemonic, just for API coverage)
	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	acct2, err := types.NewAccountFromHDWallet(mnemonic, "m/44'/195'/0'/0/0", "0")
	if err != nil {
		t.Logf("NewAccountFromHDWallet failed (expected with dummy mnemonic): %v", err)
	} else {
		t.Logf("NewAccountFromHDWallet: %+v", acct2)
	}

	// MustNewAddress
	addr := types.MustNewAddress("TXBwCB1RxvMPZTZE79aJn9KjLbdSXMax55")
	t.Logf("MustNewAddress: %s", addr.String())

	// MustNewAddressFromHex
	hexAddr := addr.Hex()
	addr2 := types.MustNewAddressFromHex(hexAddr)
	t.Logf("MustNewAddressFromHex: %s", addr2.String())

	// MustNewAddressFromBytes
	addr3 := types.MustNewAddressFromBytes(addr.Bytes())
	t.Logf("MustNewAddressFromBytes: %s", addr3.String())

	// Additional address operations
	t.Logf("Address string: %s", addr.String())
	t.Logf("Address hex: %s", addr.Hex())
	t.Logf("Address bytes: %x", addr.Bytes())
}
