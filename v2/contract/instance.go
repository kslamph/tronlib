package contract

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	eABI "github.com/ethereum/go-ethereum/accounts/abi"
	eCommon "github.com/ethereum/go-ethereum/common"
	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/v2/event"
	"github.com/kslamph/tronlib/v2/rpc"
	"github.com/kslamph/tronlib/v2/tron"
	"github.com/kslamph/tronlib/v2/tx"
)

// Instance is a typed view of one deployed contract: its ABI (fetched
// lazily from the chain or supplied via UseABI) drives encode, call,
// invoke and decode. Construct with NewInstance; the zero value is not
// usable.
type Instance struct {
	cp      rpc.ConnProvider
	address tron.Address

	mu      sync.Mutex
	abiJSON string // the loaded ABI JSON ("" until loaded)
	parsed  *eABI.ABI
}

// NewInstance returns an Instance for the contract at address. The ABI is
// NOT fetched here — loading happens lazily on first use (Call, Invoke,
// Decode), so constructing instances for contracts that are never touched
// costs nothing. Supply the ABI up front with UseABI to skip the network
// fetch; without it the first ABI-using call performs a GetContract RPC
// and fails with contract.not_found (no contract at the address),
// contract.no_abi (contract exists, publishes no ABI) or contract.bad_abi
// (ABI present but unparseable).
func NewInstance(cp rpc.ConnProvider, address tron.Address) (*Instance, error) {
	const op = "contract.NewInstance"
	if cp == nil {
		return nil, &tron.Error{Code: tron.CodeChainConnection, Op: op, Hint: "cp is nil; pass a connected *rpc.Client"}
	}
	if address.IsZero() {
		return nil, &tron.Error{Code: tron.CodeAddressInvalid, Op: op, Hint: "contract address is unset; parse it with tron.ParseAddress"}
	}
	return &Instance{cp: cp, address: address}, nil
}

// DynamicEnergy reads the contract's TIP-491 dynamic-energy state: the
// surcharge factor the node's VM is currently applying to this contract,
// with the tracked usage and the maintenance cycle it is effective for.
// It is a constant-call-cheap read (GetContractInfo) — no key, no
// signature, no spend. A fresh contract reads as the zero DynamicEnergy
// (factor 0, no penalty); an address with no contract is
// contract.not_found. See tx.DynamicEnergy for the factor semantics and
// tx.DynamicEnergy.PredictPenalty for turning it into a surcharge.
func (i *Instance) DynamicEnergy(ctx context.Context) (*tx.DynamicEnergy, error) {
	return tx.DynamicEnergyOf(i.cp, ctx, i.address)
}

