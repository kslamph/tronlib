package tx

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/rpc"
	"github.com/kslamph/tronlib/v2/tron"
)

// CostPreview predicts what broadcasting a ContractTx will cost the owner in
// SUN, combining four read-only node answers (architecture §7.3):
//
//  1. ContractTx.Simulate — the accurate ENERGY ESTIMATOR
//     (TriggerConstantContract.EnergyUsed) AND the revert check.
//     Simulate.Energy (Estimate.Energy) is the energy the call actually
//     consumes during a deterministic dry run — live-verified to match the
//     post-broadcast ResourceReceipt.EnergyUsageTotal exactly (architecture §7.5).
//     If Simulate errors, CostPreview returns the error: a cost prediction
//     for a call that cannot run would be noise.
//     Simulate.Penalty supplies the TIP-491 penalty split (when > 0).
//
//  2. rpc.GetAccountResource (owner) — the owner's staked energy
//     (EnergyLimit−EnergyUsed → EnergyAvailable) AND the staked/free
//     bandwidth (Net/FreeNet limits minus usage → Bandwidth half), one read
//     feeding both halves.
//
//  3. EnergyPriceOf — the current SunPerEnergy (the network governance
//     parameter that sets the energy→SUN burn ratio, independent of any
//     specific contract or transaction).
//
//  4. BandwidthPriceOf — the current SunPerByte (getTransactionFee's
//     history), pricing the bandwidth shortfall the same way
//     BandwidthCostOf prices it on a signed transaction.
//
// Energy estimator vs Energy burn calculator:
//
//	The energy estimator (Simulate) returns the accurate energy units the
//	call will consume.  The burn calculator (EnergyPrice.CostOf) converts
//	energy that must be purchased into SUN at the network's current
//	SunPerEnergy ratio — a property of the TRON network's operating
//	parameters, not of any specific contract or transaction (see the design
//	separation in §7.3: EnergyPrice.CostOf is the pure batching primitive).
//
//	CostPreview combines both roles: it feeds the estimator's accurate
//	EnergyNeeded through the burn calculator to produce TronToBurn.
//
// TronToBurn = max(0, EnergyNeeded − EnergyAvailable) converted to SUN at
// the network's current SunPerEnergy price via the checked multiply in
// EnergyPrice.CostOf.  An overflow returns amount.overflow.
//
// PricedAt timestamps the price read — the preview is a FLOOR, not a
// ceiling: the price is a governance parameter and energy prices only ever
// move in the caller's favor at the margins between preview and broadcast.
//
// Fee-limit floor-check (architecture §6.4): PreviewCost returns tx.fee_limit_too_low
// when TronToBurn exceeds the transaction's fee_limit — the 150-TRX default
// is a floor that is checked, not trusted.
//
// live-verified: §7.5 item 2-3 (energy matches actual receipt), corrected
// from the prior EstimateEnergy RPC to the accurate Simulate.Energy source.
type CostPreview struct {
	// EnergyNeeded is the total energy the call is expected to consume — the
	// accurate dry-run EnergyUsed from TriggerConstantContract (Simulate.Energy),
	// live-verified to match the post-execution EnergyUsageTotal exactly.
	EnergyNeeded int64
	// EnergyBase is EnergyNeeded − EnergyPenalty: the call's own consumption.
	EnergyBase int64
	// EnergyPenalty is the TIP-491 dynamic-model surcharge (from Simulate).
	EnergyPenalty int64
	// EnergyAvailable is the owner's staked energy (EnergyLimit−EnergyUsed).
	// It is reported as computed; a negative value means the account is
	// already over its limit.
	EnergyAvailable int64
	// EnergyToBuy is max(0, EnergyNeeded − EnergyAvailable): the energy that
	// will be purchased with TRX.
	EnergyToBuy int64
	// TronToBurn is EnergyToBuy × the latest unit price, in SUN.
	TronToBurn tron.SUN
	// SunPerEnergy is the latest unit price the preview used.
	SunPerEnergy int64
	// PricedAt is when the price was read; staleness is explicit.
	PricedAt time.Time
	// Bandwidth predicts the bandwidth half of the broadcast, on the same
	// charging model BandwidthCostOf applies to a signed transaction:
	// staked bandwidth first, then the free quota, any shortfall burning at
	// SunPerByte. BytesNeeded is measured on a ONE-SIGNATURE estimate of the
	// broadcast bytes (the preview runs before signing); a predicted burn
	// the owner cannot cover is account.insufficient_bandwidth — a number
	// the node would refuse is not reported as a prediction.
	Bandwidth *BandwidthCost
	// TotalFloor is TronToBurn + Bandwidth.Burn: the all-in floor. Both
	// halves under-state at worst — the energy price is a governance value,
	// and bandwidth availability only grows between the read and the
	// broadcast — while extra signatures beyond the first add ~67 bytes
	// each. It is a floor, not a total: it carries no governance fees
	// (TotalCostOf on the signed transaction is the all-in answer).
	TotalFloor tron.SUN
	// BandwidthNote states the preview's estimation caveat (see
	// BandwidthEstimateNote): the bandwidth half is priced on a
	// single-signature estimate of the broadcast bytes.
	BandwidthNote string
}

