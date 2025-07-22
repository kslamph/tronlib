package helper

import (
	"encoding/hex"

	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/smartcontract"
)

type TransactionEvent struct {
	BlockNumber     uint64
	BlockTimestamp  uint64
	TransactionHash string
	ContractAddress string
	Event           *smartcontract.DecodedEvent
}

func ContractsSliceToMap(contracts []*smartcontract.Contract) map[[20]byte]*smartcontract.Contract {
	contractsMap := make(map[[20]byte]*smartcontract.Contract)
	for _, contract := range contracts {
		addrArray := [20]byte{}
		copy(addrArray[:], contract.AddressBytes[1:21])
		contractsMap[addrArray] = contract
	}
	return contractsMap
}

func ParseTransactionInfoLog(transactionInfo *core.TransactionInfo, contracts map[[20]byte]*smartcontract.Contract) []*TransactionEvent {
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
