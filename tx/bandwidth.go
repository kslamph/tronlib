package tx

import (
	"context"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/rpc"
	"github.com/kslamph/tronlib/v2/tron"
	"google.golang.org/protobuf/proto"
)

// ResultSizePerContract mirrors java-tron's Constant.MAX_RESULT_SIZE_IN_TX
// (64): the node adds 64 bandwidth bytes per non-shielded contract on top
// of the serialized transaction (BandwidthProcessor.consume).
const ResultSizePerContract = 64

// BandwidthSize returns the bandwidth a signed transaction consumes, in
// bytes: the protobuf size with ret cleared plus ResultSizePerContract per
// contract (java-tron BandwidthProcessor.consume: clearRet().serializedSize
// + 64 per non-shielded contract). Shielded contracts are out of scope
// (spec §13); every builder-produced kind counts its contracts.
//
// Measure the EXACT bytes you will broadcast: the transaction must already
// carry its signatures (~65 bytes each), so an unsigned transaction is
// tx.invalid_argument rather than a silent undercount. The rejection names
// the unsigned size and the per-signature delta, so the failure itself
// teaches the model instead of merely blocking it.
func BandwidthSize(signed *core.Transaction) (int64, error) {
	const op = "tx.BandwidthSize"
	if signed == nil {
		return 0, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op, Hint: "transaction is nil"}
	}
	cleared := proto.Clone(signed).(*core.Transaction)
	cleared.Ret = nil
	size := int64(proto.Size(cleared))
	contracts := int64(len(cleared.GetRawData().GetContract()))
	if contracts == 0 {
		return 0, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op, Hint: "transaction holds no contract message"}
	}
	if size > math.MaxInt64-contracts*ResultSizePerContract {
		return 0, &tron.Error{Code: tron.CodeAmountOverflow, Op: op, Hint: "serialized size plus result overhead overflows int64"}
	}
	size += contracts * ResultSizePerContract
	if len(signed.GetSignature()) == 0 {
		return 0, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op,
			Hint: "transaction is unsigned at " + itoa(size) + " bytes; sign first — each signature adds ~65 bytes and the burn is priced on the broadcast bytes",
		}
	}
	return size, nil
}

// BandwidthPrice is the current bandwidth unit price read from the node's
// governance price history (getTransactionFee): the entry with the greatest
// timestamp in rpc.GetBandwidthPrices' "timestamp:price" comma-list — the
// same shape as the energy price history (java-tron Wallet.getBandwidthPrices
// serves DynamicPropertiesStore.getBandwidthPriceHistory).
type BandwidthPrice struct {
	// SunPerByte is the latest unit price in SUN per bandwidth byte.
	SunPerByte int64
	// EffectiveAt is the timestamp of the price entry that won.
	EffectiveAt time.Time
	// FetchedAt is when the price was read; staleness is explicit.
	FetchedAt time.Time
}

// BandwidthPriceOf fetches the bandwidth price history and returns the
// current price (the latest ts:price entry). A malformed or empty price
// list is contract.bad_metadata — a silent zero price would zero every
// burn prediction.
func BandwidthPriceOf(cp rpc.ConnProvider, ctx context.Context) (*BandwidthPrice, error) {
	const op = "tx.BandwidthPriceOf"
	msg, err := rpc.GetBandwidthPrices(cp, ctx, &api.EmptyMessage{})
	if err != nil {
		return nil, err
	}
	var bestTs, bestPrice int64
	found := false
	for _, entry := range strings.Split(msg.GetPrices(), ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		tsStr, priceStr, ok := strings.Cut(entry, ":")
		if !ok {
			return nil, badBandwidthMetadata(op, msg.GetPrices())
		}
		ts, err1 := strconv.ParseInt(tsStr, 10, 64)
		price, err2 := strconv.ParseInt(priceStr, 10, 64)
		if err1 != nil || err2 != nil {
			return nil, badBandwidthMetadata(op, msg.GetPrices())
		}
		if !found || ts >= bestTs { // latest timestamp wins; ties keep the last entry
			bestTs, bestPrice = ts, price
			found = true
		}
	}
	if !found {
		return nil, badBandwidthMetadata(op, msg.GetPrices())
	}
	return &BandwidthPrice{
		SunPerByte:  bestPrice,
		EffectiveAt: time.UnixMilli(bestTs),
		FetchedAt:   time.Now(),
	}, nil
}

