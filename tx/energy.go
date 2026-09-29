package tx

import (
	"context"
	"time"

	"github.com/kslamph/tronlib/v2/rpc"
	"github.com/kslamph/tronlib/v2/tron"
)

// MaintenancePeriod is TRON's maintenance interval: the window over which the
// dynamic-energy consumption factor is recomputed (architecture §7.1/§7.3). The unit
// energy price changes only via governance proposal, so a cached price is
// refetched once this much time has passed.
const MaintenancePeriod = 6 * time.Hour

// EnergyPrice is the current energy unit price read from the node's
// governance price history (architecture §7.1): the entry with the greatest
// timestamp in rpc.GetEnergyPrices' "timestamp:price" comma-list.
// SunPerEnergy is SUN per unit of energy; EffectiveAt timestamps the chosen
// entry — the price is a governance parameter and energy prices only ever
// move in the caller's favor at the margins, so a read is a floor, not a
// ceiling (CostPreview.PricedAt carries the same rule for previews).
// live-verified: pending (architecture §7.5).
type EnergyPrice struct {
	// SunPerEnergy is the latest unit price in SUN per energy.
	SunPerEnergy int64
	// EffectiveAt is the timestamp of the price entry that won.
	EffectiveAt time.Time
	// FetchedAt is when the price was read; staleness is explicit.
	FetchedAt time.Time
}

// EnergyPriceOf fetches the energy price history and returns the current
// price (the latest ts:price entry). It is the free-function entry point
// the facade's Client.EnergyPrice delegates to. A malformed or empty price
// list is contract.bad_metadata: the node answered, but not in the
// documented shape — a silent zero price would understate every cost.
// Exported per Task 9 controller ruling (D1): the facade's architecture §10 surface
// needs the EnergyPrice type, and the reviewed parse in cost.go is exported
// behind it rather than duplicated.
func EnergyPriceOf(ctx context.Context, cp rpc.ConnProvider) (*EnergyPrice, error) {
	const op = "tx.EnergyPriceOf"
	price, ts, err := latestEnergyPrice(ctx, cp, op)
	if err != nil {
		return nil, err
	}
	return &EnergyPrice{
		SunPerEnergy: price,
		EffectiveAt:  time.UnixMilli(ts),
		FetchedAt:    time.Now(),
	}, nil
}

// CostOf converts an energy amount into the SUN that burning that energy
// costs at this price (energy × SunPerEnergy). It is the energy-burn
// calculator: the energy→SUN ratio is a property of the network's current
// operating parameters, independent of any specific contract or
// transaction. Energy estimators (Simulate/EstimateEnergy, CostPreview)
// produce energy units; this primitive is the one place energy becomes SUN.
// An overflow (energy × price > int64 SUN) returns amount.overflow; a
// negative energy has no SUN cost and returns amount.negative.
func (p *EnergyPrice) CostOf(energy int64) (tron.SUN, error) {
	if energy < 0 {
		return 0, &tron.Error{
			Code: tron.CodeAmountNegative,
			Op:   "EnergyPrice.CostOf",
			Hint: "negative energy has no SUN cost",
		}
	}
	return tron.SUN(energy).Mul(p.SunPerEnergy)
}
