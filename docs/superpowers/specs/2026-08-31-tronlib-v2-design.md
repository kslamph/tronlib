# tronlib v2 — API Design Specification

**Status:** Revised after review (`docs/reviews/2026-08-31-tronlib-v2-design-review.md`); see §16 for the disposition of every finding. Awaiting user review.
**Date:** 2026-08-31
**Scope:** Core API redesign. Breaking changes permitted; no v1 compatibility work.
**Provenance:** Derived from a two-model adversarial design process (oracle draft + three independent attack lanes) with every P0 finding independently re-verified by execution. Evidence citations refer to the v1 tree at `/Users/kslam/goproj/tronlib`.

---

## 1. Constraints

These are owner-fixed and not subject to revision within this spec.

| ID | Constraint |
|---|---|
| **C1** | Ships as `github.com/kslamph/tronlib/v2`. Separate major module path. v1 remains installable and untouched. |
| **C2** | When human ergonomics and LLM-agent ergonomics conflict, **LLM agents win**. |
| **C3** | Core API only. Shielded/Sapling, TRC-10 asset issuance, and `cmd/` CLI tools are **out of scope**. |
| **C4** | Clean-room. Free to merge, split, rename, delete. **No `Deprecated:` shims** carried into v2.0. |
| **C5** | No v1 work. Bug fixes found during this audit are recorded as v2 design requirements, not v1 patches. |

### C2 boundary

C2 licenses optimizing for *ambiguity elimination* and *error recovery*. It does **not** license any machine-readable artifact that a human must keep in sync with the code — every generated surface must derive from a single source of truth. This boundary exists because v1's documentation drift (§10) is precisely a hand-maintained artifact failing.

---

## 2. Problems being solved

Each item was verified against source. `P` numbers are carried from the audit for traceability.

**P1 — Unit asymmetry.** `account.TransferTRX` takes `int64` SUN (raw; 1 TRX = 10⁶) while `(*trc20.TRC20Manager).Transfer` takes `decimal.Decimal` in human units (`pkg/trc20/client.go:316`). Same parameter name, opposite scales.

**P2 — `any` in core signatures.** `client.SignAndBroadcast(ctx, anytx any)` (`broadcaster.go:196`), `client.Simulate(ctx, anytx any)` (`broadcaster.go:97`), `smartcontract.NewInstance(tronClient contractClient, …, abi ...any)` where `contractClient` is an **unexported interface in an exported signature** (`contract.go:17,72`), and `utils.SetFeeLimit/SetTimestamp/SetExpiration/SetPermissionID(tx any)`.

**P3 — Redundant routes.** TRC-20 reachable three ways (`cli.TRC20Manager` / `cli.TRC20` / `trc20.NewManager`). Contracts reachable three ways. Two distinct simulate result types (`*BroadcastResult`, `*SimulateResult`).

**P4 — `utils` is a 56-symbol dumping ground.** Four near-identical formatters; ten `IsValidX()bool` / `ValidateX()error` twins; TRC-20-specific encoders in a generic package.

**P5 — Naming inconsistency.** Six managers stutter (`account.AccountManager`, `trc20.TRC20Manager`, …); `smartcontract.Manager` does not.

**P6 — Sign and verify in different packages.** `signer.SignMessageV2` vs `utils.VerifyMessageV2`.

**P7 — No entry point.** Hello-world requires four imports and ~8 steps.

**P8 — `lowlevel` is undocumented.** 131 exported functions, 6-line `doc.go`, zero examples. 25 of the 131 are shielded and therefore out of scope under C3, leaving ~106.

**P9 — Four competing error mechanisms, not two.** 16 `errors.New` sentinels with remediation prose fused into the message string (`pkg/types/errors.go:13-60`); `NewTronError` with integer codes; `WrapTransactionResult` (`types/errors.go:155`); `ValidateTransactionResult` (`lowlevel/tx.go:12`). Five sentinels are never returned by non-test code; three of those are advertised in `doc.go` error lists.

**P10 — Third-party types in public signatures.** `types.NewAddressFromEVM(go-ethereum common.Address)`; `decimal.Decimal` throughout `trc20`.

**P11 — Chronic documentation drift.** Seven verified non-compiling snippets in `README.md` and `docs/*.md`. CI compiles `example/` but not docs.

**P12 — Generic over unexported constraint.** `types.NewAddress[T addressAllowed]` renders an undefined name in godoc and silently falls through parse strategies.

### The F1 defect (drives §6)

`pkg/client/broadcaster.go:126-131` unconditionally `proto.Unmarshal`s `contract[0].Parameter.Value` into `core.TriggerSmartContract` without inspecting `Transaction_Contract.Type` (the enum field exists at `pb/core/tron.pb.go:4806` and is never read anywhere in `pkg/client/`).

`TransferContract` and `TriggerSmartContract` are wire-compatible on fields 1–3 (`bytes, bytes, varint`). Verified by execution: a `TransferContract{owner, to, amount=1_000_000}` unmarshals into `TriggerSmartContract` with `err = nil`, yielding `ContractAddress` = the recipient EOA, `CallValue` = 1 000 000, `Data` = empty. `docs/quickstart.md:455-470` recommends exactly this sequence. The call then invokes `TriggerConstantContract` against an externally-owned account and returns fabricated numbers with no error.

`Simulate` additionally populates `br.TxID` from the constant-call response and returns `Success = false, err = nil`. A simulated result therefore carries a real-looking transaction id for a transaction that was never broadcast.

---

## 3. Package architecture

**Rule:** a package exists only if it owns a concept an agent must retrieve by name — it has its own constructors, its own error codes, and it is not merely a file's worth of functions.

13 packages → **8 + root**.

```
github.com/kslamph/tronlib/v2
├── tronlib      root facade: Dial, Client, aliases, amount constructors, CostPreview
├── tron         vocabulary: Address, SUN, Error, Code, Action, constants
├── key          Signer, PrivateKey, HDWallet, SignMessage, VerifyMessage
├── rpc          transport, connection pool, ~106 1:1 gRPC wrappers
├── tx           NativeTx, ContractTx, DeployTx, AssetTx, Receipt, Estimate, EnergyEstimate, CostPreview
├── contract     Instance, Result, Arg
├── token        Handle, Amount
└── event        Log, EventDef, Decode
```

### Dependency DAG (strict, acyclic)

```
tron  (no internal deps)
  ├── key
  ├── rpc
  ├── event
  └── tx        (tron, key, rpc)
        ├── contract   (tron, rpc, tx)
        └── token      (tron, rpc, tx, contract)
              └── tronlib  (all)
```

**Why `tx` is its own package.** `token` and `contract` builders must return the transaction types. Placing them in the root facade creates an import cycle (`token → tronlib → token`); v1 already demonstrates the wall — `(*trc20.TRC20Manager).Transfer` returns the raw `*api.TransactionExtention` precisely because it cannot name the client's transaction type (`pkg/trc20/client.go:316`). Placing them in `rpc` instead would mix a curated model into the raw escape hatch, recreating the v1 `client` package's transport-plus-logic problem. A dedicated `tx` package keeps `rpc` honestly 1:1.

**Direction constraint.** `contract` and `token` depend on `tx`, never the reverse. `tx` therefore must not name any type owned by `contract` or `token` — see §7.2 for how this shapes `Estimate`.

### v1 disposition