// CostOf converts bandwidth bytes into the SUN burning them costs at this
// price (bytes × SunPerByte). An overflow returns amount.overflow; negative
// bytes return amount.negative.
func (p *BandwidthPrice) CostOf(bytes int64) (tron.SUN, error) {
	if bytes < 0 {
		return 0, &tron.Error{
			Code: tron.CodeAmountNegative,
			Op:   "BandwidthPrice.CostOf",
			Hint: "negative bandwidth has no SUN cost",
		}
	}
	return tron.SUN(bytes).Mul(p.SunPerByte)
}

func badBandwidthMetadata(op, raw string) *tron.Error {
	return &tron.Error{
		Code: tron.CodeContractBadMetadata,
		Op:   op,
		Hint: `GetBandwidthPrices returned a malformed "timestamp:price" list; cannot price bandwidth`,
	}
}

// BandwidthCost predicts what broadcasting a signed transaction will cost
// the owner in bandwidth Libraries (spec §7.3 limitation 1), following the
// node's charging order exactly (java-tron BandwidthProcessor.consume):
//
//  1. BytesNeeded = BandwidthSize of the signed transaction.
//  2. Staked bandwidth first (NetLimit−NetUsed), then the free quota
//     (FreeNetLimit−FreeNetUsed) — both read decayed-to-now via
//     GetAccountResource, so the prediction is a CEILING: usage only
//     decays, hence availability only grows toward broadcast time.
//  3. Any shortfall burns TRX at SunPerByte (getTransactionFee).
//
// Account creation (a transfer or asset transfer to an address with no
// account — detected by type-checked decode of the transfer contract,
// the F1 rule) changes step 2: staked bandwidth is tried with the
// getCreateNewAccountBandwidthRate multiple, else a flat getCreateAccountFee
// (0.1 TRX) burns; and NewAccountFee (getCreateNewAccountFeeInSystemContract,
// 1 TRX) burns on top, invisibly to the receipt. Contract-internal creation
// carries no 1 TRX fee — its 25,000 energy is inside Simulate — so
// ContractTx/DeployTx never take the creation branch.
//
// When a burn or fee is predicted, the owner's balance is read and a
// shortfall returns account.insufficient_bandwidth (ActionFund): the node
// rejects a transaction whose burn exceeds its balance, so a number without
// that check would be fiction.
type BandwidthCost struct {
	// BytesNeeded is the bandwidth the transaction consumes.
	BytesNeeded int64
	// StakedAvailable is the owner's staked bandwidth (NetLimit−NetUsed),
	// reported as computed; negative means already over the limit.
	StakedAvailable int64
	// FreeAvailable is the owner's free quota (FreeNetLimit−FreeNetUsed).
	FreeAvailable int64
	// ToBurn is max(0, BytesNeeded − StakedAvailable − FreeAvailable): the
	// bytes that will burn TRX. Zero when covered.
	ToBurn int64
	// Burn is ToBurn × SunPerByte. Zero when covered.
	Burn tron.SUN
	// SunPerByte is the unit price used.
	SunPerByte int64
	// PricedAt is when the price was read; staleness is explicit.
	PricedAt time.Time
	// CreatesAccount reports whether the transaction creates its recipient.
	CreatesAccount bool
	// NewAccountFee is the 1 TRX creation burn (0 otherwise). It never
	// appears in the receipt — it burns in the transfer actuator — so
	// receipt.NetFee covers Burn (or the flat creation fee) only.
	NewAccountFee tron.SUN
	// NetUsage is what the receipt will report: BytesNeeded (or the
	// ratio-scaled need on the creation-staked path) when covered, 0 on
	// any burn path. Compare it — and NetFee against Burn — exactly.
	NetUsage int64
}

