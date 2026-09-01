package tronlib

// The root facade: ONE import for the happy path (spec §10).
//
// The facade is an on-ramp, not a layer: aliases (type X = pkg.X) give
// facade and subpackage code zero conversion tax, and every Client method
// is a one-line delegation to the subpackage owner (spec D7 — the facade
// never reimplements). Anything beyond the happy path lives in the
// subpackages: rpc for the full 1:1 gRPC surface, tx for builders/options,
// contract for ABI-driven calls, key for message signing.

import (
	"context"
	"time"

	"github.com/kslamph/tronlib/v2/contract"
	"github.com/kslamph/tronlib/v2/event"
	"github.com/kslamph/tronlib/v2/key"
	"github.com/kslamph/tronlib/v2/rpc"
	"github.com/kslamph/tronlib/v2/token"
	"github.com/kslamph/tronlib/v2/tron"
	"github.com/kslamph/tronlib/v2/tx"
)

// Aliases, not wrappers: a tron.Address and a tronlib.Address are the same
// type, so facade and subpackage calls interoperate with zero conversion.
// This is only possible because v2 is a single module (spec C1).
type (
	Address    = tron.Address
	SUN        = tron.SUN
	Signer     = key.Signer
	Code       = tron.Code
	Action     = tron.Action
	Error      = tron.Error
	Tx         = tx.Tx
	NativeTx   = tx.NativeTx
	ContractTx = tx.ContractTx
	Receipt    = tx.Receipt
	Log        = event.Log
)

// TRX converts a whole-number TRX literal to SUN. It panics on overflow and
// is for literals and constants only; dynamic input must use ParseTRX.
// Re-exported as a one-line wrapper because generics cannot be aliased.
func TRX[T tron.Whole](n T) SUN { return tron.TRX(n) }

// ParseTRX parses an exact decimal TRX string into SUN (at most 6 decimal
// places; no float64 ever touches this path).
func ParseTRX(s string) (SUN, error) { return tron.ParseTRX(s) }

// MustTRX parses s and panics on error. For package-level amount literals
// in tests and examples; anything that handles user input must use ParseTRX.
func MustTRX(s string) SUN { return tron.MustTRX(s) }

// ParseAddress parses a base58check-encoded TRON address ("T...", 34 chars).
func ParseAddress(s string) (Address, error) { return tron.ParseAddress(s) }

// MustAddress parses s and panics on error. For package-level address
// literals in tests and examples; anything that handles user input must
// use ParseAddress.
func MustAddress(s string) Address { return tron.MustAddress(s) }

// KeyFromHex builds a Signer from a secp256k1 private key hex string
// (with or without a 0x prefix).
func KeyFromHex(hexKey string) (Signer, error) { return key.PrivateKeyFromHex(hexKey) }

// KeyFromMnemonic builds a Signer from a BIP-39 mnemonic, passphrase and
// derivation path.
func KeyFromMnemonic(mnemonic, passphrase, path string) (Signer, error) {
	return key.PrivateKeyFromMnemonic(mnemonic, passphrase, path)
}

// Dial creates a Client to a TRON node at endpoint, which must be
// grpc://host:port (plaintext) or grpcs://host:port (TLS). Dial is lazy:
// it builds the connection factory without network I/O, so reachability is
// proven by the first call. Options: WithTimeout (default 30s), WithPool
// (default 1..5).
func Dial(ctx context.Context, endpoint string, opts ...DialOption) (*Client, error) {
	c, err := rpc.Dial(ctx, endpoint, opts...)
	if err != nil {
		return nil, err
	}
	return &Client{inner: c}, nil
}

// DialOption configures a Client at Dial time (rpc.WithTimeout,
// rpc.WithPool).
type DialOption = rpc.DialOption

// WithTimeout sets the default timeout for client operations when the
// context has no deadline (default 30 seconds).
func WithTimeout(d time.Duration) DialOption { return rpc.WithTimeout(d) }

// WithPool configures the initial and maximum connections for the pool
// (default 1..5; sizes <= 0 fall back to the defaults).
func WithPool(initConnections, maxConnections int) DialOption {
	return rpc.WithPool(initConnections, maxConnections)
}

// Client is the happy-path handle to one TRON node. It wraps *rpc.Client;
// every method is a one-line delegation to the subpackage owner (spec D7).
// Network identity is NOT part of the v2.0 surface: Network/VerifyNetwork
// are deferred (see the Task 9 report, D2) — ChainTip alone is the network
// surface until the genesis-fingerprint heuristic can be live-verified.
type Client struct {
	inner *rpc.Client
}

// Close closes the client and all pooled connections. Idempotent.
func (c *Client) Close() error { return c.inner.Close() }

// Raw returns the underlying *rpc.Client — the escape hatch to the full 1:1
// gRPC wrapper surface without leaving the facade's connection pool.
func (c *Client) Raw() *rpc.Client { return c.inner }

// Endpoint returns the configured endpoint (scheme://host:port).
func (c *Client) Endpoint() string { return c.inner.GetNodeAddress() }

