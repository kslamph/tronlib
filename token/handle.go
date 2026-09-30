package token

import (
	"context"
	"fmt"
	"math/big"

	"github.com/kslamph/tronlib/v2/contract"
	"github.com/kslamph/tronlib/v2/rpc"
	"github.com/kslamph/tronlib/v2/tron"
	"github.com/kslamph/tronlib/v2/tx"
)

// trc20ABI is the minimal TRC-20 surface the Handle drives. decimals is
// deliberately declared uint256, NOT the uint8 real contracts declare:
// declared so, the same wire word decodes through Result.BigInt() for
// both the standard uint8 form (which Result.Byte also reads directly)
// and the uint256-packed form some non-standard contracts emit, and the
// malformed-metadata check becomes an explicit 0..255 range test.
const trc20ABI = `[
  {"type":"function","name":"decimals","stateMutability":"view",
   "inputs":[],"outputs":[{"name":"","type":"uint256"}]},
  {"type":"function","name":"balanceOf","stateMutability":"view",
   "inputs":[{"name":"owner","type":"address"}],
   "outputs":[{"name":"","type":"uint256"}]},
  {"type":"function","name":"transfer","stateMutability":"nonpayable",
   "inputs":[{"name":"to","type":"address"},{"name":"amount","type":"uint256"}],
   "outputs":[{"name":"","type":"bool"}]},
  {"type":"function","name":"approve","stateMutability":"nonpayable",
   "inputs":[{"name":"spender","type":"address"},{"name":"amount","type":"uint256"}],
   "outputs":[{"name":"","type":"bool"}]},
  {"type":"function","name":"allowance","stateMutability":"view",
   "inputs":[{"name":"owner","type":"address"},{"name":"spender","type":"address"}],
   "outputs":[{"name":"","type":"uint256"}]},
  {"type":"function","name":"name","stateMutability":"view",
   "inputs":[],"outputs":[{"name":"","type":"string"}]},
  {"type":"function","name":"symbol","stateMutability":"view",
   "inputs":[],"outputs":[{"name":"","type":"string"}]},
  {"type":"function","name":"totalSupply","stateMutability":"view",
   "inputs":[],"outputs":[{"name":"","type":"uint256"}]}
]`

// Handle is a pinned view of one TRC-20 contract: the token's decimals
// are fetched eagerly at construction and are immutable for the Handle's
// life. Every Amount the Handle produces carries that scale, so parsing
// and formatting never need it passed and never do I/O. Construct with
// New; the zero value is not usable.
type Handle struct {
	contract tron.Address
	instance *contract.Instance
	decimals uint8
}

// New pins the TRC-20 contract at address, fetching its decimals with one
// eager decimals() view call through the contract layer. Failures surface
// classified: chain/transport errors as rpc's codes, no contract at the
// address as contract.not_found, no ABI as contract.no_abi, and a
// decimals value outside the uint8 range (0..255) — e.g. a non-standard
// contract packing decimals as a wide integer — as contract.bad_metadata.
func New(ctx context.Context, cp rpc.ConnProvider, address tron.Address) (*Handle, error) {
	const op = "token.New"
	if cp == nil {
		return nil, &tron.Error{Code: tron.CodeChainConnection, Op: op, Hint: "cp is nil; pass a connected *rpc.Client"}
	}
	inst, err := contract.NewInstance(cp, address)
	if err != nil {
		return nil, err
	}
	if err := inst.UseABI(trc20ABI); err != nil {
		return nil, err // unreachable with the compiled-in ABI; kept for exhaustiveness
	}
	res, err := inst.Call(ctx, "decimals")
	if err != nil {
		return nil, err
	}
	raw, err := res.BigInt()
	if err != nil {
		// The token's metadata does not decode as the integer the TRC-20
		// surface declares (not a number at all) — malformed metadata.
		return nil, &tron.Error{Code: tron.CodeContractBadMetadata, Op: op,
			Hint: "the contract's decimals() did not return a number", Cause: err}
	}
	if !raw.IsUint64() || raw.Uint64() > 255 {
		return nil, &tron.Error{Code: tron.CodeContractBadMetadata, Op: op,
			Hint: fmt.Sprintf("decimals() returned %s; a token's decimals must fit the uint8 range 0..255 — the metadata is malformed (a standard TRC-20 encodes uint8, and wide-packed non-standard metadata is rejected rather than mis-scaled)", raw)}
	}
	//nolint:gosec // G115: guarded by the IsUint64 + >255 rejection directly above
	return &Handle{contract: address, instance: inst, decimals: uint8(raw.Uint64())}, nil
}

