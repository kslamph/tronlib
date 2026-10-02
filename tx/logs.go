package tx

import (
	"context"
	"encoding/hex"

	"github.com/kslamph/tronlib/v2/event"
	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/rpc"
	"github.com/kslamph/tronlib/v2/tron"
)

// LogsFor fetches the transaction's logs, decoded against the definitions
// registered for each log's emitting contract (global registry as the
// fallback) — unknown or ambiguous signatures materialize with EventName empty
// and raw bytes preserved; never dropped. It performs ONE non-polling
// GetTransactionInfoById fetch on the FullNode endpoint: a transaction that is
// not yet included yields no logs (poll with Wait/WaitForSolid first if
// inclusion is required). It is the free-function entry point the facade's
// Client.Events delegates to.
// Exported because the facade's Events surface delegates to the reviewed
// decode in receipt.go rather than duplicating it.
func LogsFor(ctx context.Context, cp rpc.ConnProvider, txid string) ([]event.Log, error) {
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
