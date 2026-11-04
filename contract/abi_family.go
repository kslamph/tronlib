package contract

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/kslamph/tronlib/v2/pb/core"
)

// abiFamilies holds validated Solidity ABI JSON keyed by a deployed contract's
// code identity. It lets factory-created instances reuse one ABI when the
// node did not publish ABI entries for each clone.
var abiFamilies = struct {
	sync.RWMutex
	byCode map[string]string
}{byCode: make(map[string]string)}

func codeIdentity(sc *core.SmartContract) (string, error) {
	if sc == nil {
		return "", fmt.Errorf("smart contract is nil")
	}
	if hash := sc.GetCodeHash(); len(hash) != 0 {
		return "codehash:" + hex.EncodeToString(hash), nil
	}
	if bytecode := sc.GetBytecode(); len(bytecode) != 0 {
		sum := sha256.Sum256(bytecode)
		return "bytecode:" + hex.EncodeToString(sum[:]), nil
	}
	return "", fmt.Errorf("the node returned no code hash or bytecode")
}

func lookupABIFamily(identity string) (string, bool) {
	abiFamilies.RLock()
	defer abiFamilies.RUnlock()
	abiJSON, ok := abiFamilies.byCode[identity]
	return abiJSON, ok
}

func storeABIFamily(identity, abiJSON string) {
	abiFamilies.Lock()
	defer abiFamilies.Unlock()
	abiFamilies.byCode[identity] = abiJSON
}
