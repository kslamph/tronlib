// Package tx is the v2 transaction pipeline: build → optionally simulate →
// sign → broadcast, expressed as types so each stage's output is the next
// stage's input and illegal transitions do not compile (architecture §6).
//
// # The four kinds
//
// NativeTx (TRX transfer and other non-contract operations), ContractTx
// (TriggerSmartContract — contract calls and all TRC-20 operations), DeployTx
// (CreateSmartContract) and AssetTx (TransferAssetContract, TRC-10 transfers)
// each wrap the pb transaction returned by the node's server-side build RPC
// and carry only the options meaningful for their kind. The builders decide
// the Kind statically; no runtime inspection is involved.
//
// The Tx interface is sealed with the unexported txInternal method: a
// hand-rolled Tx that bypasses the builders (and with them the fee-limit,
// expiration and permission-id defaults) is a compile error. This is pinned
// by a negative-compile fixture in v2/internal/compilecheck.
//
// # The F1 fix
//
// Simulate and EstimateEnergy exist ONLY on *ContractTx, so
// nativeTx.Simulate(ctx) and deployTx.Simulate(ctx) are compile errors — the
// static kind replaces the runtime dispatch v1 could forget.
//
// # Copy-on-write
//
// Every With* option and every Sign call returns a copy and leaves the
// receiver untouched, so multi-signature flows compose as
// tx = tx.Sign(a).Sign(b) and a partially-signed transaction can never be
// shared by accident. Options are part of raw_data, so every With* returns an
// error and rejects an already-signed transaction with tx.already_signed:
// mutating a signed transaction would detach its signatures from the bytes
// they authorize. Set options first, then sign.
//
// # Portable transactions (offline multi-signing)
//
// tx.Encode writes a versioned, self-describing envelope and tx.Decode
// rebuilds the correct concrete kind, carrying partial signatures; Sign and
// AttachSignature add more. Decode cross-checks the declared kind against the
// wrapped contract type, so an importer's type assertion is always sound, and
// refuses duplicate or unrecoverable signatures. SignHash plus
// AttachSignature let a remote or hardware signer produce the signature
// without the private key entering this process.
//
// # Defaults (architecture §6.4, stated so they are testable)
//
//   - fee_limit: 150_000_000 SUN (150 TRX) — v1's DefaultBroadcastOptions
//     value, applied by every builder at build time unless a later
//     ContractTx.WithFeeLimit overrides it (a node response with fee_limit 0
//     cannot purchase energy)
//   - expiration: head + 60 s — set server-side by the build RPC; WithExpiration
//     mutates raw_data.expiration post-build for long multi-signer circulation
//   - permission_id: 0 (owner); multi-sig under active permissions needs 2–9
//
// # Cost prediction in two phases
//
// Energy can be previewed before signing (simulation ignores signatures),
// but bandwidth is measured on the exact broadcast bytes and therefore
// only after signing. The API mirrors that order so the convenient call
// is also the correct one:
//
//	build → PreviewCost (energy, pre-sign OK, ContractTx only)
//	      → Sign
//	      → TotalCostOf (all-in total, every kind)
//	      → Broadcast
//
// TotalCostOf runs its bandwidth half first, so calling it pre-sign fails
// fast with tx.invalid_argument — naming the unsigned size and the
// per-signature delta — before any simulation RPC is spent. PreviewCost
// stays valid after signing, but post-sign callers should prefer the
// single all-in call.
//
// # The double-spend fix (architecture §6.4/§6.5)
//
// Broadcast performs one reconciliation poll on an ambiguous timeout. A
// timeout after the broadcast has landed returns chain.unconfirmed with the
// txid populated and Next = ActionWait: re-broadcasting the identical signed
// payload cannot double-spend (a TRON txid is a pure function of raw_data and
// the node deduplicates), but REBUILDING gets a new TAPOS reference and a new
// txid — that is what spends twice. Never rebuild-and-resign until the
// original txid's receipt is confirmed absent or failed.
//
// # Deviations from architecture §7.2/§7.3 (adjudicated)
//
//   - EnergyEstimate carries only Energy: the EstimateEnergy RPC
//     (api.EstimateEnergyMessage) exposes only the penalty-inclusive total, so
//     the architecture doc's Base/Penalty split would be fabricated (architecture §7.1 says the
//     node already applies the penalty). Use ContractTx.Simulate (Estimate
//     .Energy/.Penalty) when the split matters.
//   - EstimateEnergy is the node's CONSERVATIVE fee-limit calculator, not the
//     accurate cost: live-verified §7.5, it returns 1.5× the actual execution
//     energy. CostPreview uses Simulate.Energy (TriggerConstantContract.Energy
//     Used, the accurate dry-run cost, exact on the live run) as EnergyNeeded,
//     and the energy→SUN conversion goes through EnergyPrice.CostOf — the pure
//     burn calculator driven by the network's SunPerEnergy parameter, which is
//     independent of any specific contract or transaction.
//   - Estimate.Net is always 0: TriggerConstantContract exposes no bandwidth
//     figure. Receipt.Cost reports actual bandwidth after broadcast.
package tx
