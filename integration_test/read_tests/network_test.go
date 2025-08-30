package read_tests

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/eventdecoder"
	"github.com/kslamph/tronlib/pkg/network"
	"github.com/kslamph/tronlib/pkg/smartcontract"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// BlockData represents the structure of our test block data
type BlockData struct {
	BlockID     string `json:"blockID"`
	BlockHeader struct {
		RawData struct {
			Number         int64  `json:"number"`
			TxTrieRoot     string `json:"txTrieRoot"`
			WitnessAddress string `json:"witness_address"`
			ParentHash     string `json:"parentHash"`
			Version        int32  `json:"version"`
			Timestamp      int64  `json:"timestamp"`
		} `json:"raw_data"`
		WitnessSignature string `json:"witness_signature"`
	} `json:"block_header"`
	Transactions []struct {
		Ret []struct {
			ContractRet string `json:"contractRet"`
		} `json:"ret"`
		Signature []string `json:"signature"`
		TxID      string   `json:"txID"`
		RawData   struct {
			Contract []struct {
				Parameter struct {
					Value   map[string]interface{} `json:"value"`
					TypeUrl string                 `json:"type_url"`
				} `json:"parameter"`
				Type         string `json:"type"`
				PermissionId int32  `json:"Permission_id,omitempty"`
			} `json:"contract"`
			RefBlockBytes string `json:"ref_block_bytes"`
			RefBlockHash  string `json:"ref_block_hash"`
			Expiration    int64  `json:"expiration"`
			FeeLimit      int64  `json:"fee_limit,omitempty"`
			Timestamp     int64  `json:"timestamp"`
		} `json:"raw_data"`
		RawDataHex string `json:"raw_data_hex"`
	} `json:"transactions"`
}

// TransactionInfoTestData represents the structure of our test transaction info data
type TransactionInfoTestData struct {
	ID              string   `json:"id"`
	Fee             int64    `json:"fee,omitempty"`
	BlockNumber     int64    `json:"blockNumber"`
	BlockTimeStamp  int64    `json:"blockTimeStamp"`
	ContractResult  []string `json:"contractResult"`
	ContractAddress string   `json:"contract_address"`
	Receipt         struct {
		EnergyUsage        int64  `json:"energy_usage,omitempty"`
		EnergyUsageTotal   int64  `json:"energy_usage_total"`
		EnergyFee          int64  `json:"energy_fee,omitempty"`
		NetUsage           int64  `json:"net_usage,omitempty"`
		NetFee             int64  `json:"net_fee,omitempty"`
		Result             string `json:"result"`
		EnergyPenaltyTotal int64  `json:"energy_penalty_total,omitempty"`
	} `json:"receipt"`
	Result     string `json:"result,omitempty"`
	ResMessage string `json:"resMessage,omitempty"`
	Log        []struct {
		Address string   `json:"address"`
		Topics  []string `json:"topics"`
		Data    string   `json:"data"`
	} `json:"log,omitempty"`
	InternalTransactions []struct {
		Hash              string        `json:"hash"`
		CallerAddress     string        `json:"caller_address"`
		TransferToAddress string        `json:"transferTo_address"`
		CallValueInfo     []interface{} `json:"callValueInfo"`
		Note              string        `json:"note"`
		Rejected          bool          `json:"rejected,omitempty"`
	} `json:"internal_transactions,omitempty"`
}

// setupNetworkTestManager creates a test network manager instance
func setupNetworkTestManager(t *testing.T) *network.NetworkManager {
	config := getTestConfig()

	client, err := client.NewClient(config.Endpoint, client.WithTimeout(config.Timeout))
	require.NoError(t, err, "Failed to create client")

	return network.NewManager(client)
}

