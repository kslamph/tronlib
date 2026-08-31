package key

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/kslamph/bip39-hdwallet/bip39"
	"github.com/kslamph/bip39-hdwallet/hdwallet"

	"github.com/kslamph/tronlib/v2/tron"
)

// hdWalletSigner implements Signer with a key derived from a BIP-39 mnemonic
// along a BIP-32 derivation path.
type hdWalletSigner struct {
	mnemonic string
	path     string
	privKey  *ecdsa.PrivateKey
	address  tron.Address
}

// PrivateKeyFromMnemonic builds a Signer from a BIP-39 mnemonic, passphrase,
// and derivation path (e.g. "m/44'/195'/0'/0/0"). An invalid mnemonic, an
// invalid path, or a failed derivation returns an error with code
// key.mnemonic_invalid.
func PrivateKeyFromMnemonic(mnemonic, passphrase, path string) (Signer, error) {
	const op = "PrivateKeyFromMnemonic"
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, &tron.Error{Code: tron.CodeKeyMnemonicInvalid, Op: op, Hint: "pass a valid BIP-39 mnemonic (12 or 24 words with a valid checksum)"}
	}

	seed := bip39.NewSeed(mnemonic, passphrase)
	masterKey, err := hdwallet.NewMasterKey(seed)
	if err != nil {
		return nil, &tron.Error{Code: tron.CodeKeyMnemonicInvalid, Op: op, Hint: "check the mnemonic words and passphrase", Cause: err}
	}

	wallet, err := masterKey.DerivePath(path)
	if err != nil {
		return nil, &tron.Error{Code: tron.CodeKeyMnemonicInvalid, Op: op, Hint: `use a valid BIP-32 path such as "m/44'/195'/0'/0/0"`, Cause: err}
	}

	privKey, err := wallet.ToECDSA()
	if err != nil {
		return nil, &tron.Error{Code: tron.CodeKeyMnemonicInvalid, Op: op, Hint: "check the mnemonic words and derivation path", Cause: err}
	}

	ethAddr := crypto.PubkeyToAddress(privKey.PublicKey)

	// Add TRON prefix (0x41).
	tronBytes := append([]byte{0x41}, ethAddr.Bytes()...)

	address, err := tron.AddressFromBytes(tronBytes)
	if err != nil {
		return nil, err
	}

	return &hdWalletSigner{
		mnemonic: mnemonic,
		path:     path,
		privKey:  privKey,
		address:  address,
	}, nil
}

// Address returns the account's address.
func (s *hdWalletSigner) Address() tron.Address {
	return s.address
}

// PublicKey returns the account's public key.
func (s *hdWalletSigner) PublicKey() *ecdsa.PublicKey {
	return &s.privKey.PublicKey
}

// Sign signs a given hash with the derived private key and returns the raw
// signature bytes: 65-byte [R || S || V] with V in {0, 1}.
func (s *hdWalletSigner) Sign(hash []byte) ([]byte, error) {
	return crypto.Sign(hash, s.privKey)
}
