# tronlib v2 — verification ledger

Every on-chain claim the library makes, with the evidence behind it and
the exact commands to re-check it. Written for a reviewer who wants to
evaluate the project **without repeating the setup**: each broadcast
entry carries its full txid, block, and numbers; each read-only entry
carries the exact command that reproduces it. Nothing here runs in CI —
records are history, not gates.

Conventions: `NILE=https://nile.trongrid.io`,
`MAIN=https://api.trongrid.io`, `GRPC_NILE=grpc://grpc.nile.trongrid.io:50051`,
`GRPC_MAIN=grpc://grpc.trongrid.io:50051`.
Receipt fetch template (replace `$BASE` and `$TXID`):

```sh
curl -s -X POST $BASE/wallet/gettransactioninfobyid \
  -H 'Content-Type: application/json' -d "{\"value\":\"$TXID\"}"
```

Energy/bandwidth replays for **any** txid below need no key and no spend —
one command re-verifies both resources (see §4):

```sh
go run ./cmd/tip491probe -endpoint $GRPC -replay $TXID
```

## 1. Broadcast evidence (spend transactions)

### E1 — CostPreview accuracy, penalty-free (Nile, 2026-09-01)

- txid `738c6d0e10d2ba325577a38463b93af19209612008fd64fb02ae7f4f7153ac31`
- block 70573724 · TRC-20 `transfer` on self-issued `TWRvzd6FQcsyp7hwCtttjZGpU1kfvVEtNK` (18 decimals)
- Claim: `Simulate.Energy` (13,569) == `ResourceReceipt.EnergyUsageTotal`
  (13,569); `CostPreview.TronToBurn` (1,356,900) == `EnergyFee`
  (1,356,900). Delta **0 SUN** on all three comparisons (parsed and raw pb).
- Bandwidth side-record: `NetUsage` 0, `NetFee` 345,000 = 345 bytes ×
  1,000 (burn shape; the key had no free bandwidth left).
- Re-check: receipt fetch on `$NILE`; replay via the probe command above
  (still exact weeks later — the sender still holds the token).

### E2 — Simulate == receipt on a penalized contract (Mainnet, 2026-09-28)

- txid `41808e02d08669098860742b19bba2e16de29da6c065725e6394495d0b3ec2e8`
- block 86642154 · USDT `transfer` (`TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t`)
- Claim: replay energy=130,285 penalty=100,635 base=29,650 == receipt
  exactly. Extends E1 to TIP-491-penalized execution (§7.5 item 2).
  *(Amended by the R11 audit: this row originally said penalty 49,635 — a
  mis-transcription; energy = base + penalty reconciles only with 100,635,
  and the 2026-09-30 replay re-read the immutable receipt at 100,635.)*
- Re-check: receipt fetch on `$MAIN`; probe replay (needs the sender to
  still cover the amount — if it reverts now, see E6).

### E3 — Bandwidth burn shape (Nile, 2026-09-01)

- Same txid as E1. Claim: `NetFee` 345,000 == 345 bytes × 1,000
  (`getTransactionFee`), with `NetUsage` 0. Confirms the burn reporting
  rule and the +64 result-size overhead (345 ≈ 281 + 64).

### E4 — Bandwidth covered shape (Mainnet, 2026-09-28)

- Same txid as E2. Claim: `NetUsage` 345, `NetFee` 0 — byte count lands
  in usage exactly when resources cover it.

### E5 — Account-creation path with full balance accounting (Nile, 2026-09-28)

Funded run: fresh key `TFajiYgytkFiBpustBNQSFEDsiHBoAFrBo` received
52.3869 TRX (faucet + user top-up).

- txid `4abe41a6a6b5a3a029f312bc073a1b356c41326a4d2ff1b3c81bef805009c13b`
- block 71358212 · 10 TRX to fresh `TCvU2ENS6sksbhcz2x7jX87yrrCWZLt7Rf`
- Predicted before broadcast: CreatesAccount, Burn=100,000, NewAccountFee=1,000,000.
- Receipt: `NetUsage` 0, `NetFee` 100,000 — exact.
- Balance: 52.3869 − 10 − 0.1 − 41.2869 = **1.000000 TRX drift exact** —
  the 1 TRX creation burn appears in no receipt field and is proven only
  by this accounting.
- txid `ef1660a59cef503ba50adfa84b978a31891f4672eeb57ddfa34e724a43303420`
- block 71358223 · 1 TRX to the now-existing address above.
- Predicted free-covered (need 273). Receipt: `NetUsage` 273, `NetFee` 0
  — exact. Balance drift 0.

### E6 — Stale replay reports void, not mismatch (Mainnet, 2026-09-28)

- txid `646e6d4494a9d9d072b50f62c5a352bca4a0ef45df89fd35271d57f64eb5df3f`
- block 86642154 · the sender moved its funds after broadcasting, so a
  replay reverts (`REVERT opcode executed`, energy 8,624).
- Claim: the harness reports "comparison void" via `est.Code/Revert`
  instead of a numeric mismatch. Re-run the probe replay to observe it.

### E7 — First DeployTx broadcast (Nile, 2026-09-28)

