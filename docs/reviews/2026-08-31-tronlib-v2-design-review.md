# Review: docs/superpowers/specs/2026-08-31-tronlib-v2-design.md

**Reviewer basis:** official TRON developer documentation (developers.tron.network — transactions, encoding, confirmation semantics, resource model, contract types) + java-tron proto sources + independent re-execution of the spec's F1 experiment against the v1 tree at this repo.
**Date:** 2026-08-31
**Verdict:** The design is strong and its audit claims are real — I independently reproduced F1 and every wire-level citation checks out. But the spec has **three protocol-level gaps serious enough to block v2.0** (fee_limit is unsettable, the testnet-prefix premise is false, deployment is unassigned), plus one self-defeating cache rule and several smaller corrections.

---

## 1. What I verified as correct (credit where due)

The spec's "verified by execution" claims were themselves re-verified:

| Claim | Result |
|---|---|
| **F1**: `TransferContract{amount=1_000_000}` unmarshals into `TriggerSmartContract` with `err=nil`, `ContractAddress` = recipient, `CallValue`=1_000_000, `Data` empty | **Reproduced exactly** (independent run: `err=<nil>`, `ContractAddress=41040506`, `CallValue=1000000`, `Data=""`). Field-layout premise also confirmed against java-tron `smart_contract.proto` on GitHub: `call_value = 3` (varint) — fields 1–3 genuinely wire-compatible |
| Simulate fabricates TxID | Confirmed: `pkg/client/broadcaster.go:144-147` copies `ext.GetTxid()` from the constant-call response |
| quickstart recommends the F1 sequence | Confirmed at `docs/quickstart.md:455-470` |
| `EnergyPenalty` field 8 at `api.pb.go:1987`; `EnergyUsed`=13 / `EnergyLimit`=14 at 1767-1768; `ResourceReceipt` EnergyFee=2/NetFee=6/OriginEnergyUsage=3/EnergyUsageTotal=4/EnergyPenaltyTotal=8; `EstimateEnergy(*core.TriggerSmartContract)` at `api_grpc.pb.go:297`; `PricesResponseMessage.Prices string` | All confirmed |
| P1 (decimal vs SUN), P9 (fused advice in sentinels, `WrapTransactionResult`), P12 (`NewAddress[T addressAllowed]`) | All confirmed at cited locations |
| TIP-491 description (per-contract consumption factor; node applies penalty; `GetEnergyPrices` history; no client-side arithmetic) + honest §7.5 verification flags | Matches protocol; the two §7.5 verify-before-implement items are the right ones |
| §6.2 reasoning that methods on `Tx[K]` cannot be per-instantiation | Correct Go semantics; two named types is the right fix |
| §5.1 `Whole` constraint reasoning (no `~int64` → `TRX(someSUN)` rejected; no unsigned → wrap rejected) | Correct; compile errors confirmed by execution |

The amount model, the two-kind transaction split as the F1 fix, `ActionWait`, `NodeCode` preservation, copy-on-sign, eager `token.Handle` (matches the v1 `decimals_uint256` reality), and docgen-before-API are all sound.

---

## 2. Blocking findings (P0)

### B1 — `fee_limit` is unsettable anywhere in the v2 surface

TRON docs: `fee_limit` is **"Required for smart-contract calls"** — it caps the caller's total Energy usage (staked Energy *plus* the TRX-burn portion). A contract call broadcast with `fee_limit=0` cannot purchase energy and fails.

The spec:
- §4 dissolves `utils.SetFeeLimit` with "delete → **methods on tx types**"
- §8.3 defines the code `tx.fee_limit_too_low`
- §6.1–6.3 and §10 define **no such method and no default** — `token.Handle.Transfer(ctx, from, to, amt)` and `Client.Broadcast(ctx, t Tx)` have no fee-limit parameter, and the `Tx` interface (`ID/Kind/Extension/Transaction/Signers/IsSigned`) has no mutator.

As specified, every real TRC-20 transfer built through the curated surface broadcasts with `fee_limit=0` and fails. This is the same failure class as F1: the API makes a silent wrong default unavoidable.

**Fix:** define it, e.g. `func (t *ContractTx) WithFeeLimit(sun tron.SUN) *ContractTx` (and equivalently for expiration and permission id — see B4), plus a documented default (v1 examples use 5×10⁷–10⁸ sun; TronWeb defaults to 10⁹). State in §5/§6 that the default is a floor that `Simulate`/`CostPreview` should check against `Estimate.Energy × price`.

### B2 — The `0x65` testnet-prefix premise is false, and `Network()` has no mechanism

§10: *"the design distinguishes mainnet (`0x41`) from testnet (`0x65`) addresses via `address.wrong_prefix`"*.

