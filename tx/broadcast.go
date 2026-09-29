package tx

import (
	"context"
	"encoding/hex"
	"errors"
	"time"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/rpc"
	"github.com/kslamph/tronlib/v2/tron"
)

// pollInterval is the polling cadence of Wait/WaitForSolid. The first poll
// happens immediately, so a transaction that is already included returns
// without waiting a full interval.
const pollInterval = 500 * time.Millisecond

// Broadcast submits a signed transaction to the node and returns a Receipt.
//
// Pre-flight checks (the port of v1's broadcast flow): exactly one contract
// message (tx.invalid_argument), expiration in the future (tx.expired), and
// at least one signature (tx.no_signer).
//
// The node's broadcast Return is preserved as a Receipt, not an error: a
// node-level rejection (SIGERROR, DUP_TRANSACTION_ERROR, …) is an answer
// about the transaction, mapped onto the v2 code with the raw api.Return_*
// name in NodeCode (spec §7.4 — the three codes have different remedies).
// r.OK() reports success. Transport failures still return *tron.Error.
//
// THE DOUBLE-SPEND FIX (spec §6.4/§6.5): on an ambiguous chain.timeout from
// the broadcast, Broadcast performs ONE reconciliation poll
// (GetTransactionInfoById). If the transaction was found, the real receipt is
// returned. If not, the error is chain.unconfirmed with the txid populated
// and Next = ActionWait, with the rule the agent must learn in the Hint:
// poll Wait(txid); do NOT rebuild and resign — resending identical bytes is
// deduplicated (a txid is a pure function of raw_data), but rebuilding gets
// a new TAPOS reference and a new txid, and THAT spends twice.
func Broadcast(cp rpc.ConnProvider, ctx context.Context, t Tx) (*Receipt, error) {
	const op = "tx.Broadcast"
	if t == nil {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op, Hint: "transaction is nil"}
	}
	raw := t.Transaction().GetRawData()
	if raw == nil {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op, Hint: "transaction has no raw data; build it with the tx.Build* functions"}
	}
	if err := requireOneContract(raw, op); err != nil {
		return nil, err
	}
	if raw.GetExpiration() < time.Now().UnixMilli() {
		return nil, &tron.Error{
			Code: tron.CodeTxExpired,
			Op:   op,
			Hint: "expiration has passed; rebuild and re-sign, or use WithExpiration for longer multi-signer circulation",
		}
	}
	if !t.IsSigned() {
		return nil, &tron.Error{
			Code: tron.CodeTxNoSigner,
			Op:   op,
			Hint: "call Sign(...) with at least one key.Signer before Broadcast",
		}
	}
	txid := t.ID()

	ret, err := rpc.BroadcastTransaction(cp, ctx, t.Transaction())
	if err != nil {
		if tron.HasCode(err, tron.CodeChainTimeout) {
			// Ambiguous: the broadcast may or may not have landed. One
			// reconciliation poll, then the wait-and-do-not-resend answer.
			return reconcileAfterBroadcast(cp, ctx, op, txid)
		}
		return nil, err
	}

	if !ret.GetResult() {
		// The node spoke about the transaction: a receipt, not an error.
		return &Receipt{
			TxID:     txid,
			Code:     mappedReturnCode(ret, op),
			NodeCode: ret.GetCode().String(),
			Revert:   string(ret.GetMessage()),
		}, nil
	}
	return &Receipt{TxID: txid, NodeCode: api.Return_SUCCESS.String()}, nil
}

// reconcileAfterBroadcast is Broadcast's single reconciliation poll after an
// ambiguous timeout (spec §6.4).
func reconcileAfterBroadcast(cp rpc.ConnProvider, ctx context.Context, op, txid string) (*Receipt, error) {
	id, _ := hex.DecodeString(txid)
	info, err := rpc.GetTransactionInfoById(cp, ctx, &api.BytesMessage{Value: id})
	if err == nil && len(info.GetId()) > 0 {
		return receiptFromInfo(info, false), nil
	}
	return nil, &tron.Error{
		Code: tron.CodeChainUnconfirmed,
		Op:   op,
		TxID: txid,
		Next: tron.ActionWait,
		Hint: "poll Wait(txid) — do NOT rebuild and resign; resending identical bytes is deduplicated, rebuilding spends twice",
	}
}

// Wait polls the FullNode receipt (GetTransactionInfoById) until the
// transaction is included and executed, and returns the parsed Receipt.
// Inclusion is not finality: TRON reaches practical finality when a block is
// solidified (~a minute) — use WaitForSolid for custody or deposit-crediting
// semantics. Receipt.Solidified() is false here.
//
// Poll errors are transient (v1 semantics): a failed poll is retried until
// the context is done, which surfaces as chain.timeout.
func Wait(cp rpc.ConnProvider, ctx context.Context, txid string) (*Receipt, error) {
	return pollReceipt(cp, ctx, "tx.Wait", txid, false)
}

// WaitForSolid polls the Solidity endpoint (GetTransactionInfoById on
// WalletSolidity) until the transaction appears there — solidified
// semantics, the finality-aware variant of Wait. A receipt from the solidity
// node reports Solidified() == true.
func WaitForSolid(cp rpc.ConnProvider, ctx context.Context, txid string) (*Receipt, error) {
	return pollReceipt(cp, ctx, "tx.WaitForSolid", txid, true)
}

// pollReceipt is the shared polling loop of Wait/WaitForSolid.
func pollReceipt(cp rpc.ConnProvider, ctx context.Context, op, txid string, solid bool) (*Receipt, error) {
	id, err := hex.DecodeString(txid)
	if err != nil || len(id) == 0 {
		return nil, &tron.Error{
			Code: tron.CodeTxInvalidArgument,
			Op:   op,
			Hint: "txid must be the 64-hex-character transaction id (tx.ID())",
		}
	}
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		info, err := getTxInfo(cp, ctx, solid, id)
		if err == nil && len(info.GetId()) > 0 {
			return receiptFromInfo(info, solid), nil
		}
		select {
		case <-ctx.Done():
			return nil, &tron.Error{Code: tron.CodeChainTimeout, Op: op, Cause: ctx.Err()}
		case <-ticker.C:
		}
	}
}

// getTxInfo fetches the transaction info from the FullNode or Solidity
// endpoint depending on solid.
func getTxInfo(cp rpc.ConnProvider, ctx context.Context, solid bool, id []byte) (*core.TransactionInfo, error) {
	req := &api.BytesMessage{Value: id}
	if solid {
		return rpc.GetTransactionInfoByIdSolidity(cp, ctx, req)
	}
	return rpc.GetTransactionInfoById(cp, ctx, req)
}

// mappedReturnCode maps a failed node api.Return onto the v2 code by
// reusing rpc's reviewed mapping (rpc.ValidateTransactionResult) — one
// table, no duplicated 14-row switch to edit in lockstep (the drift class
// the Task 4 review flagged). The numeric code stays recoverable via
// rpc.NodeReturnCode on the produced error if a caller needs it.
func mappedReturnCode(ret *api.Return, op string) tron.Code {
	err := rpc.ValidateTransactionResult(&api.TransactionExtention{Result: ret}, op)
	var te *tron.Error
	if errors.As(err, &te) {
		return te.Code
	}
	return tron.CodeRPCMethodFailed
}
