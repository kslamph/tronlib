package tx

import (
	"time"

	"github.com/kslamph/tronlib/v2/key"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/tron"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// DeployParams carries the deployment inputs BuildDeploy validates and
// forwards to the node's DeployContract build RPC. Constructor-argument
// encoding is a contract-layer concern (tx must not import contract, spec
// §3): callers pass the final bytecode (init code + encoded constructor
// arguments already appended).
type DeployParams struct {
	// Name is the on-chain contract name; it may be empty. It must contain
	// only visible (non-control) characters.
	Name string
	// ABI is the parsed contract ABI; nil is allowed (v1 semantics) when no
	// constructor decoding is needed.
	ABI *core.SmartContract_ABI
	// Bytecode is the contract creation bytecode; it must be non-empty.
	Bytecode []byte
	// CallValue is the SUN transferred to the contract at creation.
	CallValue tron.SUN
	// ConsumeUserResourcePercent is the share of caller-paid energy, 0–100.
	ConsumeUserResourcePercent int64
	// OriginEnergyLimit is the maximum energy the contract itself can
	// consume per call; must be >= 0.
	OriginEnergyLimit int64
}

// DeployTx is a CreateSmartContract transaction. It is a distinct type — not
// a ContractTx variant — because it carries fields no other kind has
// (OriginEnergyLimit, ConsumeUserResourcePercent) and because deployment
// estimates by bytecode rather than by built call: DeployTx.Estimate runs
// the node's triggerConstantContract deploy path (empty contract address,
// init bytecode as data), which research 2026-09-28 verified end to end.
// There is still no EstimateEnergy RPC path for deploys, and no Simulate
// — the estimation shape differs, which is what the type system expresses.
type DeployTx struct{ baseTx }

// txInternal seals Tx: only the kinds in this package implement it.
func (*DeployTx) txInternal() {}

// clone returns a deep copy for the copy-on-write With*/Sign methods.
func (t *DeployTx) clone() *DeployTx { return &DeployTx{baseTx: *t.cloneBase()} }

// WithFeeLimit returns a COPY of t with raw_data.fee_limit set to s. Deploy
// is the most expensive call a user makes; the builder default is
// 150_000_000 SUN. Call it before Sign.
// Note: setting options after signing invalidates any signature (raw_data changes; the node rejects with SIGERROR).
func (t *DeployTx) WithFeeLimit(s tron.SUN) *DeployTx {
	c := t.clone()
	c.raw().FeeLimit = int64(s)
	return c
}

// WithExpiration returns a COPY of t whose raw_data.expiration is moved to
// now+d (milliseconds). Call it before Sign.
// Note: setting options after signing invalidates any signature (raw_data changes; the node rejects with SIGERROR).
func (t *DeployTx) WithExpiration(d time.Duration) *DeployTx {
	c := t.clone()
	setExpiration(c.raw(), d)
	return c
}

// WithPermissionID returns a COPY of t with Permission_id set on the wrapped
// contract message. Call it before Sign. It returns an error (not a panic)
// when the wrapped transaction carries no contract message — reachable only
// by replacing the node's build response through the Extension()/Transaction()
// escape hatch.
// Note: setting options after signing invalidates any signature (raw_data changes; the node rejects with SIGERROR).
func (t *DeployTx) WithPermissionID(id int32) (*DeployTx, error) {
	c := t.clone()
	if err := setPermissionID(c.raw(), "tx.DeployTx.WithPermissionID", id); err != nil {
		return nil, err
	}
	return c, nil
}

// WithOriginEnergyLimit returns a COPY of t with Origin_energy_limit set on
// the decoded CreateSmartContract parameter (the value lives inside the
// contract parameter, not in raw_data, so the parameter is decoded, mutated
// and re-encoded — it must be called before Sign). Unlike the builder, no
// Note: setting options after signing invalidates any signature (raw_data changes; the node rejects with SIGERROR).
// 0-floor is enforced here: the node is the authority for post-build
// mutations.
func (t *DeployTx) WithOriginEnergyLimit(n int64) (*DeployTx, error) {
	c := t.clone()
	if err := mutateDeployParam(c.raw(), "tx.DeployTx.WithOriginEnergyLimit", func(p *core.CreateSmartContract) {
		if p.NewContract == nil {
			p.NewContract = &core.SmartContract{}
		}
		p.NewContract.OriginEnergyLimit = n
	}); err != nil {
		return nil, err
	}
	return c, nil
}

// WithResourcePercent returns a COPY of t with Consume_user_resource_percent
// set on the decoded CreateSmartContract parameter. Unlike the builder, the
// 0–100 range is not enforced here: the node is the authority for post-build
// mutations. Call it before Sign.
// Note: setting options after signing invalidates any signature (raw_data changes; the node rejects with SIGERROR).
func (t *DeployTx) WithResourcePercent(p int64) (*DeployTx, error) {
	c := t.clone()
	if err := mutateDeployParam(c.raw(), "tx.DeployTx.WithResourcePercent", func(m *core.CreateSmartContract) {
		if m.NewContract == nil {
			m.NewContract = &core.SmartContract{}
		}
		m.NewContract.ConsumeUserResourcePercent = p
	}); err != nil {
		return nil, err
	}
	return c, nil
}

// Sign returns a COPY of t with a signature from each signer appended to the
// pb transaction; signatures accumulate. The receiver is left untouched.
func (t *DeployTx) Sign(signers ...key.Signer) (*DeployTx, error) {
	b, err := signBase(&t.baseTx, signers)
	if err != nil {
		return nil, err
	}
	return &DeployTx{baseTx: *b}, nil
}

// mutateDeployParam decodes the wrapped contract parameter as a
// CreateSmartContract, applies fn, and re-encodes it in place on raw.
// Escape-hatch corruption surfaces as tx.invalid_argument, not a panic.
func mutateDeployParam(raw *core.TransactionRaw, op string, fn func(*core.CreateSmartContract)) error {
	param, err := deployParam(raw, op)
	if err != nil {
		return err
	}
	fn(param)
	return reencodeParam(raw, op, param)
}

// deployParam decodes raw's contract parameter as a CreateSmartContract.
// The contract TYPE is checked first: a one-field payload (e.g.
// WithdrawBalanceContract{owner}) decodes cleanly as a CreateSmartContract
// because field 1 is the same owner_address, and the re-encode that follows
// would then silently rewrite the caller's transaction into deploy fields
// while the contract Type still names the original operation. An empty
// contract list is the same escape-hatch corruption class.
func deployParam(raw *core.TransactionRaw, op string) (*core.CreateSmartContract, error) {
	contracts := raw.GetContract()
	if len(contracts) == 0 {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op,
			Hint: "the wrapped transaction has no contract message; the builder always sets one — was the extention replaced via Extension()?"}
	}
	if contracts[0].GetType() != core.Transaction_Contract_CreateSmartContract {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op,
			Hint: "the wrapped contract is not a CreateSmartContract; the builder always sets it — was the extention replaced via Extension()?"}
	}
	p := new(core.CreateSmartContract)
	if err := proto.Unmarshal(contracts[0].GetParameter().GetValue(), p); err != nil {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op,
			Hint: "the wrapped contract parameter does not decode as CreateSmartContract; was the extention replaced via Extension()?", Cause: err}
	}
	return p, nil
}

// reencodeParam re-encodes msg as the wrapped contract parameter.
func reencodeParam(raw *core.TransactionRaw, op string, msg proto.Message) error {
	newAny, err := anypb.New(msg)
	if err != nil {
		return &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op,
			Hint: "re-encoding the contract parameter failed", Cause: err}
	}
	raw.GetContract()[0].Parameter = newAny
	return nil
}

// setExpiration moves raw_data.expiration to now+d (milliseconds) — the
// exact mutation of v1's utils.SetExpiration, ported as the With* mutation
// primitive.
func setExpiration(raw *core.TransactionRaw, d time.Duration) {
	raw.Expiration = time.Now().Add(d).UnixMilli()
}

// setPermissionID sets Permission_id on the wrapped contract message —
// v1's utils.SetPermissionID mutation (raw_data.contract[0].PermissionId).
// An empty contract list (escape-hatch corruption) is tx.invalid_argument,
// not a panic.
func setPermissionID(raw *core.TransactionRaw, op string, id int32) error {
	c := raw.GetContract()
	if len(c) == 0 {
		return &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op,
			Hint: "the wrapped transaction has no contract message; the builder always sets one — was the extention replaced via Extension()?"}
	}
	c[0].PermissionId = id
	return nil
}