// String renders the prediction as one line.
func (c *BandwidthCost) String() string {
	burnNote := "to burn " + itoa(c.ToBurn) + " @ " + itoa(c.SunPerByte) + " sun/byte = " + c.Burn.String() + " sun"
	if c.CreatesAccount && c.ToBurn == 0 && c.Burn > 0 {
		// Fee branch: Burn is the flat creation fee, not a per-byte burn.
		burnNote = "flat creation fee = " + c.Burn.String() + " sun (no per-byte burn)"
	}
	return strings.Join([]string{
		"bandwidth preview: need " + itoa(c.BytesNeeded) +
			" (staked " + itoa(c.StakedAvailable) + " + free " + itoa(c.FreeAvailable) + ")",
		burnNote,
		"new-account fee " + c.NewAccountFee.String() + " sun",
		"priced " + c.PricedAt.Format(time.RFC3339),
	}, "; ")
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

// BandwidthCostOf predicts the bandwidth cost of broadcasting t for owner.
// t must be signed (BandwidthSize); owner is the account whose staked/free
// resources are counted. The read sequence: account resources, bandwidth
// price, then — only on the creation path — chain parameters and the
// recipient account, then — only when a burn or fee is predicted — the
// owner's balance.
func BandwidthCostOf(cp rpc.ConnProvider, ctx context.Context, t Tx, owner tron.Address) (*BandwidthCost, error) {
	const op = "tx.BandwidthCostOf"
	if cp == nil {
		return nil, &tron.Error{Code: tron.CodeChainConnection, Op: op, Hint: "cp is nil; pass a connected *rpc.Client"}
	}
	if t == nil {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op, Hint: "transaction is nil"}
	}
	if owner.IsZero() {
		return nil, &tron.Error{Code: tron.CodeAddressInvalid, Op: op, Hint: "owner address is unset; parse it with tron.ParseAddress"}
	}
	need, err := BandwidthSize(t.Transaction())
	if err != nil {
		return nil, err
	}
	res, err := rpc.GetAccountResource(cp, ctx, &core.Account{Address: owner.Bytes()})
	if err != nil {
		return nil, err
	}
	price, err := BandwidthPriceOf(cp, ctx)
	if err != nil {
		return nil, err
	}
	cost := &BandwidthCost{
		BytesNeeded:     need,
		StakedAvailable: res.GetNetLimit() - res.GetNetUsed(),
		FreeAvailable:   res.GetFreeNetLimit() - res.GetFreeNetUsed(),
		SunPerByte:      price.SunPerByte,
		PricedAt:        price.FetchedAt,
		NetUsage:        need,
	}
	toBurn := need - cost.StakedAvailable - cost.FreeAvailable
	if toBurn < 0 {
		toBurn = 0
	}

	// Creation branch: transfers to an address with no account.
	if to, isTransfer := transferRecipient(t); isTransfer {
		creates, err := recipientMissing(cp, ctx, op, to)
		if err != nil {
			return nil, err
		}
		if creates {
			return creationCost(cp, ctx, op, cost, owner)
		}
	}

	cost.ToBurn = toBurn
	if toBurn == 0 {
		return cost, nil
	}
	burn, err := price.CostOf(toBurn)
	if err != nil {
		return nil, &tron.Error{Code: tron.CodeAmountOverflow, Op: op, Cause: err,
			Hint: "ToBurn × SunPerByte overflows SUN; the call cannot be priced",
		}
	}
	cost.Burn = burn
	cost.NetUsage = 0 // the node reports NetUsage 0 on the burn path
	return requireBandwidthBalance(cp, ctx, op, owner, cost, int64(burn))
}

// transferRecipient returns the recipient of a transfer-shaped transaction
// (NativeTx = TransferContract, AssetTx = TransferAssetContract), decoded
// after a contract-type check — the F1 rule. Other kinds carry no
// determinable recipient (contract-internal creation has no 1 TRX fee),
// reported as isTransfer == false.
func transferRecipient(t Tx) (to tron.Address, isTransfer bool) {
	var want core.Transaction_Contract_ContractType
	switch t.(type) {
	case *NativeTx:
		want = core.Transaction_Contract_TransferContract
	case *AssetTx:
		want = core.Transaction_Contract_TransferAssetContract
	default:
		return tron.Address{}, false
	}
	contracts := t.Transaction().GetRawData().GetContract()
	if len(contracts) != 1 || contracts[0].GetType() != want {
		return tron.Address{}, false
	}
	raw := contracts[0].GetParameter().GetValue()
	if want == core.Transaction_Contract_TransferContract {
		var c core.TransferContract
		if err := proto.Unmarshal(raw, &c); err != nil {
			return tron.Address{}, false
		}
		a, err := tron.AddressFromBytes(c.GetToAddress())
		if err != nil {
			return tron.Address{}, false
		}
		return a, true
	}
	var c core.TransferAssetContract
	if err := proto.Unmarshal(raw, &c); err != nil {
		return tron.Address{}, false
	}
	a, err := tron.AddressFromBytes(c.GetToAddress())
	if err != nil {
		return tron.Address{}, false
	}
	return a, true
}