// BandwidthEstimateNote is the CostPreview.BandwidthNote value: the preview
// runs before signing, so its bandwidth is priced on a one-signature
// estimate of the broadcast bytes. (Recipient activation is not a
// CostPreview concern: a contract call's activation cost lands inside
// Simulate's energy, and the transfer kinds that can create accounts take
// BandwidthCostOf's creation branch instead.)
const BandwidthEstimateNote = "bandwidth priced on a single-signature estimate; each extra signature adds ~67 bytes"

// signatureBytes is the secp256k1 signature length a broadcast carries per
// signer (R||S||V, 65 bytes). In the repeated-bytes signature field each
// entry encodes as tag + length + 65 ≈ 67 bytes of serialized size.
const signatureBytes = 65

// String renders the preview as one line, including the bandwidth half and
// the total floor so a logged preview reads as the estimate it is.
func (c *CostPreview) String() string {
	bandwidth := "bandwidth: not computed"
	if c.Bandwidth != nil {
		bandwidth = c.Bandwidth.String()
	}
	return fmt.Sprintf(
		"cost preview: energy needed %d (base %d + penalty %d), available %d, to buy %d @ %d sun/energy = %s sun; %s; total floor %s sun (priced %s; %s)",
		c.EnergyNeeded, c.EnergyBase, c.EnergyPenalty, c.EnergyAvailable,
		c.EnergyToBuy, c.SunPerEnergy, c.TronToBurn,
		bandwidth, c.TotalFloor,
		c.PricedAt.Format(time.RFC3339), c.BandwidthNote,
	)
}

