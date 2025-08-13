## TronLib integration testing: strategy, setup, and coverage

This guide documents how we test high-level public functions against real TRON networks, how the Nile testnet is provisioned for write tests, and what gaps remain to reach full coverage.

### Goals
- Validate all high-level, public APIs against live networks
- Keep mainnet tests strictly read-only; use Nile for state-changing tests
- Produce stable, reproducible tests with clear preconditions and logs

## Networks policy
- **Mainnet (read-only)**: Query-only tests that do not mutate chain state. Default endpoint: `grpc://127.0.0.1:50051` (configurable).
- **Nile (write tests)**: Transactions that spend TRX, freeze/unfreeze resources, transfer TRC20/TRC10, vote, deploy contracts, etc. Default endpoint: `grpc://grpc.nile.trongrid.io:50051`.

Environment variables (see `integration_test/test.env`):
- `MAINNET_NODE_URL` (default `grpc://127.0.0.1:50051`)
- `NILE_NODE_URL` (default `grpc://grpc.nile.trongrid.io:50051`)
- `NILE_TEST_KEY1`, `NILE_TEST_KEY2` (funded test accounts for Nile write tests)
- Contract addresses populated by the setup tool:(already deployed)
  - `TRC20_CONTRACT_ADDRESS`
  - `TESTCOMPREHENSIVETYPES_CONTRACT_ADDRESS`
  - `MINIMALCONTRACT_CONTRACT_ADDRESS` 

## Nile setup workflow (already deployed)
We ship a setup program that deploys test contracts and updates env files.
- Tool: `cmd/setup_nile_testnet` (see `README.md` in that folder)
- Deploys:
  - MinimalContract (no constructor args)
  - TRC20 (name, symbol, decimals, initial supply)
  - TestComprehensiveTypes (mixed types + events)
- Writes discovered addresses back into `cmd/setup_nile_testnet/test.env` (and intended to populate `integration_test/test.env`).

Recommended usage:
1) Dry run first to validate configuration
2) Fund `NILE_TEST_KEY1` with sufficient TRX (≥ 3000 TRX recommended)
3) Run live deployment; confirm addresses are present in env

## Test layout and conventions
- Read-only tests live under `integration_test/read_tests/` and target mainnet-like nodes
- Write tests live under `integration_test/write_tests/` and target Nile
- Conventions:
  - Always use protobuf getters in assertions (no direct struct field access)
  - Use `context.WithTimeout` (typically 30–60s)
  - Add descriptive `t.Logf` for human verification
  - Avoid brittle equality on dynamic fields; prefer non-negative and structure checks
  - For transactions: sign locally and wait for receipt; decode logs when applicable

## How to run

Mainnet read-only tests:
```bash
MAINNET_NODE_URL=grpc://127.0.0.1:50051 \
go test -v ./integration_test/read_tests/...
```

Nile write tests (state-changing):
```bash
source ./cmd/setup_nile_testnet/test.env
go test -v ./integration_test/write_tests/...
```

Tip: to avoid accidental Nile spends in CI, gate write tests behind an env flag and/or use `-run` selectors. Proposed:
```bash
RUN_NILE_WRITE_TESTS=true go test -v ./integration_test/write_tests/...
```

## Current coverage snapshot (high-level packages)

Legend: [M] Covered on mainnet (read), [N] Covered on Nile (write), [G] Gap

- `pkg/account.AccountManager`
  - GetAccount [M]
  - GetAccountNet [M]
  - GetAccountResource [M]
  - GetBalance [M (via GetAccount)]
  - TransferTRX [N] (covered in `write_tests/onchain_status_test.go`)

- `pkg/network.NetworkManager`
  - GetNowBlock [M]
  - GetBlockByNumber [M]
  - GetBlockById [M]
  - GetBlocksByLimit (Start/End) [M]
  - GetLatestBlocks (by count) [M]
  - GetTransactionInfoById [M]
  - GetNodeInfo [M]
  - GetChainParameters [M]
  - ListNodes [M]

- `pkg/smartcontract.Contract`
  - NewContract (ABI: auto-fetch, string, object) [M]
  - TriggerConstantContract [M]
  - TriggerSmartContract [N]
  - Simulate [N]
  - Encode / DecodeResult [M]
  - DecodeEventLog / DecodeEventSignature [M]

- `pkg/smartcontract.SmartContractManager`
  - DeployContract [G] (setup tool uses it; add Nile test that deploys MinimalContract)
  - EstimateEnergy [G]
  - GetContract [M] (via Contract.NewContract path but add explicit)
  - GetContractInfo [G]
  - UpdateSetting [G]
  - UpdateEnergyLimit [G]
  - ClearContractABI [G]

- `pkg/trc20.TRC20Manager`
  - Name / Symbol / Decimals [M]
  - BalanceOf [M]
  - Allowance [M]
  - Transfer [N]
  - Approve [N]

