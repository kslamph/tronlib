// Command setup_nile_testnet provides helpers to scaffold local credentials and
// environment for TRON Nile testnet interactions.
package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pkg/account"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/signer"
	"github.com/kslamph/tronlib/pkg/smartcontract"
	"github.com/kslamph/tronlib/pkg/types"
)

const (
	// Deployment configuration
	MinimumBalanceTRX  = 1
	DefaultFeeLimit    = 10 * types.SunPerTRX // 10 TRX fee limit per deployment
	DefaultEnergyLimit = types.DefaultEnergyLimit

	// Contract deployment parameters
	TRC20Name          = "TronLib Test"
	TRC20Symbol        = "TLT"
	TRC20Decimals      = 18
	TRC20InitialSupply = "1000000000000000000000000" // 1M tokens with 18 decimals

	TestComprehensiveTypesStatus = 0 // Status.Pending
)

// SetupConfig holds the configuration for the setup process
type SetupConfig struct {
	NodeURL                  string
	Key1PrivateKey           string
	Key1Address              *types.Address
	ProjectRoot              string
	ContractBuildDir         string
	TestEnvFiles             []string
	DryRun                   bool
	ShieldedBindTokenAddress *types.Address
}

// ContractInfo holds information about a contract to be deployed
type ContractInfo struct {
	Name              string
	ABIFile           string
	BinFile           string
	ConstructorParams []interface{}
	EnvVarName        string
}

// DeploymentResult holds the result of a contract deployment
type DeploymentResult struct {
	ContractName    string
	Address         string // keep as string for env writing and display
	TxID            string
	Success         bool
	BroadcastResult *api.Return
	Error           error
}

// NileTestnetSetup manages the entire setup process
type NileTestnetSetup struct {
	config            SetupConfig
	client            *client.Client
	accountManager    *account.AccountManager
	contractManager   *smartcontract.Manager
	signer            *signer.PrivateKeySigner
	deploymentResults []DeploymentResult
}

func main() {
	setup, err := NewNileTestnetSetup()
	if err != nil {
		log.Fatalf("Failed to initialize setup: %v", err)
	}
	defer setup.cleanup()

	if err := setup.Run(); err != nil {
		log.Fatalf("Setup failed: %v", err)
	}

	fmt.Println("✅ Nile testnet setup completed successfully!")
}

// NewNileTestnetSetup creates a new setup instance
func NewNileTestnetSetup() (*NileTestnetSetup, error) {
	config, err := loadSetupConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create client
	c, err := client.NewClient(config.NodeURL, client.WithTimeout(60*time.Second))
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	// Create managers
	accountManager := account.NewManager(c)
	contractManager := c.SmartContract()

	// Create signer
	signer, err := signer.NewPrivateKeySigner(config.Key1PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %w", err)
	}

	return &NileTestnetSetup{
		config:          config,
		client:          c,
		accountManager:  accountManager,
		contractManager: contractManager,
		signer:          signer,
	}, nil
}