// PreviewCost returns the predicted cost of broadcasting t (see CostPreview
// for the read sequence). It returns tx.fee_limit_too_low when the computed
// burn exceeds the transaction's fee limit — the §6.4 floor-check.
// It is the free-function entry point the facade's
// account.Handle.CostPreview wraps — the spec (§7.3) names the RESULT type
// CostPreview and the Client method CostPreview, so a package-level function
// of the same name cannot exist in Go; PreviewCost is that function.
// t must have been built by BuildTriggerSmartContract; owner is the account
// whose staked energy is counted.
func PreviewCost(ctx context.Context, cp rpc.ConnProvider, t *ContractTx, owner tron.Address) (*CostPreview, error) {
	const op = "tx.CostPreview"
	if t == nil {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op, Hint: "transaction is nil"}
	}
	// Read 1: the accurate energy estimator (Simulate.Energy =
	// TriggerConstantContract.EnergyUsed) AND the revert check.
	// Errors propagate: a call that cannot execute has no cost.
	sim, err := t.Simulate(ctx)
	if err != nil {
		return nil, err
	}
	// Read 2: the owner's staked energy AND bandwidth (one GetAccountResource
	// read feeds both halves — EnergyAvailable here, the Bandwidth half in
	// previewBandwidth below).
	res, err := rpc.GetAccountResource(cp, ctx, &core.Account{Address: owner.Bytes()})
	if err != nil {
		return nil, err
	}
	// Read 3: the energy→SUN burn ratio (network governance parameter,
	// independent of the contract or transaction).
	price, err := EnergyPriceOf(ctx, cp)
	if err != nil {
		return nil, err
	}
	available := res.GetEnergyLimit() - res.GetEnergyUsed()
	toBuy := sim.Energy - available
	if toBuy < 0 {
		toBuy = 0
	}
	// Energy burn calculator: convert energy-to-buy into SUN at the
	// network's current SunPerEnergy ratio.
	burn, err := price.CostOf(toBuy)
	if err != nil {
		return nil, &tron.Error{Code: tron.CodeAmountOverflow, Op: op, Cause: err,
			Hint: "EnergyToBuy × SunPerEnergy overflows SUN; the call cannot be priced in int64 SUN",
		}
	}
	// §6.4 floor-check: the fee_limit caps the TRX burned on energy, so the
	// computed burn must fit under it. The 150-TRX default is a floor that is
	// checked, not trusted.
	if t.FeeLimit() < burn {
		return nil, &tron.Error{
			Code: tron.CodeTxFeeLimitTooLow,
			Op:   op,
			Next: tron.ActionFixTransaction,
			Hint: "raise fee_limit via WithFeeLimit (current default 150 TRX) — the computed burn exceeds the transaction's fee limit",
		}
	}

	// Read 4: the bandwidth→SUN burn ratio, and the bandwidth half on the
	// same charging model BandwidthCostOf applies post-signing. A contract
	// call never takes the creation branch (its activation cost, if any, is
	// inside Simulate's energy), so this is the plain covered/burn split.
	bandwidth, err := previewBandwidth(ctx, cp, op, t, owner, res)
	if err != nil {
		return nil, err
	}
	total, err := burn.Add(bandwidth.Burn)
	if err != nil {
		return nil, &tron.Error{Code: tron.CodeAmountOverflow, Op: op, Cause: err,
			Hint: "TronToBurn + bandwidth Burn overflows SUN; the call cannot be priced in int64 SUN",
		}
	}
	return &CostPreview{
		EnergyNeeded:    sim.Energy,
		EnergyBase:      sim.Energy - sim.Penalty,
		EnergyPenalty:   sim.Penalty,
		EnergyAvailable: available,
		EnergyToBuy:     toBuy,
		TronToBurn:      burn,
		SunPerEnergy:    price.SunPerEnergy,
		PricedAt:        price.FetchedAt,
		Bandwidth:       bandwidth,
		TotalFloor:      total,
		BandwidthNote:   BandwidthEstimateNote,
	}, nil
}

// previewBandwidth computes the bandwidth half of a CostPreview on the
// charging model BandwidthCostOf applies to a signed transaction: staked
// bandwidth first, then the free quota (both from res, the read PreviewCost
// already made), any shortfall burning at BandwidthPriceOf's unit price. A
// predicted burn the owner cannot cover is account.insufficient_bandwidth
// — the node rejects the broadcast, so the preview reports the refusal
// instead of a number that cannot happen.
func previewBandwidth(ctx context.Context, cp rpc.ConnProvider, op string, t *ContractTx, owner tron.Address, res *api.AccountResourceMessage) (*BandwidthCost, error) {
	need, err := estimateBandwidthBytes(t.Transaction())
	if err != nil {
		return nil, err
	}
	bwPrice, err := BandwidthPriceOf(ctx, cp)
	if err != nil {
		return nil, err
	}
	cost := &BandwidthCost{
		BytesNeeded:     need,
		StakedAvailable: res.GetNetLimit() - res.GetNetUsed(),
		FreeAvailable:   res.GetFreeNetLimit() - res.GetFreeNetUsed(),
		SunPerByte:      bwPrice.SunPerByte,
		PricedAt:        bwPrice.FetchedAt,
		NetUsage:        need,
	}
	toBurnBytes := need - cost.StakedAvailable - cost.FreeAvailable
	if toBurnBytes < 0 {
		toBurnBytes = 0
	}
	cost.ToBurn = toBurnBytes
	if toBurnBytes == 0 {
		return cost, nil // covered: no burn, no balance read (the lazy rule)
	}
	bwBurn, err := bwPrice.CostOf(toBurnBytes)
	if err != nil {
		return nil, &tron.Error{Code: tron.CodeAmountOverflow, Op: op, Cause: err,
			Hint: "ToBurn × SunPerByte overflows SUN; the call cannot be priced",
		}
	}
	cost.Burn = bwBurn
	cost.NetUsage = 0 // the node reports NetUsage 0 on the burn path
	return requireBandwidthBalance(ctx, cp, op, owner, cost, int64(bwBurn))
}