- `pkg/resources.ResourcesManager`
  - FreezeBalanceV2 [N]
  - UnfreezeBalanceV2 [N]
  - DelegateResource [N]
  - UnDelegateResource [N]
  - CancelAllUnfreezeV2 [N]
  - WithdrawExpireUnfreeze [N]
  - GetDelegatedResourceV2 [N]
  - GetDelegatedResourceAccountIndexV2 [N]
  - GetCanDelegatedMaxSize [N]
  - GetAvailableUnfreezeCount [N]
  - GetCanWithdrawUnfreezeAmount [N]

- `pkg/trc10.TRC10Manager`
  - CreateAssetIssue2 [G]
  - UpdateAsset2 [G]
  - TransferAsset2 [G]
  - ParticipateAssetIssue2 [G]
  - UnfreezeAsset2 [G]
  - GetAssetIssueByAccount [G]
  - GetAssetIssueByName/List... [G]
  - GetPaginatedAssetIssueList [G]

- `pkg/voting.Manager`
  - VoteWitnessAccount2 [N]
  - WithdrawBalance2 [G]
  - CreateWitness2 [G]
  - UpdateWitness2 [G]
  - ListWitnesses [M] (covered via network; add explicit voting read test)
  - GetRewardInfo [N]
  - GetBrokerageInfo [N]
  - UpdateBrokerage [G]

- `pkg/eventdecoder`
  - RegisterABIJSON / RegisterABIObject / RegisterABIEntries [M unit]
  - DecodeEventSignature / DecodeLog / DecodeLogs [M unit; used in mainnet event test and Nile TRC20 write test]

- `pkg/client`
  - SignAndBroadcast [N]
  - Simulate [G]
  - DefaultBroadcastOptions sanity [G]

## Actionable test additions (proposed files)

Nile (write, state-changing):
- `integration_test/write_tests/resources_test.go`
  - Implemented: FreezeBalanceV2/UnfreezeBalanceV2 round-trip (small values)
  - Implemented: DelegateResource/UnDelegateResource to a temp account (lock=false)
  - Implemented: CancelAllUnfreezeV2 (no-op safe) and WithdrawExpireUnfreeze (safe assertion)

- `integration_test/write_tests/trc20_write_test.go`
  - Implemented: Approve + Allowance checks; optional TransferFrom using Key2 (pending)

- `integration_test/write_tests/smartcontract_write_test.go`
  - Implemented: Trigger `setValue(uint256)` on `MINIMALCONTRACT_CONTRACT_ADDRESS` and verify via `value()`/`getValue()`
  - Implemented: Simulate `setValue(uint256)` via `Contract.Simulate`
  - Pending: Deploy MinimalContract via SmartContractManager (fee-limited), verify `GetContract` and `GetContractInfo`, then `ClearContractABI` and `UpdateSetting`/`UpdateEnergyLimit` no-ops

- `integration_test/write_tests/trc10_test.go`
  - CreateAssetIssue2 (tiny supply), TransferAsset2, GetAssetIssueByAccount, then UnfreezeAsset2 (where applicable)

- `integration_test/write_tests/voting_test.go`
  - Implemented: VoteWitnessAccount2 with a minimal count (tolerates insufficient TRON Power)
  - Implemented: GetRewardInfo (owner)
  - Implemented: GetBrokerageInfo (first witness)
  - Pending: WithdrawBalance2 (safe if zero)

- `integration_test/write_tests/client_simulate_test.go`
  - Exercise `client.Simulate` on a TRC20 transfer to assert `EnergyUsage`, logs presence, and constant return handling

Mainnet (read-only):
- Implemented in `integration_test/read_tests/network_more_test.go`:
  - GetBlockById (known block)
  - GetBlocksByLimit (bounded window)
  - GetLatestBlocks (small count)
  - Still pending: add explicit `GetContract` read for a popular contract (e.g., USDT)

## Design patterns and guardrails
- Use small, reversible state changes on Nile; prefer no-ops where possible (e.g., `CancelAllUnfreezeV2` when none exist)
- Assert structure and invariants, not volatile values (balances may change)
- Centralize timeouts and endpoints in helpers; read from env with sane defaults
- For write tests, wait for receipts and log txid, energy, net usage, decoded events
- Gate Nile tests with `RUN_NILE_WRITE_TESTS` in CI; skip if not set

## Known stable vectors
- Popular mainnet contracts (USDT, USDD) for read-only TRC20 methods
- Stable transaction IDs captured in existing tests (see `integration_test/read_tests/network_test.go`)

## Maintenance
- When contracts change on Nile, re-run `cmd/setup_nile_testnet` to refresh addresses
- Keep `integration_test/test.env` updated; never commit private keys beyond designated test keys
- Prefer adding tests alongside the high-level manager that owns the behavior

---
This guide should be used as the source of truth to close [G]aps and reach full high-level API coverage across mainnet (read) and Nile (write) suites.