// recipientMissing reports whether no account exists at to. A missing
// account arrives as an empty message over gRPC (the node returns null),
// and every stored account carries its address — so an empty address is
// the missing signal.
func recipientMissing(cp rpc.ConnProvider, ctx context.Context, op string, to tron.Address) (bool, error) {
	acct, err := rpc.GetAccount(cp, ctx, &core.Account{Address: to.Bytes()})
	if err != nil {
		return false, err
	}
	return len(acct.GetAddress()) == 0, nil
}

// creationCost finishes a BandwidthCost on the account-creation path:
// staked bandwidth tried with the getCreateNewAccountBandwidthRate
// multiple, else the flat getCreateAccountFee burns; the
// getCreateNewAccountFeeInSystemContract fee burns on top in all cases.
func creationCost(cp rpc.ConnProvider, ctx context.Context, op string, cost *BandwidthCost, owner tron.Address) (*BandwidthCost, error) {
	params, err := chainParamMap(cp, ctx, op,
		"getCreateNewAccountFeeInSystemContract", "getCreateAccountFee", "getCreateNewAccountBandwidthRate")
	if err != nil {
		return nil, err
	}
	newAccountFee, createFee, ratio := params["getCreateNewAccountFeeInSystemContract"], params["getCreateAccountFee"], params["getCreateNewAccountBandwidthRate"]
	if ratio < 0 || newAccountFee < 0 || createFee < 0 {
		return nil, &tron.Error{Code: tron.CodeContractBadMetadata, Op: op, Hint: "a creation chain parameter is negative; the node answered outside its documented range"}
	}
	cost.CreatesAccount = true
	cost.NewAccountFee = tron.SUN(newAccountFee)
	// Staked-with-ratio path: need×ratio covered by staked bandwidth.
	scaled := cost.BytesNeeded
	if ratio > 0 {
		if cost.BytesNeeded > math.MaxInt64/ratio {
			return nil, &tron.Error{Code: tron.CodeAmountOverflow, Op: op, Hint: "BytesNeeded × bandwidth rate overflows int64"}
		}
		scaled = cost.BytesNeeded * ratio
	}
	if scaled <= cost.StakedAvailable {
		cost.NetUsage = scaled // the receipt reports the ratio-scaled usage
		return requireBandwidthBalance(cp, ctx, op, owner, cost, 0)
	}
	// Fee path: flat creation fee burns as NetFee.
	cost.ToBurn = 0
	cost.Burn = tron.SUN(createFee)
	cost.NetUsage = 0
	return requireBandwidthBalance(cp, ctx, op, owner, cost, createFee)
}

// requireBandwidthBalance checks the owner can cover the predicted SUN
// outlay (bandwidth burn plus any creation fees). A shortfall is
// account.insufficient_bandwidth — the node rejects the transaction, so a
// bare number would be fiction. The balance is read lazily: covered
// predictions cost no extra RPC.
func requireBandwidthBalance(cp rpc.ConnProvider, ctx context.Context, op string, owner tron.Address, cost *BandwidthCost, outlay int64) (*BandwidthCost, error) {
	if outlay <= 0 && cost.NewAccountFee <= 0 {
		return cost, nil
	}
	acct, err := rpc.GetAccount(cp, ctx, &core.Account{Address: owner.Bytes()})
	if err != nil {
		return nil, err
	}
	total := outlay + int64(cost.NewAccountFee)
	if total < 0 || acct.GetBalance() < total {
		return nil, &tron.Error{
			Code: tron.CodeAccountInsufficientBandwidth,
			Op:   op,
			Next: tron.ActionFund,
			Hint: "owner balance covers neither the predicted bandwidth burn nor the creation fee; fund the account first",
		}
	}
	return cost, nil
}

// chainParamMap reads the named governance parameters into a map. A missing
// key is contract.bad_metadata: required parameters the node did not answer.
func chainParamMap(cp rpc.ConnProvider, ctx context.Context, op string, keys ...string) (map[string]int64, error) {
	msg, err := rpc.GetChainParameters(cp, ctx, &api.EmptyMessage{})
	if err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(keys))
	for _, p := range msg.GetChainParameter() {
		out[p.GetKey()] = p.GetValue()
	}
	for _, k := range keys {
		if _, ok := out[k]; !ok {
			return nil, &tron.Error{Code: tron.CodeContractBadMetadata, Op: op,
				Hint: "GetChainParameters did not answer required key " + k,
			}
		}
	}
	return out, nil
}
