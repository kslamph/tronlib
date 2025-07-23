we need an full application to do some real onchain transactions , this is not a test package, you shall make it an executable main package under advanced_integration_test
what we have are test.env and test_contract
the application shall ensure it has enviropmetns ready, and proceed to next steps of tests.

## Implementation Plan

### 1. Environment Preparation & Validation

#### 1.1. Check key1
- Ensure key1 (from test.env) is activated.
- Ensure key1 has at least 4000 TRX.
- If not, stop and prompt user to top up key1.

#### 1.2. Check key2
- Ensure key2 (from test.env) has exactly 500 TRX.
- If not, transfer to/from key1 to make key2’s balance exactly 500 TRX.

#### 1.3. Deploy/Check Contracts
- Check if testalltypescontractaddress is present in test.env.
  - If not, deploy TestAllTypes contract and update test.env with its address.
- Check if trc20contractaddress is present in test.env.
  - If not, deploy TRC20 contract and update test.env with its address.

---

### 2. Basic Transaction Tests

#### 2.1. TRX Transfer
- Test transferring TRX between key1 and key2.

#### 2.2. TRC20 Transfers
- Test transferring TRC20 tokens between key1 and key2.

#### 2.3. TRC20 Approve & TransferFrom
- Test approving TRC20 tokens and using transferFrom.

---

### 3. Advanced Contract Tests

#### 3.1. TestAllTypes Contract
- Trigger functions on TestAllTypes contract.
- Read values from the contract.

#### 3.2. Event Parsing
- Parse and validate emitted events from contract interactions.

---

### 4. Resource Management Tests

#### 4.1. Resource Operations
- Freeze TRX for resources.
- Delegate resources.
- Reclaim delegated resources.
- Unfreeze resources.

#### 4.2. Voting & Rewards
- Vote with resources.
- Claim rewards (if claimable rewards > 1).

---

### 5. Extensibility
- Plan for adding more tests as new functions are added.

---

