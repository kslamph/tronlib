package tx

import (
	"context"
	"encoding/hex"

	"github.com/kslamph/tronlib/v2/event"
	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/rpc"
	"github.com/kslamph/tronlib/v2/tron"
)

// LogsFor fetches the transaction's logs, decoded leniently — unknown
// signatures materialize with EventName empty and raw bytes preserved; never
// dropped. It performs ONE non-polling GetTransactionInfoById fetch on the
// FullNode endpoint: a transaction that is not yet included yields no logs
// (poll with Wait/WaitForSolid first if inclusion is required). It is the
// free-function entry point the facade's Client.Events delegates to.
// Exported per Task 9 controller ruling (D3 precedent, D1 class): the
// facade's spec §10 Events surface delegates to the reviewed decode in
// receipt.go rather than duplicating it.
func LogsFor(cp rpc.ConnProvider, ctx context.Context, txid string) ([]event.Log, error) {
	const op = "tx.LogsFor"
	id, err := hex.DecodeString(txid)
	if err != nil || len(id) == 0 {
		return nil, &tron.Error{
			Code: tron.CodeTxInvalidArgument,
			Op:   op,
			Hint: "txid must be the 64-hex-character transaction id (tx.ID())",
		}
	}
	info, err := rpc.GetTransactionInfoById(cp, ctx, &api.BytesMessage{Value: id})
	if err != nil {
		return nil, err
	}
	var logs []event.Log
	for _, l := range info.GetLog() {
		logs = append(logs, decodeReceiptLog(l))
	}
	return logs, nil
}
