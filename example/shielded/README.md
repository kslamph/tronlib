# Shielded TRC-20 walkthrough

A minimal end-to-end tour of TRON shielded TRC-20, built only on the low-level
gRPC wrappers in [`pkg/client/lowlevel`](../../pkg/client/lowlevel/shielded.go).

Five modes, nothing else:

```bash
go run ./example/shielded -mode=walletgen                          # derive keys
go run ./example/shielded -mode=mint     -amount=10000000          # t -> s
go run ./example/shielded -mode=scan     -by=ivk                   # find notes
go run ./example/shielded -mode=transfer -amount=4000000 -to=ztron1...   # s -> s
go run ./example/shielded -mode=burn     -amount=4000000           # s -> t
```

> **Read [docs/shielded.md](../../docs/shielded.md) before pointing this at
> anything you care about.** Every key in this example is derived by, and
> visible to, the node it talks to. Nothing here is safe for mainnet funds.

## Prerequisites

- A TRON node that has `allowShieldedTransactionApi = true` in `config.conf`.
  Since java-tron GreatVoyage-v4.8.2 the default is `false`, which disables every
  shielded RPC this example uses.
- A funded transparent account on Nile, exported as `NILE_TEST_KEY1`. **Funded
  means TRX for fees:** the three transactions below burned 115.77 TRX between
  them (Nile's `getEnergyFee` is 100 sun per energy unit, and a shielded call
  costs 265k–475k energy each). An account with a few TRX will get
  `OUT_OF_ENERGY` on the first mint.
- A deployed TRC-20 token and a `ShieldedTRC20` contract bound to it. The
  defaults match the deployment created by
  [`cmd/setup_nile_testnet`](../../cmd/setup_nile_testnet).