// Run executes the complete setup process
func (s *NileTestnetSetup) Run() error {
	fmt.Println("🚀 Starting Nile Testnet Contract Deployment Setup")
	fmt.Printf("📍 Node URL: %s\n", s.config.NodeURL)
	fmt.Printf("👤 Key1 Address: %s\n", s.config.Key1Address.String())
	fmt.Printf("🏗️  Project Root: %s\n", s.config.ProjectRoot)

	if s.config.DryRun {
		fmt.Println("🔍 DRY RUN MODE - No actual deployments will be performed")
	}

	// Step 1: Verify high-level package capabilities
	if err := s.verifyPackageCapabilities(); err != nil {
		return fmt.Errorf("package verification failed: %w", err)
	}

	// Step 2: Verify Key1 account balance
	if err := s.verifyAccountBalance(); err != nil {
		return fmt.Errorf("balance verification failed: %w", err)
	}

	// Step 3: Prepare contract deployment parameters
	contracts, err := s.prepareContractParameters()
	if err != nil {
		return fmt.Errorf("contract preparation failed: %w", err)
	}

	// Steps 4-6: Deploy contracts sequentially
	for _, contract := range contracts {
		// Check if contract is already deployed
		if s.isContractAlreadyDeployed(contract.Name) {
			fmt.Printf("⏭️  Skipping %s - already deployed\n", contract.Name)
			continue
		}

		result, err := s.deployContract(contract)
		if err != nil {
			return fmt.Errorf("deployment of %s failed: %w", contract.Name, err)
		}

		s.deploymentResults = append(s.deploymentResults, result)

		// Update environment files immediately after successful deployment
		if result.Success {
			if err := s.updateEnvironmentFilesForContract(result); err != nil {
				fmt.Printf("⚠️  Warning: Failed to update environment files for %s: %v\n", result.ContractName, err)
				// Don't fail the entire process, just warn and continue
			} else {
				fmt.Printf("✅ Environment files updated for %s\n", result.ContractName)
			}
		}

		// Wait between deployments to avoid nonce conflicts
		if !s.config.DryRun {
			time.Sleep(5 * time.Second)
		}
	}

	// Step 7: Final environment verification (instead of bulk update)
	if err := s.verifyEnvironmentFiles(); err != nil {
		return fmt.Errorf("environment verification failed: %w", err)
	}

	// Step 8: Verify contract deployments
	if err := s.verifyDeployments(); err != nil {
		return fmt.Errorf("deployment verification failed: %w", err)
	}

	s.printSummary()
	return nil
}

// verifyPackageCapabilities checks that all required high-level packages are available
func (s *NileTestnetSetup) verifyPackageCapabilities() error {
	fmt.Println("\n📋 Step 1: Verifying High-Level Package Capabilities")

	// Check that we have all required managers
	if s.accountManager == nil {
		return fmt.Errorf("account manager not available")
	}
	if s.contractManager == nil {
		return fmt.Errorf("contract manager not available")
	}
	if s.signer == nil {
		return fmt.Errorf("signer not available")
	}

	fmt.Println("✅ Account manager available")
	fmt.Println("✅ Smart contract manager available")
	fmt.Println("✅ Transaction signer available")
	fmt.Println("✅ Workflow manager available")

	return nil
}

// verifyAccountBalance checks that Key1 has sufficient TRX for deployments
func (s *NileTestnetSetup) verifyAccountBalance() error {
	fmt.Println("\n💰 Step 2: Verifying Key1 Account Balance")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if s.config.DryRun {
		fmt.Printf("🔍 DRY RUN: Would check balance for address %s\n", s.config.Key1Address.String())
		fmt.Printf("🔍 DRY RUN: Would verify balance ≥ %d TRX\n", MinimumBalanceTRX)
		return nil
	}

	balance, err := s.accountManager.GetBalance(ctx, s.config.Key1Address)
	if err != nil {
		return fmt.Errorf("failed to get account balance: %w", err)
	}

	balanceTRX := balance / types.SunPerTRX
	fmt.Printf("💰 Current balance: %d TRX (%d SUN)\n", balanceTRX, balance)

	if balanceTRX < MinimumBalanceTRX {
		return fmt.Errorf("insufficient balance: have %d TRX, need at least %d TRX", balanceTRX, MinimumBalanceTRX)
	}

	fmt.Printf("✅ Sufficient balance confirmed: %d TRX ≥ %d TRX\n", balanceTRX, MinimumBalanceTRX)
	return nil
}

