# Signer Package

The `signer` package provides the `Signer` interface and two concrete implementations: `PrivateKeySigner` (hex key) and `HDWalletSigner` (BIP-39 mnemonic). A `Signer` signs transaction hashes and exposes the derived TRON address and public key.

## Import

```go
import "github.com/kslamph/tronlib/pkg/signer"
```

## Interface

```go
type Signer interface {
    Address() *types.Address
    PublicKey() *ecdsa.PublicKey
    Sign(hash []byte) ([]byte, error)
}
```

Both implementations satisfy this interface. `Sign` returns the raw 65-byte signature (R+S+V) suitable for inclusion in `Transaction.Signature`.

## PrivateKeySigner

```go
signer, err := signer.NewPrivateKeySigner("ab12cd34ef56...")
// 0x prefix is tolerated
if err != nil {
    // types.ErrInvalidPrivateKey — bad hex or invalid ECDSA key
    log.Fatal(err)
}
fmt.Println(signer.Address().String()) // TRON base58 address
```

### Methods

| Method | Returns |
|---|---|
| `Address()` | `*types.Address` — derived TRON address (0x41 prefix) |
| `PublicKey()` | `*ecdsa.PublicKey` |
| `Sign(hash)` | `([]byte, error)` — 65-byte ECDSA signature |
| `PrivateKeyHex()` | `string` — hex-encoded private key |

## HDWalletSigner

```go
mnemonic := "abandon abandon abandon ... about" // 12 BIP-39 words
path := "m/44'/195'/0'/0/0"                    // TRON standard path
signer, err := signer.NewHDWalletSigner(mnemonic, "", path)
// passphrase is optional (empty string = no passphrase)
if err != nil {
    log.Fatal(err) // invalid mnemonic or derivation failure
}
fmt.Println(signer.Address().String())
```

### Methods

| Method | Returns |
|---|---|
| `Address()` | `*types.Address` |
| `PublicKey()` | `*ecdsa.PublicKey` |
| `Sign(hash)` | `([]byte, error)` |

### BIP-44 Path

TRON uses the standard BIP-44 path:

```
m / purpose' / coin_type' / account' / change / address_index
m / 44'      / 195'       / 0'        / 0      / 0
```

- `purpose` = 44 (BIP-44)
- `coin_type` = 195 (TRON)

## Usage with SignAndBroadcast

```go
result, err := cli.SignAndBroadcast(ctx, txExt, client.DefaultBroadcastOptions(), signer)
```

`SignAndBroadcast` accepts one or more `signer.Signer` — multiple signers are required for multi-signature contracts.

## Error Handling

| Sentinel | Meaning |
|---|---|
| `types.ErrInvalidPrivateKey` | Non-hex string, malformed key, or nil ECDSA key |

The error is wrapped with `%w` so `errors.Is(err, types.ErrInvalidPrivateKey)` works.

## Security

- **Never log or serialize private keys.** The `PrivateKeyHex()` method is provided for local testing only; treat the return value as confidential.
- **Testnet-only keys** in test fixtures — never commit funded mainnet keys.
- `NewPrivateKeySignerFromECDSA(nil)` returns `types.ErrInvalidPrivateKey`.

## See Also

- [Client Package](client.md) — `SignAndBroadcast` and `Simulate`
- [Account Package](account.md) — creating transactions to sign
- [TRC20 Package](trc20.md) — token transfers with signing
- [Types Package](types.md) — `*types.Address` and error sentinels
