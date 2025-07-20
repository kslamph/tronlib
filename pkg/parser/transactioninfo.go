package parser

import (
	"fmt"

	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
)

func ContractsSliceToMap(contracts []*types.Contract) map[[20]byte]*types.Contract {
	contractsMap := make(map[[20]byte]*types.Contract)
	for _, contract := range contracts {
		addrArray := [20]byte{}
		copy(addrArray[:], contract.AddressBytes[1:21])
		contractsMap[addrArray] = contract
	}
	return contractsMap
}

func ParseTransactionInfoLog(transactionInfo *core.TransactionInfo, contracts map[[20]byte]*types.Contract) []*types.DecodedEvent {
	var decodedEvents []*types.DecodedEvent
	log := transactionInfo.GetLog()

	for _, log := range log {

		addrArray := [20]byte{}
		copy(addrArray[:], log.GetAddress()[0:20])
		fmt.Printf("%x\n", addrArray)

		contract, ok := contracts[addrArray]
		if !ok {
			continue
		}
		topics := log.GetTopics()
		data := log.GetData()

		// Decode event log using the new DecodeEventLog method
		decodedEvent, err := contract.DecodeEventLog(topics, data)
		if err != nil {
			// If event decoding fails, try function decoding as fallback
			if len(topics) > 0 {
				contract.DecodeInputData(topics[0])
			}
			continue
		}

		// Process the decoded event
		decodedEvents = append(decodedEvents, decodedEvent)
	}
	return decodedEvents
}