// ChainTip returns the latest known block number.
func (c *Client) ChainTip(ctx context.Context) (uint64, error) {
	return rpc.ChainTip(c.inner, ctx)
}

// TronBalance returns the account's TRX balance in SUN.
func (c *Client) TronBalance(ctx context.Context, a Address) (SUN, error) {
	return rpc.TronBalance(c.inner, ctx, a)
}

// Witnesses returns one page of the current witness list. page.Offset and
// page.Limit pass through to the node; Limit 0 means the node's rpc
// default, never "all" (spec §10.1).
func (c *Client) Witnesses(ctx context.Context, page Page) ([]Witness, error) {
	return rpc.Witnesses(c.inner, ctx, page.Offset, page.Limit)
}

// Page is one explicit pagination cursor: the cursor is a parameter, never
// hidden client state (spec §10.1). Limit 0 delegates to the rpc default —
// a caller cannot express "give me everything"; that is the point of the
// List verb contract.
type Page struct {
	Offset int64
	Limit  int64 // 0 means the rpc default; never means "all"
}

// Witness is one super-representative candidate (a decoded view of the
// node's core.Witness: address, vote count, isJobs).
type Witness = rpc.Witness

// TransferTRX builds a TRX transfer (NativeTx) from from to to for amt.
// The node fills raw_data (TAPOS reference, timestamp, expiration); sign
// the result with Sign before Broadcast.
func (c *Client) TransferTRX(ctx context.Context, from, to Address, amt SUN) (*NativeTx, error) {
	return tx.BuildTransfer(c.inner, ctx, from, to, amt)
}

// TransferToken builds a TRC-10 transfer (AssetTx) of qty units of
// assetName (the token's id or name form as the node expects it) from from
// to to. TRC-20 tokens go through Token instead.
func (c *Client) TransferToken(ctx context.Context, from, to Address, assetName string, qty int64) (*tx.AssetTx, error) {
	return tx.BuildAssetTransfer(c.inner, ctx, from, to, assetName, qty)
}

// Token pins a TRC-20 handle for the token contract at address, fetching
// its decimals with one eager view call. Amounts minted by the Handle carry
// that scale.
func (c *Client) Token(ctx context.Context, address Address) (*token.Handle, error) {
	return token.New(c.inner, ctx, address)
}

// Contract returns a typed view of the deployed contract at addr. The ABI
// loads lazily on first use unless the instance is given one with UseABI.
func (c *Client) Contract(ctx context.Context, addr Address) (*contract.Instance, error) {
	return contract.NewInstance(c.inner, addr)
}

// Deploy builds a CreateSmartContract transaction (DeployTx). p.Bytecode is
// the final creation bytecode: constructor arguments already appended
// (encoding is a contract-layer concern).
func (c *Client) Deploy(ctx context.Context, owner Address, p tx.DeployParams) (*tx.DeployTx, error) {
	return tx.BuildDeploy(c.inner, ctx, owner, p)
}

// Broadcast submits a signed transaction to the node and returns a Receipt.
// A node-level rejection is a Receipt (r.OK() reports it), not an error.
func (c *Client) Broadcast(ctx context.Context, t Tx) (*Receipt, error) {
	return tx.Broadcast(c.inner, ctx, t)
}

// Wait polls until the transaction is included and executed, and returns
// the parsed Receipt. Inclusion is not finality; use WaitForSolid for
// custody or deposit-crediting semantics.
func (c *Client) Wait(ctx context.Context, txid string) (*Receipt, error) {
	return tx.Wait(c.inner, ctx, txid)
}

// WaitForSolid polls the Solidity endpoint until the transaction appears
// there — solidified semantics, the finality-aware variant of Wait.
func (c *Client) WaitForSolid(ctx context.Context, txid string) (*Receipt, error) {
	return tx.WaitForSolid(c.inner, ctx, txid)
}

// CostPreview predicts what broadcasting t will cost owner in SUN,
// combining the simulation (energy split + revert check), the authoritative
// energy estimate and the owner's staked energy (spec §7.3).
func (c *Client) CostPreview(ctx context.Context, t *ContractTx, owner Address) (*tx.CostPreview, error) {
	return tx.PreviewCost(c.inner, ctx, t, owner)
}

// EnergyPrice returns the current energy unit price (the latest governance
// "ts:price" entry; a read is a floor, not a ceiling).
func (c *Client) EnergyPrice(ctx context.Context) (*tx.EnergyPrice, error) {
	return tx.EnergyPriceOf(c.inner, ctx)
}

// Events fetches the transaction's logs, decoded leniently — unknown
// signatures materialize with EventName empty and raw bytes preserved;
// never dropped. One non-polling fetch: a transaction that is not yet
// included yields no logs (poll Wait/WaitForSolid first if inclusion is
// required).
func (c *Client) Events(ctx context.Context, txid string) ([]Log, error) {
	return tx.LogsFor(c.inner, ctx, txid)
}
