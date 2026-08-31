# tronlib v2 — API Design Specification

**Status:** Approved for spec; awaiting user review
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
├── tx           NativeTx, ContractTx, Receipt, Estimate, EnergyEstimate, CostPreview
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
| `trc10` (14) | out of scope | — (C3) |

### `utils` dissolution

| v1 symbol | v2 disposition |
|---|---|
| `EncodeAddress` / `DecodeAddress` | delete → `tron.Address` methods |
| `HumanReadableBalance`, `HumanReadableNumber`, `HumanReadableTokenAmount`, `FormatBigInt` | collapse to **two** methods: `String()` (canonical) and `Formatted()` (display) on the type owning the scale. Not one: v1's output carries comma separators *and* rounds (`conversion.go:22-25`), so a single method cannot be both round-trippable and display-friendly. |
| 10 `IsValidX` / `ValidateX` twins | keep `ValidateX() error` only. A bool cannot tell an agent *why*. |
| `SetFeeLimit`, `SetTimestamp`, `SetExpiration`, `SetPermissionID` (`tx any`) | delete → methods on `tx` types. `SetTimestamp` deleted outright — timestamps are derived. |
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
| `TRX(10e15)` | runtime panic | literal-only function |
| `ParseTRX("1.6666666")` | error | `amount.too_many_decimals` |
| `ParseTRX("+1.6")` | error | `amount.invalid` |
| `ParseTRX("1,234")` | error | `amount.invalid` |
| `ParseTRX("1e-7")` | error | `amount.too_many_decimals` |

### 5.3 Display

```go
func (s SUN) String() string      // value receiver; canonical; round-trips with ParseTRX
func (s SUN) Formatted() string   // display only: thousands separators, rounding
func (s SUN) Add(o SUN) SUN
func (s SUN) Sub(o SUN) SUN
func (s SUN) Mul(n int64) (SUN, error)
func (s SUN) Int64() int64
```

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

`decimals` is validated to ≤ 18 at construction; a malformed on-chain response produces `contract.bad_metadata` (§7.3).

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
// is informational only — safety comes from the two distinct Go types, not
// from this value. Open enum: a third kind is a new type, not a change here.
type Kind int

const (
    KindNative Kind = iota   // non-contract transaction, no energy cost
    KindContract             // TriggerSmartContract; Simulate/EstimateEnergy apply
)

// Tx is the sealed interface satisfied by both kinds.
type Tx interface {
    ID() string
    Kind() Kind
    Extension() *api.TransactionExtention   // escape hatch
    Transaction() *core.Transaction         // escape hatch
    Signers() ([]tron.Address, error)
    IsSigned() bool
}
```

The kind is decided **statically by the builder**, not by runtime inspection:

```go
func (c *Client) TransferTRX(ctx, from, to Address, amt tron.SUN) (*tx.NativeTx, error)
func (h *token.Handle) Transfer(ctx, from, to Address, amt token.Amount) (*tx.ContractTx, error)
func (i *contract.Instance) Invoke(ctx, owner Address, value tron.SUN, method string, args ...Arg) (*tx.ContractTx, error)
```

### 6.2 This is the F1 fix

`Simulate` and `EstimateEnergy` are declared **only** on `*ContractTx`. `nativeTx.Simulate(ctx)` is a compile error because the method does not exist. No dispatch to forget, no type check to omit.

> **Implementation note:** methods on a generic type instantiated in the receiver do **not** restrict the method to that instantiation — Go registers the method for all type arguments. Verified: `func (t *Tx[Contract]) Simulate()` remains callable on `*Tx[Native]`. Two distinct named types are therefore mandatory; a generic `Tx[K]` does not achieve compile-time exclusion.

### 6.3 Signing and broadcasting

```go
func (t *NativeTx)   Sign(signers ...key.Signer) (*NativeTx, error)
func (t *ContractTx) Sign(signers ...key.Signer) (*ContractTx, error)

