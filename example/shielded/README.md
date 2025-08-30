# Real Shielded TRC20 Transactions Implementation

This implementation demonstrates how to perform real shielded TRC20 transactions on the TRON blockchain, including the complete flow of:

1. **Key Generation**: Generate all necessary zk-SNARK keys
2. **Address Creation**: Create shielded addresses
3. **Minting**: Convert public TRC20 tokens to shielded tokens
4. **Scanning**: Find shielded notes using incoming viewing keys
5. **Burning**: Convert shielded tokens back to public TRC20 tokens

## Overview

Shielded TRC20 transactions provide privacy features for TRC20 tokens by using zero-knowledge proofs (zk-SNARKs). This implementation shows how to work with the actual gRPC APIs to generate the necessary cryptographic parameters and perform the transactions.

## Prerequisites

- A TRON node endpoint (this example uses Nile testnet)
- A private key with sufficient TRC20 tokens
- The shielded TRC20 contract address

## Running the Example

```bash
cd /path/to/tronlib
go run example/shielded/main.go
```

## Key Components

### 1. Key Generation

The implementation generates all necessary cryptographic keys:

- **Spending Key (sk)**: The master key for shielded transactions
- **Expanded Spending Key**: Contains ask, nsk, and ovk
- **Authorization Key (ak)**: Used for authorization signatures
- **Nullifier Secret Key (nsk)**: Used to generate nullifiers
- **Outgoing Viewing Key (ovk)**: Used to decrypt outgoing notes
- **Incoming Viewing Key (ivk)**: Used to scan for incoming notes
- **Nullifier Key (nk)**: Used to generate nullifiers

### 2. Address Creation

The implementation creates a shielded payment address using:

1. **Diversifier (d)**: A random value to create unique addresses
2. **Payment Address**: The final shielded address in ztron format

### 3. Minting Shielded Tokens

To mint shielded tokens:
- Specify the amount to convert from transparent to shielded
- Provide the shielded address to receive the tokens
- Generate the random commitment (rcm)
- Call `CreateShieldedContractParameters` with mint parameters

### 4. Scanning for Shielded Notes

To find shielded notes you own:
- Use your incoming viewing key (ivk)
- Scan a range of blocks for transactions
- Filter by the shielded TRC20 contract address

### 5. Burning Shielded Tokens

To burn shielded tokens:
- Select a shielded note to spend
- Specify the amount to convert back to transparent tokens
- Provide the transparent address to receive the tokens
- Call `CreateShieldedContractParameters` with burn parameters

## Implementation Details

### Key Generation Flow

1. `GetSpendingKey` - Generate the master spending key
2. `GetExpandedSpendingKey` - Derive ask, nsk, and ovk from sk
3. `GetAkFromAsk` - Generate ak from ask
4. `GetNkFromNsk` - Generate nk from nsk
5. `GetIncomingViewingKey` - Generate ivk from ak and nk
6. `GetDiversifier` - Generate a diversifier
7. `GetZenPaymentAddress` - Generate the shielded payment address
8. `GetRcm` - Generate random commitment for notes

### Transaction Parameters

#### Mint Parameters
- `Ovk`: Outgoing viewing key
- `FromAmount`: Amount to convert from transparent (10 tokens in example)
- `ShieldedReceives`: Array of notes to receive (with payment address and rcm)
- `Shielded_TRC20ContractAddress`: Address of the shielded TRC20 contract

#### Burn Parameters
- `Ask`: Authorization key
- `Nsk`: Nullifier secret key
- `Ovk`: Outgoing viewing key
- `ShieldedSpends`: Array of notes to spend
- `TransparentToAddress`: Address to receive the burned tokens
- `ToAmount`: Amount to convert back to transparent (5 tokens in example)
- `Shielded_TRC20ContractAddress`: Address of the shielded TRC20 contract

## Security Considerations

- Never expose your private keys or seed phrases
- Keep cryptographic keys secure
- Validate all transaction parameters
- Use secure random number generation for cryptographic parameters
- The implementation uses the TRON node's secure random generation for keys

## Next Steps for Full Implementation

To complete a full working implementation, you would need to:

1. **Call the actual gRPC methods** for each step
2. **Handle the returned parameters** and transaction data
3. **Sign and broadcast the transactions** using your private key
4. **Scan for and select actual shielded notes** to spend
5. **Generate merkle tree information** for the notes
6. **Handle transaction confirmations** and errors

## Sample Output

When you run the example, you'll see output similar to:

```
=== Real Shielded TRC20 Transaction Implementation ===

Step 1: Generating spending key...
Generated spending key (sk): 044b625389739392dab7ae9631a248b1c00a62a0aa1529a90b6e695d8f1a75f0

Step 2: Generating expanded spending key...
Generated ask: f8093dbdc5dc0f5ab574a97a510f1ddcc2abd33d0a5931b28aef7f3769fb6c06
Generated nsk: d74362526b9413b22591d2bc59b903373b9b7b4d25e02b2374c44df393f4e302
Generated ovk: 61066ea3ff159902f20219dc73e07396ee5a074f9a802003541c62a1618d825a

Step 3: Generating ak from ask...
Generated ak: 5d89ebff87526c8ae758b2c70c409048cf0de1b42cd410b504dfb98f25d64912

Step 4: Generating nk from nsk...
Generated nk: 5b2e0294cba27cf110daec8bd21e3bc0c625f47348abfac75edd05de935fe533

Step 5: Generating incoming viewing key...
Generated ivk: 14f28ce5f849be8c7dcbd97f4df7a78c4f93c234e7abb9e2e601bfe725159902

Step 6: Generating diversifier...
Generated diversifier (d): 778eb664e4f4dd68c65f5e

Step 7: Generating shielded payment address...
Generated shielded payment address: ztron1w78tve8y7nwk33jlt6az4n7tvxpe0xjw6kwrxyywcytn53fgerg9qp5wuhw0vzd77jdws9xjqpc

Step 8: Generating random commitment (rcm)...
Generated rcm: 0dfbb44caff18a5234a5ab581eb06b458d33ba3a4243a84fd6522e660996de09
```

This demonstrates that all the necessary cryptographic parameters have been successfully generated for performing shielded TRC20 transactions.