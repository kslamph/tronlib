# Nile Testnet Setup Program

This program automates the deployment of test contracts to the TRON Nile testnet and configures the testing environment for integration tests.

## Overview

The setup program deploys 3 smart contracts to Nile testnet:
1. **MinimalContract** - Simple contract with no constructor parameters
2. **TRC20** - Token contract with constructor parameters (name, symbol, decimals, initialSupply)
3. **TestAllTypes** - Complex contract for testing various data types

## Prerequisites

1. **Account Balance**: Key1 account must have at least 3000 TRX for deployment fees
2. **Contract Files**: All contract ABI and bytecode files must be present in `integration_test/setup_nile_test/test_contract/build/`
3. **Environment Files**: Key1 private key must be configured in `integration_test/setup_nile_test/test.env`

## Usage

### Dry Run Mode (Recommended First)

```bash
# Run in dry-run mode to verify configuration without actual deployments
./scripts/setup_nile_testnet.sh --dry-run
```

### Live Deployment

```bash
# Run actual deployments (requires sufficient TRX balance)
./scripts/setup_nile_testnet.sh
```

### Direct Program Execution

```bash
# Build and run the program directly
go build -o bin/setup_nile_testnet ./cmd/setup_nile_testnet
DRY_RUN=true ./bin/setup_nile_testnet  # Dry run
./bin/setup_nile_testnet               # Live deployment
```

## Configuration

The program automatically loads configuration from:

- **Node URL**: `grpc.nile.trongrid.io:50051`
- **Key1 Private Key**: From `integration_test/setup_nile_test/test.env`
- **Contract Files**: From `integration_test/setup_nile_test/test_contract/build/`
- **Environment Files**: Updates both `integration_test/test.env` and `integration_test/setup_nile_test/test.env`

## Contract Deployment Parameters

### MinimalContract
- No constructor parameters
- Energy limit: 10,000,000
- Fee limit: 10 TRX

### TRC20 Token
- **Name**: "TronLib Test"
- **Symbol**: "TLT"
- **Decimals**: 18
- **Initial Supply**: 1,000,000 tokens (1,000,000,000,000,000,000,000,000 with 18 decimals)
- Energy limit: 10,000,000
- Fee limit: 10 TRX

### TestAllTypes Contract
- **myAddress**: Key1 address
- **myBool**: true
- **myUint**: 42
- Energy limit: 10,000,000
- Fee limit: 10 TRX

## Process Flow

1. **Package Verification**: Confirms all high-level SDK packages are available
2. **Balance Check**: Verifies Key1 account has ≥3000 TRX
3. **Contract Preparation**: Loads and validates all contract files
4. **Sequential Deployment**: Deploys contracts one by one with 5-second delays
5. **Environment Update**: Updates test.env files with deployed contract addresses
6. **Verification**: Confirms all contracts are accessible on the network
7. **Summary Report**: Displays deployment results and next steps

## Output Files

After successful deployment, the following environment variables will be updated:

### `integration_test/test.env`
```
TESTALLTYPES_CONTRACT_ADDRESS=T...
TRC20_CONTRACT_ADDRESS=T...
```

### `integration_test/setup_nile_test/test.env`
```
MINIMAL_CONTRACT_ADDRESS=T...
TESTALLTYPES_CONTRACT_ADDRESS=T...
TRC20_CONTRACT_ADDRESS=T...
```

## Error Handling

The program includes comprehensive error handling for:

- **Insufficient Balance**: Stops if Key1 has <3000 TRX
- **Missing Files**: Validates all required contract files exist
- **Network Issues**: Retries with timeouts for network operations
- **Deployment Failures**: Reports specific errors for each contract
- **Address Parsing**: Validates contract addresses from deployment receipts

## Verification

After deployment, verify the setup by:

1. **Check Environment Files**: Ensure all contract addresses are populated
2. **Run Integration Tests**: Execute `go test ./integration_test/...`
3. **Manual Verification**: Use TRON block explorer to confirm contracts exist

## Troubleshooting

### Common Issues

1. **"Insufficient balance"**
   - Solution: Add more TRX to Key1 account

2. **"Failed to read ABI/bytecode file"**
   - Solution: Ensure contract files are compiled and present in build directory

3. **"Connection failed"**
   - Solution: Check internet connection and Nile testnet status

4. **"Transaction failed"**
   - Solution: Check gas/energy limits and account permissions

### Debug Mode

Set environment variable for verbose logging:
```bash
export DEBUG=true
./scripts/setup_nile_testnet.sh
```

## Security Notes

- Private keys are loaded from local environment files only
- No private keys are logged or transmitted
- All transactions are signed locally
- Dry-run mode performs no network operations

## Development

To modify the setup program:

1. Edit `cmd/setup_nile_testnet/main.go`
2. Update contract parameters in the constants section
3. Test with dry-run mode first
4. Build and deploy: `go build ./cmd/setup_nile_testnet`