package parser

import (
	"encoding/hex"

	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
)

type TransactionEvent struct {
	BlockNumber     uint64
	BlockTimestamp  uint64
	TransactionHash string
	ContractAddress string
	Event           *types.DecodedEvent
}

func ContractsSliceToMap(contracts []*types.Contract) map[[20]byte]*types.Contract {
	contractsMap := make(map[[20]byte]*types.Contract)
	for _, contract := range contracts {
		addrArray := [20]byte{}
		copy(addrArray[:], contract.AddressBytes[1:21])
		contractsMap[addrArray] = contract
	}
	return contractsMap
}

func ParseTransactionInfoLog(transactionInfo *core.TransactionInfo, contracts map[[20]byte]*types.Contract) []*TransactionEvent {
	var decodedEvents []*TransactionEvent
	log := transactionInfo.GetLog()

	for _, log := range log {

		addrArray := [20]byte{}
		copy(addrArray[:], log.GetAddress()[0:20])

		contract, ok := contracts[addrArray]
		if !ok {
			continue
		}

		decodedEvent, err := contract.DecodeEventLog(log.GetTopics(), log.GetData())
		if err != nil {
			continue
		}

		// Create TransactionEvent struct with all the required fields
		transactionEvent := &TransactionEvent{
			BlockNumber:     uint64(transactionInfo.GetBlockNumber()),
			BlockTimestamp:  uint64(transactionInfo.GetBlockTimeStamp()),
			TransactionHash: hex.EncodeToString(transactionInfo.GetId()),
			ContractAddress: contract.Address,
			Event:           decodedEvent,
		}

		decodedEvents = append(decodedEvents, transactionEvent)
	}
	return decodedEvents
}
