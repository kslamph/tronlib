# tronlib v2 — Phase 2 (Core API) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the operational core of tronlib v2 — `key`, `event`, `rpc` (transport + all 103 non-shielded gRPC wrappers), `tx` (four transaction kinds with the compile-time F1 fix), `contract`, `token`, and the root facade — so that a newcomer can `Dial`, build, simulate, sign, broadcast, and read a receipt through one import.

**Architecture:** Strict DAG per spec §3, built in dependency order. `key`/`event` need only `tron`; `rpc` is the first pb importer (the `replace` directive lands there); `tx` composes tron+key+rpc and owns the four-kind model that makes the broadcaster's wire-type confusion (F1) a compile error; `contract`/`token` return `*tx` kinds; the facade aliases everything. Server-side transaction building via `CreateTransaction2` is retained (v1 behavior — the node fills `raw_data`), so `tx` build constructors take `ctx` + a connection while `Sign`/`With*` remain pure.

**Tech Stack:** Go 1.25, gRPC/protobuf (pb regenerated in Phase 1 at v4.8.2), go-ethereum `crypto` (secp256k1) + `accounts/abi` (ABI codec), `kslamph/bip39-hdwallet` (HD derivation), testify, bufconn fakes ported from v1 for hermetic tests.

**Spec:** `docs/superpowers/specs/2026-08-31-tronlib-v2-design.md` (§3 packages/DAG, §6 tx pipeline, §7 energy/cost, §9 contract results, §10 facade, §14 steps 6–11). This plan implements **steps 6–10 and 12**; step 11 (live §7.5 verification on Nile) is USER-GATED — see Task 10.

## Global Constraints

- **NO GIT PUSH. The user directed all work stays local on branch `v2`.** No subagent or command may run `git push` in any form.
- C2: LLM-agent ergonomics win. C3 scope: core API only — **shielded functions are excluded from the rpc port** (25 funcs in `lowlevel/shielded.go`). C4 clean-room: no `Deprecated:` shims.
- DAG is law: `key`/`event` import only `tron`; `rpc` imports `tron` (+ pb via module path `github.com/kslamph/tronlib/pb/...`); `tx` imports `tron, key, rpc`; `contract` imports `tron, rpc, tx`; `token` imports `tron, rpc, tx, contract`; root imports all. **No package may import the root facade.**
- pb is imported as `github.com/kslamph/tronlib/pb/api` and `.../pb/core` — the **v1 module path** — made resolvable by `replace github.com/kslamph/tronlib => ../` in `v2/go.mod` (ruling P11 execution).
- N5 carve-out: `any` is permitted ONLY as a type-parameter constraint (`rpc.Call[T any]`), never in exported param/result types.
- Every v2 command is `go -C v2 <cmd>`. Coverage floor 80% applies to `./v2/...` after every task (CI enforces; run `go -C v2 test -coverprofile=/tmp/c.out ./... && go tool cover -func=/tmp/c.out | grep total` before committing).
- docgen discipline continues: any new `Example*` function must get a marker in `v2/docs/examples.md` (the coverage check fails otherwise) and `go -C v2 run ./cmd/docgen sync-docs -pkg ./tron -docs ./docs/errors.md -docs ./docs/examples.md -check` must pass. New packages are NOT docgen targets in Phase 2 (docgen remains scoped to `tron`; generalization is deferred — noted, not a finding).
- Hermetic tests: every network-touching test uses bufconn fakes ported from v1 (`pkg/client/test_fakes_test.go`, `pkg/account/manager_bufconn_test.go`). No test may dial a real node. The ONLY network exception is the user-gated live verification (Task 10), which this plan builds but does not run.
- Error model: every exported function returns `*tron.Error` with a Code from §8.3 on failure; `Hint` carries remediation; `Op` names the function.

## File Structure