// Contract returns the token contract's address.
func (h *Handle) Contract() tron.Address {
	return h.contract
}

// Decimals returns the token's scale: raw units per whole token, fetched
// once at construction. 0..255.
func (h *Handle) Decimals() uint8 {
	return h.decimals
}

// Amount parses an exact decimal token string ("1.6") against the token's
// decimals. The rules mirror tron.ParseTRX, parameterized by the scale:
// plain decimal strings only ("+", "," and "_" are rejected), scientific
// notation is not accepted here, no rounding — more fractional digits
// than the token's decimals is amount.too_many_decimals, malformed input
// amount.invalid, and a negative value is amount.invalid (token amounts
// are non-negative). Parsing does no I/O: the scale is already cached.
func (h *Handle) Amount(s string) (Amount, error) {
	return parseAmount(s, h.decimals, "token.Handle.Amount")
}

// Whole converts a whole-token count (3 means 3 tokens) into an Amount at
// the token's scale. It is the token-layer analogue of tron.TRX.
//
// Accepts integer constants and int64 values only — float literals and
// foreign numeric types (including tron.SUN) are rejected at compile
// time, preserving the amount-safety property without generic methods
// (which require go 1.27+; this module pins 1.25). A negative n is
// amount.invalid (token amounts are non-negative) and a scaled value
// beyond the uint256 the ABI carries is amount.overflow.
func (h *Handle) Whole(n int64) (Amount, error) {
	const op = "token.Handle.Whole"
	if n < 0 {
		return Amount{}, &tron.Error{Code: tron.CodeAmountInvalid, Op: op,
			Hint: "a token amount cannot be negative; pass a positive whole-token count"}
	}
	raw := new(big.Int).Mul(big.NewInt(n), scale(h.decimals))
	if raw.BitLen() > 256 {
		return Amount{}, &tron.Error{Code: tron.CodeAmountOverflow, Op: op,
			Hint: fmt.Sprintf("%d whole tokens at %d decimals exceeds the uint256 the TRC-20 ABI carries", n, h.decimals)}
	}
	return newAmount(raw, h.decimals), nil
}

// metaErr classifies a failed metadata read. Contract-layer shape
// failures (the contract answered, but not as TRC-20 declares) become
// contract.bad_metadata — "this is not a TRC-20" is the actionable fact
// for an agent. Transport errors, reverts, and not_found pass through
// untouched: a reverted call means the contract RAN and chose to fail,
// which is different information from malformed metadata.
func metaErr(op, what string, err error) error {
	for _, c := range []tron.Code{
		tron.CodeContractArgMismatch,
		tron.CodeContractMethodUnknown,
		tron.CodeContractNoABI,
		tron.CodeContractBadABI,
		tron.CodeContractResultTypeMismatch,
	} {
		if tron.HasCode(err, c) {
			return &tron.Error{Code: tron.CodeContractBadMetadata, Op: op,
				Hint: what + " did not answer as TRC-20 declares; the contract does not implement TRC-20", Cause: err}
		}
	}
	return err
}

// BalanceOf reads owner's token balance as an Amount at the Handle's
// scale. balanceOf returns uint256, so contract.Result.BigInt() is its
// accessor (decimals() needed the uint8-gap workaround instead — see the
// package doc).
func (h *Handle) BalanceOf(ctx context.Context, owner tron.Address) (Amount, error) {
	const op = "token.Handle.BalanceOf"
	res, err := h.instance.Call(ctx, "balanceOf", contract.AddressArg(owner))
	if err != nil {
		return Amount{}, metaErr(op, "balanceOf()", err)
	}
	raw, err := res.BigInt()
	if err != nil {
		return Amount{}, &tron.Error{Code: tron.CodeContractBadMetadata, Op: op,
			Hint: "balanceOf() did not return a number; the contract does not implement TRC-20", Cause: err}
	}
	return newAmount(raw, h.decimals), nil
}