Official TRON docs (encoding): the 21-byte address prefix is **`0x41` for all current TRON networks — Mainnet, Shasta, and Nile**. The only documented alternative is `0xa0`, a legacy `net.type = testnet` config value that **no active network uses**. There is no `0x65` anywhere in the protocol. (Likely confusion: `0x41` hex = 65 decimal.)

Consequences:
1. `address.wrong_prefix` can only mean "not `0x41`" — a format check, not a network check. It cannot distinguish mainnet from testnet.
2. `Client.Network(ctx)` has no defined protocol source. TRON has **no chain ID**; `GetChainParams` carries ~30 governance parameters and no network identity. `Dial` performing `GetChainParams` proves reachability, not network identity.

**Fix:** (a) restate the prefix rule as "all TRON addresses begin `0x41`" and validate that only; (b) make network identity explicit configuration — `DialOption WithNetwork(...)` — or derive it heuristically from a network-fingerprint (genesis block hash via `GetBlockByNum(0)`), and document that it is heuristic, not protocol.

### B3 — Deployment (`CreateSmartContract`) is in scope but unassigned

C3 excludes shielded, TRC-10 issuance, and CLI — **not** deployment. But §6.1 assigns `ContractTx` to *"TriggerSmartContract — contract calls and all TRC-20 ops"* and lists `NativeTx` contents without deployment; the `Kind` enum has no slot for it; no deploy builder appears anywhere; and the §7 justification ("Simulate/EstimateEnergy take `*core.TriggerSmartContract`") correctly implies deploy has **no** simulation path at the protocol level.

Deployment also carries fields no other tx type has — `consume_user_resource_percent`, `origin_energy_limit` (must be > 0), `call_value` to the constructor, and a required `fee_limit` — which the two-kind model has nowhere to put.

**Fix:** either (a) generalize `ContractTx` to "TVM transactions" with a `Deploy` builder that populates the deploy-specific fields, or (b) state deployment explicitly out of scope in §13 and route it through `rpc` raw. As written it is neither.

### B4 — Three "methods on tx types" promised by §4 are never defined

The §4 dissolution table deletes `SetFeeLimit`, `SetExpiration`, and `SetPermissionID` with "delete → methods on tx types", but §6 defines the `Tx` interface and `Sign`/`Broadcast` without any of them. Beyond fee_limit (B1):

- **Expiration**: TRON default is head+60 s, max 24 h. The official docs name extended expiration as the remedy for *"large multi-signer flows where the unsigned transaction needs to circulate among signers for more than 60 s"* — precisely the `tx = tx.Sign(a).Sign(b)` composition §6.3 advertises. One-process multi-sig fits in 60 s; cross-process doesn't.
- **PermissionId**: multi-sig under active permissions requires `Permission_id` ∈ 2–9; with 0 the node validates against owner permission. v1 exposed this; v2 has nowhere to set it, so multi-sig is owner-permission-only and the spec doesn't say so.

**Fix:** add the three `With*` methods (they compose naturally with copy-on-sign) or explicitly document the limitation in §13.

---

## 3. Design gaps (P1)

### G1 — `Wait` semantics and finality are unspecified; `Receipt` has no block data

Official confirmation semantics: broadcast ack ≠ inclusion ≠ execution ≠ **solidified** (19/27 SRs, ~1 min lag). The spec's `Receipt` carries no `BlockNumber`/`BlockTimestamp`, and `Wait(ctx, txid)` does not say which stage it targets. For the custody/deposit use case the docs call out, that distinction *is* the feature. Note also: `EstimateEnergy` exists **only** on `Wallet`, not `WalletSolidity` (verified in `api_grpc.pb.go`) — estimates are head-based by construction.

**Fix:** `Receipt.BlockNum`, `Receipt.BlockTimestamp`; state that `Wait` polls the FullNode receipt and add a finality-aware variant (or option) backed by the Solidity service, which `rpc` already wraps.

### G2 — No read path for view functions

N2 lists `Call` as a permitted domain verb, and §9 provides `Instance.Decode` — but no method actually *executes* a read. The implied path is `Invoke(owner, 0, "balanceOf", …) → Simulate → Decode → BigInt()`: four steps and raw-bytes plumbing for the single most common agent operation after transfers. That contradicts C2 more than any naming rule in §4 does.

**Fix:** `func (i *Instance) Call(ctx context.Context, method string, args ...Arg) (*Result, error)` — wraps `triggerconstantcontract` + decode in one step. `Simulate` remains for pre-broadcast estimates.

### G3 — The ABI `0x41`-prefix rule is absent from §9

