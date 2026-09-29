package tx

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/rpc"
	"github.com/kslamph/tronlib/v2/tron"
)

// CostPreview predicts what broadcasting a ContractTx will cost the owner in
// SUN, combining three read-only node answers (architecture §7.3):
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
//  2. rpc.GetAccountResource (owner) — EnergyLimit−EnergyUsed = staked
//     energy available without buying (EnergyAvailable).
//
//  3. EnergyPriceOf — the current SunPerEnergy (the network governance
//     parameter that sets the energy→SUN burn ratio, independent of any
//     specific contract or transaction).
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
	// BandwidthNote states what the preview does not cover. The energy-only
	// preview is a FLOOR: a live run measured a 345,000 SUN NetFee delta the
	// preview never mentioned (architecture §7.3 limitation 1).
	BandwidthNote string
}

// BandwidthNotModelled is the CostPreview.BandwidthNote value: the preview
// prices energy only, and RecipientActivation is a separate unmodelled cost
// (architecture §7.3 limitations 1 and 2).
const BandwidthNotModelled = "bandwidth (NetFee) and recipient activation are not included"

// String renders the preview as one line, including the bandwidth note so a
// logged preview cannot be read as a total.
func (c *CostPreview) String() string {
	return fmt.Sprintf(
		"cost preview: energy needed %d (base %d + penalty %d), available %d, to buy %d @ %d sun/energy = %s sun (priced %s; %s)",
		c.EnergyNeeded, c.EnergyBase, c.EnergyPenalty, c.EnergyAvailable,
		c.EnergyToBuy, c.SunPerEnergy, c.TronToBurn,
		c.PricedAt.Format(time.RFC3339), c.BandwidthNote,
	)
}

// PreviewCost returns the predicted cost of broadcasting t (see CostPreview
// for the read sequence). It returns tx.fee_limit_too_low when the computed
// burn exceeds the transaction's fee limit — the §6.4 floor-check.
// It is the free-function entry point the facade's
// Client.CostPreview wraps — the spec (§7.3) names the RESULT type
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
	// Read 2: the owner's staked energy.
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
	return &CostPreview{
		EnergyNeeded:    sim.Energy,
		EnergyBase:      sim.Energy - sim.Penalty,
		EnergyPenalty:   sim.Penalty,
		EnergyAvailable: available,
		EnergyToBuy:     toBuy,
		TronToBurn:      burn,
		SunPerEnergy:    price.SunPerEnergy,
		PricedAt:        price.FetchedAt,
		BandwidthNote:   BandwidthNotModelled,
	}, nil
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