- txid `e40e069a17318bfccf1fff259ebde7d761a30fb20513f73a68b6b04cfb34b34a`
- block 71359418 · minimal 1-byte-STOP init, name `tronlib-probe`.
- Deployed contract `TGHsNCnqw9fnwB86NcKU36xdoT9BPQUPqV`, runtime `00`
  (verify: `wallet/getcontractinfo` with the address).
- Receipt: energy=221, penalty=0, bandwidth=312 covered, EnergyFee 2,210,
  NetFee 0. Balance drift 0. Total spend 2,210 SUN.
- Claim: top-level deploys pay init-exec only — no 32,000 internal-CREATE
  price; Simulate==receipt holds for deploys (constant 221 == receipt 221).

## 2. Read-only evidence (no key, no spend, re-runnable)

### R1 — TIP-491 factor live on mainnet USDT (2026-09-28)

```sh
go run ./cmd/tip491probe -endpoint $GRPC_MAIN \
  -contract TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t \
  -owner TLibQrqpdqPyg11VBJR97Q4H2714xa9GT1 \
  -data a9059cbb00000000000000000000000075e5d9f6ffa84694803e32e07aadf1b5d89528cf
```

Observed: `Simulate` energy=63,999 penalty=49,415; stored factor=34,000
(= `getDynamicEnergyMaxFactor`, pinned at the ceiling); predicted 49,585
vs actual 49,415; derived 33,883 vs stored 34,000. Probe exit 0.
(The public gateway rate-limits; the probe retries read-only steps.)

### R2 — Deploy estimation matrix (Nile, 2026-09-28)

`triggerconstantcontract` with init bytecode as `data` and no contract
address returns the complete deploy energy (init + 200/byte deposit).
Observed `energy_used`: 12B init + 0B runtime → 15; +1B → 221; +32B →
6,421 (6,200/31 = 200.0/byte exact); +1000B → 200,209. Re-check any row:

```sh
curl -s -X POST $NILE/wallet/triggerconstantcontract \
  -H 'Content-Type: application/json' \
  -d '{"owner_address":"413d90d3d2b78e2a2d1c0b6315c80a2bd43ecf1113","data":"<hex>"}'
```

### R3 — TRC-20 metadata + selectors (Mainnet USDT, 2026-09-28)

decimals 6, `Tether USD` / `USDT`, totalSupply 94,258,350,557.07,
allowance 0 for the probed pair; approve calldata selector `095ea7b3`,
transfer `a9059cbb`. Via `token.Handle` + `contract.Instance` (any agent
reproduces with `cli.Token`).

### R4 — Chain parameters (Mainnet = Nile, 2026-09-28)

`getTransactionFee` 1,000, `getCreateAccountFee` 100,000,
`getCreateNewAccountFeeInSystemContract` 1,000,000,
`getCreateNewAccountBandwidthRate` 1, `getFreeNetLimit` 600,
`getDynamicEnergyThreshold` 5,000,000,000,
`getDynamicEnergyIncreaseFactor` 2,000, `getDynamicEnergyMaxFactor` 34,000.
Re-check: `wallet/getchainparameters` on either gateway.

### R5 — Management builds reach the node (Mainnet USDT, 2026-09-28)

`UpdateSetting` / `UpdateEnergyLimit` / `ClearABI` builds return the
node's deployer-only rejection (not a client error) — wire path proven;
the success path is now live-verified (R5 below).

### R6–R8 — §7.5 live-verification closeout (2026-09-29, Nile, funded keys)

Run on branch `v2` via a throwaway `go run` harness using the repo's
committed throwaway Nile keys (key1 =
`TLibQrqpdqPyg11VBJR97Q4H2714xa9GT1`, key2 = `TLibCZ2i2dFp6a9KZeKriSms5peeXSibks`).
These close the three "NOT proven" items below.

- **R6 — staked-energy branch (`max(0, needed − available)`, nonzero
  availability).** Key1 carries `EnergyLimit=12005` of staked energy. A
  TLT transfer (`TWRvzd6FQcsyp7hwCtttjZGpU1kfvVEtNK`, 0.1 TLT to key2):
  `CostPreview` → EnergyNeeded **28569**, EnergyBase 28569, EnergyPenalty 0,
  EnergyAvailable **12005** (nonzero, < needed), EnergyToBuy **16564**,
  TronToBurn **1,656,400 SUN**, SunPerEnergy 100,
  PricedAt 2026-09-29T15:02:06+08:00. Broadcast
  `5b1e65f9826dc0c94366f247f716b81718930d11d34faf744764a1e5847e7763`
  (block **71379452**): receipt `energy_usage` **12005** (exactly the
  staked energy consumed), `energy_usage_total` **28569**, `energy_fee`
  **1,656,400**, `net_usage` 345 (free-covered, netFee 0). Raw pb
  `ResourceReceipt.EnergyFee` = 1,656,400. Predicted burn == actual
  energyFee, **delta 0** — the stake consumes 12005 and only the 16564
  shortfall is bought, exactly as `EnergyToBuy` predicted.
