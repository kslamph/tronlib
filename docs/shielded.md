# 🛡️ Shielded TRC-20

How to use TRON's shielded TRC-20 through tronlib's low-level gRPC wrappers.

This document exists because the shielded API is not discoverable: it is a dozen
RPCs that must be called in a specific order, an amount model with two different
units, and a transaction that is assembled from a blob the node hands you. None
of that is obvious from the method names.

## 📚 Learning Path

1. [Quick Start Guide](quickstart.md) - Basic usage
2. [Architecture Overview](architecture.md) - Understanding the design
3. **Shielded TRC-20** (this document)
4. [Runnable walkthrough](../example/shielded) - `example/shielded`

---

## ⚠️ Read this first: what this costs you

**There is no client-side cryptography in tronlib's shielded support.** Key
derivation, note commitments, nullifiers and zk-proof generation all happen on
the full node you connect to. The criterion for the table below is simple: a
call transmits secrets when its request carries `sk`, `ask`, `nsk`, `ovk`,
`ivk`, or the `ak`+`nk` pair that derives `ivk`. On that criterion:

| RPC | Secret the node receives |
|---|---|
| `GetSpendingKey` | generates `sk` on your behalf |
| `GetExpandedSpendingKey` | `sk` |
| `GetAkFromAsk` | `ask` |
| `GetNkFromNsk` | `nsk` |
| `GetIncomingViewingKey` | `ak` + `nk` |
| `GetZenPaymentAddress` | `ivk` |
| `GetNewShieldedAddress` | generates the whole hierarchy on the node |
| `ScanShieldedTRC20NotesByIvk` | `ivk` + `ak` + `nk` |
| `ScanShieldedTRC20NotesByOvk` | `ovk` |
| `IsShieldedTRC20ContractNoteSpent` | `ak` + `nk` |
| `CreateShieldNullifier` | `ak` + `nk` |
| `IsSpend` | `ak` + `nk` |
| `ScanNoteByIvk` | `ivk` |
| `ScanAndMarkNoteByIvk` | `ivk` + `ak` + `nk` |
| `ScanNoteByOvk` | `ovk` |
| `CreateShieldedContractParameters` | `ask` + `nsk` + `ovk` |
| `CreateShieldedTransaction` | `ask` + `nsk` + `ovk` |
| `CreateShieldedContractParametersWithoutAsk` | `nsk` + `ovk` (see below) |
| `CreateShieldedTransactionWithoutSpendAuthSig` | `nsk` + `ovk` |
| `CreateSpendAuthSig` | `ask` |

From those, the node can reconstruct your entire spending authority and decrypt
every note you own. **A malicious or compromised node can steal your shielded
balance outright, and can de-anonymise every transaction you make.** The
"shielded" property only holds against other chain observers, not against the
node you talk to.

Consequences you have to accept:

- **Run your own node.** Do not do this against a public endpoint. The example
  defaults to Nile testnet so that the worst case is testnet tokens; nothing
  here is safe for mainnet funds.
- **Fees are real even on testnet.** Nile's `getEnergyFee` is 100 sun per energy
  unit and a shielded call costs 265k–475k energy, so one mint → transfer →
  burn cycle measured 115.77 TRX. An account with a few TRX fails on the first
  mint.
- **Use `grpcs://`.** The example's default is `grpc://`, which is plaintext
  (`pkg/client/client.go` uses `insecure.NewCredentials()` for that scheme).