Official docs: ABI-encoded addresses are the **20-byte form (prefix dropped)**, and *"Sending `0x41...` (21 bytes padded to 32) is the most common ABI-encoding mistake."* §9 defines `AddressArg(v tron.Address)` and `Result.Address()` without stating that encoding strips the prefix and decoding re-prepends it. Given that the spec's stated purpose is killing exactly this class of silent wrong answer, it needs an explicit requirement plus a round-trip test in §14's acceptance for step 8.

### G4 — §6.4's justification is protocol-wrong (the design is still right)

*"following the documented advice would resend a signed transaction and duplicate a transfer"* — no. On TRON a transaction ID is a pure function of `raw_data`; re-broadcasting the **identical signed payload** is deduplicated (`DUPLICATE_TRANSACTION`) and can never execute twice. The double-spend hazard is **rebuilding** the transfer (new TAPOS ref block → new txid → second spend), not resending. `ActionWait` remains the right instruction — but the spec must not teach agents a false fact to justify it. Correct framing: *resend of the same bytes is safe-but-futile (deduped or `TaposException`); never rebuild-and-resign after an ambiguous timeout until the txid's receipt is confirmed absent/failed.*

### G5 — The §7.3 `EnergyPrice` cache rule is self-defeating as written

*"memoised against the head block number returned by the most recent ChainTip, and refetched when that number changes"* — the head number changes **every block (~3 s)**, so this condition fires constantly and the memoisation never hits; the second condition (older than one maintenance period) is then unreachable. The unit price changes only via governance proposal. **Fix:** TTL-only (refetch when `FetchedAt` older than one maintenance period, or on explicit fresh request), and drop the head-number trigger.

### G6 — TRC-10 transfer has no home, and C3 vs §3 disagree

C3 excludes TRC-10 **issuance**; §3 dispositions the whole `trc10` package (14 symbols, including transfer paths) as out of scope. §6.1's `NativeTx` list omits `TransferAssetContract`, and `token` is TRC-20-only. If TRC-10 transfers are intentionally excluded, §13 must say so; if not, they belong in `NativeTx` with a `TransferAsset` builder.

---

## 4. Smaller corrections (P2)

1. **§5.2 rejection matrix, `TRX(10e15)` row is wrong** — verified by execution: `10e15` is an untyped *float* constant whose default type is `float64`, so this is a **compile error** (`float64 does not satisfy Whole`), not a runtime panic. The genuine runtime-panic case is an *int* literal above `maxSafeTRX`, e.g. `TRX(9_223_372_036_855)` → panic confirmed. Ironic in a spec that polices "verified by execution" claims.
2. **`Amount` decimals cap of 18** — TRC-20 `decimals()` is `uint8`; values up to 255 are legal on-chain. Rejecting >18 as `contract.bad_metadata` conflates *unusual but valid* with *malformed*. Either cap at 255 or document 18 as a deliberate strictness with an override.
3. **`SUN.Add`/`Sub` unchecked vs `Mul` checked** — inconsistent overflow discipline; two near-`MaxInt64` SUNs add-wrap silently. Return `error` or document why the supply bound makes it unreachable (the type admits arbitrary int64, so it isn't).
4. **"Tx is the sealed interface" (§6.1)** — it isn't: every method is exported, so anyone can implement `Tx` and hand it to `Broadcast`. If sealing matters (it should, per the F1 lesson), add an unexported marker method as §9 correctly does for `Arg`. The two halves of the spec currently disagree on sealing technique.
5. **Activation cost unmodeled** — transferring to an unactivated address burns ~1.1 TRX (creation fee + bandwidth shortfall), and contract transfers to unactivated addresses cost 25,000 extra Energy. `CostPreview` and the transfer docs should mention it; agents transferring to fresh addresses will otherwise see unexplained cost deltas.
6. **`CostPreview` ignores bandwidth** — `Receipt.Cost` includes `NetFee`, so the preview→actual delta will show bandwidth cost the preview never mentioned. Either add a bandwidth line or state that the preview is energy-only.
7. **`Network` type (§10) is undefined** — which package owns it, and what are its values, given B2?

---

## 5. Recommendation

Fix B1–B4 and G3 (one page of spec changes — they are all local edits, not structural rework) before tagging the spec as implementable. The package DAG, amount model, error taxonomy, and F1 fix are right and should not move. Everything I could execute against the v1 tree or the official protocol docs held up; the failures are all in the *last mile* where the spec promises a mechanism in one section and doesn't define it in another (fee_limit/expiration/permission_id, `Network`), or states a protocol fact the chain contradicts (`0x65`, identical-resend double-spend).

Re-run the §5.2 matrix as actual compile/runtime tests in step 3's acceptance (the spec already demands this for F1 in step 7 — extend it to the amount model), and add the ABI prefix round-trip test to step 8.
