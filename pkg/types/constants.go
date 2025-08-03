// Package types provides shared types and utilities for the TRON SDK
package types

import "time"

// SDK-wide constants

const (
	// Network constants
	TronMainNet = "mainnet"
	TronTestNet = "testnet"
	TronNileNet = "nile"

	// Address constants
	AddressPrefixByte   = 0x41
	AddressLength       = 21
	AddressHexLength    = 42 // Including 0x prefix
	AddressBase58Length = 34

	// Transaction constants
	DefaultFeeLimit           = 1000000 // 1 TRX in SUN
	DefaultTransactionTimeout = 30 * time.Second
	DefaultExpiration         = 10 * time.Minute

	// Energy constants
	DefaultEnergyLimit = 10000000
	EnergyPerByte      = 1

	// Bandwidth constants
	DefaultBandwidthLimit = 5000
	BandwidthPerByte      = 1

	// Resource constants
	SunPerTRX = 1000000 // 1 TRX = 1,000,000 SUN

	// Contract constants
	DefaultContractCallValue = 0
	MaxContractSize          = 65536 // 64KB

	// TRC20 constants
	TRC20TransferMethodID    = "a9059cbb" // transfer(address,uint256)
	TRC20BalanceOfMethodID   = "70a08231" // balanceOf(address)
	TRC20ApproveMethodID     = "095ea7b3" // approve(address,uint256)
	TRC20AllowanceMethodID   = "dd62ed3e" // allowance(address,address)
	TRC20TotalSupplyMethodID = "18160ddd" // totalSupply()
	TRC20NameMethodID        = "06fdde03" // name()
	TRC20SymbolMethodID      = "95d89b41" // symbol()
	TRC20DecimalsMethodID    = "313ce567" // decimals()

	// Block constants
	BlockTimeMS = 3000 // 3 seconds per block

	// Voting constants
	VotesPerTRX = 1 // 1 TRX = 1 vote

	// Freeze constants
	MinFreezeAmount   = 1000000 // 1 TRX minimum
	FreezeMinDuration = 3       // 3 days minimum

	// Permission constants
	OwnerPermissionID   = 0
	WitnessPermissionID = 1
	ActivePermissionID  = 2

	MaxResultSize = 64 // used for bandwidth estimation
)

// Network represents a TRON network
type Network struct {
	Name     string
	ChainID  string
	NodeURLs []string
}

// Predefined networks
var (
	MainNet = Network{
		Name:    TronMainNet,
		ChainID: "0x2b6653dc",
		NodeURLs: []string{
			"grpc.trongrid.io:50051",
			"grpc.shasta.trongrid.io:50051",
		},
	}

	TestNet = Network{
		Name:    TronTestNet,
		ChainID: "0x94a9059e",
		NodeURLs: []string{
			"grpc.shasta.trongrid.io:50051",
		},
	}

	NileNet = Network{
		Name:    TronNileNet,
		ChainID: "0xcd8690dc",
		NodeURLs: []string{
			"grpc.nile.trongrid.io:50051",
		},
	}
)

// GetNetwork returns a predefined network by name
func GetNetwork(name string) *Network {
	switch name {
	case TronMainNet:
		return &MainNet
	case TronTestNet:
		return &TestNet
	case TronNileNet:
		return &NileNet
	default:
		return nil
	}
}

// ResourceType represents the type of resource
type ResourceType int

const (
	ResourceBandwidth ResourceType = iota
	ResourceEnergy
)

// String returns the string representation of ResourceType
func (r ResourceType) String() string {
	switch r {
	case ResourceBandwidth:
		return "BANDWIDTH"
	case ResourceEnergy:
		return "ENERGY"
	default:
		return "UNKNOWN"
	}
}

// ContractType represents the type of smart contract
type ContractType int

const (
	ContractTypeUnknown ContractType = iota
	ContractTypeTRC20
	ContractTypeTRC721
	ContractTypeTRC1155
	ContractTypeCustom
)

// String returns the string representation of ContractType
func (c ContractType) String() string {
	switch c {
	case ContractTypeTRC20:
		return "TRC20"
	case ContractTypeTRC721:
		return "TRC721"
	case ContractTypeTRC1155:
		return "TRC1155"
	case ContractTypeCustom:
		return "CUSTOM"
	default:
		return "UNKNOWN"
	}
}

// TransactionStatus represents the status of a transaction
type TransactionStatus int

const (
	TransactionStatusUnknown TransactionStatus = iota
	TransactionStatusPending
	TransactionStatusConfirmed
	TransactionStatusFailed
	TransactionStatusReverted
)

// String returns the string representation of TransactionStatus
func (s TransactionStatus) String() string {
	switch s {
	case TransactionStatusPending:
		return "PENDING"
	case TransactionStatusConfirmed:
		return "CONFIRMED"
	case TransactionStatusFailed:
		return "FAILED"
	case TransactionStatusReverted:
		return "REVERTED"
	default:
		return "UNKNOWN"
	}
}