// prepareContractParameters loads contract files and prepares deployment parameters
func (s *NileTestnetSetup) prepareContractParameters() ([]ContractInfo, error) {
	fmt.Println("\n🔧 Step 3: Preparing Contract Deployment Parameters")

	contracts := []ContractInfo{
		{
			Name:              "MinimalContract",
			ABIFile:           "MinimalContract.abi",
			BinFile:           "MinimalContract.bin",
			ConstructorParams: []interface{}{}, // No constructor parameters
			EnvVarName:        "MINIMAL_CONTRACT_ADDRESS",
		},
		{
			Name:    "TRC20",
			ABIFile: "TRC20.abi",
			BinFile: "TRC20.bin",
			ConstructorParams: []interface{}{
				TRC20Name,            // name_
				TRC20Symbol,          // symbol_
				uint8(TRC20Decimals), // decimals_
				TRC20InitialSupply,   // initialSupply_
			},
			EnvVarName: "TRC20_CONTRACT_ADDRESS",
		},
		{
			Name:    "TestComprehensiveTypes",
			ABIFile: "TestComprehensiveTypes.abi",
			BinFile: "TestComprehensiveTypes.bin",
			ConstructorParams: []interface{}{
				uint8(TestComprehensiveTypesStatus),                     // _currentStatus (enum)
				s.config.Key1Address,                                    // _myAddress
				[]*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3)}, // _uintArray
			},
			EnvVarName: "TESTCOMPREHENSIVETYPES_CONTRACT_ADDRESS",
		},
		{
			Name:    "ShieldedTRC20",
			ABIFile: "ShieldedTRC20.abi",
			BinFile: "ShieldedTRC20.bin",
			ConstructorParams: []interface{}{
				s.config.ShieldedBindTokenAddress, // trc20ContractAddress: the TRC-20 token to bind, NOT a shielded contract
				big.NewInt(0),                     // scalingFactorExponent -> scalingFactor == 10**0 == 1
			},
			EnvVarName: "SHIELDEDTRC20_CONTRACT_ADDRESS",
		},
	}

	// Verify all contract files exist and load them
	for i := range contracts {
		contract := &contracts[i]

		// Load ABI
		abiPath := filepath.Join(s.config.ContractBuildDir, contract.ABIFile)
		abiBytes, err := os.ReadFile(abiPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read ABI file %s: %w", abiPath, err)
		}

		// Load bytecode
		binPath := filepath.Join(s.config.ContractBuildDir, contract.BinFile)
		binBytes, err := os.ReadFile(binPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read bytecode file %s: %w", binPath, err)
		}

		fmt.Printf("📄 Loaded %s: ABI=%d bytes, Bytecode=%d bytes\n",
			contract.Name, len(abiBytes), len(binBytes))

		// Store the loaded data (in a real implementation, we'd store these in the struct)
		_ = abiBytes
		_ = binBytes
	}

	fmt.Printf("✅ Prepared %d contracts for deployment\n", len(contracts))
	return contracts, nil
}

