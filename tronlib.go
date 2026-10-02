package tronlib

// The root facade: ONE import for the happy path.
//
// The facade is an on-ramp, not a layer: aliases (type X = pkg.X) give
// facade and subpackage code zero conversion tax, and every Client method
// is a one-line delegation to the subpackage owner — the facade
// never reimplements. Anything beyond the happy path lives in the
// subpackages: rpc for the full 1:1 gRPC surface, tx for builders/options,
// contract for ABI-driven calls, key for message signing.

import (
	"context"
	"sync"
	"time"

	"github.com/kslamph/tronlib/v2/account"
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
// This is only possible because v2 is a single module.
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

	// Account-scoped handles, reached through Client.Account(owner).
	Account     = account.Handle
	Resources   = account.Resources
	Permissions = account.Permissions
	Voting      = account.Voting

	// Account-scoped values.
	AccountState   = account.State
	Stake          = account.Stake
	Unstake        = account.Unstake
	ResourceState  = tx.ResourceState
	SignatureState = account.SignatureStatus
	Resource       = tx.Resource
	DelegateParams = tx.DelegateOptions
	Vote           = tx.Vote
	Permission     = tx.Permission
	PermissionSet  = tx.PermissionSet
	PermissionKey  = tx.PermissionKey
	ContractType   = tx.ContractType
	ChainParams    = tx.ChainParams
	Delegation     = tx.Delegation
	DelegationList = tx.DelegationIndex
	TotalCost      = tx.TotalCost
	CostPreview    = tx.CostPreview
	BandwidthCost  = tx.BandwidthCost
)

// Resource names a stakeable/delegatable resource. TRON Power is
// deliberately absent: it is not delegatable, and staking grants it under the
// current Mainnet resource model.
const (
	Energy    = tx.ResourceEnergy
	Bandwidth = tx.ResourceBandwidth
)

// Contract types for an active permission's operations bitmap
// (tx.OperationsBitmap). These are the operations this SDK can build; the
// bitmap itself accepts any protocol id through ContractType.
const (
	TypeAccountCreate           = tx.TypeAccountCreate
	TypeTransfer                = tx.TypeTransfer
	TypeTransferAsset           = tx.TypeTransferAsset
	TypeVoteWitness             = tx.TypeVoteWitness
	TypeCreateWitness           = tx.TypeCreateWitness
	TypeUpdateWitness           = tx.TypeUpdateWitness
	TypeFreezeBalance           = tx.TypeFreezeBalance
	TypeUnfreezeBalance         = tx.TypeUnfreezeBalance
	TypeWithdrawBalance         = tx.TypeWithdrawBalance
	TypeProposalCreate          = tx.TypeProposalCreate
	TypeProposalApprove         = tx.TypeProposalApprove
	TypeProposalDelete          = tx.TypeProposalDelete
	TypeCreateSmartContract     = tx.TypeCreateSmartContract
	TypeTriggerSmartContract    = tx.TypeTriggerSmartContract
	TypeUpdateSetting           = tx.TypeUpdateSetting
	TypeUpdateEnergyLimit       = tx.TypeUpdateEnergyLimit
	TypeAccountPermissionUpdate = tx.TypeAccountPermissionUpdate
	TypeClearABI                = tx.TypeClearABI
	TypeUpdateBrokerage         = tx.TypeUpdateBrokerage
	TypeFreezeBalanceV2         = tx.TypeFreezeBalanceV2
	TypeUnfreezeBalanceV2       = tx.TypeUnfreezeBalanceV2
	TypeWithdrawExpireUnfreeze  = tx.TypeWithdrawExpireUnfreeze
	TypeDelegateResource        = tx.TypeDelegateResource
	TypeUnDelegateResource      = tx.TypeUnDelegateResource
	TypeCancelAllUnfreezeV2     = tx.TypeCancelAllUnfreezeV2
)

// OperationsBitmap builds the 32-byte active-permission bitmap for the given
// contract types (tx.OperationsBitmap). Re-exported as a one-line wrapper
// because a type alias cannot carry a function.
func OperationsBitmap(types ...ContractType) ([]byte, error) { return tx.OperationsBitmap(types...) }