// estimateBandwidthBytes measures what BandwidthSize will report for this
// transaction once it carries one signature: the SAME model — a clone with
// ret cleared plus ResultSizePerContract per contract — with one 65-byte
// placeholder signature attached, so a preview that runs before signing
// prices the bytes the broadcast will actually be charged for. Each
// additional signature grows the broadcast by ~67 bytes (tag + length +
// 65), priced as bandwidth when uncovered — the caveat BandwidthEstimateNote
// states.
func estimateBandwidthBytes(unsigned *core.Transaction) (int64, error) {
	const op = "tx.PreviewCost"
	if unsigned == nil {
		return 0, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op, Hint: "transaction is nil"}
	}
	clone := proto.Clone(unsigned).(*core.Transaction)
	clone.Ret = nil
	clone.Signature = [][]byte{make([]byte, signatureBytes)}
	return BandwidthSize(clone)
}

// latestEnergyPrice fetches the energy price history and returns the price
// and timestamp of the entry with the greatest timestamp. Malformed entries
// are contract.bad_metadata: the node answered, but not in the documented
// "timestamp:price" comma-list shape. Two shapes need an explicit rule because
// they parse without error yet resolve to a meaningless answer: a list holding
// no parsable entry (",,") and an entry outside the documented range (a
// negative price or timestamp). Either would otherwise return the zero
// baseline as a real price — see EnergyPriceOf's promise that a silent zero
// price is not acceptable.
func latestEnergyPrice(ctx context.Context, cp rpc.ConnProvider, op string) (int64, int64, error) {
	msg, err := rpc.GetEnergyPrices(cp, ctx, &api.EmptyMessage{})
	if err != nil {
		return 0, 0, err
	}
	var bestTs, bestPrice int64
	parsed := 0
	for _, entry := range strings.Split(msg.GetPrices(), ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		tsStr, priceStr, ok := strings.Cut(entry, ":")
		if !ok {
			return 0, 0, badPriceMetadata(op, msg.GetPrices())
		}
		ts, err1 := strconv.ParseInt(tsStr, 10, 64)
		price, err2 := strconv.ParseInt(priceStr, 10, 64)
		if err1 != nil || err2 != nil {
			return 0, 0, badPriceMetadata(op, msg.GetPrices())
		}
		// Governance prices and their timestamps are nonnegative; a negative
		// price inverts every cost, and a negative timestamp can never win the
		// comparison below (the baseline is 0), so both shapes must be refused
		// rather than parsed.
		if ts < 0 || price < 0 {
			return 0, 0, badPriceMetadata(op, msg.GetPrices())
		}
		parsed++
		if ts >= bestTs { // latest timestamp wins; ties keep the last entry
			bestTs, bestPrice = ts, price
		}
	}
	// Count the entries that PARSED, not the length of the raw string: ",," and
	// " " have length but contain nothing, and the zero baseline would then be
	// handed back as a price of 0 SUN per energy.
	if parsed == 0 {
		return 0, 0, badPriceMetadata(op, msg.GetPrices())
	}
	return bestPrice, bestTs, nil
}

func badPriceMetadata(op, raw string) *tron.Error {
	return &tron.Error{
		Code:  tron.CodeContractBadMetadata,
		Op:    op,
		Hint:  `GetEnergyPrices returned a malformed "timestamp:price" list; cannot price energy`,
		Cause: fmt.Errorf("energy prices: %q", raw),
	}
}
