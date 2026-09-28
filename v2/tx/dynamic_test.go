package tx

// Tests for the TIP-491 dynamic-energy read path (DynamicEnergyOf) and the
// penalty arithmetic (PredictPenalty, Estimate.EffectiveFactor).
//
// The penalty vectors are hand-computed from the node's formula
// (java-tron VM.play: penalty = floor(base*(factor+10_000)/10_000) - base
// per opcode): the test pins the aggregate form the library implements,
// with the per-opcode flooring bound documented on PredictPenalty.

import (
	"context"
	"math"
	"testing"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/v2/tron"
)

func TestPredictPenaltyVectors(t *testing.T) {
	cases := []struct {
		name   string
		factor int64
		base   int64
		want   int64
	}{
		{"no factor, live-run base", 0, 13569, 0},
		{"no factor, zero base", 0, 0, 0},
		{"factor, zero base", 5000, 0, 0},
		// 13569*15000/10000 = 20353.5 -> floor 20353 - 13569
		{"50% surcharge on live-run base", 5000, 13569, 6784},
		{"100% surcharge", 10000, 100, 100},
		{"900% surcharge", 90000, 1000, 9000},
		// 100*10001/10000 = 100.01 -> floor 100: a tiny factor on a
		// small call floors to nothing. The node behaves identically
		// per opcode, so small calls on lightly-penalized contracts
		// genuinely cost no surcharge.
		{"tiny factor floors to zero", 1, 100, 0},
		{"max-range factor", 100000, 1000000, 10000000},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := (&DynamicEnergy{Factor: tc.factor}).PredictPenalty(tc.base)
			if err != nil {
				t.Fatalf("PredictPenalty: %v", err)
			}
			if got != tc.want {
				t.Fatalf("PredictPenalty(base=%d, factor=%d) = %d, want %d", tc.base, tc.factor, got, tc.want)
			}
		})
	}
}

func TestPredictPenaltyErrors(t *testing.T) {
	if _, err := (*DynamicEnergy)(nil).PredictPenalty(100); !tron.HasCode(err, tron.CodeTxInvalidArgument) {
		t.Fatalf("nil receiver: want tx.invalid_argument, got %v", err)
	}
	if _, err := (&DynamicEnergy{Factor: -1}).PredictPenalty(100); !tron.HasCode(err, tron.CodeTxInvalidArgument) {
		t.Fatalf("negative factor: want tx.invalid_argument, got %v", err)
	}
	if _, err := (&DynamicEnergy{Factor: 5000}).PredictPenalty(-1); !tron.HasCode(err, tron.CodeAmountNegative) {
		t.Fatalf("negative base: want amount.negative, got %v", err)
	}
	if _, err := (&DynamicEnergy{Factor: 1}).PredictPenalty(math.MaxInt64); !tron.HasCode(err, tron.CodeAmountOverflow) {
		t.Fatalf("overflowing multiply: want amount.overflow, got %v", err)
	}
	if _, err := (&DynamicEnergy{Factor: math.MaxInt64}).PredictPenalty(1); !tron.HasCode(err, tron.CodeAmountOverflow) {
		t.Fatalf("overflowing factor scale: want amount.overflow, got %v", err)
	}
}

func TestHasPenalty(t *testing.T) {
	if (*DynamicEnergy)(nil).HasPenalty() {
		t.Fatal("nil state must not report a penalty")
	}
	if (&DynamicEnergy{}).HasPenalty() {
		t.Fatal("zero state must not report a penalty")
	}
	if !(&DynamicEnergy{Factor: 1}).HasPenalty() {
		t.Fatal("positive factor must report a penalty")
	}
}

func TestEffectiveFactor(t *testing.T) {
	cases := []struct {
		name       string
		energy     int64
		penalty    int64
		wantFactor int64
		wantOK     bool
	}{
		{"penalty-free live run derives zero", 13569, 0, 0, true},
		// 10000*20353/13569 = 14999 (floor) - 10000 = 4999: the
		// aggregate derivation sits just below the true stored factor
		// 5000, exactly the documented lower-bound behavior.
		{"penalized call derives just below stored factor", 20353, 6784, 4999, true},
		{"zero energy has nothing to derive", 0, 0, 0, false},
		{"negative penalty is degenerate", 100, -1, 0, false},
		{"penalty meeting energy is degenerate", 100, 100, 0, false},
		{"penalty above energy is degenerate", 100, 101, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := (&Estimate{Energy: tc.energy, Penalty: tc.penalty}).EffectiveFactor()
			if ok != tc.wantOK || got != tc.wantFactor {
				t.Fatalf("EffectiveFactor() = (%d, %v), want (%d, %v)", got, ok, tc.wantFactor, tc.wantOK)
			}
		})
	}
	if _, ok := (*Estimate)(nil).EffectiveFactor(); ok {
		t.Fatal("nil estimate must not derive a factor")
	}
}

func TestDynamicEnergyOf(t *testing.T) {
	ctx := t.Context()

	t.Run("maps state fields", func(t *testing.T) {
		f := &fakeWalletServer{
			ContractInfo: func(ctx context.Context, in *api.BytesMessage) (*core.SmartContractDataWrapper, error) {
				return &core.SmartContractDataWrapper{
					SmartContract: &core.SmartContract{},
					ContractState: &core.ContractState{EnergyUsage: 6000000000, EnergyFactor: 5000, UpdateCycle: 12345},
				}, nil
			},
		}
		got, err := DynamicEnergyOf(newTxTestClient(t, f), ctx, testTo)
		if err != nil {
			t.Fatalf("DynamicEnergyOf: %v", err)
		}
		if got.Factor != 5000 || got.Usage != 6000000000 || got.UpdateCycle != 12345 {
			t.Fatalf("DynamicEnergyOf = %+v, want factor 5000 usage 6000000000 cycle 12345", got)
		}
		if !got.HasPenalty() {
			t.Fatal("factor 5000 must report a penalty")
		}
	})

	t.Run("absent state is a fresh contract", func(t *testing.T) {
		got, err := DynamicEnergyOf(newTxTestClient(t, &fakeWalletServer{}), ctx, testTo)
		if err != nil {
			t.Fatalf("DynamicEnergyOf: %v", err)
		}
		if *got != (DynamicEnergy{}) {
			t.Fatalf("DynamicEnergyOf = %+v, want zero state", got)
		}
	})

	t.Run("nil wrapper is not found", func(t *testing.T) {
		f := &fakeWalletServer{
			ContractInfo: func(ctx context.Context, in *api.BytesMessage) (*core.SmartContractDataWrapper, error) {
				return nil, nil
			},
		}
		if _, err := DynamicEnergyOf(newTxTestClient(t, f), ctx, testTo); !tron.HasCode(err, tron.CodeContractNotFound) {
			t.Fatalf("want contract.not_found, got %v", err)
		}
	})

	t.Run("zero address rejected", func(t *testing.T) {
		if _, err := DynamicEnergyOf(newTxTestClient(t, &fakeWalletServer{}), ctx, tron.Address{}); !tron.HasCode(err, tron.CodeAddressInvalid) {
			t.Fatalf("want address.invalid, got %v", err)
		}
	})

	t.Run("nil conn rejected", func(t *testing.T) {
		if _, err := DynamicEnergyOf(nil, ctx, testTo); !tron.HasCode(err, tron.CodeChainConnection) {
			t.Fatalf("want chain.connection, got %v", err)
		}
	})
}
