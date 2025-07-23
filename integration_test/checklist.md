# Integration Test Coverage Checklist for Safe Functions in `pkg`

This checklist tracks which safe (read-only, non-mutating) functions from `pkg` are covered by integration tests.

## pkg/client

- [x] `GetAccount`
- [ ] `GetAccountNet`
- [ ] `GetAccountResource`
- [ ] `GetRewardInfo`
- [ ] `ListWitnesses`
- [x] `GetNodeInfo`
- [x] `ListNodes`
- [x] `GetChainParameters`
- [ ] `GetBlockById`
- [ ] `GetBlockByNum`
- [ ] `GetNowBlock`
- [ ] `NewContractFromAddress`
- [ ] `TriggerConstantSmartContract`
- [ ] `EstimateEnergy`
- [ ] `GetContractInfo`
- [ ] `GetTransactionById`
- [ ] `GetTransactionInfoById`
- [ ] `GetTransactionInfoByBlockNum`
- [ ] `WaitForTransactionInfo`
- [ ] `GetDelegatedResourceV2`
- [ ] `GetDelegatedResourceAccountIndexV2`
- [ ] `GetCanDelegatedMaxSize`
- [ ] `GetCanWithdrawUnfreezeAmount`

## pkg/smartcontract

- [ ] `DecodeABI`
- [x] `NewContract`
- [ ] `NewContractFromABI`
- [x] `DecodeEventSignature`
- [ ] `DecodeEventLog`
- [ ] `DecodeInputData`
- [ ] `DecodeResult`
- [x] `EncodeInput`

## pkg/types

- [ ] `NewAccountFromPrivateKey`
- [ ] `NewAccountFromHDWallet`
- [x] `NewAddress`
- [ ] `NewAddressFromBytes`
- [ ] `NewAddressFromHex`
- [ ] `MustNewAddress`
- [ ] `MustNewAddressFromBytes`
- [ ] `MustNewAddressFromHex`
- [ ] `Bytes`
- [ ] `Hex`
- [ ] `String`
- [ ] `PublicKey`
- [ ] `SignMessageV2`
- [ ] `PrivateKeyHex`

## pkg/helper

- [x] `ContractsSliceToMap`
- [x] `GetTxid`
- [x] `SunToTrx`
- [x] `TrxToSun`
- [x] `SunToTrxString`
- [x] `SunToTrxStringCommas`
- [x] `ParseTransactionInfoLog`

## pkg/crypto

- [ ] `VerifyMessageV2`

## pkg/trc20 (TRC20Contract, read-only methods only)

- [x] `Allowance`
- [x] `BalanceOf`
- [x] `Decimals`
- [x] `Name`
- [x] `Symbol`
- [x] `TotalSupply`

---

- [x] = Covered by integration tests
- [ ] = Not yet covered
