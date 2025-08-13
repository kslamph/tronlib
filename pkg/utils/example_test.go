package utils_test

import (
	"math/big"

	"github.com/kslamph/tronlib/pkg/utils"
)

// ExampleABIProcessor demonstrates encoding a simple method call.
func ExampleABIProcessor() {
	abiJSON := `{"entrys":[{"type":"function","name":"set","inputs":[{"name":"v","type":"uint256"}]}]}`
	abi, _ := utils.NewABIProcessor(nil).ParseABI(abiJSON)
	proc := utils.NewABIProcessor(abi)
	_, _ = proc.EncodeMethod("set", []string{"uint256"}, []interface{}{big.NewInt(1)})
}
