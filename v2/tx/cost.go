package tx

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/v2/rpc"
	"github.com/kslamph/tronlib/v2/tron"
)

// CostPreview predicts what broadcasting a ContractTx will cost the owner in
// SUN, combining three read-only node answers (spec §7.3):
//
//  1. ContractTx.Simulate — the Energy/Penalty split AND the revert check. If
//     Simulate errors (including a node rejection surfaced as an error),
//     CostPreview returns the error: the caller then knows the call will not
//     run, and a cost prediction for a call that cannot run would be noise.
//  2. ContractTx.EstimateEnergy — the authoritative penalty-INCLUSIVE total
//     (EnergyNeeded). Simulate supplies the split; the total comes from
//     EstimateEnergy because that RPC is the node's dedicated estimate. This
//     costs ONE extra RPC versus using EstimateEnergy alone — the price of
//     knowing the penalty share.
//  3. rpc.GetAccountResource (owner) — EnergyLimit−EnergyUsed = staked
//     energy available without buying.
//
// The unit price comes from rpc.GetEnergyPrices: its history is a
// comma-separated "timestamp:price" list and the entry with the GREATEST
// timestamp is the current price (SUN per energy).
//
// TronToBurn = max(0, EnergyNeeded − EnergyAvailable) × price, computed with
// SUN's checked multiply: an overflow (huge energies × high prices) returns
// amount.overflow rather than wrapping into a silently wrong amount.
// PricedAt timestamps the price read — the preview is a FLOOR, not a ceiling:
// the price is a governance parameter and energy prices only ever move in the
// caller's favor at the margins between preview and broadcast.
//
// Fee-limit floor-check (spec §6.4): PreviewCost returns tx.fee_limit_too_low
// when TronToBurn exceeds the transaction's fee_limit — the 150-TRX default
// is a floor that is checked, not trusted.
//
// live-verified: pending (spec §7.5).
type CostPreview struct {
	// EnergyNeeded is the penalty-inclusive total energy (EstimateEnergy).
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
}

// String renders the preview as one line.
func (c *CostPreview) String() string {
	return fmt.Sprintf(
		"cost preview: energy needed %d (base %d + penalty %d), available %d, to buy %d @ %d sun/energy = %s sun (priced %s)",
		c.EnergyNeeded, c.EnergyBase, c.EnergyPenalty, c.EnergyAvailable,
		c.EnergyToBuy, c.SunPerEnergy, c.TronToBurn,
		c.PricedAt.Format(time.RFC3339),
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
func PreviewCost(cp rpc.ConnProvider, ctx context.Context, t *ContractTx, owner tron.Address) (*CostPreview, error) {
	const op = "tx.CostPreview"
	if t == nil {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op, Hint: "transaction is nil"}
	}
	// Read 1: the split and the revert check. Errors propagate.
	sim, err := t.Simulate(ctx)
	if err != nil {
		return nil, err
	}
	// Read 2: the authoritative inclusive total. Errors propagate.
	est, err := t.EstimateEnergy(ctx)
	if err != nil {
		return nil, err
	}
	// Read 3: the owner's staked energy.
	res, err := rpc.GetAccountResource(cp, ctx, &core.Account{Address: owner.Bytes()})
	if err != nil {
		return nil, err
	}
	price, _, err := latestEnergyPrice(cp, ctx, op)
	if err != nil {
		return nil, err
	}
	available := res.GetEnergyLimit() - res.GetEnergyUsed()
	toBuy := est.Energy - available
	if toBuy < 0 {
		toBuy = 0
	}
	burn, err := tron.SUN(toBuy).Mul(price)
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
		EnergyNeeded:    est.Energy,
		EnergyBase:      est.Energy - sim.Penalty,
		EnergyPenalty:   sim.Penalty,
		EnergyAvailable: available,
		EnergyToBuy:     toBuy,
		TronToBurn:      burn,
		SunPerEnergy:    price,
		PricedAt:        time.Now(),
	}, nil
}

// latestEnergyPrice fetches the energy price history and returns the price
// and timestamp of the entry with the greatest timestamp. Malformed entries
// are contract.bad_metadata: the node answered, but not in the documented
// "timestamp:price" comma-list shape.
func latestEnergyPrice(cp rpc.ConnProvider, ctx context.Context, op string) (int64, int64, error) {
	msg, err := rpc.GetEnergyPrices(cp, ctx, &api.EmptyMessage{})
	if err != nil {
		return 0, 0, err
	}
	var bestTs, bestPrice int64
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
		if ts >= bestTs { // latest timestamp wins; ties keep the last entry
			bestTs, bestPrice = ts, price
		}
	}
	if len(msg.GetPrices()) == 0 {
		return 0, 0, badPriceMetadata(op, "")
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
