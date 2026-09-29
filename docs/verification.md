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
- Claim: replay energy=130,285 penalty=49,635 base=29,650 == receipt
  exactly. Extends E1 to TIP-491-penalized execution (§7.5 item 2).
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

## 3. Negative records (what is NOT proven)

- **Nile TIP-491**: factor 0 on Nile USDT (usage 7,506 vs 5e9 threshold);
  both independent reads agree at zero — inconclusive by network
  parameters, not by funds. Never cite Nile for penalty behavior.
- ~~**Staked-energy cost runs**~~ — resolved by R6 above (Nile, 2026-09-29).
- ~~**`insufficient_bandwidth` live**~~ — resolved by R7 above.
- ~~**UpdateSetting success path**~~ — resolved by R8 above.
- **Arg constructors** for `address[]`/`bytes`/`bytesN`/`int256` and
  `Result.Byte` for `uint8`: **shipped 2026-09-29** (see `PHASE2.md`
  Phase 2.1).

## 4. Reviewer toolbox (no setup)

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