// UseABI loads the contract's ABI from a Solidity JSON string, replacing
// any previously loaded ABI. Loading parses the JSON (contract.bad_abi on
// failure) and registers the ABI's event definitions with the event
// package's registry, so event.Decode works for this contract's events.
func (i *Instance) UseABI(abiJSON string) error {
	parsed, err := parseAndRegisterABI(abiJSON)
	if err != nil {
		return err
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	i.abiJSON = abiJSON
	i.parsed = parsed
	return nil
}

// parseAndRegisterABI validates the JSON (parse + event registration) and
// returns the parsed ABI — everything UseABI must do, WITHOUT taking the
// instance lock (the lazy load path calls it while i.mu is already held;
// locking here would deadlock).
func parseAndRegisterABI(abiJSON string) (*eABI.ABI, error) {
	const op = "contract.UseABI"
	if strings.TrimSpace(abiJSON) == "" {
		return nil, &tron.Error{Code: tron.CodeContractBadABI, Op: op, Hint: "ABI JSON is empty; pass the contract's Solidity ABI JSON"}
	}
	parsed, err := parseABIJSON(abiJSON)
	if err != nil {
		return nil, &tron.Error{Code: tron.CodeContractBadABI, Op: op, Hint: "the ABI JSON does not parse as a Solidity ABI", Cause: err}
	}
	if len(parsed.Methods) == 0 && len(parsed.Events) == 0 {
		return nil, &tron.Error{Code: tron.CodeContractBadABI, Op: op, Hint: "the ABI JSON carries no functions or events"}
	}
	// Feed the event registry so Decode/DecodeLenient resolve this
	// contract's events without a second registration step.
	if err := event.RegisterABIJSON(abiJSON); err != nil {
		return nil, &tron.Error{Code: tron.CodeContractBadABI, Op: op, Hint: "the ABI JSON's event entries do not parse", Cause: err}
	}
	return parsed, nil
}

// loadABI performs the lazy network load. Ported from v1
// smartcontract.NewInstance's network path: GetContract returns the
// SmartContract; no contract at the address is contract.not_found, a
// missing/empty ABI is contract.no_abi, and an unparseable one is
// contract.bad_abi. Failures are NOT cached — a transient transport error
// (chain.connection) retries on the next use.
func (i *Instance) loadABI(ctx context.Context) error {
	const op = "contract.loadABI"
	msg := &api.BytesMessage{Value: i.address.Bytes()}
	sc, err := rpc.GetContract(i.cp, ctx, msg)
	if err != nil {
		return err
	}
	if sc == nil || (len(sc.GetAbi().GetEntrys()) == 0 && len(sc.GetBytecode()) == 0) {
		return &tron.Error{
			Code: tron.CodeContractNotFound,
			Op:   op,
			Hint: "no contract exists at this address; check the address (tron.ParseAddress) and the network",
		}
	}
	if sc.GetAbi() == nil || len(sc.GetAbi().GetEntrys()) == 0 {
		return &tron.Error{
			Code: tron.CodeContractNoABI,
			Op:   op,
			Hint: "the contract publishes no ABI on-chain; supply it with UseABI(json)",
		}
	}
	abiJSON, err := pbABIToJSON(sc.GetAbi())
	if err == nil {
		parsed, perr := parseAndRegisterABI(abiJSON)
		if perr == nil {
			// i.mu is already held by ensureABI — set the fields directly.
			i.abiJSON = abiJSON
			i.parsed = parsed
		} else {
			err = perr
		}
	}
	if err != nil {
		var te *tron.Error
		if !errors.As(err, &te) {
			err = &tron.Error{Code: tron.CodeContractBadABI, Op: op, Hint: "the on-chain ABI does not parse", Cause: err}
		}
		return err
	}
	return nil
}

// ensureABI returns the parsed ABI, loading it lazily on first use.
func (i *Instance) ensureABI(ctx context.Context) (*eABI.ABI, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.parsed == nil {
		if err := i.loadABI(ctx); err != nil {
			return nil, err
		}
	}
	return i.parsed, nil
}

// ABI returns the loaded ABI JSON — empty until an ABI is loaded (UseABI,
// or the lazy fetch triggered by the first Call/Invoke/Decode). It
// performs no I/O.
func (i *Instance) ABI() string {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.abiJSON
}

// Methods returns the loaded ABI's function names, sorted (spec §7.6:
// valid method values are enumerable). Empty until an ABI is loaded
// (UseABI, or the lazy fetch triggered by the first Call/Invoke/Decode).
// It performs no I/O.
func (i *Instance) Methods() []string {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.parsed == nil {
		return nil
	}
	names := make([]string, 0, len(i.parsed.Methods))
	for name := range i.parsed.Methods {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// method looks a method up by name, triggering the lazy ABI load when
// needed. contract.method_unknown when absent.
func (i *Instance) method(ctx context.Context, name string) (*eABI.Method, error) {
	parsed, err := i.ensureABI(ctx)
	if err != nil {
		return nil, err
	}
	if m, ok := parsed.Methods[name]; ok {
		return &m, nil
	}
	return nil, &tron.Error{
		Code: tron.CodeContractMethodUnknown,
		Op:   "contract.Instance.method",
		Hint: fmt.Sprintf("method %q is not in the contract ABI; enumerate with Methods()", name),
	}
}

// encodeCall ABI-encodes method(args...) into call data: 4-byte selector
// + packed arguments (ported from v1 utils.EncodeMethod). Count and type
// mismatches are contract.arg_mismatch; the packing itself (dynamic
// offsets, the 0x41-stripped address form) is geth's.
func (i *Instance) encodeCall(ctx context.Context, name string, args []Arg) ([]byte, error) {
	const op = "contract.Instance.encode"
	m, err := i.method(ctx, name)
	if err != nil {
		return nil, err
	}
	if len(args) != len(m.Inputs) {
		return nil, &tron.Error{
			Code: tron.CodeContractArgMismatch,
			Op:   op,
			Hint: fmt.Sprintf("method %s takes %d argument(s), got %d; the declared types are (%s)",
				name, len(m.Inputs), len(args), strings.Join(inputTypeNames(m), ", ")),
		}
	}
	values := make([]any, len(args))
	for idx, arg := range args {
		declared := m.Inputs[idx].Type.String()
		got := arg.argABI()
		if declared != got {
			return nil, &tron.Error{
				Code: tron.CodeContractArgMismatch,
				Op:   op,
				Hint: fmt.Sprintf("argument %d of %s is declared %s but the Arg encodes %s", idx, name, declared, got),
			}
		}
		v, err := toEVMValue(arg)
		if err != nil {
			return nil, &tron.Error{Code: tron.CodeContractArgMismatch, Op: op, Hint: fmt.Sprintf("argument %d of %s: %v", idx, name, err)}
		}
		values[idx] = v
	}
	packed, err := m.Inputs.Pack(values...)
	if err != nil {
		return nil, &tron.Error{Code: tron.CodeContractArgMismatch, Op: op, Hint: fmt.Sprintf("arguments do not pack for %s's declared types", name), Cause: err}
	}
	return append(append([]byte{}, m.ID...), packed...), nil
}

func inputTypeNames(m *eABI.Method) []string {
	names := make([]string, len(m.Inputs))
	for idx, in := range m.Inputs {
		names[idx] = in.Type.String()
	}
	return names
}

// nullOwner is the owner_address Call sends: the node's
// triggerconstantcontract RPC requires the field, view calls spend
// nothing, and spec §9's Call signature has no owner. See the package doc.
var nullOwner = func() tron.Address {
	b := make([]byte, 21) // 20 zero bytes
	b[0] = 0x41           // the network prefix — the 0x41-prefixed null address
	a, err := tron.AddressFromBytes(b)
	if err != nil {
		panic("contract: null owner address is not constructible: " + err.Error())
	}
	return a
}()

// Call executes a view (constant) method end to end and returns the
// decoded result (spec §9, review G2: one step, not four). It is the
// v1 Instance.Call shape: triggerconstantcontract, then decode the
// ConstantResult against the method's declared outputs. The call value is
// always 0 — a read cannot spend.
//
// The owner address is the null address (see the package doc). A
// node-level rejection (including a contract revert) surfaces as a
// *tron.Error classified through rpc's Return table; tx.Simulate's
// in-band Code shape is a transaction-layer concern and is not repeated
// here. live-verified: pending (spec §7.5).
func (i *Instance) Call(ctx context.Context, method string, args ...Arg) (*Result, error) {
	const op = "contract.Instance.Call"
	data, err := i.encodeCall(ctx, method, args)
	if err != nil {
		return nil, err
	}
	req := &core.TriggerSmartContract{
		OwnerAddress:    nullOwner.Bytes(),
		ContractAddress: i.address.Bytes(),
		Data:            data,
		CallValue:       0,
	}
	ext, err := rpc.TriggerConstantContract(i.cp, ctx, req)
	if err != nil {
		return nil, err
	}
	if err := rpc.ValidateTransactionResult(ext, op); err != nil {
		return nil, err
	}
	// The node returns one entry per output value; v1 concatenated them
	// and so does this decode path (a single-blob decode).
	var blob []byte
	for _, r := range ext.GetConstantResult() {
		blob = append(blob, r...)
	}
	return i.decodeResult(ctx, method, blob)
}

// CallAtBlock is the block-anchored read (spec §9). The v4.8.2 Wallet API
// has NO block anchor on triggerconstantcontract — the pb
// TriggerSmartContract message carries owner/contract/value/data/token
// fields only — so v2 refuses instead of silently calling at head, which
// would return the wrong block's state with no signal. Archive-node reads
// need an upstream API addition.
func (i *Instance) CallAtBlock(ctx context.Context, block uint64, method string, args ...Arg) (*Result, error) {
	const op = "contract.Instance.CallAtBlock"
	return nil, &tron.Error{
		Code:  tron.CodeRPCMethodFailed,
		Op:    op,
		Hint:  fmt.Sprintf("block-anchored constant calls are not supported by the node's Wallet API: TriggerSmartContract carries no block anchor (block %d requested); use Call for head state", block),
		Cause: errors.New("triggerconstantcontract has no block-anchored variant in the Wallet API"),
	}
}

// Invoke builds a state-changing call transaction (spec §9): encode, then
// the tx builder's TriggerSmartContract. Returns *tx.ContractTx — the DAG
// direction contract → tx (spec §3). The ABI is loaded (and its events
// registered with the event package) before encoding, so the
// transaction's eventual receipt decodes this contract's events.
func (i *Instance) Invoke(ctx context.Context, owner tron.Address, value tron.SUN, method string, args ...Arg) (*tx.ContractTx, error) {
	const op = "contract.Instance.Invoke"
	if owner.IsZero() {
		return nil, &tron.Error{Code: tron.CodeAddressInvalid, Op: op, Hint: "owner address is unset; Invoke needs the account that will sign and broadcast"}
	}
	data, err := i.encodeCall(ctx, method, args)
	if err != nil {
		return nil, err
	}
	return tx.BuildTriggerSmartContract(i.cp, ctx, owner, i.address, data, value)
}

// Decode decodes a raw ABI return value for a known method into a typed
// Result — the partner of tx.Estimate.ConstantResult (spec §9), which tx
// cannot decode itself (the DAG direction contract → tx forbids it).
// If no ABI is loaded yet, the lazy network fetch runs against a
// background context; load it with UseABI first to decode offline.
func (i *Instance) Decode(method string, data []byte) (*Result, error) {
	return i.decodeResult(context.Background(), method, data)
}

// decodeResult unpacks data against method's declared outputs (ported
// from v1 abiProcessor.DecodeResult) and maps the decoded values onto
// Result's accessor shapes: addresses re-prepended with 0x41, integers as
// *big.Int, bytesN normalized to slices, multi-output returns as []any.
func (i *Instance) decodeResult(ctx context.Context, method string, data []byte) (*Result, error) {
	const op = "contract.Instance.Decode"
	m, err := i.method(ctx, method)
	if err != nil {
		return nil, err
	}
	if len(m.Outputs) == 0 {
		return &Result{}, nil
	}
	raw, err := m.Outputs.Unpack(data)
	if err != nil {
		return nil, &tron.Error{
			Code:  tron.CodeContractArgMismatch,
			Op:    op,
			Hint:  fmt.Sprintf("data does not decode as %s's declared output types (%s)", method, strings.Join(outputTypeNames(m), ", ")),
			Cause: err,
		}
	}
	return &Result{val: convertDecoded(raw, m.Outputs)}, nil
}

func outputTypeNames(m *eABI.Method) []string {
	names := make([]string, len(m.Outputs))
	for idx, out := range m.Outputs {
		names[idx] = out.Type.String()
	}
	return names
}

// convertDecoded maps geth's unpacked values onto Result's accessor shapes.
// Single-output methods yield the converted value directly; multi-output
// methods yield []any (the singular accessors then fail with
// contract.result_type_mismatch — see the package doc).
func convertDecoded(values []any, args eABI.Arguments) any {
	converted := make([]any, len(values))
	for idx, v := range values {
		var paramType string
		if idx < len(args) {
			paramType = args[idx].Type.String()
		}
		converted[idx] = convertOne(v, paramType)
	}
	if len(args) == 1 && len(values) == 1 {
		return converted[0]
	}
	return converted
}

func convertOne(v any, paramType string) any {
	switch val := v.(type) {
	case eCommon.Address:
		a, err := tronAddressFromEVM(val)
		if err != nil {
			return v
		}
		return a
	case [32]byte:
		return val[:]
	case []any:
		out := make([]any, len(val))
		base := paramType
		if k := strings.Index(paramType, "["); k >= 0 {
			base = paramType[:k]
		}
		for idx, elem := range val {
			out[idx] = convertOne(elem, base)
		}
		return out
	default:
		// *big.Int (uintN/intN), bool, string, []byte, uint64... pass
		// through as geth decoded them.
		return v
	}
}
