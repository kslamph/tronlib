package crypto

import (
	"fmt"
	"log"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/kslamph/tronlib/pkg/types"
)

const (
	TronMessagePrefix = "\x19TRON Signed Message:\n"
)

// VerifyMessageV2 verifies a message signature using TIP-191 format
func VerifyMessageV2(address string, message string, hexSignature string) (bool, error) {

	addr, err := types.NewAddress(address)
	if err != nil {
		return false, fmt.Errorf("invalid address: %w", err)
	}

	// 1. Prepare the message.
	var data []byte
	if strings.HasPrefix(message, "0x") {
		data = common.FromHex(message)
	} else {
		data = []byte(message)
	}

	// 2. Prefix the message.
	messageLen := len(data)
	prefixedMessage := []byte(fmt.Sprintf("%s%d%s", TronMessagePrefix, messageLen, string(data)))

	// 3. Hash the prefixed message.
	hash := crypto.Keccak256Hash(prefixedMessage)

	// 4. Decode the signature.
	signature := common.FromHex(strings.TrimPrefix(hexSignature, "0x"))
	if len(signature) != 65 {
		return false, fmt.Errorf("invalid signature length: %d", len(signature))
	}

	// 5. Adjust the recovery ID (v) back to 0 or 1.
	signature[64] -= 27

	// 6. Recover the public key.
	recoveredPublicKey, err := crypto.SigToPub(hash.Bytes(), signature)
	if err != nil {
		return false, fmt.Errorf("failed to recover public key: %w", err)
	}

	// 7. Get the address from the recovered public key.
	recoveredAddress := crypto.PubkeyToAddress(*recoveredPublicKey)

	log.Println("recoveredAddress", mustEthhexToTronhex(recoveredAddress.Hex()))
	log.Println("hexAddr", addr.Hex())
	// 8. Compare the recovered address with the provided address.
	return strings.EqualFold(mustEthhexToTronhex(recoveredAddress.Hex()), addr.Hex()), nil
}

func mustEthhexToTronhex(ethhex string) string {
	//convert to small case and remove 0x prefix
	ethhex = strings.TrimPrefix(strings.ToLower(ethhex), "0x")
	//prefix with 41
	tronhex := "41" + ethhex
	return tronhex
}