```
v2/
├── go.mod                        # + replace directive (Task 3)
├── key/
│   ├── doc.go  key.go            # Signer interface, sign/verify primitives
│   ├── privatekey.go             # hex private key signer
│   ├── hdwallet.go               # mnemonic + derivation path
│   └── message.go                # TIP-191 v2 sign/verify
├── event/
│   ├── doc.go  event.go          # Log, Decode
│   ├── registry.go               # EventDef registry (port of SimpleABIParser)
│   └── builtin.go                # TRC-20 Transfer/Approval defs
├── rpc/
│   ├── doc.go  client.go         # Dial, Client (ConnProvider impl), pool, Call[T]
│   ├── wallet.go                 # Wallet + WalletSolidity service projections
│   ├── {account,asset,block,transaction,tx,witness}.go      # Task 4 ports
│   ├── {contract,proposal,resource,other}.go                # Task 5 ports
│   └── fakes_test.go             # bufconn fake wallet (ported)
├── tx/
│   ├── doc.go  kind.go           # Kind, Tx sealed interface
│   ├── native.go  contract.go  deploy.go  asset.go   # four kinds
│   ├── options.go                # With* (copy-on-write)
│   ├── sign.go                   # Sign (copy) using key.Signer
│   ├── broadcast.go              # Broadcast/Wait/WaitForSolid + chain.* codes
│   ├── build.go                  # server-side builders (CreateTransaction2)
│   ├── estimate.go               # Simulate/EstimateEnergy (ContractTx only)
│   ├── cost.go                   # CostPreview, EnergyPrice
│   └── receipt.go                # Receipt, ActualCost
├── contract/
│   ├── doc.go  instance.go       # Instance, Call/Invoke/Decode
│   ├── result.go  arg.go         # Result accessors, sealed Arg
│   └── abi.go                    # UseABI, 0x41-strip round-trip
├── token/
│   ├── doc.go  handle.go         # eager decimals Handle
│   └── amount.go                 # Amount
├── docs/examples.md              # + facade examples (Task 9)
├── tronlib.go                    # root: Dial, Client, aliases
└── doc.go                        # facade package doc
```

---

### Task 1: `key` package

**Files:** Create `v2/key/{doc.go,key.go,privatekey.go,hdwallet.go,message.go}`, `v2/key/{key_test.go,message_test.go}`; Modify `v2/go.mod` (add `github.com/ethereum/go-ethereum`, `github.com/kslamph/bip39-hdwallet` — same versions as root).

**Interfaces (Consumes: `tron.Address`. Produces — used by Tasks 6+):**
```go
type Signer interface {
    Address() tron.Address
    PublicKey() *ecdsa.PublicKey
    Sign(hash []byte) ([]byte, error)   // raw 65-byte [R || S || V] over the given hash, no extra hashing
}
func PrivateKeyFromHex(hexKey string) (Signer, error)          // CodeKeyInvalid
func PrivateKeyFromMnemonic(mnemonic, passphrase, path string) (Signer, error) // CodeKeyMnemonicInvalid
func SignMessageV2(s Signer, message string) (string, error)   // TIP-191 v2, returns base64
func VerifyMessageV2(message, sigBase64 string, addr tron.Address) (bool, error)
```
Port from `pkg/signer/privatekey.go`, `hdwallet_signer.go`, `transaction.go` (SignMessageV2), `pkg/utils/validation.go` (VerifyMessageV2). **Value change:** `Address() tron.Address` returns the value type (v1 returned `*types.Address`). **Dep change:** the geth `common` blank import becomes a real address conversion via `tron.AddressFromHex`.

