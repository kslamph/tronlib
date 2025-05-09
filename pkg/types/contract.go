package types

// SmartContractClient defines the interface for smart contract interactions
type SmartContractClient interface {
	TriggerConstantSmartContract(contract interface{}, ownerAddress *Address, data []byte) ([][]byte, error)
}