- A key file, which you create: `-mode=walletgen` writes one to
  `tmp/shielded_keys.json`. Nothing is bundled — see
  [Keys](#keys).

## Flags

| Flag | Default | Meaning |
|---|---|---|
| `-mode` | *required* | `walletgen` \| `mint` \| `scan` \| `transfer` \| `burn` |
| `-node` | `grpc://grpc.nile.trongrid.io:50051` | endpoint; `grpcs://` for TLS |
| `-private-key` | `$NILE_TEST_KEY1` | transparent account; pays fees, owns the TRC-20 |
| `-token` | `TWRvzd6FQcsyp7hwCtttjZGpU1kfvVEtNK` | TRC-20 contract |
| `-contract` | `TV5mhPAhsK2rXKx1FAAgz58reKwW6zSTp2` | ShieldedTRC20 contract |
| `-keyfile` | `tmp/shielded_keys.json` | where shielded keys live, relative to where you run from |
| `-amount` | | **raw** token units, exactly `from_amount` / `to_amount` |
| `-to` | | destination `ztron1...` address (`transfer`) |
| `-begin` / `-end` | key file / chain head | scan block window |
| `-by` | `ivk` | `ivk` = notes received, `ovk` = notes sent |
| `-check-spent` | `false` | `scan`: re-derive each nullifier and confirm spend status via `IsShieldedTRC20ContractNoteSpent` (ivk scans only) |
| `-fee-limit` | `350000000` | energy fee limit in sun (350 TRX) |
| `-timeout` | `5m` | overall deadline, **and** the receipt-wait window |
| `-force` | `false` | `walletgen`: replace an existing key file |
| `-show-keys` | `false` | `walletgen`: print full key material instead of 4-byte fingerprints |

Five flags fall back to an environment variable when not given on the command
line: `-node` ← `TRON_NODE`, `-private-key` ← `NILE_TEST_KEY1`, `-token` ←
`SHIELDED_TOKEN`, `-contract` ← `SHIELDED_CONTRACT`, `-keyfile` ←
`SHIELDED_KEYFILE`. The rest are flag-only.

## Keys

`walletgen` derives the sapling hierarchy through the node and writes it to
`-keyfile` with `0600` permissions. The default path is under `tmp/`, which
`.gitignore` already covers, and `example/shielded/shielded_keys.json` is
gitignored too — a key file must never sit on a path the repository tracks,
because the next `-mode=walletgen -force` would write somebody's secrets
straight into a commit (CODING_STANDARDS.md §8).

By default the derived keys are reported as 4-byte fingerprints; the payment
address is the only value printed in full, because it is the only one that is
safe to paste anywhere. `-show-keys` opts into the real material.

`walletgen` also records the chain head as `startBlock` in the file, so a later
scan knows the earliest block that could contain its notes.

## Amounts

`-amount` is in **raw token units**, the same number the API calls
`from_amount` / `to_amount`. There is no decimal conversion in this example. The
only conversion is the contract's `scalingFactor`, which is read from the chain
rather than assumed:

```
raw_amount == note_value * scalingFactor
```

Amounts that are not an exact multiple of `scalingFactor` are rejected before
anything is broadcast, because the contract's `rawValueToValue` would revert.
Note values are `int64`, and this example additionally refuses anything at or
above `2**62` (`maxNoteValue`): `transfer` can combine two notes, and their sum
is what the change note is derived from, so halving the contract's own
`INT64_MAX` ceiling keeps that sum from wrapping. Notes minted elsewhere can
still arrive near the contract ceiling, so every value is re-checked by
`noteValue` before it goes into a note field.

The bundled deployment is a plain `TRC20` with **18 decimals** and
`scalingFactor == 1`, so `-amount=10000000` there is 0.00000001 tokens, not 10.

## What each mode does

### `walletgen`

Walks the seven derivation RPCs so you can see the key hierarchy being built:

```
GetSpendingKey → GetExpandedSpendingKey → GetAkFromAsk → GetNkFromNsk
→ GetIncomingViewingKey → GetDiversifier → GetZenPaymentAddress
```

Refuses to overwrite an existing key file without `-force`, because that orphans
every note the old address owns.

### `mint` (t → s)

Transparent → shielded. Checks the TRC-20 allowance and, if the shielded contract
cannot pull the amount, **broadcasts an `approve` transaction first** — so mint
can cost two transactions and two fees. Then:

```
GetRcm → CreateShieldedContractParameters{ovk, from_amount, shielded_receives}
→ TriggerContract(mint) → sign → broadcast
```

Mint has no spends, so it needs neither `ask` nor `nsk`.

### `scan`

```
ScanShieldedTRC20NotesByIvk   (or)   ScanShieldedTRC20NotesByOvk
```

Walks the block window in 1000-block chunks and merges, because nodes cap the
range a single scan covers. Prints each note's value, position, spend status and
txid, then a total over the unspent ones.

`ivk` finds notes paid to you. `ovk` finds notes you sent, including change, and
also reports the transparent half of your burns as entries with no note. They
are not interchangeable. An `ovk` scan also leaves `position` at 0 and
`is_spent` at false, so `-check-spent` is refused on that path rather than
reporting numbers that mean nothing.

**Bound the range.** The default start is the key file's `startBlock`, which for
an address created at block 59808727 means 10,693 chunked RPCs to reach the
current head — measured at roughly 1.1 s per window, so about three hours. Pass
`-begin` for a working session:

```bash
go run ./example/shielded -mode=scan -by=ivk -begin=70501000 -check-spent
```

### `transfer` (s → s)

Shielded → shielded. Scans, picks notes, fetches each one's merkle path, and
balances the equation:

```
Σ spend.value == Σ receive.value
```

The contract allows 1–2 inputs and 1–2 outputs, so a payment plus change is the
most one transaction can express. Selection prefers a single sufficient note and
minimises the change; only when no single note covers the payment does it try
pairs.

### `burn` (s → t)

Shielded → transparent. The contract allows exactly one input and at most one
output, so `burn` cannot combine notes; the example picks the smallest note that
covers the withdrawal and returns the remainder to your own address as a change
note:

```
spend.value * scalingFactor == change.value * scalingFactor + to_amount
```

Omitting that change note is how examples here previously lost the difference.
The transparent half always goes to the `-private-key` account, which is why
`burn` refuses to start without one.

## How the transaction is assembled

The node returns `trigger_contract_input`, which is the ABI-encoded arguments
**without** a function selector. The gRPC `TriggerSmartContract` message has one
`data` field holding selector + arguments, so the selector is prepended here,
derived from the method signature rather than hardcoded:

```go
data := calldata(sig, triggerInput)   // keccak256(sig)[:4] ‖ triggerInput
```

From there it is an ordinary contract call: `TriggerContract` → sign locally →
broadcast. Nothing about steps 6–7 is shielded-specific, which is the part most
people expect to be.

Note that `smartcontract.Instance.Invoke` cannot be used for this: it ABI-encodes
its arguments itself, and yours are already encoded.

## Verified run

Everything below was executed against `grpc://grpc.nile.trongrid.io:50051` on
2026-08-29 with the contract and token named above. The three flows were run
twice, once before and once after the note-selection refactor; the call-data
sizes were identical both times.

```
$ go run ./example/shielded -mode=mint -amount=10000000
transparent account: TLibQrqpdqPyg11VBJR97Q4H2714xa9GT1
node:              grpc://grpc.nile.trongrid.io:50051
shielded contract: TV5mhPAhsK2rXKx1FAAgz58reKwW6zSTp2
scalingFactor 1: from_amount 10000000 creates a note worth 10000000 scaled units
current allowance for the shielded contract: 999999999999970000000
mint: transaction built, 1060 bytes of call data
mint: confirmed, txid 74ec05b28b05e34648db045d8da6a556f8d7d65c70d6d1b8104bb34f01b7d1f5 (energy 265334, net 0)

$ go run ./example/shielded -mode=transfer -amount=4000000 -to=ztron14870lt2... -begin=70501000
transfer plan: spend position 18 (value 10000000), pay 4000000 scaled, change 6000000 scaled
spend: position 18, value 10000000, anchor bacc05fbb61544f4ab56ac584b45ac7f5d8d6aa7b1a0b371be0bf1c2d5f71136
transfer: transaction built, 2628 bytes of call data
transfer: confirmed, txid 89e4bf811d8a09ec400914c6614098a2805869f9b4b41843be97da12f2f3186f (energy 418533, net 0)

$ go run ./example/shielded -mode=burn -amount=4000000 -begin=70501000
burn plan: spend position 20 (value 6000000), to_amount 4000000 scaled, change 2000000 scaled
spend: position 20, value 6000000, anchor dbfebefc1acc6552d707bc3b5b3a9b0f95075f62c1c0242bd14fcbfab4ea4a05
returning 2000000 scaled units of change to ztron18d55gr94mpszwge0xd37xlr6pjxg77x5mlm3spkf378htfkrrsdqecpgs76s92yx8szvxf0d7gr
burn: transaction built, 1700 bytes of call data
burn: confirmed, txid b14ad171ccab72ee8077f94b8018e17f1da76b21edc13f43a73a3692cb8f6eb6 (energy 365156, net 0)
```

The call-data sizes are exactly what the ABI says they must be: `4 + 32 × words`
— mint 33 words (1060), transfer 82 words (2628), burn 53 words (1700).

The value equation closes across the three transactions: 10,000,000 minted =
4,000,000 paid to the destination + 4,000,000 burned back to transparent +
2,000,000 left as the change note. The transparent token balance moved by
`-10000000 + 4000000 = -6000000`, and the contract's `leafCount` grew by four
(the mint note, the payment note, the transfer's change note and the burn's
change note).

The scans that confirmed it, over blocks 70501000-70501615:

```
$ go run ./example/shielded -mode=scan -by=ivk -begin=70501000 -check-spent
6 note(s)
  4. value 10000000  position 18  spent=true   txid 74ec05b28b05e34648db045d8da6a556f8d7d65c70d6d1b8104bb34f01b7d1f5
  5. value 6000000   position 20  spent=true   txid 89e4bf811d8a09ec400914c6614098a2805869f9b4b41843be97da12f2f3186f
  6. value 2000000   position 21  spent=false  txid b14ad171ccab72ee8077f94b8018e17f1da76b21edc13f43a73a3692cb8f6eb6
2 unspent note(s), 4000000 scaled units total
re-checking spend status against the nullifier table:
  position 21: is_spent=false

$ go run ./example/shielded -mode=scan -by=ivk -keyfile=tmp/dest_keys.json -begin=70501000
2 note(s)
  2. value 4000000  position 19  spent=false  txid 89e4bf811d8a09ec400914c6614098a2805869f9b4b41843be97da12f2f3186f
     payment address ztron14870lt2sweuarzqjca4vaynngmg67s5939ucyv2r88ly5v9h6w678saypkyanknplk5sydadc4w
2 unspent note(s), 8000000 scaled units total
```

The second scan is the one that matters: it reads the *destination* address's
incoming key, and finds the 4,000,000 note there. A transfer that only looked
right in the sender's own history has not paid anybody.

For the record, the older claim that scanning 59808727-59840000 returns 13 notes
was wrong by one: measured over that exact 32-window range, the bundled address
has **12 notes, 11 unspent, 110,000,000 scaled units total**.

## Files

| File | Contents |
|---|---|
| `main.go` | flags, connection, dispatch |
| `keys.go` | the key hierarchy, `walletgen`, persistence |
| `shielded.go` | chain reads, call data, scanning, note selection, the amount model |
| `flows.go` | `mint`, `scan`, `transfer`, `burn` |
| `*_test.go` | hermetic tests for the pure logic: selection, scaling, ABI encoding, revert detection, windowing |

`tmp/shielded_keys.json` is generated by `walletgen` and is not part of the
repository.

## Running the tests

```bash
go test ./example/shielded/...
```

No network, no node, no keys: the tests cover note selection, the amount model,
ABI call-data assembly, revert detection and scan windowing — the parts where a
mistake costs money rather than returning an error.