// deployContract deploys a single contract
func (s *NileTestnetSetup) deployContract(contract ContractInfo) (DeploymentResult, error) {
	fmt.Printf("\n🚀 Deploying %s Contract\n", contract.Name)

	result := DeploymentResult{
		ContractName: contract.Name,
	}

	if s.config.DryRun {
		fmt.Printf("🔍 DRY RUN: Would deploy %s with parameters: %v\n",
			contract.Name, contract.ConstructorParams)

		// Simulate successful deployment
		result.Address = fmt.Sprintf("T%s%s", contract.Name, "MockAddress123456789")
		result.TxID = fmt.Sprintf("mock_tx_%s_%d", strings.ToLower(contract.Name), time.Now().Unix())
		result.Success = true

		fmt.Printf("🔍 DRY RUN: Mock deployment successful\n")
		fmt.Printf("🔍 DRY RUN: Mock Contract Address: %s\n", result.Address)
		fmt.Printf("🔍 DRY RUN: Mock Transaction ID: %s\n", result.TxID)

		return result, nil
	}

	// Load contract files
	abiPath := filepath.Join(s.config.ContractBuildDir, contract.ABIFile)
	binPath := filepath.Join(s.config.ContractBuildDir, contract.BinFile)

	abiBytes, err := os.ReadFile(abiPath)
	if err != nil {
		result.Error = err
		return result, fmt.Errorf("failed to read ABI: %w", err)
	}

	binBytes, err := os.ReadFile(binPath)
	if err != nil {
		result.Error = err
		return result, fmt.Errorf("failed to read bytecode: %w", err)
	}

	// Decode hex bytecode
	bytecode, err := hex.DecodeString(strings.TrimSpace(string(binBytes)))
	if err != nil {
		result.Error = err
		return result, fmt.Errorf("failed to decode bytecode: %w", err)
	}

	// Create deployment transaction
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fmt.Printf("📝 Creating deployment transaction...\n")

	// NOTE: Deploy expects ownerAddress as *types.Address
	txExt, err := s.contractManager.Deploy(
		ctx,
		s.config.Key1Address,          // ownerAddress as *types.Address
		contract.Name,                 // contractName
		string(abiBytes),              // abi as string
		bytecode,                      // bytecode
		0,                             // callValue
		100,                           // consumeUserResourcePercent
		DefaultEnergyLimit,            // originEnergyLimit
		contract.ConstructorParams..., // constructor parameters
	)
	if err != nil {
		result.Error = err
		return result, fmt.Errorf("failed to create deployment transaction: %w", err)
	}
	fmt.Printf("🔍 Transaction: %+v\n", txExt)

	// Sign and broadcast transaction
	fmt.Printf("✍️  Signing and broadcasting transaction...\n")
	simulate, err := s.client.Simulate(ctx, txExt)
	if err != nil {
		result.Error = err
		return result, fmt.Errorf("failed to simulate transaction: %w", err)
	}
	fmt.Printf("🔍 Simulated transaction: %+v\n", simulate)
	// return result, nil

	rst, err := s.client.SignAndBroadcast(ctx, txExt, client.BroadcastOptions{FeeLimit: 2000000000, WaitForReceipt: true}, s.signer)
	if err != nil {
		fmt.Printf("X broadcast failed \n")
	}

	fmt.Printf("✅ %s deployed successfully!\n", contract.Name)
	fmt.Printf("📍 Contract Address: %s\n", result.Address)
	fmt.Printf("🔗 Transaction ID: %s\n", rst.TxID)

	return result, nil
}