// OperationsList decodes an active-permission bitmap back into the contract
// types it allows, in ascending order (tx.OperationsList) — the read side of
// OperationsBitmap, for showing what a permission actually grants.
func OperationsList(bitmap []byte) ([]ContractType, error) { return tx.OperationsList(bitmap) }

// Encode renders a transaction as a portable, versioned envelope that carries
// the declared kind and every signature attached so far (tx.Encode). It is the
// interchange format for offline multi-signing: persist it, or hand it to the
// next signer on another machine.
func Encode(t Tx) ([]byte, error) { return tx.Encode(t) }

// Decode rebuilds a transaction from an Encode envelope, restoring the
// concrete kind and any partial signatures, and rejecting an envelope whose
// declared kind contradicts the contract it wraps (tx.Decode).
func Decode(data []byte) (Tx, error) { return tx.Decode(data) }

// Sign adds a signature from each signer to t and returns the same concrete
// kind (tx.Sign). It is the entry point for a transaction recovered by Decode,
// whose static type is Tx rather than a named kind, and it rejects a signer
// that is already present (a duplicate signature invalidates the transaction).
func Sign(t Tx, signers ...Signer) (Tx, error) { return tx.Sign(t, signers...) }

// SignHash returns the 32-byte digest a signature must cover for t
// (tx.SignHash): sha256 of raw_data. A hardware wallet or remote signer signs
// it, and AttachSignature attaches the result — the private key never enters
// this process.
func SignHash(t Tx) ([]byte, error) { return tx.SignHash(t) }

// AttachSignature returns a copy of t with the raw 65-byte [R || S || V]
// signature attached for addr, rejecting a signature that does not recover to
// that address and a duplicate signer (tx.AttachSignature).
func AttachSignature(t Tx, addr Address, sig []byte) (Tx, error) {
	return tx.AttachSignature(t, addr, sig)
}

// ChainParamsOf reads the governance parameters that price and bound
// transactions: the permission-update and multi-signature fees, the unstake
// cooldown, and the maximum delegation lock (tx.ChainParamsOf). They are live
// values, not constants.
func ChainParamsOf(ctx context.Context, cp *rpc.Client) (*ChainParams, error) {
	return tx.ChainParamsOf(ctx, cp)
}

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
// (default 1..5), WithNetwork. Dial does NOT verify the network — call
// Client.VerifyNetwork explicitly when a declaration was made.
func Dial(ctx context.Context, endpoint string, opts ...DialOption) (*Client, error) {
	var rpcOpts []rpc.DialOption
	var network Network
	for _, o := range opts {
		rpcOpts = append(rpcOpts, o.rpcOpts...)
		if o.network != "" {
			network = o.network
		}
	}
	c, err := rpc.Dial(ctx, endpoint, rpcOpts...)
	if err != nil {
		return nil, err
	}
	return &Client{inner: c, network: network}, nil
}

// DialOption configures a Client at Dial time. It is opaque: build one with
// WithTimeout, WithPool, or WithNetwork.
type DialOption struct {
	rpcOpts []rpc.DialOption
	network Network
}

// WithTimeout sets the default timeout for client operations when the
// context has no deadline (default 30 seconds).
func WithTimeout(d time.Duration) DialOption {
	return DialOption{rpcOpts: []rpc.DialOption{rpc.WithTimeout(d)}}
}

// WithPool configures the initial and maximum connections for the pool
// (default 1..5; sizes <= 0 fall back to the defaults).
func WithPool(initConnections, maxConnections int) DialOption {
	return DialOption{rpcOpts: []rpc.DialOption{rpc.WithPool(initConnections, maxConnections)}}
}