- **R7 — `insufficient_bandwidth` live.** Key2 drained to **526,000 SUN**
  with free bandwidth exhausted (586/600). A raw
  `TriggerSmartContract` to TLT with 604 bytes of calldata (a contract
  call does not pre-validate the bandwidth fee; a `TransferContract` of
  the same shape fails earlier as `CONTRACT_VALIDATE_ERROR`): broadcast
  `895e09bd62e84618fe997d6168e8fc1de7d24704482bed3ce97d14d185792594` →
  nodeCode **BANDWITH_ERROR**, Code **`account.insufficient_bandwidth`**,
  Revert `Account resource insufficient error.` — the mapped code is
  exactly `tron.CodeAccountInsufficientBandwidth`.
- **R8 — `UpdateSetting` success path.** Key2 deployed a minimal 1-byte
  STOP contract (`6001600c60003960016000f300`):
  `cda8755bd871b3b4ae0278c5efc30b56d8870be1094c1b9bf86cbf674f05e72b`
  (block **71379455**, contract **TL8gvgsYyzVDdupcFEUiYmGHdehLySW5P3**,
  energy 221, energyFee 22,100). `BuildUpdateSetting(percent=42)` →
  sign → broadcast
  `260cc2111aaff22a4b263435e3f8856cd12c38e08c9245ba22a7207d1ae6aa4b`
  (block **71379456**, ok, netFee 0). `GetContractInfo` then reports
  `consume_user_resource_percent` = **42** (was 0) — the success path a
  deployer key is required for.

### R9 — every documented example flow, live-checked (Nile, 2026-09-30)

`cmd/examplecheck` walks the flows the examples teach against a live node:
reads, builds, local signing, simulation, cost pricing, the portable-envelope
round trip and the node's signature-weight verdict. It broadcasts only with
`-broadcast`, so the default run spends nothing and needs no funded key.

```sh
go run ./cmd/examplecheck                         # spend-free, fresh signer — notes, not failures
go run ./cmd/examplecheck -key <hex>              # spend-free, existing account — the variant recorded here
go run ./cmd/examplecheck -key <hex> -broadcast   # full run, spends TRX
```

*(Corrected by the R11 audit: the table below was recorded from the `-key`
variant — an existing owner account. The no-flags fresh-signer run cannot
read a permission set that does not exist, and the node rejects state-
changing builds from an account that does not exist; both are notes, and
before the audit the harness miscounted the first as failures.)*

Observed at Nile block **71,399,777** — 34 steps OK, 4 notes, **0 failed**:

| Flow | Live evidence |
|---|---|
| Account reads | `TLibQrqp…GT1` state, resource state, staking summary and delegation index all decoded; EnergyLimit 8520, free Bandwidth 600 |
| Chain parameters | `getUnfreezeDelayDays` **1** (Nile, not 14), `getMaxDelegateLockPeriod` 144000, `getMultiSignFee` 1 TRX, `getUpdateAccountPermissionFee` 100 TRX |
| Energy price | 100 sun/energy, `EffectiveAt` 2025-08-08 (a governance entry, not "now") |
| TRC-20 handle | `Tether USD (USDT)`, decimals 6, totalSupply 1,000,000,000,000,000,035,993,266,846; `Amount("1.5")` → 1500000 raw units |
| ABI from the node | 20 methods loaded from `getcontractinfo` and used by `Invoke`/`Decode` |
| Simulate (success) | `approve`: energy 22506, decoded ABI result `true` |
| CostPreview | 22506 needed, 8520 staked, buys 13986 @ 100 sun = **1.3986 TRX** |
| TotalCostOf | 1.3985 TRX; for the permission update it reported the **100 TRX** `getUpdateAccountPermissionFee` — the V2 reversal, live (a bandwidth-only total would have said 0) |
| Simulate (revert) | `transfer` from a 0-balance owner: `REVERT opcode executed`; decoding that payload fails with `contract.arg_mismatch` |
| Portable envelope | 215 bytes, kind `native`, signer recovered from the bytes after `Decode`, duplicate signer refused with `tx.already_signed` |
| Remote signer | `SignHash` → `AttachSignature` verified the 65-byte signature against the stated address |
| SignWeight | `PERMISSION_ERROR`, threshold 1, `enough=false` — a signature from a key outside the permission list does not authorize, which is the point of the step |
| Staking builds | stake / unstake / delegate accepted by the node's build RPCs and each signed locally |
| Event decoding | E1's txid decoded to `Transfer [from=TKgHdpAqr7… to=TBkfmcE7pM8… value=1000000000000000000]` |

**Defect caught by this run:** `ExampleClient_Contract` decoded
`Simulate`'s `ConstantResult` unconditionally. A reverting simulation returns
the revert payload, not the method's return value, so the example failed with
`contract.arg_mismatch` instead of reporting the revert. The example now
checks `Estimate.Revert` before decoding; the harness keeps a step that pins
both branches. Compilation could not have found this — only execution could.

**Notes (all four are node-side state validation, not SDK behaviour):**
`withdrawexpireunfreeze`, `cancelallunfreezev2`, `undelegateresource` and
`votewitnessaccount` were rejected at *build* time with `tx.invalid_argument`
because the reference account has nothing pending to withdraw, cancel or
undelegate, and is not voting. These builds are state-validated by the node,
which is why the examples order the reads before the writes.

### R10 — the chain-updating flows, broadcast on Nile (2026-09-30)