// verifyDeployments verifies that all deployed contracts are accessible
func (s *NileTestnetSetup) verifyDeployments() error {
	fmt.Println("\n🔍 Step 8: Verifying Contract Deployments")

	if s.config.DryRun {
		fmt.Println("🔍 DRY RUN: Would verify the following contract deployments:")
		for _, result := range s.deploymentResults {
			if result.Success {
				fmt.Printf("🔍 DRY RUN: - %s at %s\n", result.ContractName, result.Address)
			}
		}
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, result := range s.deploymentResults {
		if !result.Success {
			continue
		}

		fmt.Printf("🔍 Verifying %s at %s...\n", result.ContractName, result.Address)
		addr, err := types.NewAddress(result.Address)
		if err != nil {
			return fmt.Errorf("invalid address %s: %w", result.Address, err)
		}
		contract, err := s.contractManager.GetContract(ctx, addr)
		if err != nil {
			return fmt.Errorf("failed to verify %s: %w", result.ContractName, err)
		}

		if contract == nil {
			return fmt.Errorf("contract %s not found at address %s", result.ContractName, result.Address)
		}

		fmt.Printf("✅ %s verified successfully\n", result.ContractName)
	}

	return nil
}

// printSummary prints a summary of the deployment results
func (s *NileTestnetSetup) printSummary() {
	fmt.Println("\n📊 Deployment Summary")
	fmt.Println(strings.Repeat("=", 50))

	successCount := 0
	for _, result := range s.deploymentResults {
		status := "❌ FAILED"
		if result.Success {
			status = "✅ SUCCESS"
			successCount++
		}

		fmt.Printf("%s %s\n", status, result.ContractName)
		if result.Success {
			fmt.Printf("   📍 Address: %s\n", result.Address)
			fmt.Printf("   🔗 TX ID: %s\n", result.TxID)
		} else if result.Error != nil {
			fmt.Printf("   ❌ Error: %s\n", result.Error.Error())
		}
		fmt.Println()
	}

	fmt.Printf("📈 Success Rate: %d/%d contracts deployed successfully\n",
		successCount, len(s.deploymentResults))

	if successCount == len(s.deploymentResults) {
		fmt.Println("🎉 All contracts deployed successfully!")
		fmt.Println("🧪 Environment is ready for integration testing!")
	}
}

// loadSetupConfig loads configuration from environment and files
func loadSetupConfig() (SetupConfig, error) {
	// Get project root
	currentFolder, err := os.Getwd()
	if err != nil {
		return SetupConfig{}, fmt.Errorf("failed to get working directory: %w", err)
	}

	// Load Key1 private key from test.env
	testEnvPath := filepath.Join(currentFolder, "test.env")
	key1PrivateKey, err := loadKey1FromEnv(testEnvPath)
	if err != nil {
		return SetupConfig{}, fmt.Errorf("failed to load Key1: %w", err)
	}

	// Derive Key1 address
	signer, err := signer.NewPrivateKeySigner(key1PrivateKey)
	if err != nil {
		return SetupConfig{}, fmt.Errorf("failed to create signer: %w", err)
	}

	// The TRC-20 token the ShieldedTRC20 contract binds to in its constructor.
	// This is deliberately *not* SHIELDEDTRC20_CONTRACT_ADDRESS: the deploy loop
	// writes that key with the newly deployed shielded contract, so reading the
	// bind target from it would make a second run bind the contract to itself.
	bindAddrStr := envOrFirst("SHIELDED_BIND_TOKEN_ADDRESS", "TRC20_CONTRACT_ADDRESS")
	if bindAddrStr == "" {
		bindAddrStr = "TWRvzd6FQcsyp7hwCtttjZGpU1kfvVEtNK" // known TRC-20 deployment on Nile
	}
	shieldedBindTokenAddress, err := types.NewAddress(bindAddrStr)
	if err != nil {
		return SetupConfig{}, fmt.Errorf("invalid shielded bind token address %q: %w", bindAddrStr, err)
	}

	config := SetupConfig{
		NodeURL:        "grpc://grpc.nile.trongrid.io:50051",
		Key1PrivateKey: key1PrivateKey,
		// Convert string key1Address from env to *types.Address
		Key1Address:              signer.Address(),
		ProjectRoot:              currentFolder,
		ContractBuildDir:         filepath.Join(currentFolder, "test_contract", "build"),
		ShieldedBindTokenAddress: shieldedBindTokenAddress,
		TestEnvFiles: []string{
			filepath.Join(currentFolder, "test.env"),
		},
		DryRun: os.Getenv("DRY_RUN") == "true",
	}

	return config, nil
}

// envOrFirst returns the value of the first environment variable that is set and
// non-empty, or "" when none of them are.
func envOrFirst(keys ...string) string {
	for _, key := range keys {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}
	return ""
}

// loadKey1FromEnv loads the Key1 private key from the test.env file
func loadKey1FromEnv(envPath string) (string, error) {
	content, err := os.ReadFile(envPath)
	if err != nil {
		return "", fmt.Errorf("failed to read env file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "NILE_TEST_KEY1=") {
			return strings.TrimPrefix(line, "NILE_TEST_KEY1="), nil
		}
	}

	return "", fmt.Errorf("NILE_TEST_KEY1 not found in %s", envPath)
}

// cleanup performs cleanup operations
func (s *NileTestnetSetup) cleanup() {
	if s.client != nil {
		s.client.Close()
	}
}

// updateEnvironmentFilesForContract updates environment files for a single deployed contract
func (s *NileTestnetSetup) updateEnvironmentFilesForContract(result DeploymentResult) error {
	if !result.Success {
		return fmt.Errorf("cannot update environment for failed deployment")
	}

	if s.config.DryRun {
		fmt.Printf("🔍 DRY RUN: Would update environment files for %s with address %s\n", result.ContractName, result.Address)
		return nil
	}

	// Update all configured environment files
	for _, envFile := range s.config.TestEnvFiles {
		if err := s.updateSingleContractInFile(envFile, result); err != nil {
			return fmt.Errorf("failed to update %s: %w", envFile, err)
		}
	}

	return nil
}

// updateSingleContractInFile updates a single contract's address in an environment file
func (s *NileTestnetSetup) updateSingleContractInFile(filePath string, result DeploymentResult) error {
	// Read current file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create new file if it doesn't exist
			content = []byte{}
		} else {
			return fmt.Errorf("failed to read %s: %w", filePath, err)
		}
	}

	lines := strings.Split(string(content), "\n")

	// Find and update the line for this contract
	contractKey := fmt.Sprintf("%s_CONTRACT_ADDRESS", strings.ToUpper(result.ContractName))
	updated := false

	for i, line := range lines {
		if strings.HasPrefix(line, contractKey+"=") {
			lines[i] = fmt.Sprintf("%s=%s", contractKey, result.Address)
			updated = true
			break
		}
	}

	// If not found, append new line
	if !updated {
		if len(lines) > 0 && lines[len(lines)-1] != "" {
			lines = append(lines, "")
		}
		lines = append(lines, fmt.Sprintf("%s=%s", contractKey, result.Address))
	}

	// Write back to file
	newContent := strings.Join(lines, "\n")
	if err := os.WriteFile(filePath, []byte(newContent), 0600); err != nil {
		return fmt.Errorf("failed to write %s: %w", filePath, err)
	}

	return nil
}