// Client is the happy-path handle to one TRON node. It wraps *rpc.Client;
// every method is a one-line delegation to the subpackage owner.
// The declared network is explicit configuration recorded here by
// WithNetwork; VerifyNetwork checks it against the endpoint's genesis. The
// energy-price cache is one memoised read per maintenance period.
type Client struct {
	inner   *rpc.Client
	network Network

	priceMu sync.Mutex
	price   *tx.EnergyPrice
	priceAt time.Time
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

// Account returns the account-scoped handle for owner: transfers, deployment,
// staking and delegation, permissions and voting, plus the account-shaped
// reads. It performs no I/O and stores no key material — the account being
// operated on is not the key that signs, which is what makes multi-signature
// work.
//
// The shared pipeline is unchanged: every state-changing method returns an
// unsigned transaction, and signing and broadcasting stay on the transaction
// and on Client.
func (c *Client) Account(owner Address) *account.Handle {
	return account.New(c.inner, owner)
}

// Witnesses returns one page of the current witness list. page.Offset and
// page.Limit pass through to the node; Limit 0 means the node's rpc
// default, never "all".
func (c *Client) Witnesses(ctx context.Context, page Page) ([]Witness, error) {
	return rpc.Witnesses(c.inner, ctx, page.Offset, page.Limit)
}

// Page is one explicit pagination cursor: the cursor is a parameter, never
// hidden client state. Limit 0 delegates to the rpc default —
// a caller cannot express "give me everything"; that is the point of the
// List verb contract.
type Page struct {
	Offset int64
	Limit  int64 // 0 means the rpc default; never means "all"
}

// Witness is one super-representative candidate (a decoded view of the
// node's core.Witness: address, vote count, isJobs).
type Witness = rpc.Witness

// Token pins a TRC-20 handle for the token contract at address, fetching
// its decimals with one eager view call. Amounts minted by the Handle carry
// that scale.
func (c *Client) Token(ctx context.Context, address Address) (*token.Handle, error) {
	return token.New(ctx, c.inner, address)
}

// Contract returns a typed view of the deployed contract at addr. The ABI
// loads lazily on first use unless the instance is given one with UseABI.
func (c *Client) Contract(_ context.Context, addr Address) (*contract.Instance, error) {
	return contract.NewInstance(c.inner, addr)
}

// Broadcast submits a signed transaction to the node and returns a Receipt.
// A node-level rejection is a Receipt (r.OK() reports it), not an error.
func (c *Client) Broadcast(ctx context.Context, t Tx) (*Receipt, error) {
	return tx.Broadcast(ctx, c.inner, t)
}

// Wait polls until the transaction is included and executed, and returns
// the parsed Receipt. Call it only on a broadcast the node accepted
// (Receipt.OK()) — a rejected transaction never lands and Wait would poll
// until the context deadline. Inclusion is not finality; use WaitForSolid
// for custody or deposit-crediting semantics.
func (c *Client) Wait(ctx context.Context, txid string) (*Receipt, error) {
	return tx.Wait(ctx, c.inner, txid)
}

// WaitForSolid polls the Solidity endpoint until the transaction appears
// there — solidified semantics, the finality-aware variant of Wait.
func (c *Client) WaitForSolid(ctx context.Context, txid string) (*Receipt, error) {
	return tx.WaitForSolid(ctx, c.inner, txid)
}

// EnergyPrice returns the current energy unit price (the latest governance
// "ts:price" entry). The unit price changes only via governance proposal, so
// the read is cached for one maintenance period: the
// cache is TTL-only, never keyed on the head block. A caller wanting a
// guaranteed-fresh read calls tx.EnergyPriceOf(ctx, c.Raw()) directly.
func (c *Client) EnergyPrice(ctx context.Context) (*tx.EnergyPrice, error) {
	c.priceMu.Lock()
	defer c.priceMu.Unlock()
	if c.price != nil && time.Since(c.priceAt) < tx.MaintenancePeriod {
		cp := *c.price // defensive copy: the cache stays un-mutable through the handle
		return &cp, nil
	}
	p, err := tx.EnergyPriceOf(ctx, c.inner)
	if err != nil {
		return nil, err
	}
	c.price = p
	c.priceAt = time.Now()
	cp := *p // never hand the cached pointer to a caller (Amount.Raw's copy rule)
	return &cp, nil
}

// Events fetches the transaction's logs, decoded leniently — unknown
// signatures materialize with EventName empty and raw bytes preserved;
// never dropped. One non-polling fetch: a transaction that is not yet
// included yields no logs (poll Wait/WaitForSolid first if inclusion is
// required).
func (c *Client) Events(ctx context.Context, txid string) ([]Log, error) {
	return tx.LogsFor(ctx, c.inner, txid)
}
