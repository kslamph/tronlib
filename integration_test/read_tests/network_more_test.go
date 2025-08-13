package read_tests

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMainnetGetBlockById tests retrieving a block by its ID (hash)
func TestMainnetGetBlockById(t *testing.T) {
	manager := setupNetworkTestManager(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Provided known mainnet block ID (hex)
	blockIdHex := "00000000047516e375a0301a5ba8e2e309a5b97333395cdc7afeaad99142154c"
	blockIdBytes, err := hex.DecodeString(blockIdHex)
	require.NoError(t, err, "block id hex should decode")

	block, err := manager.GetBlockById(ctx, blockIdBytes)
	require.NoError(t, err, "GetBlockById should succeed")
	require.NotNil(t, block, "Block should not be nil")

	header := block.GetBlockHeader()
	require.NotNil(t, header, "Block header should not be nil")
	raw := header.GetRawData()
	require.NotNil(t, raw, "Raw data should not be nil")

	// Validate basic invariants
	number := raw.GetNumber()
	assert.Greater(t, number, int64(0), "Block number should be positive")
	timestamp := raw.GetTimestamp()
	assert.Greater(t, timestamp, int64(0), "Timestamp should be positive")
	witness := raw.GetWitnessAddress()
	assert.NotEmpty(t, witness, "Witness address should not be empty")
	assert.Len(t, witness, 21, "Witness address should be 21 bytes")

	t.Logf("Fetched block by ID %s -> number=%d timestamp=%d", blockIdHex, number, timestamp)
}

// TestMainnetGetBlocksByLimit tests retrieving a bounded window of blocks
func TestMainnetGetBlocksByLimit(t *testing.T) {
	manager := setupNetworkTestManager(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	now, err := manager.GetNowBlock(ctx)
	require.NoError(t, err, "GetNowBlock should succeed")
	require.NotNil(t, now, "Now block should not be nil")

	nowNum := now.GetBlockHeader().GetRawData().GetNumber()
	// Choose a small safe window within the 100-block limit
	var window int64 = 5
	start := nowNum - (window - 1)
	if start < 0 {
		start = 0
	}
	end := nowNum

	list, err := manager.GetBlocksByLimit(ctx, start, end)
	require.NoError(t, err, "GetBlocksByLimit should succeed")
	require.NotNil(t, list, "Block list should not be nil")

	blocks := list.GetBlock()
	require.NotNil(t, blocks, "Blocks slice should not be nil")
	assert.Greater(t, len(blocks), 0, "Should return at least one block")
	assert.LessOrEqual(t, len(blocks), int(window), "Should not exceed requested window size")

	// Validate monotonicity and bounds
	var prevNum int64 = -1
	for i, be := range blocks {
		hdr := be.GetBlockHeader()
		require.NotNil(t, hdr, "block[%d] header should not be nil", i)
		rd := hdr.GetRawData()
		require.NotNil(t, rd, "block[%d] raw data should not be nil", i)

		num := rd.GetNumber()
		assert.GreaterOrEqual(t, num, start, "block[%d] number should be >= start", i)
		assert.LessOrEqual(t, num, end, "block[%d] number should be <= end", i)
		if prevNum >= 0 {
			assert.GreaterOrEqual(t, num, prevNum, "block numbers should be non-decreasing")
		}
		prevNum = num
	}

	t.Logf("Fetched %d blocks in range [%d, %d] (now=%d)", len(blocks), start, end, nowNum)
}

// TestMainnetGetLatestBlocks tests retrieving the latest N blocks
func TestMainnetGetLatestBlocks(t *testing.T) {
	manager := setupNetworkTestManager(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var count int64 = 5
	list, err := manager.GetLatestBlocks(ctx, count)
	require.NoError(t, err, "GetLatestBlocks should succeed")
	require.NotNil(t, list, "Block list should not be nil")

	blocks := list.GetBlock()
	require.NotNil(t, blocks, "Blocks slice should not be nil")
	assert.Greater(t, len(blocks), 0, "Should return at least one block")
	assert.LessOrEqual(t, int64(len(blocks)), count, "Should not exceed requested count")

	// Basic structural checks
	for i, be := range blocks {
		hdr := be.GetBlockHeader()
		require.NotNil(t, hdr, "block[%d] header should not be nil", i)
		rd := hdr.GetRawData()
		require.NotNil(t, rd, "block[%d] raw data should not be nil", i)
		assert.Greater(t, rd.GetNumber(), int64(0), "block[%d] number should be positive", i)
		assert.Greater(t, rd.GetTimestamp(), int64(0), "block[%d] timestamp should be positive", i)
	}

	t.Logf("Fetched latest %d blocks (returned=%d)", count, len(blocks))
}
