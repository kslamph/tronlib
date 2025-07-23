package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/kslamph/tronlib/pkg/types"
)

type Env struct {
	Key1                        string
	Key2                        string
	MainnetNodeURL              string
	NileNodeURL                 string
	ShastaNodeURL               string
	TestAllTypesContractAddress string
	TRC20ContractAddress        string
}

var env Env

func loadEnv() error {
	data, err := os.ReadFile("advanced_integration_test/test.env")
	if err != nil {
		return fmt.Errorf("failed to read test.env: %w", err)
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "INTEGRATION_TEST_KEY1=") {
			env.Key1 = strings.TrimPrefix(line, "INTEGRATION_TEST_KEY1=")
		}
		if strings.HasPrefix(line, "INTEGRATION_TEST_KEY2=") {
			env.Key2 = strings.TrimPrefix(line, "INTEGRATION_TEST_KEY2=")
		}
		if strings.HasPrefix(line, "MAINNET_NODE_URL=") {
			env.MainnetNodeURL = strings.TrimPrefix(line, "MAINNET_NODE_URL=")
		}
		if strings.HasPrefix(line, "NILE_NODE_URL=") {
			env.NileNodeURL = strings.TrimPrefix(line, "NILE_NODE_URL=")
		}
		if strings.HasPrefix(line, "SHASTA_NODE_URL=") {
			env.ShastaNodeURL = strings.TrimPrefix(line, "SHASTA_NODE_URL=")
		}
		if strings.HasPrefix(line, "TESTALLTYPES_CONTRACT_ADDRESS=") {
			env.TestAllTypesContractAddress = strings.TrimPrefix(line, "TESTALLTYPES_CONTRACT_ADDRESS=")
		}
		if strings.HasPrefix(line, "TRC20_CONTRACT_ADDRESS=") {
			env.TRC20ContractAddress = strings.TrimPrefix(line, "TRC20_CONTRACT_ADDRESS=")
		}
	}
	if env.Key1 == "" || env.Key2 == "" {
		return fmt.Errorf("INTEGRATION_TEST_KEY1 or INTEGRATION_TEST_KEY2 missing in test.env")
	}
	return nil
}

// DeployAndSaveContract deploys a contract and updates test.env
func DeployAndSaveContract(contractName string, constructorParams ...interface{}) (string, error) {
	// Load contract bytecode and ABI
	binPath := fmt.Sprintf("advanced_integration_test/test_contract/build/%s.bin", contractName)
	binData, err := os.ReadFile(binPath)
	if err != nil {
		return "", fmt.Errorf("failed to read contract bin: %w", err)
	}
	binBytes, err := hex.DecodeString(string(binData))
	if err != nil {
		return "", fmt.Errorf("failed to decode contract bin: %w", err)
	}
	abiPath := fmt.Sprintf("advanced_integration_test/test_contract/build/%s.abi", contractName)
	abi, err := os.ReadFile(abiPath)
	if err != nil {
		return "", fmt.Errorf("failed to read contract abi: %w", err)
	}

	tronClient, err := getTronClient()
	if err != nil {
		return "", fmt.Errorf("failed to create tron client: %w", err)
	}
	defer tronClient.Close()
	owner, err := types.NewAccountFromPrivateKey(env.Key1)
	if err != nil {
		return "", fmt.Errorf("invalid owner private key: %w", err)
	}

	ctx := context.Background()
	tx, err := tronClient.DeployContract(
		ctx,
		owner.Address(),
		binBytes,
		string(abi),
		contractName,
		1000000000000, // OriginEnergyLimit
		100,           // ConsumeUserResourcePercent
		constructorParams...,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create deploy contract transaction: %w", err)
	}
	signed, err := owner.Sign(tx.GetTransaction())
	if err != nil {
		return "", fmt.Errorf("failed to sign contract deploy transaction: %w", err)
	}
	receipt, err := tronClient.BroadcastTransaction(ctx, signed)
	if err != nil {
		return "", fmt.Errorf("failed to broadcast contract deploy transaction: %w", err)
	}
	if !receipt.GetResult() {
		return "", fmt.Errorf("contract deploy failed: %s", string(receipt.GetMessage()))
	}
	// TODO: Wait for confirmation and get contract address from tx info
	contractAddr := "DEPLOYED_CONTRACT_ADDRESS" // Placeholder
	// Update test.env
	updateEnvFile(contractName, contractAddr)
	return contractAddr, nil
}

func updateEnvFile(contractName, contractAddr string) {
	// TODO: Actually update the test.env file with the new contract address
	// This is a placeholder for demonstration
	fmt.Printf("Would update test.env: %s address = %s\n", contractName, contractAddr)
}

// Update PrepareAndValidateEnv to use DeployAndSaveContract for MinimalContract
func PrepareAndValidateEnv() error {
	if err := loadEnv(); err != nil {
		return err
	}
	acc1, err := CheckKey1()
	if err != nil {
		return err
	}

	// Deploy MinimalContract (no constructor params)
	fmt.Println("Testing minimal contract deployment...")
	minimalAddr, err := DeployAndSaveContract("MinimalContract")
	if err != nil {
		return fmt.Errorf("minimal contract deployment test failed: %w", err)
	}
	fmt.Printf("Minimal contract deployed successfully at: %s\n", minimalAddr)
	fmt.Println("Minimal contract deployment test completed. Stopping here.")

	acc2, err := CheckKey2(acc1)
	if err != nil {
		return err
	}
	fmt.Printf("Key1: %s, Key2: %s\n", acc1.Address().String(), acc2.Address().String())
	if env.TestAllTypesContractAddress == "" {
		// Example params for TestAllTypes
		myAddress := acc1.Address().Hex()
		myBool := true
		myUint := uint64(12345)
		addr, err := DeployAndSaveContract("TestAllTypes", myAddress, myBool, myUint)
		if err != nil {
			return fmt.Errorf("failed to deploy TestAllTypes contract: %w", err)
		}
		env.TestAllTypesContractAddress = addr
	}

	if env.TRC20ContractAddress == "" {
		// Example params for TRC20 (update as needed)
		name := "MyToken"
		symbol := "MTK"
		decimals := uint8(6)
		supply := uint64(10_000_000_000_000_000)
		addr, err := DeployAndSaveContract("TRC20", name, symbol, decimals, supply)
		if err != nil {
			return fmt.Errorf("failed to deploy TRC20 contract: %w", err)
		}
		env.TRC20ContractAddress = addr
	}

	return nil
}