| v1 package | Fate | Destination |
|---|---|---|
| `types` (46) | split | `tron` |
| `utils` (56) | **deleted** | dissolved — see §4 |
| `client` (28) | split | `rpc` (transport) + `tronlib` (facade) + `tx` (result types) |
| `client/lowlevel` (131) | rename + cut | `rpc` — minus 25 shielded (C3) = ~106 |
| `signer` (16) | rename | `key` — fixes P6 by co-locating sign and verify |
| `smartcontract` (20) | rename | `contract` |
| `trc20` (15) | rename + dissolve free functions | `token` |
| `eventdecoder` (13) | rename | `event` |
| `account` (8) | fold | `tronlib` (balance, transfer) + `rpc` (raw) |
| `network` (13) | fold | `rpc` + `tronlib` (`ChainTip`) |
| `resources` (15) | fold | `rpc` + `tx` (`CostPreview`) |
| `voting` (12) | fold | `rpc` |
| `trc10` (14) | split | issuance out of scope (C3); **transfer** in scope as `tx.AssetTx` — see §13 |

### `utils` dissolution

| v1 symbol | v2 disposition |
|---|---|
| `EncodeAddress` / `DecodeAddress` | delete → `tron.Address` methods |
| `HumanReadableBalance`, `HumanReadableNumber`, `HumanReadableTokenAmount`, `FormatBigInt` | collapse to **two** methods: `String()` (canonical) and `Formatted()` (display) on the type owning the scale. Not one: v1's output carries comma separators *and* rounds (`conversion.go:22-25`), so a single method cannot be both round-trippable and display-friendly. |
| 10 `IsValidX` / `ValidateX` twins | keep `ValidateX() error` only. A bool cannot tell an agent *why*. |
| `SetFeeLimit`, `SetTimestamp`, `SetExpiration`, `SetPermissionID` (`tx any`) | replaced by the typed `With*` methods in **§6.4** — fee limit, expiration and permission id are all required by real flows and are defined there, not merely promised. `SetTimestamp` deleted outright: timestamps are derived from the head block. |
| `EncodeTRC20Transfer`, `EncodeTRC20BalanceOf`, `DecodeTRC20Balance` | move to `token` |
| `EncodeParameters`, `DecodeParameters`, `EncodeMethodSignature` | move to `contract` |
| `VerifyMessageV2` | move to `key`, beside `SignMessageV2` |
| `ExtractSigners`, `GetTransactionID` | methods `(*Tx).Signers()`, `(*Tx).ID()` |
| `JSONToMap`, `MapToJSON` | delete → `encoding/json` |
| `PadLeft`, `PadRight`, `HexToBytes`, `BytesToHex`, `DecodeString`, `EncodeString` | delete → `encoding/hex`, `bytes` |
| `IsValidNodeURL` / `ValidateNodeURL` | delete — `Dial` validates and returns a code; a pre-check is a second source of truth |

`trc20`'s exported free functions `ToWei`, `FromWei`, `ToWeiWithDecimals`, `FromWeiWithDecimals` are dissolved into `token.Amount` / `Handle.Amount` under the same principle. `decimal.Decimal` appears in **no** v2 exported signature.

---

## 4. Naming and convention contract

Each rule states its real enforcement mechanism. Rules marked *review* are not mechanically decidable and must not be misrepresented as lint rules — v1's design claimed seven lint-enforced rules of which two were implementable as stated.