func (c *Client) Broadcast(ctx context.Context, t Tx) (*Receipt, error)
func (c *Client) Wait(ctx context.Context, txid string) (*Receipt, error)
```

`Sign` **returns a copy** and leaves the receiver unmodified, so a partially-signed transaction cannot be shared by accident and multi-sig composes as `tx = tx.Sign(a).Sign(b)`. The kind is preserved by the return type.

**No `SignedTx` type.** Broadcasting an unsigned transaction already fails at the node with a distinct code; it is not a silent-wrong-answer defect, and four types where two suffice is not justified.

### 6.4 Retry safety

`Broadcast` performs one reconciliation poll on timeout. A timeout after the broadcast has landed returns `chain.unconfirmed` **with the txid populated** and `Next = ActionWait`, instructing the caller to poll for the receipt rather than resend.

This closes a defect in the prior design where the taxonomy marked `chain.timeout` retryable while the default broadcast path waits *after* submission — following the documented advice would resend a signed transaction and duplicate a transfer.

### 6.5 Receipt

```go
type Receipt struct {
    TxID     string
    Code     tron.Code
    NodeCode string          // raw api.Return_* name, preserved — see §7.4
    Cost     ActualCost
    Logs     []event.Log
    Revert   string
}
func (r *Receipt) OK() bool   // derived from Code; no stored Success field
```

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

**Cache rule, stated so it is testable:** `EnergyPrice` is memoised against the head block number returned by the most recent `ChainTip`, and refetched when that number changes or when `FetchedAt` is older than one maintenance period. It is never cached across a `CostPreview` that the caller has asked to be fresh.

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
func (c *Client) Token(ctx, tokenAddr Address) (*token.Handle, error)
func (c *Client) Contract(ctx, contractAddr Address) (*contract.Instance, error)
func (c *Client) Broadcast(ctx, t tx.Tx) (*tx.Receipt, error)
func (c *Client) Wait(ctx, txid string) (*tx.Receipt, error)
func (c *Client) CostPreview(ctx, t *tx.ContractTx, owner Address) (*tx.CostPreview, error)
func (c *Client) EnergyPrice(ctx) (*tx.EnergyPrice, error)
func (c *Client) Events(ctx, txid string) ([]event.Log, error)
```

**Aliases, not wrappers.** `type Address = tron.Address` makes a `tron.Address` and a `tronlib.Address` the same type, so facade and subpackage calls interoperate with zero conversion. This is only possible because v2 is a single module (C1); the decision is coupled to it.

**`Dial` performs one round trip** (`GetChainParams`) unless `WithLazyDial()` is passed, and `Network` is exposed. Validating URL *shape* is not validating reachability: a `Dial` to a dead node returning `err == nil` sends the first `TronBalance` to `chain.connection → retry`, so the agent retries the wrong operation instead of switching endpoint. `Network` is required because the design distinguishes mainnet (`0x41`) from testnet (`0x65`) addresses via `address.wrong_prefix` but must give callers a way to ask which network they are on.

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
- Provide TRC-10 asset issuance (C3).
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
| 3 | `tron` — Address, SUN, Error, Code, Action, constants | §5.2 rejection matrix passes as tests; `codes_gen.go` generates |
| 4 | `cmd/docgen` + `docgen -check` in CI **now, not later** | a deliberately stale doc fails the build |
| 5 | `key`, `event` | sign/verify round-trip in one package |
| 6 | `rpc` — transport, pool, ~106 wrappers | bufconn fake passes |
| 7 | `tx` — kinds, Sign, Broadcast, Wait, Receipt, cost types | F1 is a compile error in a negative test; §6.4 timeout path tested against a fake |
| 8 | `contract`, `token` | `Result` accessors; `Amount` round-trips through `String`/parse |
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
| **R2** | Tier A ≤120 may still be missed once `tx` and `contract` are enumerated. | The ratchet in step 3 records the real number; the budget is a target, and missing it is visible rather than silent. |
| **R3** | `Code` is an open string type, so a downstream typo (`"chain.conection"`) falls through an agent's switch. | `codes_gen.go` exports `AllCodes` as the authoritative set; `HasCode` never panics on unknown codes. |
| **R4** | Two transaction kinds may not cover a future contract type cleanly. | `Kind()` is an open enum; a third kind is a new type, not a change to the existing two. |
| **R5** | `rpc` exemption from N1/N3 means the drop-down violates rules the curated surface enforces. | Intentional and stated: its names are a mechanical projection, not designer choices. |
| **R6** | `Sign` returning a copy allocates per signature on multi-sig paths. | Multi-sig counts are single digits; correctness of non-mutation is worth one allocation. |