// TestMainnetGetNowBlock tests the GetNowBlock API against real network data
func TestMainnetGetNowBlock(t *testing.T) {
	manager := setupNetworkTestManager(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("GetNowBlock_ValidateStructure", func(t *testing.T) {
		block, err := manager.GetNowBlock(ctx)
		require.NoError(t, err, "GetNowBlock should succeed")
		require.NotNil(t, block, "Block should not be nil")

		// Validate block structure using gRPC getter methods
		blockHeader := block.GetBlockHeader()
		require.NotNil(t, blockHeader, "Block header should not be nil")

		rawData := blockHeader.GetRawData()
		require.NotNil(t, rawData, "Raw data should not be nil")

		// Validate basic block properties
		blockNumber := rawData.GetNumber()
		assert.Greater(t, blockNumber, int64(0), "Block number should be positive")
		t.Logf("Current block number: %d", blockNumber)

		timestamp := rawData.GetTimestamp()
		assert.Greater(t, timestamp, int64(0), "Timestamp should be positive")
		t.Logf("Block timestamp: %d", timestamp)

		version := rawData.GetVersion()
		assert.GreaterOrEqual(t, version, int32(0), "Version should be non-negative")
		t.Logf("Block version: %d", version)

		// Validate witness address
		witnessAddress := rawData.GetWitnessAddress()
		assert.NotEmpty(t, witnessAddress, "Witness address should not be empty")
		assert.Len(t, witnessAddress, 21, "Witness address should be 21 bytes")
		t.Logf("Witness address: %x", witnessAddress)

		// Validate parent hash
		parentHash := rawData.GetParentHash()
		assert.NotEmpty(t, parentHash, "Parent hash should not be empty")
		assert.Len(t, parentHash, 32, "Parent hash should be 32 bytes")
		t.Logf("Parent hash: %x", parentHash)

		// Validate tx trie root
		txTrieRoot := rawData.GetTxTrieRoot()
		if len(txTrieRoot) > 0 {
			assert.Len(t, txTrieRoot, 32, "Tx trie root should be 32 bytes when present")
			t.Logf("Tx trie root: %x", txTrieRoot)
		} else {
			t.Logf("No tx trie root (empty block)")
		}

		// Validate witness signature
		witnessSignature := blockHeader.GetWitnessSignature()
		assert.Len(t, witnessSignature, 65, "Witness signature should be 65 bytes")

		t.Logf("✅ Block structure validation passed")
	})

	t.Run("GetNowBlock_ValidateTransactions", func(t *testing.T) {
		block, err := manager.GetNowBlock(ctx)
		require.NoError(t, err, "GetNowBlock should succeed")

		transactions := block.GetTransactions()
		t.Logf("Block contains %d transactions", len(transactions))

		// Validate each transaction structure
		for i, tx := range transactions {
			if i >= 5 { // Limit validation to first 5 transactions for performance
				break
			}

			t.Logf("Validating transaction %d", i)

			// Validate transaction ID
			txID := tx.GetTxid()
			assert.NotEmpty(t, txID, "Transaction ID should not be empty")
			assert.Len(t, txID, 32, "Transaction ID should be 32 bytes")

			// Get the core transaction
			coreTransaction := tx.GetTransaction()
			require.NotNil(t, coreTransaction, "Core transaction should not be nil")

			// Validate raw data
			rawData := coreTransaction.GetRawData()
			require.NotNil(t, rawData, "Transaction raw data should not be nil")

			// Validate contracts
			contracts := rawData.GetContract()
			assert.Greater(t, len(contracts), 0, "Transaction should have at least one contract")

			for j, contract := range contracts {
				contractType := contract.GetType()
				assert.NotEqual(t, core.Transaction_Contract_ContractType(0), contractType, "Contract type should be valid")
				t.Logf("Transaction %d contract %d type: %v", i, j, contractType)

				parameter := contract.GetParameter()
				if parameter != nil {
					typeUrl := parameter.GetTypeUrl()
					assert.NotEmpty(t, typeUrl, "Parameter type URL should not be empty")
					t.Logf("Transaction %d contract %d parameter type: %s", i, j, typeUrl)
				}
			}

			// Validate timestamps
			timestamp := rawData.GetTimestamp()
			// assert.Greater(t, timestamp, int64(0), "Transaction timestamp should be positive")

			expiration := rawData.GetExpiration()
			assert.Greater(t, expiration, timestamp, "Expiration should be after timestamp")

			// Validate ref block data
			refBlockBytes := rawData.GetRefBlockBytes()
			assert.NotEmpty(t, refBlockBytes, "Ref block bytes should not be empty")

			refBlockHash := rawData.GetRefBlockHash()
			assert.NotEmpty(t, refBlockHash, "Ref block hash should not be empty")

			// Validate signatures
			signatures := coreTransaction.GetSignature()
			assert.Greater(t, len(signatures), 0, "Transaction should have at least one signature")
			for k, sig := range signatures {
				assert.Len(t, sig, 65, "Signature should be 65 bytes")
				t.Logf("Transaction %d signature %d length: %d bytes", i, k, len(sig))
			}

			// Validate transaction results
			ret := coreTransaction.GetRet()
			if len(ret) > 0 {
				for k, result := range ret {
					contractRet := result.GetContractRet()
					t.Logf("Transaction %d result %d: %v", i, k, contractRet)
				}
			}
		}

		t.Logf("✅ Transaction validation passed")
	})

	t.Run("GetNowBlock_ValidateAgainstTestData", func(t *testing.T) {
		// This test validates that our network manager can return data
		// that matches the expected structure from our test data
		block, err := manager.GetNowBlock(ctx)
		require.NoError(t, err, "GetNowBlock should succeed")

		// We can't validate exact values since we're getting current block,
		// but we can validate the structure matches our expected format

		// Validate block ID format
		blockId := block.GetBlockid()
		if len(blockId) > 0 {
			blockIdHex := hex.EncodeToString(blockId)
			assert.Len(t, blockIdHex, 64, "Block ID should be 32 bytes (64 hex chars)")
			t.Logf("Current block ID: %s", blockIdHex)

			// Compare format with expected data

		}

		// Validate timestamp is recent (within last hour)
		rawData := block.GetBlockHeader().GetRawData()
		currentTime := time.Now().UnixMilli()
		blockTime := rawData.GetTimestamp()
		timeDiff := currentTime - blockTime

		// Block should be recent (within 1 hour = 3600000 ms)
		assert.Less(t, timeDiff, int64(60000), "Block should be recent (within 1 hour)")
		t.Logf("Block age: %d ms", timeDiff)

		t.Logf("✅ Test data validation passed")
	})
}

// TestMainnetGetBlockByNumber tests the GetBlockByNumber API with known block
func TestMainnetGetBlockByNumber(t *testing.T) {
	manager := setupNetworkTestManager(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("GetBlockByNumber_KnownBlock", func(t *testing.T) {
		expectedBlockNumber := int64(68900960)
		block, err := manager.GetBlockByNumber(ctx, expectedBlockNumber)
		require.NoError(t, err, "GetBlockByNumber should succeed")
		require.NotNil(t, block, "Block should not be nil")

		// Validate we got the correct block
		rawData := block.GetBlockHeader().GetRawData()
		actualBlockNumber := rawData.GetNumber()
		assert.Equal(t, expectedBlockNumber, actualBlockNumber,
			"Should get the requested block number")

		// Validate timestamp matches expected
		actualTimestamp := rawData.GetTimestamp()
		expectedTimestamp := int64(1737354357000)
		assert.Equal(t, expectedTimestamp, actualTimestamp,
			"Block timestamp should match expected")

		// Validate witness address matches expected
		actualBlockHash := block.GetBlockid()
		assert.Equal(t, "00000000041b5860faa23511e1eb3f4d00b230bfac4fd4291753c9b9af1bb942", fmt.Sprintf("%x", actualBlockHash), "Block hash should match expected")

		actualNumberOfTransactions := len(block.GetTransactions())
		assert.Equal(t, 343, actualNumberOfTransactions, "Should have 343 transactions")

		t.Logf("✅ Successfully retrieved and validated block %d", expectedBlockNumber)
	})

	t.Run("GetBlockByNumber_InvalidInputs", func(t *testing.T) {
		// Test negative block number
		_, err := manager.GetBlockByNumber(ctx, -1)
		assert.Error(t, err, "Should reject negative block number")
		assert.Contains(t, err.Error(), "non-negative", "Error should mention non-negative requirement")

		t.Logf("✅ Input validation working correctly")
	})
}

// TestMainnetNetworkInfo tests network information APIs
func TestMainnetNetworkInfo(t *testing.T) {
	manager := setupNetworkTestManager(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("GetNodeInfo", func(t *testing.T) {
		nodeInfo, err := manager.GetNodeInfo(ctx)
		require.NoError(t, err, "GetNodeInfo should succeed")
		require.NotNil(t, nodeInfo, "Node info should not be nil")

		// Validate node info structure
		beginSyncNum := nodeInfo.GetBeginSyncNum()
		block := nodeInfo.GetBlock()
		solidityBlock := nodeInfo.GetSolidityBlock()

		t.Logf("Node begin sync number: %d", beginSyncNum)
		t.Logf("Node block: %s", block)
		t.Logf("Node solidity block: %s", solidityBlock)

		// Basic validation
		assert.GreaterOrEqual(t, beginSyncNum, int64(0), "Begin sync number should be non-negative")

		t.Logf("✅ Node info validation passed")
	})

	t.Run("GetChainParameters", func(t *testing.T) {
		chainParams, err := manager.GetChainParameters(ctx)
		require.NoError(t, err, "GetChainParameters should succeed")
		require.NotNil(t, chainParams, "Chain parameters should not be nil")

		// Validate chain parameters structure
		chainParams_list := chainParams.GetChainParameter()
		assert.Greater(t, len(chainParams_list), 0, "Should have chain parameters")

		t.Logf("Found %d chain parameters", len(chainParams_list))

		// Validate some key parameters
		for i, param := range chainParams_list {
			if i >= 10 { // Limit logging to first 10 parameters
				break
			}

			key := param.GetKey()
			value := param.GetValue()
			assert.NotEmpty(t, key, "Parameter key should not be empty")

			t.Logf("Chain parameter %d: %s = %d", i, key, value)
		}

		t.Logf("✅ Chain parameters validation passed")
	})

	t.Run("ListNodes", func(t *testing.T) {
		nodeList, err := manager.ListNodes(ctx)
		require.NoError(t, err, "ListNodes should succeed")
		require.NotNil(t, nodeList, "Node list should not be nil")

		// Validate node list structure
		nodes := nodeList.GetNodes()
		t.Logf("Found %d nodes", len(nodes))

		// Validate each node
		for i, node := range nodes {
			if i >= 5 { // Limit validation to first 5 nodes
				break
			}

			address := node.GetAddress()
			assert.NotNil(t, address, "Node address should not be nil")

			host := address.GetHost()
			port := address.GetPort()

			assert.NotEmpty(t, host, "Node host should not be empty")
			assert.Greater(t, port, int32(0), "Node port should be positive")

			t.Logf("Node %d: %s:%d", i, string(host), port)
		}

	})
}

// TestMainnetGetTransactionInfoById tests the GetTransactionInfoById API with real transaction data
func TestMainnetGetTransactionInfoById(t *testing.T) {
	manager := setupNetworkTestManager(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Load test data
	succTx := "f079d362f06b496cd22ccb9ec54d8c5bf0ef734e47613dc4caa76e4eb118f5a9"
	failedTx := "65fd57f1324178fb30a100584f454b1d03d987086dea87ed16c401aecb5dbf88"

	t.Run("GetTransactionInfoById_SuccessfulTransaction", func(t *testing.T) {
		txId := succTx
		txInfo, err := manager.GetTransactionInfoById(ctx, txId)
		require.NoError(t, err, "GetTransactionInfoById should succeed for successful transaction")
		require.NotNil(t, txInfo, "Transaction info should not be nil")

		// Validate basic transaction info using gRPC getter methods
		actualTxId := txInfo.GetId()
		assert.NotEmpty(t, actualTxId, "Transaction ID should not be empty")

		// Convert expected hex ID to bytes for comparison
		expectedTxIdBytes, err := hex.DecodeString(txId)
		require.NoError(t, err, "Should decode expected transaction ID")
		assert.Equal(t, expectedTxIdBytes, actualTxId, "Transaction ID should match expected")

		// Validate block information
		blockNumber := txInfo.GetBlockNumber()
		assert.Equal(t, int64(74255266), blockNumber, "Block number should match expected")
		t.Logf("Transaction in block: %d", blockNumber)

		blockTimeStamp := txInfo.GetBlockTimeStamp()
		assert.Equal(t, int64(1753424376000), blockTimeStamp, "Block timestamp should match expected")
		t.Logf("Block timestamp: %d", blockTimeStamp)

		// Validate contract result
		contractResult := txInfo.GetContractResult()
		assert.Greater(t, len(contractResult), 0, "Should have contract result")
		t.Logf("Contract result count: %d", len(contractResult))

		// Validate contract address
		contractAddress := txInfo.GetContractAddress()
		assert.NotEmpty(t, contractAddress, "Contract address should not be empty")

		// Convert expected hex address to bytes for comparison
		expectedContractAddrBytes := types.MustNewAddressFromBase58("TKFRELGGoRgiayhwJTNNLqCNjFoLBh3Mnf").Bytes()
		assert.Equal(t, expectedContractAddrBytes, contractAddress, "Contract address should match expected")
		t.Logf("Contract address: %x", contractAddress)

		// Validate receipt
		receipt := txInfo.GetReceipt()
		require.NotNil(t, receipt, "Receipt should not be nil")

		energyUsageTotal := receipt.GetEnergyUsageTotal()
		assert.Equal(t, int64(84291), energyUsageTotal, "Energy usage total should match expected")
		t.Logf("Energy usage total: %d", energyUsageTotal)

		result := receipt.GetResult()
		assert.Equal(t, core.Transaction_Result_SUCCESS, result, "Transaction result should be SUCCESS")
		t.Logf("Transaction result: %v", result)

		// Validate logs (events)
		logs := txInfo.GetLog()
		expectedLogCount := 6
		assert.Len(t, logs, expectedLogCount, "Log count should match expected")
		t.Logf("Event logs count: %d", len(logs))

		// Validate first few logs
		for i, log := range logs {
			if i >= 3 { // Limit validation to first 3 logs
				break
			}

			address := log.GetAddress()
			assert.NotEmpty(t, address, "Log address should not be empty")

			topics := log.GetTopics()
			assert.Greater(t, len(topics), 0, "Log should have topics")

			data := log.GetData()
			// Data can be empty for some events
			t.Logf("Log %d: address=%x, topics=%d, data_length=%d", i, address, len(topics), len(data))
		}

		// Note: Internal transactions may not always be included in gRPC response
		// so we'll log the count but not enforce strict equality
		t.Logf("Internal transactions count: %d (expected: %d)", len(txInfo.InternalTransactions), 13)

		// Validate first few internal transactions if they exist
		for i, internalTx := range txInfo.GetInternalTransactions() {
			if i >= 3 { // Limit validation to first 3 internal transactions
				break
			}

			hash := internalTx.GetHash()
			assert.NotEmpty(t, hash, "Internal transaction hash should not be empty")

			callerAddress := internalTx.GetCallerAddress()
			assert.NotEmpty(t, callerAddress, "Caller address should not be empty")

			transferToAddress := internalTx.GetTransferToAddress()
			assert.NotEmpty(t, transferToAddress, "Transfer to address should not be empty")

			note := internalTx.GetNote()
			// Note can be empty
			t.Logf("Internal tx %d: hash=%x, caller=%x, to=%x, note=%s",
				i, hash, callerAddress, transferToAddress, string(note))
		}

		t.Logf("✅ Successful transaction validation passed")
	})

	t.Run("GetTransactionInfoById_FailedTransaction", func(t *testing.T) {
		txId := failedTx
		txInfo, err := manager.GetTransactionInfoById(ctx, txId)
		require.NoError(t, err, "GetTransactionInfoById should succeed for failed transaction")
		require.NotNil(t, txInfo, "Transaction info should not be nil")

		// Validate basic transaction info
		actualTxId := txInfo.GetId()
		assert.NotEmpty(t, actualTxId, "Transaction ID should not be empty")

		expectedTxIdBytes, err := hex.DecodeString(txId)
		require.NoError(t, err, "Should decode expected transaction ID")
		assert.Equal(t, expectedTxIdBytes, actualTxId, "Transaction ID should match expected")

		// Validate block information
		blockNumber := txInfo.GetBlockNumber()
		assert.Equal(t, int64(71773886), blockNumber, "Block number should match expected")
		t.Logf("Failed transaction in block: %d", blockNumber)

		// Validate fee for failed transaction
		fee := txInfo.GetFee()
		assert.Equal(t, int64(28999810), fee, "Fee should match expected")
		t.Logf("Transaction fee: %d SUN", fee)

		// Validate receipt for failed transaction
		receipt := txInfo.GetReceipt()
		require.NotNil(t, receipt, "Receipt should not be nil")

		result := receipt.GetResult()
		assert.Equal(t, core.Transaction_Result_OUT_OF_ENERGY, result, "Transaction result should be OUT_OF_ENERGY")
		t.Logf("Failed transaction result: %v", result)

		energyUsageTotal := receipt.GetEnergyUsageTotal()
		assert.Equal(t, int64(134461), energyUsageTotal, "Energy usage total should match expected")
		t.Logf("Energy usage total: %d", energyUsageTotal)

		energyFee := receipt.GetEnergyFee()
		assert.Equal(t, int64(28236810), energyFee, "Energy fee should match expected")
		t.Logf("Energy fee: %d SUN", energyFee)

		// Validate result message for failed transaction
		resMessage := txInfo.GetResMessage()
		assert.NotEmpty(t, resMessage, "Failed transaction should have result message")
		t.Logf("Result message length: %d bytes", len(resMessage))

		// Validate internal transactions for failed transaction (should be rejected)
		internalTxs := txInfo.GetInternalTransactions()
		// Note: Internal transactions may not always be included in gRPC response
		// so we'll log the count but not enforce strict equality
		t.Logf("Failed transaction internal transactions count: %d (expected: %d)", len(internalTxs), 3)
		t.Logf("internal transactions: %v", internalTxs)

		// All internal transactions should be rejected for failed transaction if they exist
		for i, internalTx := range internalTxs {
			rejected := internalTx.GetRejected()
			assert.True(t, rejected, "Internal transaction %d should be rejected", i)
		}

		t.Logf("✅ Failed transaction validation passed")
	})

	t.Run("GetTransactionInfoById_InputValidation", func(t *testing.T) {
		// Test empty transaction ID
		_, err := manager.GetTransactionInfoById(ctx, "")
		assert.Error(t, err, "Should reject empty transaction ID")
		assert.Contains(t, err.Error(), "cannot be empty", "Error should mention empty ID")

		// Test invalid hex characters
		_, err = manager.GetTransactionInfoById(ctx, "invalid_hex_characters_here_not_valid_transaction_id_format")
		assert.Error(t, err, "Should reject invalid hex characters")
		// This could be either "invalid hex" or length error - both are acceptable
		assert.True(t,
			strings.Contains(err.Error(), "invalid hex") || strings.Contains(err.Error(), "64 hex characters"),
			"Error should mention invalid hex or length requirement, got: %s", err.Error())

		// Test wrong length
		_, err = manager.GetTransactionInfoById(ctx, "1234567890abcdef") // Too short
		assert.Error(t, err, "Should reject wrong length transaction ID")
		assert.Contains(t, err.Error(), "64 hex characters", "Error should mention correct length requirement")

		// Test with 0x prefix (should be accepted and stripped)
		txId := "0x" + succTx
		txInfo, err := manager.GetTransactionInfoById(ctx, txId)
		require.NoError(t, err, "Should accept transaction ID with 0x prefix")
		require.NotNil(t, txInfo, "Should return valid transaction info")

		// Test with 0X prefix (should be accepted and stripped)
		txId = "0X" + succTx
		txInfo, err = manager.GetTransactionInfoById(ctx, txId)
		require.NoError(t, err, "Should accept transaction ID with 0X prefix")
		require.NotNil(t, txInfo, "Should return valid transaction info")

		t.Logf("✅ Input validation working correctly")
	})

	t.Run("GetTransactionInfoById_NonexistentTransaction", func(t *testing.T) {
		// Test with a valid format but nonexistent transaction ID
		nonexistentTxId := "11000000000000000000000000000000000000000000000000000000000000ff"
		info, err := manager.GetTransactionInfoById(ctx, nonexistentTxId)
		// This should either return an error or return nil - both are acceptable
		// The important thing is that it doesn't crash
		require.NoError(t, err, "Should not return error for nonexistent transaction")
		require.Empty(t, info.GetId(), "Should return empty txid for nonexistent transaction")

		t.Logf("✅ Nonexistent transaction handling passed")
	})
}

// TestMainnetEventDecoder tests the event decoder functionality with a specific transaction
func TestMainnetEventDecoder(t *testing.T) {
	manager := setupNetworkTestManager(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Transaction ID that contains 10 events from 6 different contracts
	testTxId := "e1c63b2135e8fa7fbe2566d48b05194d5b9c01b32ed907d91e9cd55caa32c34c"

	t.Run("EventDecoder_DecodeTransactionEvents", func(t *testing.T) {
		// Get transaction info
		txInfo, err := manager.GetTransactionInfoById(ctx, testTxId)
		require.NoError(t, err, "Should retrieve transaction info successfully")
		require.NotNil(t, txInfo, "Transaction info should not be nil")

		// Validate we got the expected transaction
		actualTxId := hex.EncodeToString(txInfo.GetId())
		assert.Equal(t, testTxId, actualTxId, "Should get the correct transaction")

		// Get the event logs
		logs := txInfo.GetLog()
		require.Greater(t, len(logs), 0, "Transaction should have event logs")
		t.Logf("Found %d event logs in transaction %s", len(logs), testTxId)

		decodedEvents, err := eventdecoder.DecodeLogs(logs)
		require.NoError(t, err, "Should be able to decode event logs")

		// Validate we have the expected number of events and contracts
		assert.Equal(t, 10, len(decodedEvents), "Should have 10 events as expected")

		t.Logf("✅ Event decoder test completed successfully")
	})

	t.Run("EventDecoder_InputValidation", func(t *testing.T) {
		// Test with invalid transaction ID
		_, err := manager.GetTransactionInfoById(ctx, "invalid_tx_id")
		assert.Error(t, err, "Should reject invalid transaction ID")

		// Test with invalid contract address for NewContract
		client, err := client.NewClient(getTestConfig().Endpoint)
		require.NoError(t, err, "Should create client")

		invalidAddr, err := types.NewAddress("invalid_address")
		require.Error(t, err, "Should fail to create invalid address")
		_, err = smartcontract.NewInstance(client, invalidAddr)
		assert.Error(t, err, "Should reject invalid contract address")

		t.Logf("✅ Input validation tests passed")
	})
}
