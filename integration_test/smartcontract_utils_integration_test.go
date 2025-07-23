package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/smartcontract"
	"github.com/kslamph/tronlib/pkg/types"
)

const (
	UtilsUSDT_ABI = `[
		{"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"type":"function"},
		{"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"type":"function"},
		{"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"type":"function"},
		{"constant":true,"inputs":[{"name":"owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"type":"function"},
		{"constant":true,"inputs":[{"name":"owner","type":"address"},{"name":"spender","type":"address"}],"name":"allowance","outputs":[{"name":"","type":"uint256"}],"type":"function"}
	]`
	UtilsUSDTContractAddress = "TXDk8mbtRbXeYuMNS83CfKPaYYT8XWv9Hz"
	UtilsNodeEndpoint        = "127.0.0.1:50051"
	UtilsTestTxID            = "60a3be8dcf42cc13a38c35a00b89e115520756ae4106a7896d750fd9d8463f9c"
)

func TestSmartContractUtils(t *testing.T) {
	// DecodeABI
	abi, err := smartcontract.DecodeABI(UtilsUSDT_ABI)
	if err != nil {
		t.Errorf("DecodeABI failed: %v", err)
	} else {
		t.Logf("DecodeABI: %+v", abi)
	}

	// NewContractFromABI
	contract, err := smartcontract.NewContractFromABI(abi, UtilsUSDTContractAddress)
	if err != nil {
		t.Errorf("NewContractFromABI failed: %v", err)
	} else {
		t.Logf("NewContractFromABI: %+v", contract)
	}

	// EncodeInput
	data, err := contract.EncodeInput("symbol")
	if err != nil {
		t.Errorf("EncodeInput failed: %v", err)
	} else {
		t.Logf("EncodeInput: %x", data)
	}

	// DecodeEventSignature (simulate Transfer event)
	sig := []byte{0xa9, 0x05, 0x9c, 0xbb}
	event, err := contract.DecodeEventSignature(sig)
	if err != nil {
		t.Logf("DecodeEventSignature failed (may not exist): %v", err)
	} else {
		t.Logf("DecodeEventSignature: %+v", event)
	}

	// Create client for real data
	c, err := client.NewClient(client.ClientConfig{
		NodeAddress: UtilsNodeEndpoint,
		Timeout:     15 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), c.GetTimeout())
	defer cancel()

	// Get transaction info for real event data
	txInfo, err := c.GetTransactionInfoById(ctx, UtilsTestTxID)
	if err != nil {
		t.Fatalf("Failed to get transaction info: %v", err)
	}

	// Get contract instance from address for event decoding
	contractAddr := types.MustNewAddress("TXJgMdjVX5dKiQaUi9QobwNxtSQaFqccvd")
	contractFromAddr, err := c.NewContractFromAddress(ctx, contractAddr)
	if err != nil {
		t.Fatalf("Failed to create contract from address: %v", err)
	}

	// DecodeEventLog using real transaction data
	if len(txInfo.GetLog()) > 0 {
		log := txInfo.GetLog()[0]
		topics := log.GetTopics()
		dataBytes := log.GetData()

		decodedEvents, err := contractFromAddr.DecodeEventLog(topics, dataBytes)
		if err != nil {
			t.Errorf("DecodeEventLog failed: %v", err)
		} else {
			t.Logf("DecodeEventLog: %+v", decodedEvents)
		}
	}

	// DecodeInputData using real transaction data
	if len(txInfo.GetLog()) > 1 {
		log := txInfo.GetLog()[1]
		dataBytes := log.GetData()

		decodedInput, err := contractFromAddr.DecodeInputData(dataBytes)
		if err != nil {
			t.Errorf("DecodeInputData failed: %v", err)
		} else {
			t.Logf("DecodeInputData: %+v", decodedInput)
		}
	}

	// DecodeResult using real contract calls
	symbolData, err := contract.EncodeInput("symbol")
	if err != nil {
		t.Errorf("Failed to encode symbol input: %v", err)
	} else {
		result, err := c.TriggerConstantSmartContract(ctx, contract, types.MustNewAddress("TQrY8tryqsYVCYS3MFbtffiPp2ccyn4STm"), symbolData)
		if err != nil {
			t.Errorf("TriggerConstantSmartContract failed: %v", err)
		} else {
			decodedResult, err := contract.DecodeResult("symbol", result)
			if err != nil {
				t.Errorf("DecodeResult failed: %v", err)
			} else {
				t.Logf("DecodeResult: %+v", decodedResult)
			}
		}
	}
}