`cmd/examplecheck -broadcast` performs the flows for real, with the repo's
throwaway keys (`v1-legacy:integration_test/test.env`, `NILE_TEST_KEY1/2`).
Every state change is paired with the operation that reverses it, and the
harness prints the before/after state so the claim is checkable.

```sh
K1=... K2=...   # from v1-legacy:integration_test/test.env
go run ./cmd/examplecheck -key "$K1" -payee TLibCZ2i2dFp6a9KZeKriSms5peeXSibks \
  -payee-key "$K2" -token TWRvzd6FQcsyp7hwCtttjZGpU1kfvVEtNK -broadcast
```

Three consecutive runs, each exit 0 (**55 steps OK, 6 notes, 0 failed**).
Transaction ids from the clean run:

| Flow | Evidence |
|---|---|
| Rebalance in (TRX transfer) | `d4474957…92f6` — 2.6 TRX to the payee; payee 0.4963 → 3.0963 TRX |
| Approve | `6ca05692…1806` — allowance then read back as 0.1 |
| transferFrom (spender pulls) | `bec02b52…de3c` — the payee's own key signs and pays |
| Transfer (direct) | `22178712…1ec6` — owner 997,387.35 → 997,387.15, payee 0.6 |
| Stake 1 TRX Energy | `79f4fdde…76b5` — energy limit rose to 12226 |
| Unstake | `e3dd027d…1189` — 1 pending unstake, **0 withdrawable** (cooldown 1 day on Nile) |
| CancelUnstake | `e0f4388a…43e0` — pending unstakes back to 0, stake restored |
| Delegate (unlocked) | `6820f3c1…bcca` |
| Undelegate | `32be1d96…0d0d` — immediate for an unlocked delegation |
| Delegate (locked, 20 blocks) | `15ebb92f…4a72` |
| Undelegate while locked | **refused at build** with `tx.invalid_argument` — the lock is enforced |
| Undelegate after the lock expires | `1aac51fd…40d6` — succeeded after the 60 s lock elapsing |
| SetVotes (whole-list replace) | `9acc69f5…b529` |
| ClaimRewards | `dbb69af4…b5ef` (earlier run, when rewards were non-zero) |

**The payee's cost, and why the float has to be 3 TRX.** `CostPreview` for the
payee's `transferFrom` reported *21257 energy needed, 0 staked, buys 21257 =
2.1257 TRX* — an account holding staked Energy pays nothing for those 21257
units; this one bought them all with TRX, which is exactly the balance drop the
run shows. That is the measured basis for the default `-float 3`.

**Rebalancing: repeated runs do not drain either key.** The flow tops the payee
up to the float when it is short and returns anything above float + reserve, so
the only net movement is the fees burnt:

- payee: 0.526 → 0.4963 → 3.0963 → 0.5926 TRX across runs — always inside the band, never accumulating
- owner: 96.2485 → 87.876026 → 78.617126 → 68.257826 TRX — ~9–10.4 TRX per run for 12 broadcasts (≈4 TRX bandwidth + ≈3 TRX bought energy + the payee's ≈2.5 TRX of bought energy)

**Position restored:** `stakes 3 → 3`, `pending unstakes 0 → 0`, only balances
differ, by exactly the burnt fees. The harness asserts both.

**Left deliberately for a later run:** one 1 TRX unstake is pending from the
`-leave-unstaked` run (`1678a578…79fc`), so a run after its 1-day cooldown can
prove `WithdrawUnstaked` positively — nothing matures inside a single run, which
is why that method had no positive evidence before.

### R11 — ledger audit against the current tree (Nile, 2026-09-30)

Parts of this ledger predate the v2.0.0 code fixes, so every checkable
claim was re-verified, read-only, against the tree as it stands:

- **R9 reproduced.** `go run ./cmd/examplecheck -key <K1>` (spend-free,
  existing owner): **34 OK / 4 notes / 0 failed** — the recorded numbers,
  against the post-fix tree. The note set shifted with account state, not
  code: CancelUnstake now builds (the deliberate pending unstake exists)
  and ClaimRewards notes (nothing claimable).
- **The no-flags fresh-signer run is a note-path, not a failure path.** It
  reported 2 FAILs at the audit — `Permissions().Current` answers
  `contract.bad_metadata` by design for an account that does not exist —
  so the harness learned that this is the documented "no account" note
  class. Now 22 OK / 16 notes / 0 failed, exit 0. R9's inline comment
  wrongly implied the fresh-signer invocation produced its table;
  corrected above.
- **`WithdrawUnstaked` still open, with a date**: live `GetAccount` shows
  the pending 1 TRX ENERGY unstake expiring **2026-10-01 08:30 (+08)** — a
  `-broadcast` run after that timestamp closes the row.
- **Permission-update broadcast still refused, against fresh numbers**:
  `getUpdateAccountPermissionFee` re-read at 100 TRX; key1 holds
  59.364826 TRX.
- **Nile TIP-491 negative row re-confirmed**: `getDynamicEnergyThreshold`
  is still 5×10⁹ on Nile — ordinary usage (≈7.5k energy) sits far below
  it, factor 0; never cite Nile for penalty behavior.
- **Stale pointers fixed**: §3 pointed at `PHASE2.md` (removed in
  `302b0f5`); the token package still documented a "uint8 accessor gap"
  that `contract.Result.Byte` closed on 2026-09-29. §4's names re-checked:
  `tx.BandwidthCostOf`, `tx.TotalCostOf`, `DeployTx.Estimate` exist as
  documented.
- **Mainnet TIP-491 re-proven**: the probe replay of E2's txid still
  matches the immutable receipt exactly (energy 130,285 / penalty 100,635 /
  base 29,650 — PASS), which also exposed E2's mis-transcribed penalty
  figure; the row above is amended. The Nile negative row stands: it is
  about citing *Nile* for penalty behaviour, and Mainnet carries the proof.

### R12 — permission-update broadcast, add + verify + reverse (Nile, 2026-09-30)

Funded for this run: the owner account received 200 TRX (259.364626 TRX
before the run). `cmd/examplecheck -broadcast -permission-update
-leave-unstaked` — **56 steps OK, 9 notes, 0 failed**, 9 transactions.

The permission cycle (the last §3 broadcast row, now closed):

| Step | Evidence |
|---|---|
| Price gate | fee 100 TRX ×2 (add + reverse), account held 250.96 TRX after the earlier flows |
| SignWeight (node verdict before sending) | `signature weight 1/1 (ENOUGH_PERMISSION) — authorized` |
| Add active `examplecheck` (Transfer + TriggerSmartContract bitmap, payee key, weight 1) | `2364ff3da3bd58725a7b0957712684019b83981a0c9a6ae99df94e511a22d9ff` |
| On-chain verify | 2 actives, last `examplecheck` threshold 1; **owner permission asserted unchanged** (same threshold, same key list) |
| Reverse (submit the saved original set) | `0816b512e177d2b04a1573dcaa1d74a4dfe614a3c734fd93b9ac3e844c58ddea` |
| Verify reversal | actives back to 1; permission fees burnt **200.829 TRX** |

The lockout risk §3 refused to take never materialized because the flow
never modifies the owner permission: it appends one active permission to
the set read from the chain, signs with the owner key, asks the node for
the sign-weight verdict first, and the reverse submits the saved original
set back.

Rest of the run (re-proven post-R11): rebalance-in `8ec07e56…1ffa`,
approve `8adf8c60…2890`, transferFrom `c384f85c…0368`, transfer
`576f1d01…cb15`, stake `79f6c8cb…08ff`, unstake `65620750…83d6`, SetVotes
`181ce8bc…592e`. Owner 259.364626 → 50.135726 TRX (209.2289 spent:
200 permission fees + ≈9.2 run fees). Stakes 3 → 3; pending unstakes 1 → 2
(`-leave-unstaked`: the original 1 TRX matures 2026-10-01 08:30 (+08), the
fresh one ≈12:05 — the next run proves `WithdrawUnstaked` on both).

New node-side observation (notes, not failures): with a pending unstake on
the ENERGY resource the node refuses `delegate resource` /
`undelegate resource` builds with `tx.invalid_argument` — a delegation
must be backed by staked balance not in unstaking — so the delegation
proofs were skipped this run (R10 carries them); `ClaimRewards` with zero
accrued is likewise refused at build.

### R13 — CostPreview bandwidth line, live (Nile, 2026-09-30)

The last v2.1 code TODO (P2.6) landed early: `CostPreview` now carries a
`Bandwidth *BandwidthCost` half — the same charging model `BandwidthCostOf`
applies post-signing: staked bandwidth first, then the free quota, shortfall
at `SunPerByte` — measured on a one-signature estimate of the broadcast
bytes, plus `TotalFloor = TronToBurn + Burn`. A predicted burn the balance
cannot cover fails with `account.insufficient_bandwidth` (the fiction rule
`BandwidthCostOf` already followed).

Keyed spend-free run (`-key` K1): **34 OK / 4 notes / 0 failed** (totals
unchanged from R9/R11). New lines, live:

```text
bandwidth preview: need 345 (staked 0 + free 36); to burn 309 @ 1000 sun/byte = 0.309 sun
total floor 2.5038 TRX (energy + bandwidth; bandwidth priced on a single-signature estimate; …)
```

- **Estimate exactness**: the harness signs the same transaction with one
  key and asserts `preview.Bandwidth.BytesNeeded ==` the signed
  `BandwidthSize` measurement — **345 == 345, PASS**. The 345-byte shape is
  E3/E4's live-measured burn model (281 + 64 result overhead).
- **Fresh-signer run: 21 OK / 17 notes / 0 failed** — `CostPreview` now
  correctly notes `account.insufficient_bandwidth` for an account with no
  bandwidth and no balance, where it previously reported an energy-only
  number with a caveat string. The step counts moved with the honesty, not
  with a regression.
- Gates: full suite + `-race` + `golangci-lint` 0 issues; `tx` coverage
  84.7% (floor 80%).

### R14 — eventtool live capture against Envoy (mainnet, 2026-09-30)

The v1 4-byte event remnants were replaced by the 32-byte `cmd/eventtool`
pipeline (spec `.superpowers/specs/2026-09-30-eventtool-design.md`, local). Node
access went through the local Envoy gRPC proxy `~/envoy` (listener
`grpc://127.0.0.1:50051` → 19 mainnet full nodes, no rate limits):

```bash
docker compose -f ~/envoy/compose.yaml up -d
go run ./cmd/eventtool contracts --limit 100 --out internal/eventdata/top_contracts.json
go run ./cmd/eventtool capture --node grpc://127.0.0.1:50051 \
  --in internal/eventdata/top_contracts.json \
  --out internal/eventdata/events_registry.json
go run ./cmd/eventtool generate --in internal/eventdata/events_registry.json \
  --out event/builtin_gen.go
```

Measured, live:

- `contracts`: **100 contracts** ranked by `trxCount` (TronScan's default list;
  two 50-row pages), snapshot written to `internal/eventdata/top_contracts.json`
  (`fetched_at` 2026-09-30T07:16:56Z). Rank 1 USDT
  `TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t`; rank 100
  `TTnSmkMNBoxeBMbEA84x29wmR98WbQYdbQ`.
- `capture`: **100 contracts / 77 with events / 155 new events / 23 skipped**
  in ~40 s sequential (one `GetContract` per address). 23 addresses carry an
  ABI with no event entries (proxy/treasury-shaped contracts).
- corpus **747 → 902 entries** (no duplicate signatures); the curated baseline
  survived first-wins.
- `generate`: **902 definitions** → `event/builtin_gen.go`, keys as 32-byte
  array literals; re-running is byte-identical (idempotent).
- The whole corpus verifies against `event.SignatureKey`
  (`TestTrackedCorpusVerifies`), and the regenerated table is consistent and
  distinct-keyed (`TestBuiltinTableCountAndKeys`).

### R15 — event corpus coverage census + official-ecosystem map (mainnet, 2026-09-30)

Two questions R14 left open: *what is the corpus actually made of*, and *how
much chain activity does it fail to decode*. Both answered by reading the chain
through Envoy (`grpc://127.0.0.1:50051`, 19 mainnet full nodes) at tip
**86,698,614**. The probe was throwaway and is **not tracked** — the recipe is
in §4 and a re-check means re-deriving it.

**A. Corpus composition (902 entries, attributed by re-reading every ABI).**
For each of the 100 snapshotted addresses: `GetContract`, derive
`event.SignatureKey(name, types)` per named non-anonymous event, mark which
corpus entries it explains (first claim wins; duplicates within one ABI deduped
by sighash).

- **204** of 902 entries are declared by a top-100 contract — that is the whole
  reproducible contribution of the current pipeline (R14's 155 new + 49 already
  present).
- **698 of 902** are declared by *no* top-100 contract: they are inherited from
  the v1 curated baseline, whose provenance no workflow in this tree
  reproduces. **This is the corpus's real reproducibility hole** — an order of
  magnitude larger than the emitter gap the harvest design targets.
- Re-read found **0 declared-but-missing** across the top-100, confirming R14's
  capture completeness. *Discrepancy to note:* this re-read has **69** addresses
  yielding usable events where R14 recorded **77 with events**; ABI drift or a
  transient `GetContract` failure are the candidates and it is unexplained —
  treat R14's 77 as the capture-time figure.

**B. Official-ecosystem coverage, per contract, declared events only.** Names
from `GetContractInfo`; "source" = top-100 pipeline vs v1 baseline.

| Contract | On-chain name | Declared | In corpus | Source |
|---|---|---|---|---|
| USDT | `TetherToken` | 12 | **12/12** | top-100 |
| USDD | `USDD` | 4 | **4/4** | 2 top-100 + 2 v1 (`Deposit`, `Withdrawal`) |
| USDD PSM | `UsddPsm` | 5 | **5/5** — `BuyGem`, `SellGem`, `File`, `Rely`, `Deny` | v1 baseline |
| JST | `JST` | 6 | **6/6** | top-100 |
| SUN | `SunToken` | 2 | **2/2** | top-100 |
| WTRX | `WTRX` | 4 | **4/4** | 2 top-100 + 2 v1 |
| WINK / BTT | `WINK`, `BTT` | 4 / 2 | **4/4, 2/2** | top-100 |
| SR price oracle | `PriceOracle` | 8 | **8/8** — `PricePosted`, `CappedPricePosted`, `NewPendingAnchor`, `OracleFailure`, `Failure`, `SetPaused` | v1 baseline |
| FiatTokenProxy | `FiatTokenProxy` | 19 unique (104 raw entries) | **19/19** | top-100 |
| SmartExchangeRouter / Bridgers | — | 6 / 5 | **6/6, 5/5** | top-100 |
| sTRX | `STRXProxy` (blueTag *JustLend DAO*) | **0** | **0** | undecodable — see §3 |
| SunSwap v2 router A / B | `UniswapV2Router02` ×2 | **0** / **0** | — | proxy ABIs empty; pool-level events survive only via the v1 baseline |

Lending- and staking-shaped v1 entries are also present and match the official
ecosystem: `Borrow(address,uint256,uint256,uint256,uint256)`,
`LiquidateBorrow(address,address,uint256,address,uint256)`,
`RepayBorrow(address,address,uint256,uint256,uint256,uint256)`,
`Redeem(address,uint256)`, `Mint(uint256)`, `Supply(uint256,uint256)`,
`SetPaused(bool)`, plus **55** staking/governance events (`Staked`, `Unstaked`,
`DelegatorRestaked`, `Restaked`, `RewardClaimed`, `AuditRewardPaid`, …).

**C. Coverage census — 500 blocks (~25 min chain time), tip 86,698,614.**
Walk `GetTransactionInfoByBlockNum` downward; tally per emitter normalized to
21 bytes (`0x41` + the 20-byte `TransactionInfo_Log.address` — comparing raw
makes every log look indirect); weight each signature by log count.

- **59,489 logs · 186 distinct emitters · 128 distinct signatures · 74 unknown.**
- **Log-weighted: 97.0% of emitted logs decode today (57,724); 3.0% do not
  (1,765).** The "58% of signatures are unknown" framing is a long-tail
  artifact, not a coverage statement — always quote the log-weighted figure.
- USDT alone is **94%** of all log volume in the window and is 12/12 covered.

**D. Harvest ceiling, measured (the number the emitter-harvest design lacked).**
For the top 60 emitters (1,686 undecodable logs), asking each emitter's own
on-chain ABI which of the sighashes *it actually emitted* that ABI explains:

| Mechanism | Recovers | Share of 1,686 |
|---|---|---|
| Capture each emitter's own ABI | **118** | 7.0% |
| + one-hop `proxy_implementation` (TronScan) | **+967** | → **1,085 = 64.4%** |
| …of which **945 of the 967 is ONE contract** | `TFFAMQLZy…jF3U` `UpgradableProxy` (blueTag *GasFree*) → impl `TUGNUUoS…VJEJw` `GasFreeController`, 15 events → **945/945** | |

- Only **6** emitters would contribute via their own ABI; **15** have empty
  ABIs. Of the empty-ABI ones, **10 are `is_proxy: false`** factory clones named
  `CreatedByContract` with **no implementation pointer at all** — a recursive
  proxy follow cannot touch them.
- Where a proxy *is* recorded, the hop is often just as empty: `MarketProxy`'s
  implementation declares **0** events (288 unknown logs, 0 recovered); sTRX's
  implementation `TUAV6ZSCX…bF4HtG` declares **0** (23 logs, 0 recovered). So
  D3's yield is real but concentrated: its ceiling here is one hand-addable
  address, not a subsystem.
- Net: the full harvest would move **~0.3% of all emitted logs** (1,085 of
  59,489), ~97% of which is that single GasFree implementation.

**E. Local (no network) checks.** All 7 seed ABIs under
`internal/eventdata/abi/` re-parsed: **16 event definitions, 16 already in the
corpus, 0 missing** — D5 of the harvest design ("reference-only, duplicates
the corpus") is confirmed, and they are read by no workflow.

**Methodology notes (so a re-check does not repeat them).** Three defects in
the throwaway probe produced wrong numbers before being fixed, each of them
plausible in real tooling: (1) counting ABI entries without deduping by sighash
reported "85 declared-but-missing" where the true figure is 0 —
`FiatTokenProxy` declares 104 entries for 19 unique signatures; (2) first-claim
attribution made shared signatures (`Transfer`, `Approval`) look missing on
every contract after the first; (3) the worker pool both wrote a shared map
outside the mutex (concurrent map write) and overwrote instead of accumulating
per-emitter counts, understating log-weighted coverage by orders of magnitude.

**Re-check:** no tracked artifact — see §4. The deterministic parts (A, B, E)
need only `GetContract`/`GetContractInfo` for 100 addresses plus the tracked
corpus; the census (C, D) is wall-clock dependent and differs per window.

### R16 — WithdrawUnstaked positive proof, broadcast on Nile (2026-10-02)

The last open §3 row, closed. Both 1 TRX unstakes pending since R10/R12
(`1678a578…79fc` + R12's `65620750…83d6`) had matured — the spend-free
run showed `withdrawable 2 TRX` — so the standard broadcast run withdrew
them for real:

```sh
K1=... K2=...   # from v1-legacy:integration_test/test.env
go run ./cmd/examplecheck -key "$K1" -payee TLibCZ2i2dFp6a9KZeKriSms5peeXSibks \
  -payee-key "$K2" -token TWRvzd6FQcsyp7hwCtttjZGpU1kfvVEtNK -broadcast
```

- withdraw-unstaked `031406ceedd1d36ca2295f382debf139b29dad3384e6abd26792061a6fcf1047`
  (block **71463200**): receipt `withdraw_expire_amount` **2000000** (= 2 TRX),
  `net_usage` 253. The full-run log prints `2 TRX matured` before the step.
- Rest of the run re-proven: 13 transactions (rebalance-in, approve,
  transferFrom, transfer, stake, unstake, cancel-unstake, delegate/undelegate
  unlocked + locked with the build-time refusal while locked, undelegate after
  expiry, set-votes), owner 50.135726 → 43.363026 TRX (6.7727 spent: bought
  energy + bandwidth), stakes 3 → 3, pending unstakes 2 → 0.
- Harness verdict **57 OK / 4 notes / 1 FAILED** — the FAIL is the
  position-restored assertion (`pending unstakes changed: 2 → 0`), an
  artifact of withdrawing two pre-existing matured unstakes at the start of
  the run, not a chain or SDK failure. Every broadcast landed; the run's own
  fresh 1 TRX unstake was cancelled back to staked, ending at 0 pending.
- Re-check: receipt fetch on `$NILE` for the txid above.

## 3. Negative records (what is NOT proven)

- **Nile TIP-491**: factor 0 on Nile USDT (usage 7,506 vs 5e9 threshold);
  both independent reads agree at zero — inconclusive by network
  parameters, not by funds. Never cite Nile for penalty behavior.
- ~~**Staked-energy cost runs**~~ — resolved by R6 above (Nile, 2026-09-29).
- ~~**`insufficient_bandwidth` live**~~ — resolved by R7 above.
- ~~**UpdateSetting success path**~~ — resolved by R8 above.
- **Broadcast-verified (R10, R12, R16):** TRX transfer/rebalance, TRC-20 approve +
  transferFrom + transfer, stake, unstake, withdraw-unstaked (R16), cancel-unstake, delegate
  (unlocked and locked), undelegate, vote replace, reward claim, and the
  permission update (add active + verify + reverse, SignWeight-checked).
- ~~**`WithdrawUnstaked` has no positive on-chain evidence yet.**~~ — resolved by R16
  above (Nile, 2026-10-02): withdraw-unstaked `031406ce…cf1047`, receipt
  `withdraw_expire_amount` 2000000 (= 2 TRX). No run spans the cooldown
  (1 day on Nile, 14 on Mainnet); the proof came from unstakes left pending
  by earlier runs.
- ~~**The permission-update broadcast is not verified.**~~ — resolved by R12
  above (funded 200 TRX; add + on-chain verify + reverse, 2×100 TRX fees
  burnt; owner permission untouched throughout).
- **Arg constructors** for `address[]`/`bytes`/`bytesN`/`int256` and
  `Result.Byte` for `uint8`: **shipped 2026-09-29** (the historical PHASE2.md
  was removed with the v1-era docs in `302b0f5`; the constructors live in
  `contract/arg.go` and the accessor in `contract/result.go` — presence
  re-verified at the R11 audit).
- **sTRX events are undecodable from any source available to us** (R15 §B/§D).
  `STRXProxy`'s on-chain ABI declares 0 events and so does its TronScan-recorded
  implementation `TUAV6ZSCX…bF4HtG`, while the proxy emits 7–12 distinct
  signatures. This is the flagship "official ecosystem gap" and it is *not* a
  harvesting defect: there is no ABI to capture. Fixing it needs an external ABI
  source (verified source repo, verified-deployment registry), which is a
  different decision from anything in the emitter-harvest design.
- **698 of the 902 corpus entries have no reproducible provenance** (R15 §A).
  They came from the v1 curated baseline; no workflow in this tree regenerates
  them, and their source ABI set was not preserved. Do not cite the corpus as
  reproducible until that set is either recovered or rebuilt.
- **Coverage was never stated as a number before R15.** No prior record claims
  what fraction of chain log volume the corpus decodes; the only coverage
  claims were R14's per-contract capture completeness (still true: 0
  declared-but-missing). The measured figure is 97.0% log-weighted over a
  500-block mainnet window.

## 4. Reviewer toolbox (no setup)

- Event-corpus composition (A), official-ecosystem coverage (B) and the
  log-weighted decode rate (C) from R15 — read-only, no key: `Dial` the node,
  `rpc.ChainTip`, then per address `rpc.GetContract` (ABI) and
  `rpc.GetContractInfo` (on-chain name); membership is
  `event.SignatureKey(name, types)` against
  `internal/eventdata/events_registry.json`. The census needs a
  `GetTransactionInfoByBlockNum` walk with `0x41`-prefixed log addresses. No
  probe is tracked for this — it was a one-off.
- **There is no programmatic source for the official-ecosystem contract list.**
  TronScan's `/api/contract?contract=<addr>` returns rich classification
  (`is_proxy`, `proxy_implementation`, `blueTag`, `publicTag`, `methodMap`) but
  **no `abi`**, and `/api/contracts?blueTag=<tag>` **silently ignores the
  filter**, returning the same unfiltered `trxCount` list every time. Recorded
  here because a reviewer will try it: `methodMap` is function selectors only
  and is useless for event decoding.
- Energy + bandwidth for any historical txid: the probe replay command
  at the top (exact assertions, void-on-revert).
- Factor + penalty cross-check for any contract: the R1 command with a
  `balanceOf` calldata for that contract.
- Deploy estimate for any bytecode: the R2 curl, or
  `cli.Deploy` → `DeployTx.Estimate` (no key until broadcast).
- Bandwidth prediction for a signed tx: `tx.BandwidthCostOf`
  (pre-broadcast, exact); all-in total: `tx.TotalCostOf`.

## Appendix — addresses (no private keys anywhere in this repo)

- Nile test key (funded, ~40 TRX remainder):
  `TFajiYgytkFiBpustBNQSFEDsiHBoAFrBo`
- Creation recipient: `TCvU2ENS6sksbhcz2x7jX87yrrCWZLt7Rf`
- Deployed probe contract (Nile): `TGHsNCnqw9fnwB86NcKU36xdoT9BPQUPqV`
- Self-issued test token (Nile): `TWRvzd6FQcsyp7hwCtttjZGpU1kfvVEtNK`
- Official USDT (Nile): `TXLAQ63Xg1NAzckPwKHvzw7CSEmLMEqcdj`
- Official USDT (Mainnet): `TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t`