- [ ] Step 1: Write failing tests — Signer from hex derives the known address for a fixture key (port v1's fixtures); wrong-hex → `HasCode(err, CodeKeyInvalid)`; bad mnemonic → `CodeKeyMnemonicInvalid`; SignMessageV2 round-trips through VerifyMessageV2 (sign with key, verify against `key.Address()` → true; tamper message → false; wrong address → false); two signers with same key produce equal Addresses.
- [ ] Step 2: Run to confirm fail (`go -C v2 test ./key/ -v`).
- [ ] Step 3: Implement; port TIP-191 prefix logic verbatim from v1 (`pkg/signer/transaction.go:109+` and `pkg/utils/validation.go:40+`) — this is crypto-adjacent; do not improve the scheme.
- [ ] Step 4: `go -C v2 test ./key/ -v && go -C v2 vet ./... && gofmt -l v2/` green; coverage ≥80% for ./key/.
- [ ] Step 5: Write report to `.superpowers/sdd/2026-08-31-tronlib-v2-phase2/task-1-report.md` (workspace created at plan start); update `v2/docs/examples.md` with an Example marker if an Example is added; commit `feat(v2/key): signer interface with hex and HD-wallet keys`.

### Task 2: `event` package

**Files:** Create `v2/event/{doc.go,event.go,registry.go,builtin.go}` + tests; Modify `v2/go.mod` (geth `accounts/abi`).

**Interfaces (Produces — used by `tx.Receipt.Logs` and Task 7):**
```go
type Log struct {
    Address tron.Address        // emitting contract
    Topics  [][]byte
    Data    []byte
    EventName string
    Parameters []Param          // name -> decoded value
}
type Param struct{ Name string; Value any }   // any IS the decoded ABI value; documented, spec §5 exception class
func Decode(topics [][]byte, data []byte) (*Log, error)        // via registry
func RegisterABIJSON(abiJSON string) error
func RegisterABIObject(abi *core.SmartContract_ABI) error
func BuiltinTRC20() // registers Transfer/Approval
```
Port from `pkg/eventdecoder/{decoder.go,builtin.go}` (registry + builtin defs). Port the deprecation of the global-mutable-registry pattern ONLY if cheap; otherwise keep v1 semantics (global registry, documented) — decide in-task and record.

- [ ] Step 1: Failing tests — register TRC-20 ABI, decode a Transfer log (fixture topics/data from v1's `eventdecoder` tests) → EventName "Transfer", params from/to/value; unknown signature → error `CodeEventUnknown` (NEW code — add to `codes.go` + `AllCodes`, which auto-flows to codes_gen.go via docgen); tampered data length → decode error.
- [ ] Step 2: Confirm fail; implement; **run `go -C v2 run ./cmd/docgen generate-codes -pkg ./tron -out ./tron/codes_gen.go` after editing codes.go** (the fixed-point test enforces this — spec §7.4 discipline working as designed).
- [ ] Step 3: Full suite green; coverage ≥80%; report to workspace; commit `feat(v2/event): log decoding with builtin TRC-20 registry`.

### Task 3: `rpc` transport core + the P11 replace directive

**Files:** Create `v2/rpc/{doc.go,client.go,fakes_test.go}`; Modify `v2/go.mod` (+`replace github.com/kslamph/tronlib => ../`, +google.golang.org/grpc).

**Interfaces (Produces):**
```go
type ConnProvider interface {                       // 1:1 with v1 lowlevel's interface
    GetConnection(ctx context.Context) (*grpc.ClientConn, error)
    ReturnConnection(conn *grpc.ClientConn)
    GetTimeout() time.Duration
}
type Client struct{ /* pool */ }
func Dial(ctx context.Context, endpoint string, opts ...DialOption) (*Client, error)  // grpc:// | grpcs://, validates scheme
func (c *Client) Close() error
func (c *Client) GetConnection(ctx) (*grpc.ClientConn, error)     // ConnProvider impl
func (c *Client) ReturnConnection(conn *grpc.ClientConn)
func (c *Client) GetTimeout() time.Duration
func Call[T any](cp ConnProvider, ctx context.Context, operation string, call func(api.WalletClient, context.Context) (T, error), validate ...ValidationFunc[T]) (T, error)
func WithTimeout(d time.Duration) DialOption
func WithPool(init, max int) DialOption
```
Port from `pkg/client/{client.go,connection.go,common.go}` + `pkg/client/lowlevel/lowlevel.go` (Call). Pool semantics, timeout default (30s), scheme validation: port verbatim. Errors: `chain.connection`, `chain.timeout`, `chain.closed` (codes exist). Add `Network()`/`VerifyNetwork` ONLY if cheap (genesis fingerprint per spec §10) — else defer to facade task; record decision.

- [ ] Step 1: Add replace directive FIRST; `go -C v2 mod tidy`; confirm `go -C v2 build ./...` still green and pb resolves locally (`go -C v2 list -m github.com/kslamph/tronlib` shows the replace).
- [ ] Step 2: Port the bufconn fake (`pkg/client/test_fakes_test.go` → `rpc/fakes_test.go`) — tests need it; this is also Phase 2's hermetic-test foundation.
- [ ] Step 3: TDD: Dial validates scheme (grpc://grpcs://, else `chain.connection` with Hint), Call propagates conn errors as `chain.connection` and honors ctx deadlines, pool returns connections.
- [ ] Step 4: Green; coverage; report; commit `feat(v2/rpc): transport core with connection pool and Call lifecycle`.

### Task 4: `rpc` wrappers — wallet core (mechanical port, batched)

**Files:** Create `v2/rpc/{wallet.go,account.go,asset.go,block.go,transaction.go,tx.go,witness.go}`.

**Scope:** Port 1:1 from `pkg/client/lowlevel/{account,asset,block,transaction,tx,witness}.go` (10+11+16+11+2+8 = 58 funcs). **Ruling: wrappers stay FREE FUNCTIONS taking `ConnProvider`** (v1 shape) — `rpc.Client` implements ConnProvider; method-set explosion rejected (reviewer finding #8's concern, resolved by NOT converting). Signature change: `*types.Address` → `tron.Address` (pass `addr.Bytes()`); `*big.Int` stays; error returns become `*tron.Error` via the Call wrapper's operation string (Call port wraps fmt errors → wrap as `&Error{Code: CodeRPCMethodFailed, Op: operation, Cause: err}`).

- [ ] Step 1: Port `block.go` + `witness.go` first (includes `GetPaginatedNowWitnessList` — the v4.8.2 addition, both Wallet and WalletSolidity variants; spec §3). Test each with the bufconn fake: happy path returns the fake's message; node error → `HasCode(err, CodeRPCMethodFailed)` with Op = wrapper name.
- [ ] Step 2: Port `account.go`, `asset.go`, `transaction.go`, `tx.go` the same way. `TxCall`/`ValidateTransactionResult` port from `lowlevel/tx.go` — the `api.Return_*` → `tron.Code` mapping table belongs HERE (one table; spec §8.5) with `NodeCode` preserved for the Receipt later.
- [ ] Step 3: Coverage ≥80% across ./rpc/; report; commit `feat(v2/rpc): wallet-core gRPC wrappers (1:1 port)`.

### Task 5: `rpc` wrappers — remaining services + exclusion

**Files:** Create `v2/rpc/{contract.go,proposal.go,resource.go,other.go}`.

**Scope:** Port `pkg/client/lowlevel/{contract,proposal,resource,other}.go` (9+6+13+17 = 45 funcs). **EXCLUDE all 25 shielded funcs** (C3) — delete, do not port; the ledger records the count (103 ported + 25 excluded = 128 total v1 surface). `other.go` may contain node-info/misc calls — port all non-shielded. Include `DeployContract` (Task 6's DeployTx needs it).

- [ ] Step 1: Port with tests per service group (bufconn fake).
- [ ] Step 2: Verify count: `grep -c "^func " v2/rpc/*.go` — record the number in the report; confirm zero shielded names (`grep -ri shielded v2/rpc/` returns nothing).
- [ ] Step 3: Full suite; commit `feat(v2/rpc): remaining service wrappers; shielded excluded per C3`.

### Task 6: `tx` package — the four kinds (the F1 fix, design task)

**Files:** Create `v2/tx/{doc.go,kind.go,native.go,contract.go,deploy.go,asset.go,options.go,sign.go,broadcast.go,build.go,estimate.go,cost.go,receipt.go}` + tests; this is the largest design task.

**Interfaces (Consumes: tron, key, rpc. Produces: everything contract/token/facade need):**
```go
type Kind int; const ( KindNative Kind = iota+1; KindContract; KindDeploy; KindAssetTransfer )  // spec §6.1
type Tx interface { txInternal(); ID() string; Kind() Kind; Extension() *api.TransactionExtention; Transaction() *core.Transaction; Signers() ([]tron.Address, error); IsSigned() bool; FeeLimit() tron.SUN; Expiration() time.Time; PermissionID() int32 }
type NativeTx struct{...}; type ContractTx struct{...}; type DeployTx struct{...}; type AssetTx struct{...}
// builders (server-side build via CreateTransaction2 — ctx+ConnProvider, I/O):
func BuildTransfer(cp rpc.ConnProvider, ctx context.Context, from, to tron.Address, amt tron.SUN) (*NativeTx, error)
func BuildTriggerSmartContract(cp, ctx, owner tron.Address, contract tron.Address, data []byte, callValue tron.SUN) (*ContractTx, error)
func BuildDeploy(cp, ctx, owner tron.Address, p DeployParams) (*DeployTx, error)
func BuildAssetTransfer(cp, ctx, from, to tron.Address, assetName string, qty int64) (*AssetTx, error)
// options (copy-on-write, pure):
func (t *ContractTx) WithFeeLimit(s tron.SUN) *ContractTx   // default 150_000_000
func (t *ContractTx) WithExpiration(d time.Duration) *ContractTx  // default head+60s
func (t *ContractTx) WithPermissionID(id int32) *ContractTx
// NativeTx/AssetTx: WithExpiration + WithPermissionID only. DeployTx: +WithOriginEnergyLimit +WithResourcePercent.
// sign (copy-on-write; pure):
func (t *NativeTx) Sign(signers ...key.Signer) (*NativeTx, error)   // same for other kinds
// lifecycle:
func Broadcast(cp rpc.ConnProvider, ctx context.Context, t Tx) (*Receipt, error)
func Wait(cp, ctx, txid string) (*Receipt, error)                   // FullNode receipt
func WaitForSolid(cp, ctx, txid string) (*Receipt, error)           // Solidity endpoint
// contract-only (F1 fix — methods exist ONLY on *ContractTx):
func (t *ContractTx) Simulate(ctx context.Context) (*Estimate, error)
func (t *ContractTx) EstimateEnergy(ctx context.Context) (*EnergyEstimate, error)
func CostPreview(cp, ctx, t *ContractTx, owner tron.Address) (*CostPreview, error)
```
Semantics per spec §6: `Sign` returns a copy; `Broadcast` = submit + one reconciliation poll, timeout → `chain.unconfirmed` + TxID + `Next: ActionWait` (§6.4 — the double-spend fix); `Receipt` has `BlockNum/BlockTime/Solidified()/NodeCode/Cost/Logs/Revert`, no stored `Success`/`TxID` on Estimate (§7.2 B5 fix); `ActualCost` from `ResourceReceipt.EnergyFee/NetFee/EnergyPenaltyTotal` (§7.4). **Kind dispatch:** builders construct the kind directly (statically known) — `Broadcast` accepts the sealed `Tx` interface; `Simulate`/`EstimateEnergy` exist only on `*ContractTx`, so `nativeTx.Simulate` is a compile error (spec §6.2). `Estimate`/`EnergyEstimate` have NO TxID field.

- [ ] Step 1: Kind + Tx interface + the four structs (wrapping pb types via `Extension()`/`Transaction()` accessors) with `txInternal()` seal. Test: a foreign type cannot satisfy Tx (negative-compile fixture in the existing compilecheck suite — extend it).
- [ ] Step 2: Options — copy-on-write (test: original unchanged after With*).
- [ ] Step 3: Builders via `CreateTransaction2` through rpc wrappers (bufconn fake returns a canned TransactionExtention). Test: built NativeTx has Kind KindNative, ID() = hex txid, fee limit default 150_000_000.
- [ ] Step 4: Sign — copy semantics (test: receiver unchanged, new tx has signature), multi-sig composes, `tx.already_signed` on... (no such state — signatures accumulate; skip), PermissionID honored via `WithPermissionID` before Sign.
- [ ] Step 5: Broadcast/Wait/WaitForSolid against the fake: success receipt, node reject → mapped code + NodeCode, timeout → `chain.unconfirmed` + TxID + Next=Wait (the §6.4 test the spec demands).
- [ ] Step 6: Estimate/EnergyEstimate on ContractTx; `nativeTx.Simulate` must NOT compile (extend compilecheck).
- [ ] Step 7: CostPreview per §7.3 (three reads: EstimateEnergy + GetAccountResource + GetEnergyPrices — all faked); `contract.bad_metadata` on malformed decimals-width metadata (§7.3's v1-tested case). **§7.5 live verification is NOT run here (user-gated); mark both functions' docs with "live-verified: pending (spec §7.5)".**
- [ ] Step 8: Full suite + coverage; report; commit `feat(v2/tx): four transaction kinds with static-kind simulation and copy-on-sign`.

### Task 7: `contract` package

**Files:** Create `v2/contract/{doc.go,instance.go,result.go,arg.go,abi.go}` + tests.

**Interfaces (Produces):**
```go
type Arg interface{ argABI() string }   // sealed — spec §9
func BoolArg(bool) Arg; func StringArg(string) Arg; func BigIntArg(*big.Int) Arg; func AddressArg(tron.Address) Arg; func Uint64Arg(uint64) Arg
type Result struct{...}  // Bool/String/BigInt/Address/Uint64/Bytes/IsNil accessors, (T, error) + CodeContractResultTypeMismatch
type Instance struct{...}
func UseABI(json string) error   // or constructor variant per spec §5 table: NewInstance fetches ABI lazily on first use
func (i *Instance) Call(ctx, method string, args ...Arg) (*Result, error)           // read path (review G2)
func (i *Instance) CallAtBlock(ctx, block uint64, method string, args ...Arg) (*Result, error)
func (i *Instance) Invoke(ctx, owner tron.Address, value tron.SUN, method string, args ...Arg) (*tx.ContractTx, error)
func (i *Instance) Decode(method string, data []byte) (*Result, error)
```
ABI address rule (spec §9.1, normative): `AddressArg` strips the 0x41 prefix (20-byte encode); `Result.Address()` re-prepends. **Round-trip test is mandatory** (named in spec §14 step 8 acceptance).

- [ ] Step 1: Arg types + seal test (foreign impl must not compile — compilecheck fixture).
- [ ] Step 2: ABI round-trip test: `AddressArg(a)` encodes 20 bytes; `Result.Address()` of that returns `a` — mainnet and testnet-form addresses.
- [ ] Step 3: Instance.Call against bufconn fake (TriggerConstantContract canned); Result accessors incl. `contract.result_type_mismatch` on wrong accessor.
- [ ] Step 4: Invoke returns `*tx.ContractTx` (proving the DAG direction contract → tx works).
- [ ] Step 5: Green; report; commit `feat(v2/contract): typed ABI calls with sealed args and the 0x41 round-trip rule`.

### Task 8: `token` package

**Files:** Create `v2/token/{doc.go,handle.go,amount.go}` + tests.

**Interfaces:**
```go
type Amount struct{ raw *big.Int; decimals uint8 }   // raw never nil; 0..255 decimals accepted (review finding)
func (a Amount) Raw() *big.Int   // COPY
func (a Amount) String() string  // canonical
type Handle struct{...}          // immutable; decimals fetched at construction
func New(cp rpc.ConnProvider, ctx context.Context, contract tron.Address) (*Handle, error)  // eager decimals(); contract.bad_metadata on wrong width
func (h *Handle) Contract() tron.Address
func (h *Handle) Decimals() uint8
func (h *Handle) Amount(s string) (Amount, error)
func (h *Handle) Whole[T tron.Whole](n T) Amount
func (h *Handle) BalanceOf(ctx, owner tron.Address) (Amount, error)
func (h *Handle) Transfer(ctx, from, to tron.Address, amt Amount) (*tx.ContractTx, error)
```
Port the v1-tested decimal-width validation (`decimals_uint256_test.go` case → `contract.bad_metadata`).

- [ ] Step 1: Amount + Handle with fake decimals(); malformed metadata test. Step 2: Transfer returns *tx.ContractTx. Step 3: green; report; commit `feat(v2/token): TRC-20 handle with eager decimals and immutable amounts`.

### Task 9: root facade

**Files:** Create `v2/tronlib.go`, `v2/doc.go` (already has stub from Phase 1 — extend), `v2/facade_test.go`, examples in `v2/example_test.go`; Modify `v2/docs/examples.md` (markers for new Examples — coverage check enforces).

**Surface (spec §10, exact):** aliases (Address, SUN, Signer, Code, Action, Error, Tx, NativeTx, ContractTx, Receipt, Log), `TRX/ParseTRX/MustTRX`, `ParseAddress/MustAddress/KeyFromHex/KeyFromMnemonic`, `Dial`, and Client methods: Close/Raw/Endpoint/Network/VerifyNetwork/ChainTip/TronBalance/Witnesses/TransferTRX/TransferToken/Token/Contract/Deploy/Broadcast/Wait/WaitForSolid/CostPreview/EnergyPrice/Events. Thin delegations ONLY (spec D7: facade never reimplements).

- [ ] Step 1: Aliases + constructors (one-liners over subpackages). Step 2: Client wrapping rpc.Client with the passthroughs; facade methods one-line delegate. Step 3: Happy-path Example (spec §10's 6-line program) — compiles against the bufconn fake (Example bodies must not dial; use the fake via a non-network Example form or keep the Example compile-only with a comment — decide: Examples must COMPILE (docgen extracts bodies; they need not run without network since `go test` skips non-Output examples — use no `// Output:` comment so it compiles but doesn't execute).
- [ ] Step 4: docgen sync + `-check` green (new example markers added). Step 5: full suite; report; commit `feat(v2): root facade — one import for the happy path`.

### Task 10: Phase 2 closeout — verification checklist (USER-GATED item inside)

**Files:** Create `v2/PHASE2.md` (short); Modify nothing else.

- [ ] Step 1: Full verification: `go -C v2 build ./... && go -C v2 test ./... -count=1 && go -C v2 vet ./... && gofmt -l v2/` (known testdata exception only); coverage floor ≥80%; docgen `-check` green.
- [ ] Step 2: Write `v2/PHASE2.md`: what shipped, the P11 tag-time decision (pb dependency story: v1.9.1+ re-tag OR v2/pb OR pb-module — spec §14 step 12 references it), the migration-guide note (step 12: generated by AST diff — deferred to the tag cycle), and the **§7.5 live-verification checklist** for the user: two Nile transactions (one with staked energy, one without) comparing `CostPreview.TronToBurn` vs `ResourceReceipt.EnergyFee`, confirming `EnergyRequired` includes the TIP-491 penalty. **Do not run it** — it needs a funded Nile key only the user controls.
- [ ] Step 3: Commit `docs(v2): phase 2 closeout and live-verification checklist`. **NO PUSH.**

---

## Deferred (explicitly not in Phase 2)

- Step 11 (live §7.5 verification) — user-gated, checklist provided in Task 10.
- Step 12 full migration guide (AST diff v1→v2) — deferred to the tag cycle with the P11 decision.
- Step 13 (tag v2.0.0) — gated on step 11 + P11.
- docgen generalization to non-`tron` packages (hard-coded `package tron` check) — Phase 2.1.
- Shielded (C3), TRC-10 issuance (C3), CLI (C3).

## Self-Review

**Spec coverage (§14 steps 6–10, 12):** 6→Tasks 1–2; 7→Tasks 3–5; 8→Task 6; 9→Task 7; 10→Tasks 8–9; 12 deferred-with-note (Task 10 records the decision gate). **P11 execution** lands in Task 3 (first pb importer). **§9.1 round-trip test** lands in Task 7. **§6.4 timeout test** lands in Task 6 Step 5. **F1 compile-error proof** lands in Task 6 Step 6 via compilecheck extension. **No task touches v1.**

**Placeholder scan:** port tasks (4–5) reference existing in-repo source files as their spec — that is a complete specification for mechanical ports. Builder signatures in Task 6 are complete. No TBDs.

**Type consistency:** `Tx` interface members match spec §6.1 exactly; `Receipt` fields match §6.6; `CostPreview` fields match §7.3; `Result` accessors match §9; facade methods match §10 (minus `Sign` which lives on tx types per spec §6.3). `CodeEventUnknown` is the one new Code (Task 2) — flows through docgen parity automatically.

**Known risk:** Task 6 is the largest design task since the original Task 7 timeout — it is split into 8 internal steps and may need the same continuation pattern if it stalls. Budget accordingly.
