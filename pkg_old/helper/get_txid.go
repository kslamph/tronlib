package helper

import (
	"crypto/sha256"
	"fmt"

	"github.com/kslamph/tronlib/pb/core"
	"google.golang.org/protobuf/proto"
)

func GetTxid(tx *core.Transaction) string {
	rawData, err := proto.Marshal(tx.GetRawData())
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%x", sha256.Sum256(rawData))
}
