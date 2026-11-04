package contract

// Revert classification for constant calls. A TRON node answers a
// constant call that hit require()/revert()/panic() with
// Return{result: true, message: "REVERT opcode executed"} and the
// Solidity revert payload in ConstantResult — Error(string) (selector
// 08c379a0) for require/revert, Panic(uint256) (selector 4e487b71) for
// panic/assert. ValidateTransactionResult sees result=true and passes;
// without classification the payload falls through to the ABI output
// decode and surfaces as a bogus contract.arg_mismatch (or, for
// permissive output shapes, silently misdecodes). The inner
// Transaction.Ret code is FAILED on trongrid (not REVERT), so it is not
// a usable signal on the observed node; classification therefore requires
// both the node's revert marker and a valid payload. Payload bytes alone are
// not enough because a successful method may legitimately return them.

import (
	"fmt"

	eABI "github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/kslamph/tronlib/v2/tron"
)

// classifyRevertBlob returns a contract.reverted *tron.Error only when the
// node response says that the VM executed a revert and the payload decodes as
// Solidity Error(string) or Panic(uint256). The marker is essential: a
// successful method is allowed to return bytes that happen to be a valid
// revert payload. Offline Decode has no node marker and therefore must not
// call this helper.
func classifyRevertBlob(op string, nodeReverted bool, data []byte) error {
	if !nodeReverted {
		return nil
	}
	reason, err := eABI.UnpackRevert(data)
	if err != nil {
		return nil
	}
	return &tron.Error{
		Code: tron.CodeContractReverted,
		Op:   op,
		Hint: fmt.Sprintf("the contract reverted the call: %s", reason),
	}
}
