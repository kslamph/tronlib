# tronlib v2 — Review records & deferred work

This document holds the **process records**: what the reviews found, how each
finding was dispositioned, and what work is deliberately deferred. The design
itself lives in [architecture.md](architecture.md); on-chain evidence in
[verification.md](verification.md); maintainer operations in
[runbook.md](runbook.md).

Sections 1 and 2 were moved out of architecture.md (2026-09-30) so that
document describes only the design; the `(architecture §16)` / `(§17.1)`
references preserved here explain where they came from.

## 1. Original design review — disposition (from architecture §16)

Every finding from the original design review, with its verification status.
Load-bearing claims were re-checked against source or the live protocol
before being accepted; one review claim was found to be wrong.

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
| **G6** | TRC-10 transfer homeless; C3 vs §3 disagree | **Accepted** | `AssetTx` + `TransferTRC10` in scope; §13 separates transfer from issuance |
| **P2.1** | `TRX(10e15)` is a compile error, not a panic | **Accepted — verified by execution** | §5.2 corrected; real panic row added |
| **P2.2** | Decimals cap 18 rejects valid uint8 tokens | **Accepted** | §5.4 accepts 0–255; `bad_metadata` reserved for wrong encoding width |
| **P2.3** | `Add`/`Sub` unchecked vs `Mul` checked | **Accepted** | §5.3 all three checked |
| **P2.4** | `Tx` claimed sealed but is not | **Accepted** | §6.1 adds `txInternal()` marker |
| **P2.5** | Activation cost unmodelled | **Accepted as a documented limitation** | §7.3 note; not modelled in v2.0 |
| **P2.6** | `CostPreview` ignores bandwidth | **Accepted as a documented limitation** | §7.3 note; bandwidth line deferred to v2.1 |
| **P2.7** | `Network` type undefined | **Accepted** | §10 defines it |

**Net effect on the design's spine:** none. The package DAG, the amount
model, the error taxonomy, the four-kind F1 fix and docgen-before-API all
survived review unchanged. Every accepted finding was a last-mile gap — a
mechanism promised in one section and not defined in another, or a protocol
fact stated from recollection. That is the same failure mode as P11,
appearing in a document written to eliminate P11, which is worth noting as
the real lesson here: **a spec that polices unverified claims still has to
make them, and every one needs a source.**

## 2. Quickstart review (2026-09-30) — findings (from architecture §17.1)

The curated layer covered transfers, deployment, contract calls and cost
preview, and `rpc` carried a 1:1 wrapper for essentially everything else.
What was missing was not node support but **workflows**: the compositions of
rpc + tx + sign + broadcast that a user actually performs. The inventory:

| Workflow | Before this revision |
|---|---|
| TRX / TRC-10 / TRC-20 transfers, calls, deploy | Curated path existed |
| Stake 2.0: stake, unstake, withdraw matured, cancel unstake | Raw `rpc` calls only; no builders, no facade |
| Delegate / undelegate, delegation reads | Raw `rpc` calls only |
| Multi-signature | `Sign(signers...)` and `WithPermissionID` existed; configuring permissions, checking weight and **moving a partial transaction between machines** did not |
| Voting and reward claiming | `Witnesses` was curated; everything else was raw |
| Account state (stake, unstake, votes, delegation totals) | Protobuf reads only, except `TronBalance` |

Two structural consequences made this more than a naming gap:

1. `Client.Broadcast` accepts the sealed `tx.Tx` interface, so a transaction
   returned by a raw `rpc` build call **cannot** enter the curated signing
   pipeline. A user following the quickstart could not stake without
   dropping to protobufs.
2. Multi-signature was only demonstrated within one process. There was no
   safe way to hand a partially signed transaction to another signer.

The revision these findings produced is architecture §17.2–§17.5 (the
account-scoped facade, portable transactions, naming). The 2026-09-29
seven-lane code review's fixes and the R11 ledger audit are recorded in
[verification.md](verification.md).

## 3. Deferred work (the consolidated TODO list)

Everything deliberately not built, in one place. Sources: the disposition
table above (P2.5/P2.6), architecture §7.3's notes, §13 non-goals, §17.6
deliberately deferred.

| Item | Status | Where documented |
|---|---|---|
| `CostPreview` bandwidth line | **Done (2026-09-30, R13 — landed ahead of v2.1)** — `Bandwidth` half on `BandwidthCostOf`'s charging model + `TotalFloor`; the one-signature estimate is live-verified exact against the signed measurement | architecture §7.3, P2.6 |
| Recipient-activation cost in `CostPreview` | **Open** — `TotalCostOf` models account creation (E5 proved it exact); `CostPreview` documents the delta instead | architecture §7.3, P2.5 |
| Machine manifest (`schema.json`) | **Deferred** until a tool-call layer reaches the roadmap; `codes_gen.go` carries the recoverable value today | architecture §12 |
| Shielded/Sapling operations | **Deferred (v2.1)**; reachable through `rpc` | architecture §13, §17.6 |
| TRC-10 asset issuance | **Excluded**; transfer only (`TransferTRC10`) | architecture §13 |
| CLI | **Excluded** | architecture §13 |
| Stake 1.0 legacy freeze/unfreeze | **Excluded**; new staking is Stake 2.0 only | architecture §17.6 |
| `TRON_POWER` resource / new resource model | **Excluded**; `Resource` rejects it rather than silently mapping | architecture §17.6 |
| Governance proposals, witness administration | **Excluded**; raw via `rpc` | architecture §17.6 |
| `Client`-level chain-parameters read beyond `ChainParamsOf`'s priced subset | **Excluded** | architecture §17.6 |
| `WithdrawUnstaked` positive on-chain proof | **Verification follow-up** — two 1 TRX unstakes pending, maturing 2026-10-01 08:30 / ≈12:05 (+08); the next `-broadcast` run closes it | verification.md §3, R12 |

Residual *design* risks (accepted trade-offs, not work items) remain in
architecture §15 — they document why the design is shaped as it is.