// Transfer builds the transfer(address,uint256) transaction: from will be
// the signing owner, the token moves to to. The call value is 0 — a
// TRC-20 transfer spends no TRX beyond fees. An Amount minted against a
// different scale is amount.decimals_mismatch (never silently re-scaled);
// a zero from address is address.invalid.
func (h *Handle) Transfer(ctx context.Context, from, to tron.Address, amt Amount) (*tx.ContractTx, error) {
	const op = "token.Handle.Transfer"
	if from.IsZero() {
		return nil, &tron.Error{Code: tron.CodeAddressInvalid, Op: op, Hint: "from address is unset; Transfer needs the account that will sign and broadcast"}
	}
	if amt.Decimals() != h.decimals {
		return nil, &tron.Error{Code: tron.CodeAmountDecimalsMismatch, Op: op,
			Hint: fmt.Sprintf("the Amount carries %d decimals but this token has %d; mint amounts with this Handle (Amount/Whole/BalanceOf) — they are never re-scaled", amt.Decimals(), h.decimals)}
	}
	return h.instance.Invoke(ctx, from, 0, "transfer", contract.AddressArg(to), contract.BigIntArg(amt.Raw()))
}

// Approve builds the approve(address,uint256) transaction: owner authorizes
// spender to move up to amt of the owner's tokens (the DEX allowance
// flow). Same rules as Transfer: the call value is 0, the Amount must
// carry this token's scale, and a zero owner is address.invalid.
func (h *Handle) Approve(ctx context.Context, owner, spender tron.Address, amt Amount) (*tx.ContractTx, error) {
	const op = "token.Handle.Approve"
	if owner.IsZero() {
		return nil, &tron.Error{Code: tron.CodeAddressInvalid, Op: op, Hint: "owner address is unset; Approve needs the account that will sign and broadcast"}
	}
	if amt.Decimals() != h.decimals {
		return nil, &tron.Error{Code: tron.CodeAmountDecimalsMismatch, Op: op,
			Hint: fmt.Sprintf("the Amount carries %d decimals but this token has %d; mint amounts with this Handle (Amount/Whole/BalanceOf) — they are never re-scaled", amt.Decimals(), h.decimals)}
	}
	return h.instance.Invoke(ctx, owner, 0, "approve", contract.AddressArg(spender), contract.BigIntArg(amt.Raw()))
}

// Allowance reads how much of owner's tokens spender may currently move,
// as an Amount at the Handle's scale.
func (h *Handle) Allowance(ctx context.Context, owner, spender tron.Address) (Amount, error) {
	const op = "token.Handle.Allowance"
	res, err := h.instance.Call(ctx, "allowance", contract.AddressArg(owner), contract.AddressArg(spender))
	if err != nil {
		return Amount{}, metaErr(op, "allowance()", err)
	}
	raw, err := res.BigInt()
	if err != nil {
		return Amount{}, &tron.Error{Code: tron.CodeContractBadMetadata, Op: op,
			Hint: "allowance() did not return a number; the contract does not implement TRC-20", Cause: err}
	}
	return newAmount(raw, h.decimals), nil
}

// Name reads the token's name. A non-string answer is contract.bad_metadata.
func (h *Handle) Name(ctx context.Context) (string, error) {
	const op = "token.Handle.Name"
	res, err := h.instance.Call(ctx, "name")
	if err != nil {
		return "", metaErr(op, "name()", err)
	}
	name, err := res.String()
	if err != nil {
		return "", &tron.Error{Code: tron.CodeContractBadMetadata, Op: op,
			Hint: "name() did not return a string; the contract does not implement TRC-20", Cause: err}
	}
	return name, nil
}

// Symbol reads the token's symbol. A non-string answer is contract.bad_metadata.
func (h *Handle) Symbol(ctx context.Context) (string, error) {
	const op = "token.Handle.Symbol"
	res, err := h.instance.Call(ctx, "symbol")
	if err != nil {
		return "", metaErr(op, "symbol()", err)
	}
	symbol, err := res.String()
	if err != nil {
		return "", &tron.Error{Code: tron.CodeContractBadMetadata, Op: op,
			Hint: "symbol() did not return a string; the contract does not implement TRC-20", Cause: err}
	}
	return symbol, nil
}

// TotalSupply reads the token's total supply as an Amount at the Handle's
// scale. A non-numeric answer is contract.bad_metadata.
func (h *Handle) TotalSupply(ctx context.Context) (Amount, error) {
	const op = "token.Handle.TotalSupply"
	res, err := h.instance.Call(ctx, "totalSupply")
	if err != nil {
		return Amount{}, metaErr(op, "totalSupply()", err)
	}
	raw, err := res.BigInt()
	if err != nil {
		return Amount{}, &tron.Error{Code: tron.CodeContractBadMetadata, Op: op,
			Hint: "totalSupply() did not return a number; the contract does not implement TRC-20", Cause: err}
	}
	return newAmount(raw, h.decimals), nil
}