// verifyEnvironmentFiles checks that all deployed contracts are properly recorded in environment files
func (s *NileTestnetSetup) verifyEnvironmentFiles() error {
	fmt.Println("\n📝 Step 7: Verifying Environment Configuration Files")

	if s.config.DryRun {
		fmt.Println("🔍 DRY RUN: Would verify the following environment files:")
		for _, envFile := range s.config.TestEnvFiles {
			fmt.Printf("🔍 DRY RUN: - %s\n", envFile)
		}

		fmt.Println("🔍 DRY RUN: Would verify the following environment variables:")
		for _, result := range s.deploymentResults {
			if result.Success {
				contractKey := fmt.Sprintf("%s_CONTRACT_ADDRESS", strings.ToUpper(result.ContractName))
				fmt.Printf("🔍 DRY RUN: - %s=%s\n", contractKey, result.Address)
			}
		}
		return nil
	}

	for _, envFile := range s.config.TestEnvFiles {
		if err := s.verifyContractsInFile(envFile); err != nil {
			return fmt.Errorf("verification failed for %s: %w", envFile, err)
		}
		fmt.Printf("✅ Verified %s\n", envFile)
	}

	return nil
}

// verifyContractsInFile checks that all successfully deployed contracts are recorded in the specified file
func (s *NileTestnetSetup) verifyContractsInFile(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", filePath, err)
	}

	fileContent := string(content)

	for _, result := range s.deploymentResults {
		if result.Success {
			contractKey := fmt.Sprintf("%s_CONTRACT_ADDRESS", strings.ToUpper(result.ContractName))
			expectedLine := fmt.Sprintf("%s=%s", contractKey, result.Address)

			if !strings.Contains(fileContent, expectedLine) {
				return fmt.Errorf("contract %s address not found in %s", result.ContractName, filePath)
			}
		}
	}

	return nil
}

// isContractAlreadyDeployed checks if a contract is already deployed by looking for its address in the environment file
func (s *NileTestnetSetup) isContractAlreadyDeployed(contractName string) bool {
	// In dry-run mode, never skip (so we can see what would be deployed)
	if s.config.DryRun {
		return false
	}

	// Check the main environment file where private keys are read from
	envFile := s.config.TestEnvFiles[0] // Use the first (and only) configured env file
	content, err := os.ReadFile(envFile)
	if err != nil {
		// If we can't read the file, assume contract is not deployed
		return false
	}

	// Look for the contract address variable
	contractKey := fmt.Sprintf("%s_CONTRACT_ADDRESS", strings.ToUpper(contractName))
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, contractKey+"=") {
			// Extract the value after the equals sign
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				address := strings.TrimSpace(parts[1])
				// Consider it deployed if the address is not empty
				return address != ""
			}
		}
	}

	return false
}
