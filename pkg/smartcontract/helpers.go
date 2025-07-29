package smartcontract

import (
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/utils"
)

// DecodeABI is a helper function for backward compatibility
// It wraps the utils.ABIParser.ParseABI method
func DecodeABI(abi string) (*core.SmartContract_ABI, error) {
	processor := utils.NewABIProcessor(nil)
	return processor.ParseABI(abi)
}