| # | Rule | Enforcement |
|---|---|---|
| **N1** | No exported type repeats its package name. `token.Handle`, `token.Amount`, `contract.Instance`, `key.Signer`. | lint (prefix-only) |
| **N2** | **Lifecycle verbs are a closed set** and must be chosen from: `Dial`, `New`, `Parse`, `Must`, `Sign`, `Broadcast`, `Simulate`, `Estimate`, `Wait`, `List`, `Cost`. These are the operations where picking the wrong one is a *semantic* error with a cost attached — `Broadcast` vs `Wait`, `Simulate` vs `Estimate`, `Parse` vs `Must`. Domain verbs (`Transfer`, `Approve`, `Invoke`, `Call`, `Decode`, `Balance`) are **not** constrained beyond N6; they name blockchain actions, not library mechanics. | review |
| **N3** | `Get*` forbidden as a zero-argument field accessor (`GetBalance()` → `Balance()`). **Permitted** when it takes a lookup key (`GetByID(id)`). **`rpc` is exempt entirely** — its names are a mechanical projection of the TRON gRPC service; the absolute ban would rename ~46% of it and destroy the 1:1 property that justifies the package. | lint (arity-based, package-scoped) |
| **N4** | `ctx context.Context` is the first parameter of every function performing I/O. Never stored on a struct. Never `context.Background()` inside the library. | lint |
| **N5** | Zero `any` / `interface{}` in exported parameter or result types. **Carve-out:** `any` inside a type-parameter constraint (`rpc.Call[T any]`) is permitted — it is not the same hazard. | lint |
| **N6** | Nouns must be qualified when two distinct concepts could share them: `TronBalance` / `TokenBalance`, never a bare `Balance`. | review |
| **N7** | No `float32` / `float64` in any amount path. | lint (package-scoped, **not** name-scoped — v1's `ToWei`/`FromWei` would slip past a name-based rule) |
| **N8** | Receiver names consistent per type across a package. Interfaces have no receiver entry. | lint |

**N6 is load-bearing, not cosmetic.** An unqualified `Balance` returning `tron.SUN` is the same failure shape as F1: the intent "what is their USDT balance" resolves to the wrong symbol and returns a plausible number with no error.

**Deleted from the prior design:** the "no two exported symbols share a 4-character prefix" rule and the absolute `Get` ban, both of which failed on their own package map. Also deleted: the claim that the *entire* verb vocabulary is closed — the spec's own API uses domain verbs (`Transfer`, `Approve`, `Decode`) that no finite lifecycle list could cover, and pretending otherwise is the "rigorous-sounding but post-hoc rule" failure this section exists to avoid.

---

## 5. Amount model

### 5.1 TRX

```go
package tron

// SUN is TRX in the atomic on-chain unit. 1 TRX = 1_000_000 SUN.
// It is the only in-memory representation of a TRX value.
type SUN int64

// Whole admits the predeclared signed integer types and nothing else.
//
// Two deliberate exclusions:
//   - No tilde. With ~int64, SUN's own underlying type satisfies the
//     constraint, so TRX(someSUN) compiles and re-scales an already-scaled
//     value — turning 1 TRX into 1,000,000 TRX with the overflow guard
//     passing. Verified by execution.
//   - No unsigned kinds. int64(n) of a uint64 above MaxInt64 wraps negative,
//     and the wrapped value passes the bound check, yielding a negative
//     amount silently. Verified by execution.
type Whole interface{ int | int8 | int16 | int32 | int64 }

const maxSafeTRX = math.MaxInt64 / 1_000_000 // 9_223_372_036_854

func TRX[T Whole](n T) SUN
func ParseTRX(s string) (SUN, error)
func MustTRX(s string) SUN
```

`int64` is sufficient: the ceiling is ~9.2 × 10¹² TRX against a total supply of ~10¹¹.

`TRX` panics on overflow and is documented for **literals and constants only**; anything dynamic uses `ParseTRX`. This follows the `Must`/`Parse` split in N2.

`ParseTRX` rejects, with distinct codes: unparseable strings, a leading `+`, thousands separators, more than 6 decimal places, and int64 overflow. It accepts scientific notation down to 1 SUN (`"1e-6"` → 1). Implementation is a scale shift on exact decimal arithmetic; no float appears in the path (N7).

### 5.2 Rejection matrix

| Input | Rejected by | Signal |
|---|---|---|
| `TRX(1.6)` | compiler | `float64 does not satisfy Whole` |
| `TRX(someSUN)` | compiler | `SUN does not satisfy Whole (possibly missing ~ for int64 in Whole)` |
| `TRX(uint64(x))` | compiler | `uint64 does not satisfy Whole` |
| `TRX(10e15)` | compiler | `float64 does not satisfy Whole` — `10e15` is an **untyped float constant**, so it is rejected at compile time, not at runtime |
| `TRX(9_223_372_036_855)` | runtime panic | an *integer* literal above `maxSafeTRX`; the only genuine panic case |
| `ParseTRX("1.6666666")` | error | `amount.too_many_decimals` |
| `ParseTRX("+1.6")` | error | `amount.invalid` |
| `ParseTRX("1,234")` | error | `amount.invalid` |
| `ParseTRX("1e-7")` | error | `amount.too_many_decimals` |

### 5.3 Display

```go
func (s SUN) String() string        // value receiver; canonical; round-trips with ParseTRX
func (s SUN) Formatted() string     // display only: thousands separators, rounding
func (s SUN) Add(o SUN) (SUN, error) // checked — see overflow note below
func (s SUN) Sub(o SUN) (SUN, error) // checked
func (s SUN) Mul(n int64) (SUN, error)
func (s SUN) Int64() int64
```

**Overflow discipline is uniform.** `Add` and `Sub` are checked and return `error` like `Mul`. `SUN` is an `int64` and the type admits arbitrary values, so two near-`MaxInt64` amounts would otherwise wrap silently — and an unchecked pair of binary operators next to a checked `Mul` is exactly the inconsistency this spec exists to remove. The supply bound makes these cases unreachable *in practice*; it does not make them unreachable *in the type*, and the difference is the whole point of §5.

`String()` **must** use a value receiver. With a pointer receiver, `fmt.Sprintf("%v", sun)` on a non-addressable `SUN` does not select the method and prints the raw integer — every log line then silently lies about units.

### 5.4 Token amounts

```go
package token

// Amount is an immutable token quantity. raw is always integer-valued and
// never nil; the zero value represents 0 with 0 decimals.
type Amount struct {
    raw      *big.Int
    decimals uint8
}

func (h *Handle) Amount(s string) (Amount, error)   // "1.6" — scale from h, no I/O
func (h *Handle) Whole[T tron.Whole](n T) Amount    // whole tokens; same compile-time float rejection
func (a Amount) Raw() *big.Int                      // returns a COPY
func (a Amount) Decimals() uint8
func (a Amount) String() string                     // canonical, round-trips
func (a Amount) Formatted() string                  // display
```

`Raw()` returns a copy; exposing the internal pointer lets a caller mutate the amount in place.

`Handle` is **eager and immutable**: `Client.Token(ctx, addr)` performs the `decimals()` read at construction, which is why it takes `ctx` and returns `error`. There is no `Refresh`. This is required by the amount model — `Amount(s string)` cannot compute a scale-shifted integer without knowing the token's decimals, and laziness would force I/O into a pure parse function.

**Decimals validation.** `decimals()` returns `uint8`, so anything in `0..255` is representable on-chain and must not be rejected as malformed — values above 18 are unusual, not invalid, and refusing them would make the library unable to read a token the chain accepts. `contract.bad_metadata` is reserved for the case v1 actually has a test for: a response whose **encoding width** is wrong, e.g. `decimals` returned packed as uint256 instead of uint8 (`pkg/trc20/decimals_uint256_test.go:17-24`). `Amount.String()` and `Handle.Amount` must therefore tolerate `decimals` up to 255 and are tested against a 255-decimal fixture.

---

## 6. Transaction pipeline

The flow is **build → optionally simulate → sign → broadcast**, expressed as types so each stage's output is the next stage's input and illegal transitions do not compile.

### 6.1 Two transaction kinds

```go
package tx

type NativeTx struct    // TRX transfer, freeze, delegate, vote, account ops
type ContractTx struct  // TriggerSmartContract — contract calls and all TRC-20 ops

// Kind reports which TRON contract a built transaction wraps. It is derived
// from Transaction_Contract.Type (pb/core/tron.pb.go:4806) at build time and
// is informational only — safety comes from the distinct Go types, not from
// this value. Open enum: a further kind is a new type, not a change here.
type Kind int

const (
    KindNative   Kind = iota   // non-contract transaction, no energy cost
    KindContract               // TriggerSmartContract; Simulate/EstimateEnergy apply
    KindDeploy                 // CreateSmartContract; no simulate path exists
    KindAssetTransfer          // TransferAssetContract (TRC-10 transfer, §13)
)

// Tx is the sealed interface satisfied by every transaction kind.
//
// Sealing is deliberate: without txInternal() the interface would be
// satisfiable by any caller-supplied type, and Broadcast would accept a
// hand-rolled Tx that bypasses the builders which set fee_limit, expiration
// and permission id. This is the same technique §9 uses for Arg.
type Tx interface {
    txInternal()                        // unexported; seals the set
    ID() string
    Kind() Kind
    Extension() *api.TransactionExtention   // escape hatch
    Transaction() *core.Transaction         // escape hatch
    Signers() ([]tron.Address, error)
    IsSigned() bool
    FeeLimit() tron.SUN
    Expiration() time.Time
    PermissionID() int32
}
```

Four kinds, four types. `NativeTx`, `ContractTx`, `DeployTx` and `AssetTx` each carry only the options that are meaningful for them, and each exposes read-back accessors so a caller can inspect what a builder defaulted.

The kind is decided **statically by the builder**, not by runtime inspection:

```go
func (c *Client) TransferTRX(ctx, from, to Address, amt tron.SUN) (*tx.NativeTx, error)
func (c *Client) TransferToken(ctx, from, to Address, assetName string, qty int64) (*tx.AssetTx, error)
func (h *token.Handle) Transfer(ctx, from, to Address, amt token.Amount) (*tx.ContractTx, error)
func (i *contract.Instance) Invoke(ctx, owner Address, value tron.SUN, method string, args ...Arg) (*tx.ContractTx, error)
func (c *Client) Deploy(ctx, owner Address, p DeployParams) (*tx.DeployTx, error)
```

**Deployment is in scope.** v1 exposes `smartcontract.Manager.Deploy`, `UpdateSetting` and `UpdateEnergyLimit`; omitting them would be a regression, and C3 excludes only shielded operations, TRC-10 *issuance*, and the CLI. `DeployTx` is a distinct type rather than a variant of `ContractTx` because it carries fields no other kind has — `OriginEnergyLimit` (must be > 0) and `ConsumeUserResourcePercent` — and because the protocol gives it **no simulation path**, which the type system should express rather than document.

### 6.2 This is the F1 fix

`Simulate` and `EstimateEnergy` are declared **only** on `*ContractTx`. `nativeTx.Simulate(ctx)` and `deployTx.Simulate(ctx)` are both compile errors because the methods do not exist. No dispatch to forget, no type check to omit.

> **Implementation note:** methods on a generic type instantiated in the receiver do **not** restrict the method to that instantiation — Go registers the method for all type arguments. Verified: `func (t *Tx[Contract]) Simulate()` remains callable on `*Tx[Native]`. Two distinct named types are therefore mandatory; a generic `Tx[K]` does not achieve compile-time exclusion.

### 6.3 Signing and broadcasting

```go
func (t *NativeTx)   Sign(signers ...key.Signer) (*NativeTx, error)
func (t *ContractTx) Sign(signers ...key.Signer) (*ContractTx, error)

func (c *Client) Broadcast(ctx context.Context, t Tx) (*Receipt, error)
func (c *Client) Wait(ctx context.Context, txid string) (*Receipt, error)
```

`Sign` **returns a copy** and leaves the receiver unmodified, so a partially-signed transaction cannot be shared by accident and multi-sig composes as `tx = tx.Sign(a).Sign(b)`. The kind is preserved by the return type.

**No `SignedTx` type.** Broadcasting an unsigned transaction already fails at the node with a distinct code; it is not a silent-wrong-answer defect, and doubling the type count is not justified.

### 6.4 Transaction options — fee limit, expiration, permission id

These are **not** optional in the protocol sense, and their absence was a blocking defect in the first version of this spec: TRON requires `fee_limit` for contract calls, and a transaction broadcast with `fee_limit = 0` cannot purchase energy and fails. Every builder therefore applies a documented default, and every default is overridable.

```go
// Contract-shaped transactions: fee limit, expiration and permission id.
func (t *ContractTx) WithFeeLimit(s tron.SUN) *ContractTx
func (t *ContractTx) WithExpiration(d time.Duration) *ContractTx
func (t *ContractTx) WithPermissionID(id int32) *ContractTx

// Native and asset transfers: expiration and permission id only — they
// consume no energy, so a fee limit is meaningless and is not offered.
func (t *NativeTx) WithExpiration(d time.Duration) *NativeTx
func (t *NativeTx) WithPermissionID(id int32) *NativeTx
func (t *AssetTx)  WithExpiration(d time.Duration) *AssetTx
func (t *AssetTx)  WithPermissionID(id int32) *AssetTx

// Deploy adds two fields no other kind carries.
func (t *DeployTx) WithFeeLimit(s tron.SUN) *DeployTx
func (t *DeployTx) WithExpiration(d time.Duration) *DeployTx
func (t *DeployTx) WithOriginEnergyLimit(n int64) *DeployTx
func (t *DeployTx) WithResourcePercent(p int64) *DeployTx
```

`With*` returns a copy, matching `Sign`'s copy-on-write discipline, so options compose: `ct.WithFeeLimit(tron.TRX(5)).WithPermissionID(3).Sign(a).Sign(b)`.

**Defaults, stated so they are testable:**

| Option | Default | Rationale |
|---|---|---|
| `fee_limit` (contract) | `150_000_000` SUN (150 TRX) | v1's `DefaultBroadcastOptions` value (`broadcaster.go:42`); carried over deliberately rather than invented |
| `fee_limit` (deploy) | same | deploy is the most expensive call a user makes |
| expiration | head + 60 s | protocol default |
| `permission_id` | `0` (owner) | protocol default; multi-sig under active permissions needs 2–9 |

**Expiration is a real constraint, not a nicety.** The 60 s default is the documented remedy for large multi-signer flows where an unsigned transaction must circulate between signers for longer than a minute — exactly the `With*`/`Sign` composition this spec advertises. One-process multi-sig fits inside 60 s; cross-process circulation does not, which is why `WithExpiration` is required rather than deferred.

`CostPreview` and `Simulate` must compare the effective `fee_limit` against `Estimate.Energy × SunPerEnergy` and surface `tx.fee_limit_too_low` **before** broadcast, so the default is a floor that is checked, not trusted.

### 6.5 Retry safety

`Broadcast` performs one reconciliation poll on timeout. A timeout after the broadcast has landed returns `chain.unconfirmed` **with the txid populated** and `Next = ActionWait`.

The reason is protocol-specific and the first version of this spec stated it wrongly. Re-broadcasting the **identical signed payload** cannot double-spend: a TRON txid is a pure function of `raw_data`, and the node deduplicates it with `DUP_TRANSACTION_ERROR`. The hazard is **rebuilding** — a fresh transaction gets a new TAPOS reference block and therefore a new txid, and *that* spends twice.

So `ActionWait` is correct, but the rule the agent must learn is:

> After an ambiguous timeout, never rebuild-and-resign until the original txid's receipt is confirmed absent or failed. Resending the same bytes is safe but futile; rebuilding is what loses money.

### 6.6 Receipt

```go
type Receipt struct {
    TxID     string
    Code     tron.Code
    NodeCode string          // raw api.Return_* / broadcast code, preserved — §7.4
    BlockNum uint64          // inclusion block, 0 if not yet included
    BlockTime time.Time      // block timestamp
    Cost     ActualCost
    Logs     []event.Log
    Revert   string
}
func (r *Receipt) OK() bool   // derived from Code; no stored Success field
func (r *Receipt) Solidified() bool
```

**`Wait` semantics.** `Wait(ctx, txid)` polls the **FullNode** receipt and returns as soon as the transaction is *included and executed*. That is not finality. TRON reaches practical finality when a block is **solidified** — confirmed by ≥ 19 of 27 Super Representatives, roughly a minute — and for custody or deposit-crediting use cases solidification, not inclusion, is the semantic that matters. `Receipt.Solidified()` reports it, and `WaitForSolid` is the finality-aware variant:

```go
func (c *Client) Wait(ctx context.Context, txid string) (*Receipt, error)             // included + executed
func (c *Client) WaitForSolid(ctx context.Context, txid string) (*Receipt, error)     // solidified
```

Both are served by the Solidity endpoint where the protocol provides one; `rpc` already wraps `WalletSolidity`, and `EstimateEnergy` is available on it (`pb/api/api_grpc.pb.go:6181`), so estimates are not restricted to the head-based FullNode service.

v1's `BroadcastResult` leaks `api.ReturnResponseCode` and `[]*core.TransactionInfo_Log` past the high-level boundary and represents revert data as `ConstantReturn [][]byte` requiring a nil test. v2 converts all three at the boundary.

---

## 7. Simulation, energy, and cost

Both features are contract-only, which the wire protocol confirms: `TriggerConstantContract` and `EstimateEnergy` both take `*core.TriggerSmartContract` (`pb/api/api_grpc.pb.go:297` and the `EstimateEnergy` signature at line 297's neighbour). Energy does not apply to non-contract transactions, so this is a correct restriction rather than a limitation.

### 7.1 Pricing model — what is actually true

Two independent mechanisms are commonly conflated:

- **Unit price** is a governance chain parameter, changed only by proposal. `GetEnergyPrices` returns its **history** as a comma-separated `timestamp:price` string in SUN (`pb/api/api.pb.go:1014-1016`). The entry with the greatest timestamp is the current price.
- **TIP-491 Dynamic Energy Model** is a **per-contract consumption factor**, not a price curve. A contract whose base energy use exceeds `threshold` in a maintenance period has its instruction costs scaled by `consumption_factor` (bounded by `max_factor`), decaying by `increase_factor / 4` when under threshold.

Consequence: the node already applies the penalty, so `EstimateEnergy`'s `EnergyRequired` includes it. **No client-side penalty arithmetic is required.**

### 7.2 Pre-transaction estimate

```go
func (t *ContractTx) Simulate(ctx context.Context) (*Estimate, error)
func (t *ContractTx) EstimateEnergy(ctx context.Context) (*EnergyEstimate, error)

type Estimate struct {
    ConstantResult [][]byte   // raw ABI return values
    Energy         int64      // TransactionExtention.EnergyUsed
    Penalty        int64      // TransactionExtention.EnergyPenalty (field 8, api.pb.go:1987)
    Net            int64
    Revert         string
    Code           tron.Code
}

type EnergyEstimate struct {
    Energy  int64   // total required, penalty included
    Base    int64   // pre-TIP-491
    Penalty int64   // the dynamic-model surcharge portion
}

func (e *Estimate) HasResult() bool

// Decoding is a contract concern, not a tx concern, because tx may not import
// contract (§3). The method already exists in v1 as Instance.DecodeResult.
//   inst, _ := cli.Contract(ctx, addr)
//   out,  _ := inst.Decode("balanceOf", est.ConstantResult[0])
```

`Estimate` deliberately carries **raw bytes** rather than a `*contract.Result`: `contract` depends on `tx`, so `tx` naming a `contract` type would invert the DAG and create a cycle. The cost is one extra call to decode a simulated return; the benefit is that the package graph stays acyclic and `tx` does not acquire an ABI dependency it otherwise has no reason to hold.

**Neither `Estimate` nor `EnergyEstimate` has a `TxID` field.** A simulated result cannot carry a fabricated transaction id if the field does not exist. This is the structural fix for the second half of F1.

### 7.3 Cost — the account-aware answer

"How much TRX will this cost me?" depends on the **owner's staked resources**, not only the transaction. An account holding 60,000 staked energy pays nothing for the same call. Three live reads are required:

| Read | Source | Provides |
|---|---|---|
| Energy required | `EstimateEnergy` → `EnergyRequired` | cost of the call |
| Free energy held | `GetAccountResource` → `EnergyLimit − EnergyUsed` (fields 14, 13 of `AccountResourceMessage`, `api.pb.go:1767-1768`) | what the owner already has |
| Unit price | `GetEnergyPrices` → latest `ts:price` | SUN per energy |

```
burn = max(0, energyRequired − energyAvailable) × sunPerEnergy
```

```go
func (c *Client) CostPreview(ctx context.Context, t *tx.ContractTx, owner tron.Address) (*CostPreview, error)

type CostPreview struct {
    EnergyNeeded    int64
    EnergyBase      int64
    EnergyPenalty   int64
    EnergyAvailable int64
    EnergyToBuy     int64
    TronToBurn      tron.SUN   // the answer
    SunPerEnergy    int64      // price used, so the result is auditable
    PricedAt        time.Time  // staleness is explicit
}
func (p *CostPreview) String() string   // one-line rendering for logs
```

Composable primitives for batching — a 500-transfer payout must not issue 1,500 RPCs:

```go
func (c *Client) EnergyPrice(ctx context.Context) (*EnergyPrice, error)   // cached per §7.3 cache rule
func (p *EnergyPrice) CostOf(energy int64) tron.SUN                        // pure, no I/O

// EnergyPrice is owned by tx alongside CostPreview.
type EnergyPrice struct {
    SunPerEnergy int64
    EffectiveAt  time.Time   // from the latest timestamp:price entry
    FetchedAt    time.Time
}
```

**Cache rule, stated so it is testable.** The unit price changes **only** via governance proposal, so the cache is TTL-only: refetch when `FetchedAt` is older than one maintenance period, or when the caller explicitly requests a fresh read. It must **not** be keyed on head block number — the head advances every ~3 seconds, which would refetch on every call and make the memoisation dead while the second condition stayed unreachable. That was the rule in the first version of this spec and it was self-defeating.

**Two stated limitations of `CostPreview`:**

1. **Bandwidth is not modelled.** `Receipt.Cost` reports `NetFee`, so a preview→actual delta can contain bandwidth cost the preview never mentioned. `CostPreview` therefore carries an explicit `BandwidthNote` and §12's docs must state that the preview covers energy only. Adding a bandwidth line is a v2.1 item, not a v2.0 blocker.
2. **Recipient activation is not modelled.** Sending TRX to an address that has never existed costs roughly 1.1 TRX (account creation plus bandwidth shortfall), and a contract transfer to an unactivated address costs about 25,000 extra energy. `CostPreview` cannot know the recipient's state without an account read, so it documents the delta rather than guessing it. An agent transferring to a fresh address will otherwise see an unexplained cost jump — which is exactly the class of surprise §8 exists to eliminate.

**Staleness is a stated property, not an implementation detail.** `consumption_factor` is recomputed each maintenance period, so a `CostPreview` is a **floor, not a ceiling**. `PricedAt` exists so callers can set their own tolerance.

### 7.4 Post-transaction actual cost

`core.ResourceReceipt` (`pb/core/tron.pb.go:2781-2791`) already carries `EnergyFee` (field 2) and `NetFee` (field 6) — the actual SUN burned — plus `OriginEnergyUsage` and `EnergyPenaltyTotal`. v1's `BroadcastResult` exposes only `EnergyUsage` and `NetUsage` and **discards both fee fields** (`pkg/client/broadcaster.go:72-73`).

```go
type ActualCost struct {
    EnergyFee  tron.SUN   // real TRX burned on energy
    NetFee     tron.SUN   // real TRX burned on bandwidth
    Total      tron.SUN
    Energy     int64      // EnergyUsageTotal
    BaseEnergy int64      // OriginEnergyUsage
    Penalty    int64      // EnergyPenaltyTotal
    Bandwidth  int64      // NetUsage
}
```

Exposing `CostPreview` before and `Receipt.Cost` after makes the delta between estimate and actual directly observable — which is exactly what a contract's penalty factor did to the caller mid-flight.

`NodeCode string` on `Receipt` preserves the `api.Return_*` distinction that has three different remedies: `BANDWITH_ERROR` (buy bandwidth), `CONTRACT_EXE_ERROR` (arguments), `CONTRACT_VALIDATE_ERROR` (transaction shape). Collapsing these into one mapped code loses actionable signal, so the mapped `Code` and the raw `NodeCode` are both carried.

### 7.5 To be verified by the implementor

The following two statements are derived from reading the wire protocol and TIP-491, not from execution against a node. **Verify both before implementing `EstimateEnergy` and `CostPreview`.** If either is false, adjust the affected struct's semantics — the API shape in §7.2–§7.3 stands either way.

1. **`EstimateEnergyMessage.EnergyRequired` already includes the TIP-491 penalty**, so no client-side factor multiplication is needed. Confirm by comparing `EnergyRequired` against `ResourceReceipt.EnergyUsageTotal` and `OriginEnergyUsage` for the same call on a contract known to carry a factor.
2. **`burn = max(0, required − available) × price` matches what the chain actually charges.** Confirm by executing a contract call from an account with known staked energy and comparing the computed `TronToBurn` against `ResourceReceipt.EnergyFee`.

---

## 8. Error model

### 8.1 Shape

```go
package tron

type Code string
type Action int

const (
    ActionRetry    Action = iota   // identical call may succeed; back off
    ActionWait                     // the call may have landed; poll, do NOT resend
    ActionFixCall                  // change an argument; retrying is pointless
    ActionFixTransaction           // well-formed but rejected; restructure and re-sign
    ActionFund                     // acquire a resource, then retry the identical call
    ActionBug                      // library invariant violated; stop and report
)

type Error struct {
    Code  Code
    Op    string    // "Dial", "Broadcast", "contract.Invoke"
    Hint  string    // remediation; may be empty; never duplicated in a message
    TxID  string    // populated when a transaction id is known
    Cause error
    Next  Action    // derived from Code; see 8.2
}

func (e *Error) Error() string
func (e *Error) Unwrap() error
func (e *Error) Is(target error) bool
func HasCode(err error, c Code) bool
func (c Code) Action() Action
func (c Code) Doc() string
```

**No `Msg` field.** The message is a table lookup keyed on `Code`, generated from the same source as `Action()` and `Doc()`. v1's sentinels fuse fact and advice into one string (`"invalid address: check format and ensure it's a valid TRON address"`), which means a machine cannot get the fact without the advice and a human cannot change the advice without changing the message.

**No stored `Retryable` bool and no stored `Success`.** Both are derived from `Code`, so they cannot contradict it. `Action` is an `int` enum with a `String()` method: a 6-value set gains nothing from string typing and loses nothing from a lookup.

`tron.Is` is replaced by `HasCode` — `errors.Is(err, CodeAddressInvalid)` compiles and silently returns `false`, which would disable every retry branch in the prior design.

`ActionWait` exists because "the call may have landed" is a state that neither `retry` nor `fix_call` describes, and conflating it with `retry` is the double-spend defect in §6.4.

### 8.2 Action is derived, and `Next` is the machine-readable instruction

`Action()` is a generated function over `Code`. Where an error's remedy depends on context the code alone cannot see — a `chain.timeout` before submission is `ActionRetry`, after submission it is `ActionWait` — the construction site overrides the default via `Next`, and the override is explicit in the struct rather than implied.

### 8.3 Code set

Dotted-string codes, namespaced by prefix. Prefixes are stable: every `amount.*` is `ActionFixCall`, which gives a second recovery signal for free.

```
amount.invalid          amount.too_many_decimals     amount.overflow        amount.negative
amount.decimals_mismatch

address.invalid         address.wrong_prefix

tx.no_signer            tx.already_signed            tx.expired             tx.duplicate
tx.fee_limit_too_low    tx.invalid_argument          tx.unknown_contract
tx.tapos_invalid        tx.too_large

chain.connection        chain.timeout                chain.closed           chain.unavailable
chain.unconfirmed

receipt.reverted        receipt.out_of_energy        receipt.failed

contract.not_found      contract.no_abi              contract.bad_abi       contract.bad_metadata
contract.method_unknown contract.arg_mismatch        contract.result_type_mismatch

account.insufficient_balance     account.insufficient_energy
account.insufficient_bandwidth   account.permission_denied

key.invalid             key.mnemonic_invalid
rpc.method_failed
```

`account.insufficient_*` and `account.permission_denied` **restore remediation v1 already ships and the prior design deleted.** v1's `ErrInsufficientEnergy` carries `"freeze TRX for energy or wait for energy regeneration"` — precisely an `ActionFund` hint. `contract.bad_metadata` covers a case v1 already tests: a contract returning `decimals()` packed as uint256 instead of uint8 (`pkg/trc20/decimals_uint256_test.go:17-24`).

`tx.tapos_invalid` and `tx.too_large` map the real broadcast codes `TAPOS_ERROR` and `TOO_BIG_TRANSACTION_ERROR`. The broadcast response carries a distinct enum set — `SUCCESS`, `SIGERROR`, `CONTRACT_VALIDATE_ERROR`, `CONTRACT_EXE_ERROR`, `BANDWITH_ERROR`, `DUP_TRANSACTION_ERROR`, `TAPOS_ERROR`, `TOO_BIG_TRANSACTION_ERROR`, `TRANSACTION_EXPIRATION_ERROR` — and the generated mapping in §8.5 must cover **all** of them, because `DUP_TRANSACTION_ERROR` in particular is the signal that tells an agent its resend was deduplicated rather than lost (§6.5).

### 8.4 Action semantics

| Action | Agent behaviour | Example codes |
|---|---|---|
| `retry` | Back off and repeat. Safe: the call had no effect. | `chain.connection` |
| `wait` | Poll for a result using the attached `TxID`. **Do not resend.** | `chain.unconfirmed` |
| `fix_call` | Do not retry. Re-derive an argument. | `amount.*`, `address.*`, `key.*` |
| `fix_transaction` | Rebuild with different parameters, re-sign, re-broadcast. | `receipt.*`, `tx.fee_limit_too_low` |
| `fund` | Acquire a resource, then repeat the identical call. | `account.insufficient_*` |
| `bug` | Stop. The invariant is the library's. | `tx.duplicate` on a fresh tx |

`ActionWait` is the sixth action, required because "the call may have landed" is a distinct state that neither `retry` nor `fix_call` describes, and conflating it with `retry` is the double-spend defect in §6.4.

### 8.5 Migration from v1's four mechanisms

v2 has exactly one: `*tron.Error` with a `Code`. `errors.Is(err, types.ErrInvalidAddress)` becomes `tron.HasCode(err, CodeAddressInvalid)`. `WrapTransactionResult` and `ValidateTransactionResult` are replaced by a single generated mapping from `api.Return_*` to `Code`, with the raw name preserved in `Receipt.NodeCode`.

---

## 9. Contract call results

The one place where a genuinely dynamic type appears is an ABI return value. v1 returns `interface{}`, forcing the caller to guess. v2 returns a typed `Result` with enumerable accessors, so an agent discovers the surface instead of type-switching:

```go
type Result struct{ /* raw */ }
func (r *Result) Bool() (bool, error)
func (r *Result) String() (string, error)
func (r *Result) BigInt() (*big.Int, error)
func (r *Result) Address() (tron.Address, error)
func (r *Result) Uint64() (uint64, error)
func (r *Result) Bytes() ([]byte, error)     // covers bytes32 and dynamic bytes
func (r *Result) IsNil() bool

// Decode turns a raw ABI return value into a typed Result. It is the partner
// of tx.Estimate.ConstantResult (§7.2), which tx cannot decode itself.
func (i *Instance) Decode(method string, data []byte) (*Result, error)

// Call executes a view function end to end: triggerconstantcontract, then
// decode, in one step. Without it the read path for the single most common
// agent operation after transfers is four calls plus raw-byte plumbing.
func (i *Instance) Call(ctx context.Context, method string, args ...Arg) (*Result, error)
func (i *Instance) CallAtBlock(ctx context.Context, block uint64, method string, args ...Arg) (*Result, error)

// Arg is sealed: the unexported method cannot be satisfied outside this package,
// so an agent cannot construct an invalid argument.
type Arg interface{ argABI() string }
func BoolArg(v bool) Arg
func StringArg(v string) Arg
func BigIntArg(v *big.Int) Arg
func AddressArg(v tron.Address) Arg
func Uint64Arg(v uint64) Arg
```

`Arg` constructors are named `XxxArg`, not bare `Xxx` — bare `Address(...)` collides with the `tron.Address` alias re-exported at root.

### 9.1 The ABI address rule — a requirement, not a footnote

**TRON addresses are 21 bytes with a leading `0x41`; ABI-encoded address arguments are the 20-byte form with that prefix stripped.** `AddressArg` must strip `0x41` before encoding, and `Result.Address()` must re-prepend it on decode. Sending the 21-byte value padded to 32 is documented as *the most common ABI-encoding mistake* on TRON, and it produces a call that encodes cleanly, executes, and reads the wrong slot — the exact silent-wrong-answer class this spec exists to eliminate.

This is a normative requirement with a named acceptance test: **step 8 of §14 must include a round-trip test** asserting `AddressArg(a)` encodes to the 20-byte form and `Result.Address()` of that value returns a `tron.Address` equal to `a`, for both a mainnet and a testnet address.

`Result` accessors return `(T, error)` with `contract.result_type_mismatch` on the wrong accessor, so a mistaken type assertion is a classified error rather than a zero value.

---

## 10. Root facade

```go
package tronlib // import "github.com/kslamph/tronlib/v2"

func Dial(ctx context.Context, endpoint string, opts ...DialOption) (*Client, error)

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

func TRX[T tron.Whole](n T) tron.SUN
func ParseTRX(s string) (tron.SUN, error)
func MustTRX(s string) tron.SUN

// addresses and keys
func ParseAddress(s string) (tron.Address, error)
func MustAddress(s string) tron.Address
func KeyFromHex(hex string) (key.Signer, error)
func KeyFromMnemonic(mnemonic, passphrase, path string) (key.Signer, error)

func (c *Client) Close() error
func (c *Client) Raw() *rpc.Client
func (c *Client) Endpoint() string
func (c *Client) Network(ctx context.Context) (Network, error)
func (c *Client) ChainTip(ctx context.Context) (uint64, error)
func (c *Client) TronBalance(ctx context.Context, a Address) (tron.SUN, error)
func (c *Client) TransferTRX(ctx, from, to Address, amt tron.SUN) (*tx.NativeTx, error)
func (c *Client) Deploy(ctx, owner Address, p tx.DeployParams) (*tx.DeployTx, error)
func (c *Client) Token(ctx, tokenAddr Address) (*token.Handle, error)
func (c *Client) Contract(ctx, contractAddr Address) (*contract.Instance, error)
func (c *Client) Broadcast(ctx, t tx.Tx) (*tx.Receipt, error)
func (c *Client) Wait(ctx, txid string) (*tx.Receipt, error)
func (c *Client) WaitForSolid(ctx, txid string) (*tx.Receipt, error)
func (c *Client) CostPreview(ctx, t *tx.ContractTx, owner Address) (*tx.CostPreview, error)
func (c *Client) EnergyPrice(ctx) (*tx.EnergyPrice, error)
func (c *Client) Events(ctx, txid string) ([]event.Log, error)
```

**Aliases, not wrappers.** `type Address = tron.Address` makes a `tron.Address` and a `tronlib.Address` the same type, so facade and subpackage calls interoperate with zero conversion. This is only possible because v2 is a single module (C1); the decision is coupled to it.

**`Dial` performs one round trip** (`GetChainParams`) unless `WithLazyDial()` is passed. Validating URL *shape* is not validating reachability: a `Dial` to a dead node returning `err == nil` sends the first `TronBalance` to `chain.connection → retry`, so the agent retries the wrong operation instead of switching endpoint.

**Network identity is explicit configuration, not a derivation.** An earlier version of this spec claimed mainnet addresses use `0x41` and testnet addresses use `0x65`. That is **false** and has been removed: the 21-byte address prefix is `0x41` on Mainnet, Shasta **and** Nile; the only other documented value is `0xa0`, a legacy `net.type = testnet` config value no active network uses. (`0x65` appears nowhere in the protocol — it is almost certainly `0x41` read as the decimal 65.) Two consequences:

1. `address.wrong_prefix` is a **format** check meaning "not `0x41`". It cannot discriminate networks and must not be documented as if it could.
2. `Client.Network` has no protocol source. **TRON has no chain ID**, and `GetChainParams` carries governance parameters only, no network identity. Network is therefore declared at construction and verified heuristically:

```go
// Network is explicit configuration. It is not inferred from an address byte.
type Network string

const (
    Mainnet Network = "mainnet"
    Shasta  Network = "shasta"
    Nile    Network = "nile"
    Private Network = "private"   // own genesis; no fingerprint match
)

func WithNetwork(n Network) DialOption
func (c *Client) Network() Network                    // the configured value, no I/O
func (c *Client) VerifyNetwork(ctx context.Context) error   // genesis-hash fingerprint
```

`VerifyNetwork` compares `GetBlockByNum(0)`'s block id against a table of known genesis hashes and returns `chain.network_mismatch` when it disagrees with the configured value. It is **heuristic and must be documented as such** — a private chain matches no entry, and a future testnet redeploy changes its genesis. `Dial` calls it automatically unless `WithLazyDial()` or `Network == Private`.

**`Sign` is on the transaction, not the client** (§6.3), so it is absent from this list by design. `ParseAddress`, `MustAddress`, `KeyFromHex` and `KeyFromMnemonic` are thin re-exports of `tron` and `key` constructors; they are listed here because the happy path below must compile with a single import.

### Happy path

```go
cli, err := tronlib.Dial(ctx, "grpc://grpc.nile.trongrid.io:50051")
defer cli.Close()

signer, err := tronlib.KeyFromHex(os.Getenv("TRON_PRIVATE_KEY"))
to, err := tronlib.ParseAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")

transfer, err := cli.TransferTRX(ctx, signer.Address(), to, tronlib.TRX(1))
signed, err := transfer.Sign(signer)
rec, err := cli.Broadcast(ctx, signed)
```

One import. Four imports in v1. Local identifiers deliberately avoid `tx`, `key` and `contract`, which are v2 package names and would shadow them.

---

## 11. Surface budget

**The prior design's arithmetic was wrong and this spec states real numbers.** v1's package table sums to **387** exported symbols, not 377. The prior "~120" target is unachievable: an honest count of the same design is **~265**, because folding `network` + `resources` + `voting` + `account` into `rpc` is relocation, not reduction — 48 symbols remain 48 retrievable symbols.

**Two-tier budget:**

| Tier | Contents | Budget |
|---|---|---|
| **A — curated** | `tron`, `key`, `tx`, `contract`, `token`, `event`, root | **≤ 120** exported symbols |
| **B — mechanical** | `rpc` | exempt by rule; size fixed by the TRON service |

Tier A is the surface an agent is expected to *choose between*. Tier B is a lookup table with a name.

**Measurement:** by `go/types` inside `docgen`, not by grepping `go doc`. A CI ratchet records the baseline in `api/v2.txt` and fails on growth beyond it.

**The real retrieval hazard is not total count.** Two additional rules address it directly:

- No single type exceeds **20 methods**. `rpc` fails this immediately, which is the correct signal to split it by service group.
- `Sign` / `Broadcast` / `Simulate` / `Estimate` / `Wait` must remain distinguishable by name alone; where they are not, rename rather than document.

---

## 12. Documentation as code

P11 is structural, not disciplinary: `README.md` and `docs/*.md` contain Go that nothing compiles, while `example/` is compiled.

**Rule: no hand-written Go, and no hand-written error lists, anywhere in documentation.**

1. All Go in docs lives in `Example*` functions in `_test.go`.
2. `cmd/docgen` walks the AST once and renders each example into a fenced block with a provenance comment.
3. Markdown contains markers — `<!-- go:example ExampleClient_Broadcast -->` … `<!-- /go:example -->` — that `docgen` fills.
4. `docgen` **also** fills `<!-- go:errors -->` markers in `doc.go` files and in `docs/API_REFERENCE.md` from the `tron/codes.go` const block. This is required: one dead v1 sentinel is currently advertised in four hand-maintained places (`pkg/resources/doc.go:36`, `pkg/account/doc.go:27`, `pkg/trc20/doc.go:49`, `docs/API_REFERENCE.md:1135-1163`), and an examples-only generator addresses none of them.
5. CI runs `docgen -check`; drift fails the build like `gofmt -l`.
6. The same walk produces `codes_gen.go` (`Code.Action()`, `Code.Doc()`, `AllCodes`).

**Examples must *compile*, not *run*.** "Runnable" requires a live node, which means the example does not execute in CI and P11 returns through a new door. v1 already has an in-process fake (`pkg/client/test_fakes_test.go`, `pkg/account/manager_bufconn_test.go`); the pattern exists and must be exported so examples are hermetic.

**`schema.json` is deferred to v2.1.** A bespoke machine manifest with no consumer becomes step-9 work that never lands, and its per-symbol `errors` field is wrong at package granularity — an authoritative-looking artifact that misinforms the audience it was built for. `codes_gen.go` delivers the recoverable-information value at a fraction of the maintenance cost. If a tool-call layer reaches the roadmap, emit standard JSON Schema then, not a bespoke shape now.

---

## 13. Non-goals

v2 explicitly does **not**:

- Provide shielded/Sapling operations (C3). Deferred to v2.1; `rpc` means nothing is blocked.
- Provide TRC-10 asset **issuance** (C3) — `AssetIssueCreate`, update and permission variants.
- **TRC-10 *transfer* is in scope**, as `tx.AssetTx` via `Client.TransferToken`. C3 excludes issuance, not the ability to move an asset you already hold, and `TransferAssetContract` is a native (non-TVM, no-energy) transaction that fits `NativeTx`'s category. It is a distinct type because it carries an asset-name field neither `NativeTx` nor `ContractTx` has.
- Contract **deployment is in scope** (§6.1). v1 exposes `Manager.Deploy`, so omitting it would be a regression.
- Ship a CLI (C3).
- Carry any `Deprecated:` shim (C4).
- Guarantee that a `CostPreview` equals the eventual `Receipt.Cost` — the penalty factor moves between maintenance periods (§7.3).
- Support user-defined integer types in `tron.Whole` (§5.1). Callers convert explicitly.
- Provide a `SignedTx` type (§6.3).
- Ship `schema.json` in v2.0 (§12).

---

## 14. Migration

C4 clean-room: no shims. v1 remains installable at `github.com/kslamph/tronlib` indefinitely (C1).

Build in DAG order (§3). Each step is independently shippable on the v2 branch and leaves v1 untouched.

| # | Milestone | Proves it works |
|---|---|---|
| 1 | Freeze v1 at `v1.9.0`; bugfixes only | tag exists |
| 2 | `/v2` module skeleton, `go.mod`, CI wiring | empty build green |
| 3 | `tron` — Address, SUN, Error, Code, Action, constants | **§5.2 rejection matrix executed as real compile and runtime tests** (negative-compile cases via a `go vet`-style harness or `go/types` check, panic case as a test); `codes_gen.go` generates |
| 4 | `cmd/docgen` + `docgen -check` in CI **now, not later** | a deliberately stale doc fails the build |
| 5 | `key`, `event` | sign/verify round-trip in one package |
| 6 | `rpc` — transport, pool, ~106 wrappers | bufconn fake passes |
| 7 | `tx` — four kinds, `Sign`, `With*` options, `Broadcast`, `Wait`, `WaitForSolid`, `Receipt`, cost types | F1 is a compile error in a negative test; **fee-limit default asserted non-zero on every contract-shaped builder**; ambiguous-timeout path yields `chain.unconfirmed` + txid against a fake; `Tx` sealing verified by a negative compile test |
| 8 | `contract`, `token` | `Result` accessors; `Amount` round-trips through `String`/parse; **§9.1 ABI `0x41` strip/re-prepend round-trip test**; `Instance.Call` against the bufconn fake; `Handle` accepts a 255-decimal fixture |
| 9 | Root facade + happy-path examples | one-import example compiles |
| 10 | Implementor verification of §7.5 items | recorded against a live Nile transaction |
| 11 | Generated migration guide, v1 symbol → v2 symbol, by AST diff | no hand-written table |
| 12 | Tag `v2.0.0` | |

Step 4 precedes all API work deliberately: `docgen` is the mechanism that keeps the design honest, and landing it late reintroduces the drift class it exists to prevent.

---

## 15. Residual risks

| ID | Risk | Mitigation |
|---|---|---|
| **R1** | The §7.5 pricing assumptions are unverified against a live node. | Step 10 gates `EstimateEnergy` and `CostPreview`. The API shape survives either outcome. |
| **R2** | Tier A ≤120 may still be missed once `tx` and `contract` are enumerated. Four kinds plus `With*` options raised this risk. | The ratchet in step 3 records the real number; the budget is a target, and missing it is visible rather than silent. |
| **R3** | `Code` is an open string type, so a downstream typo (`"chain.conection"`) falls through an agent's switch. | `codes_gen.go` exports `AllCodes` as the authoritative set; `HasCode` never panics on unknown codes. |
| **R4** | More transaction kinds may not cover a future contract type cleanly. | `Kind` is an open enum; a further kind is a new type, not a change to the existing four. |
| **R5** | `rpc` exemption from N1/N3 means the drop-down violates rules the curated surface enforces. | Intentional and stated: its names are a mechanical projection, not designer choices. |
| **R6** | `Sign` and `With*` returning copies allocate per call on multi-sig paths. | Multi-sig counts are single digits; correctness of non-mutation is worth the allocations. |
| **R7** | `VerifyNetwork` is heuristic. A redeployed testnet changes its genesis hash and a private chain matches nothing. | Explicit `WithNetwork` is the source of truth; `VerifyNetwork` only *detects* mismatch, never *infers* identity. |
| **R8** | The default 150 TRX fee limit is a large ceiling for small users and a possible under-estimate for heavy contracts. | It is v1's shipped value, it is overridable, and §6.4 requires `CostPreview`/`Simulate` to check it against the estimate rather than trust it. |

---

## 16. Review disposition

Every finding in `docs/reviews/2026-08-31-tronlib-v2-design-review.md`, with its verification status. Load-bearing claims were re-checked against source or the live protocol before being accepted; one review claim was found to be wrong.

| ID | Finding | Status | Action |
|---|---|---|---|
| **B1** | `fee_limit` unsettable; contract calls go out with 0 | **Accepted — verified by inspection**: §4 promised methods §6 never defined | §6.4 `With*` family + documented defaults |
| **B2** | `0x65` testnet prefix is false | **Accepted — verified against official docs**: `0x41` on Mainnet/Shasta/Nile; no `0x65`; no chain ID exists | §10 rewritten: `address.wrong_prefix` is format-only; `WithNetwork` explicit + `VerifyNetwork` heuristic |
| **B3** | Deployment in scope but unassigned | **Accepted — verified**: v1 exposes `Manager.Deploy`, so omission is a regression | §6.1 `DeployTx` as a third kind, no simulate path |
| **B4** | Expiration and permission id also undefined | **Accepted** | §6.4; expiration is the documented remedy for cross-process multi-sig circulation |
| **G1** | No block data; `Wait` finality unspecified | **Partly accepted** | §6.6 adds `BlockNum`/`BlockTime`/`Solidified()`/`WaitForSolid`. **The review's parenthetical is wrong**: `EstimateEnergy` *is* on `WalletSolidity` (`pb/api/api_grpc.pb.go:6181`), not Wallet-only |
| **G2** | No read path for view functions | **Accepted** | §9 `Instance.Call` / `CallAtBlock` |
| **G3** | ABI `0x41`-strip rule absent | **Accepted** | §9.1 as a normative requirement with a named step-8 test |
| **G4** | §6.4 rationale protocol-wrong | **Accepted — verified**: `DUP_TRANSACTION_ERROR` deduplicates identical payloads, so resend cannot double-spend; the hazard is rebuilding | §6.5 rewritten; `ActionWait` kept, reason corrected; `tx.duplicate` now mapped explicitly |
| **G5** | `EnergyPrice` cache keyed on head block refetches every ~3 s | **Accepted** | §7.3 TTL-only rule |
| **G6** | TRC-10 transfer homeless; C3 vs §3 disagree | **Accepted** | `AssetTx` + `TransferToken` in scope; §13 separates transfer from issuance |
| **P2.1** | `TRX(10e15)` is a compile error, not a panic | **Accepted — verified by execution** | §5.2 corrected; real panic row added |
| **P2.2** | Decimals cap 18 rejects valid uint8 tokens | **Accepted** | §5.4 accepts 0–255; `bad_metadata` reserved for wrong encoding width |
| **P2.3** | `Add`/`Sub` unchecked vs `Mul` checked | **Accepted** | §5.3 all three checked |
| **P2.4** | `Tx` claimed sealed but is not | **Accepted** | §6.1 adds `txInternal()` marker |
| **P2.5** | Activation cost unmodelled | **Accepted as a documented limitation** | §7.3 note; not modelled in v2.0 |
| **P2.6** | `CostPreview` ignores bandwidth | **Accepted as a documented limitation** | §7.3 note; bandwidth line deferred to v2.1 |
| **P2.7** | `Network` type undefined | **Accepted** | §10 defines it |

**Net effect on the design's spine:** none. The package DAG, the amount model, the error taxonomy, the four-kind F1 fix and docgen-before-API all survived review unchanged. Every accepted finding was a last-mile gap — a mechanism promised in one section and not defined in another, or a protocol fact stated from recollection. That is the same failure mode as P11, appearing in a document written to eliminate P11, which is worth noting as the real lesson here: **a spec that polices unverified claims still has to make them, and every one needs a source.**