- **java-tron disables these RPCs by default.** Since GreatVoyage-v4.8.2,
  [`allowShieldedTransactionApi`](https://github.com/tronprotocol/java-tron/pull/6694)
  defaults to `false`, which turns off roughly all 25 shielded endpoints
  ([motivating issue](https://github.com/tronprotocol/java-tron/issues/6616)).
  Your own node must set it to `true` in `config.conf`, and upstream recommends
  doing so only for local development. If these calls fail with an opaque gRPC
  error, that flag is the first thing to check.
- **Randomness comes from the node too.** `sk`, the diversifier, `rcm` and
  `alpha` all come from `GetSpendingKey` / `GetDiversifier` / `GetRcm`. A node
  that biases or predicts them can correlate your notes to your spends.

### What tronlib deliberately does not do

Local sapling cryptography: deriving `sk → ask/nsk/ovk` offline, computing
Pedersen commitments and nullifiers locally, generating Groth16 proofs locally.
That requires the sapling circuit parameters and a mature Go implementation of
jubjub/Poseidon/Groth16 over the Jubjub curve, which is not available in this
ecosystem. It is out of scope here, deliberately.

The upstream API has a partial mitigation,
`CreateShieldedContractParametersWithoutAsk`, which keeps `ask` off the wire by
having the client sign each spend separately with `CreateSpendAuthSig`. It is
wrapped in `pkg/client/lowlevel/shielded.go` but **not** demonstrated here,
because it still requires `nsk` and `ovk` on the node and so does not change the
conclusion above.

---

## 🧭 Scope

Covered: **Shielded TRC-20 only**, five operations.

| Mode | What it does |
|---|---|
| `walletgen` | derive the key hierarchy and a `ztron1...` payment address |
| `mint` | transparent TRC-20 → shielded note (t → s) |
| `scan` | find owned notes, by `ivk` or by `ovk` |
| `transfer` | shielded note → another `ztron` address (s → s) |
| `burn` | shielded note → transparent TRC-20 (s → t) |

Not covered: **zTRX (shielded TRX)**. That is a different set of RPCs
(`CreateShieldedTransaction`, `GetMerkleTreeVoucherInfo`, `IsSpend`,
`ScanNoteByIvk`/`ByOvk`) against the built-in `ShieldedTransferContract` rather
than a user-deployed contract. Those wrappers exist in
`pkg/client/lowlevel/shielded.go`; see the
[upstream shielded transaction doc](https://github.com/tronprotocol/documentation-en/blob/master/docs/mechanism-algorithm/shielded-transaction.md)
if you need them.

---

## 🔑 The key hierarchy

One `sk` fans out into everything. Knowing which key does what is the difference
between using this API and guessing at it.

```
sk  (spending key) ─────────────── root secret, can spend
 ├── ask ──► ak      authorizing key pair — signs spend authorizations
 ├── nsk ──► nk      nullifier key pair   — blinds nullifiers
 └── ovk             outgoing viewing key — decrypts notes you SENT
      (ak, nk) ──► ivk   incoming viewing key — decrypts notes you RECEIVED
      (ivk, d)  ──► payment address (ztron1...)
```

| Key | Length | Secret? | Role |
|---|---|---|---|
| `sk` | 32 | yes | root spending key |
| `ask` | 32 | yes | authorizes a spend |
| `nsk` | 32 | yes | derives nullifiers |
| `ovk` | 32 | yes | decrypts your outgoing notes |
| `ak` | 32 | public | verifies spend authorizations |
| `nk` | 32 | public | computes nullifiers |
| `ivk` | 32 | **yes** | decrypts your incoming notes |
| `d` | 11 | public | diversifier: one `ivk` → many addresses |

`ak` and `nk` are each only half of something, but `ivk` is derived from the
pair — so any call that carries both is carrying `ivk` in all but name. That is
why the trust table above lists them as secrets even though this table calls
them public.

`ivk` and `ovk` are not spending keys, but they are secrets: either one reveals
your entire balance and transaction history.

**`ivk` and `ovk` see different things.** `ivk` decrypts notes paid *to* you
(including change you paid yourself). `ovk` decrypts notes you *sent*, including
payments to other people. To find notes you can spend, scan with `ivk`. `ovk` is
not a fallback for that — it answers a different question.

`walletgen` walks all seven derivation RPCs explicitly so the hierarchy is
visible. `GetNewShieldedAddress` does all seven in one call and returns a
`ShieldedAddressInfo` with every field filled in.

---

## 💯 The amount model — the part that silently loses funds

There are **two units**, and every bug in this area comes from confusing them.

- **raw amount** — the TRC-20's own base units. This is `from_amount` /
  `to_amount`, which the API takes as *decimal strings*.
- **note value** — the amount inside a note, an `int64`. This is `note.value`.

They are related by the contract's `scalingFactor`:

```
raw_amount == note_value * scalingFactor
```

`scalingFactor` is fixed when the shielded contract is deployed
(`10 ** scalingFactorExponent`) and is **a public getter on-chain**. Read it.
The contract enforces the relationship in `rawValueToValue`:

```solidity
require(rawValue.mod(scalingFactor) == 0, "Value must be integer multiples of scalingFactor!");
require(value < INT64_MAX);
```

So an amount that is not an exact multiple of `scalingFactor` reverts, and
truncating the division instead of checking the remainder produces a proof over
a different amount than the one transferred. `cmd/setup_nile_testnet` deploys
with exponent `0`, i.e. `scalingFactor == 1`, which is why naive examples appear
to work and then break on any other deployment. That check lives on the
transparent boundary: it applies to `mint`'s `from_amount` and `burn`'s
`to_amount`. `transfer` never touches a raw amount at all — value moves inside
the shielded set, where the only requirement is that the notes balance.

`example/shielded` keeps `-amount` in raw units (identical to `from_amount`) and
reads `scalingFactor` from the chain, so there is exactly one conversion and it
is checked.

### The note ceiling

`note.value` is an `int64` and the contract requires `value < INT64_MAX`. This
example halves that ceiling to `2**62` (`maxNoteValue` in
`example/shielded/shielded.go`) for one reason: `transfer` can combine two
notes, and the change note is derived from their sum, so keeping every note it
accepts below `2**62` means the sum cannot wrap. Notes minted elsewhere can
still arrive near the contract's own ceiling, so the example re-checks every
value with `noteValue` before putting it in a note field rather than trusting
the cap. A silent wrap would produce a negative-value note and a proof over
it.

### The value equation

Each transaction must balance, or the binding signature fails:

| Operation | Equation |
|---|---|
| `mint` | `from_amount == Σ receive.value * scalingFactor` |
| `transfer` | `Σ spend.value == Σ receive.value` |
| `burn` | `Σ spend.value * scalingFactor == Σ receive.value * scalingFactor + to_amount` |

**The `receive` side of a spend is the change note.** If you spend a note worth
more than you need and omit the change output, the difference is not returned to
you. This is the single most common way to lose money here, and it is what the
upstream `burn` example is demonstrating when it spends a 60-note to withdraw
20 and passes a second `shielded_receives` entry of 40.

---

## 🏗️ How a shielded transaction is built

Every flow ends in the same shape, and the last half is an ordinary TRON
contract call:

```
1. read scalingFactor            TriggerConstantContract  → contract state
2. find notes to spend           ScanShieldedTRC20NotesByIvk
3. get each note's merkle path   TriggerConstantContract("getPath(uint256)")
4. get randomness                GetRcm                   → rcm, alpha
5. build the proof               CreateShieldedContractParameters
                                 → trigger_contract_input   (hex string)
6. build the transaction         TriggerContract
                                    data = selector ‖ trigger_contract_input
7. sign and broadcast            local secp256k1 sign, then BroadcastTransaction
```

Steps 1–5 are shielded-specific. **Steps 6–7 are not** — a shielded transaction
is a normal `TriggerSmartContract`. That is the thing most people miss.

Measured on Nile, the energy each of those normal transactions burns:

| Operation | Call data | Energy |
|---|---|---|
| `mint` | 1060 bytes | 265,334 |
| `transfer` (1 in, 2 out) | 2628 bytes | 418,533 – 474,033 |
| `burn` (1 in, 1 out) | 1700 bytes | 365,156 |

Call data is always `4 + 32 × words`: 33 words for mint, 82 for transfer, 53 for
burn. At Nile's 100 sun per energy unit the three together cost 115.77 TRX.

### The `data` field

The node returns `trigger_contract_input`, which is the **ABI-encoded arguments
only** — no function selector. The gRPC `TriggerSmartContract` message has a
single `data` field holding selector + arguments, so you must prepend the
4-byte selector yourself:

```go
data := append(utils.EncodeMethodSignature(sig), triggerInput...)
```

Addresses inside that ABI-encoded data use the **20-byte EVM form** (the TRON
address without its `0x41` prefix), right-aligned in a 32-byte word. Passing the
21-byte `Address.Bytes()` representation produces a well-formed but meaningless
word: the call will not revert, it will just read or authorise the wrong thing.
Use `Address.BytesEVM()`.

The HTTP API expresses the same bytes as two fields, `function_selector` (the
full text signature) and `parameter` (the `trigger_contract_input`), which is
why the upstream examples look like they are passing something different.

Selectors for the three methods, derived from their signatures rather than
memorised:

| Method | Signature | Selector |
|---|---|---|
| `mint` | `mint(uint256,bytes32[9],bytes32[2],bytes32[21])` | `855d175e` |
| `transfer` | `transfer(bytes32[10][],bytes32[2][],bytes32[9][],bytes32[2],bytes32[21][])` | `9110a55b` |
| `burn` | `burn(bytes32[10],bytes32[2],uint256,bytes32[2],address,bytes32[3],bytes32[9][],bytes32[21][])` | `cc105875` |

Do **not** route this through `smartcontract.Instance.Invoke`. `Invoke` ABI-encodes
the arguments you give it and prepends the selector itself; your arguments are
already encoded, so it produces garbage. Use `lowlevel.TriggerContract` with the
raw `data`.

### The merkle path, and what "confirmed" actually means

To spend a note you need its `root` (anchor) and `path`, from the contract's
`getPath(uint256 position)` view method. The argument is a `uint256`, so the
call data is the selector plus the position left-padded to 32 bytes; the return
is `(bytes32, bytes32[32])`, both statically sized, so it decodes as 32 + 1024
inline bytes with no offset head.

`getPath` reverts unless `position < leafCount`. **That is the confirmation
check.** A note whose leaf has not been appended to the tree cannot be spent, and
the only way to know is to ask. A block-count heuristic tells you nothing.

Watch how that revert arrives. Measured against Nile, `getPath(999999)` returns
`result.result == true` with a 132-byte `constant_result` beginning
`08c379a0` — the Solidity `Error(string)` selector, decoding to
`"Position should be smaller than leafCount!"`. Checking only the `result` flag
is not enough: you would read `0x08c379a0` as if it were return data. Treat any
`constant_result` starting with `08c379a0` (`Error(string)`) or `4e487b71`
(`Panic(uint256)`) as a failure. `example/shielded` does this in `callConstant`.
A successful `getPath` returns exactly 1056 bytes (32 + 32×32), which is the
other half of the same check.

### Contract limits

From `ShieldedTRC20.sol`, and they constrain what you can express:

| Method | Inputs | Outputs |
|---|---|---|
| `mint` | none | exactly 1 |
| `transfer` | 1–2 | 1–2 |
| `burn` | exactly 1 | 0–1 |

`burn` taking one input and one change output means **you cannot combine several
notes to make one withdrawal**. To consolidate, use `transfer` (up to two inputs,
two outputs) first. `example/shielded` picks the smallest sufficient note and
reports the change it will create, so the equation is visible before you sign.

---

## 📋 The five flows

### walletgen

```
GetSpendingKey → GetExpandedSpendingKey → GetAkFromAsk → GetNkFromNsk
→ GetIncomingViewingKey → GetDiversifier → GetZenPaymentAddress
```

Persist the result along with the current block height: notes can only exist at
or after the height the address was created, and that height is what makes
scanning cheap later.

### mint (t → s)

```
GetRcm → CreateShieldedContractParameters{ovk, from_amount, shielded_receives}
→ TriggerContract(mint) → sign → broadcast
```

Mint has no spends, so it needs **neither `ask` nor `nsk`** — only `ovk`, so you
can recover the note later. It does require the shielded contract to have an
allowance on your TRC-20 balance, since the contract pulls with `transferFrom`,
and `example/shielded` sends an `approve` transaction first when the allowance is
short — so a mint can cost two transactions and two fees.

### scan

```
ScanShieldedTRC20NotesByIvk{ivk, ak, nk, start_block_index, end_block_index}
ScanShieldedTRC20NotesByOvk{ovk, ...}
```

Returns `noteTxs` with `note{value, payment_address, rcm}`, `position`,
`is_spent`, `txid`, `index`. Nodes cap how many blocks one scan may cover, so
walk long ranges in windows and merge — do not clamp the range and quietly report
"no notes found". An `ovk` scan also yields entries with `to_amount` and
`transparent_to_address` but **no `note`**: those are the transparent half of
your burns. An `ovk` scan additionally leaves `position` at 0 and `is_spent` at
false for every entry, so it cannot tell you what is spendable at all.

Sizing a scan is not theoretical: on Nile one 1000-block window measured about
1.1 s, so the default range from a `startBlock` of 59808727 to a head of 70501119
is 10,693 windows and roughly three hours. Pass `-begin`.

To check one specific note rather than rescan, use
`IsShieldedTRC20ContractNoteSpent{note, ak, nk, position, contract}`.

### transfer (s → s)

```
ScanByIvk → getPath(each spend) → GetRcm (alpha, and rcm per output)
→ CreateShieldedContractParameters{ask, nsk, ovk, shielded_spends, shielded_receives}
→ TriggerContract(transfer) → sign → broadcast
```

No `from_amount` / `to_amount` — value moves only inside the shielded set, and
must balance exactly across spends and receives.

### burn (s → t)

```
ScanByIvk → getPath → GetRcm
→ CreateShieldedContractParameters{ask, nsk, ovk, shielded_spends,
                                    shielded_receives (change),
                                    transparent_to_address, to_amount}
→ TriggerContract(burn) → sign → broadcast
```

---

## 🚚 Running the walkthrough

```bash
go run ./example/shielded -mode=walletgen
go run ./example/shielded -mode=mint     -amount=10000000
go run ./example/shielded -mode=scan     -by=ivk -begin=70501000
go run ./example/shielded -mode=transfer -amount=4000000 -to=ztron1... -begin=70501000
go run ./example/shielded -mode=burn     -amount=4000000 -begin=70501000
```

`-amount` is always raw token units — the same number that goes into
`from_amount` / `to_amount`. The token `cmd/setup_nile_testnet` deploys has
**18 decimals** and `scalingFactor == 1`, so `10000000` there is 0.00000001
tokens; with a 6-decimal token the same number would be 10 tokens.

`walletgen` writes the key file to `tmp/shielded_keys.json` (gitignored) and
prints key material only as 4-byte fingerprints unless you pass `-show-keys`.

One caveat about `-timeout`: it is both the overall deadline and the receipt-wait
window handed to `SignAndBroadcast`. The broadcaster reports a receipt timeout as
a successful broadcast with zero energy and zero net, so the example refuses to
print "confirmed" when every usage figure came back zero and says the receipt is
missing instead.

See [example/shielded/README.md](../example/shielded/README.md) for the full
flag list and the measured transcripts of a complete mint → transfer → burn run.

---

## 🔗 Upstream references

The two documents this one summarises. They are the authoritative request and
response shapes, with full worked payloads:

- [Shielded TRC-20 Contract](https://github.com/tronprotocol/documentation-en/blob/master/docs/mechanism-algorithm/shielded-TRC20-contract.md)
- [Shielded Transaction (zTRX)](https://github.com/tronprotocol/documentation-en/blob/master/docs/mechanism-algorithm/shielded-transaction.md)

Node-side behaviour and the API gating:

- [java-tron#6616](https://github.com/tronprotocol/java-tron/issues/6616) — why the shielded APIs are gated
- [java-tron#6694](https://github.com/tronprotocol/java-tron/pull/6694) — `allowShieldedTransactionApi` defaults to `false` (GreatVoyage-v4.8.2)

Reference contract used by the example deployment:
`cmd/setup_nile_testnet/test_contract/ShieldedTRC20.sol`.
