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
// contract message. Call it before Sign.
// Note: setting options after signing invalidates any signature (raw_data changes; the node rejects with SIGERROR).
func (t *DeployTx) WithPermissionID(id int32) *DeployTx {
	c := t.clone()
	setPermissionID(c.raw(), id)
	return c
}

// WithOriginEnergyLimit returns a COPY of t with Origin_energy_limit set on
// the decoded CreateSmartContract parameter (the value lives inside the
// contract parameter, not in raw_data, so the parameter is decoded, mutated
// and re-encoded — it must be called before Sign). Unlike the builder, no
// Note: setting options after signing invalidates any signature (raw_data changes; the node rejects with SIGERROR).
// 0-floor is enforced here: the node is the authority for post-build
// mutations.
func (t *DeployTx) WithOriginEnergyLimit(n int64) *DeployTx {
	c := t.clone()
	mutateDeployParam(c.raw(), func(p *core.CreateSmartContract) {
		if p.NewContract == nil {
			p.NewContract = &core.SmartContract{}
		}
		p.NewContract.OriginEnergyLimit = n
	})
	return c
}

// WithResourcePercent returns a COPY of t with Consume_user_resource_percent
// set on the decoded CreateSmartContract parameter. Unlike the builder, the
// 0–100 range is not enforced here: the node is the authority for post-build
// mutations. Call it before Sign.
// Note: setting options after signing invalidates any signature (raw_data changes; the node rejects with SIGERROR).
func (t *DeployTx) WithResourcePercent(p int64) *DeployTx {
	c := t.clone()
	mutateDeployParam(c.raw(), func(m *core.CreateSmartContract) {
		if m.NewContract == nil {
			m.NewContract = &core.SmartContract{}
		}
		m.NewContract.ConsumeUserResourcePercent = p
	})
	return c
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
func mutateDeployParam(raw *core.TransactionRaw, fn func(*core.CreateSmartContract)) {
	param := deployParam(raw)
	fn(param)
	reencodeParam(raw, param)
}

// deployParam decodes raw's contract parameter as a CreateSmartContract.
func deployParam(raw *core.TransactionRaw) *core.CreateSmartContract {
	p := new(core.CreateSmartContract)
	if err := proto.Unmarshal(raw.GetContract()[0].GetParameter().GetValue(), p); err != nil {
		panic("tx: deploy parameter does not decode as CreateSmartContract; the builder always sets it — was the extention replaced via Extension()?")
	}
	return p
}

// reencodeParam re-encodes msg as the wrapped contract parameter.
func reencodeParam(raw *core.TransactionRaw, msg proto.Message) {
	newAny, err := anypb.New(msg)
	if err != nil {
		panic("tx: re-encoding a contract parameter failed: " + err.Error())
	}
	raw.GetContract()[0].Parameter = newAny
}

// setExpiration moves raw_data.expiration to now+d (milliseconds) — the
// exact mutation of v1's utils.SetExpiration, ported as the With* mutation
// primitive.
func setExpiration(raw *core.TransactionRaw, d time.Duration) {
	raw.Expiration = time.Now().Add(d).UnixMilli()
}

// setPermissionID sets Permission_id on the wrapped contract message —
// v1's utils.SetPermissionID mutation (raw_data.contract[0].PermissionId).
func setPermissionID(raw *core.TransactionRaw, id int32) {
	c := raw.GetContract()
	if len(c) == 0 {
		panic("tx: WithPermissionID on a transaction without a contract message; the builder always sets one — was the extention replaced via Extension()?")
	}
	c[0].PermissionId = id
}
