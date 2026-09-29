package main

import (
	"strings"
	"testing"

	"github.com/kslamph/tronlib/v2/tx"
)

func TestVerifyFactorInconclusiveAtZero(t *testing.T) {
	// The Nile shape: both independent reads agree at zero.
	v := verifyFactor(&tx.DynamicEnergy{}, &tx.Estimate{Energy: 651})
	if v.pass || v.live {
		t.Fatalf("zero state must not pass: %+v", v)
	}
	for _, want := range []string{"inconclusive", "getDynamicEnergyThreshold", "5e9"} {
		if !strings.Contains(v.detail, want) {
			t.Errorf("detail %q does not contain %q", v.detail, want)
		}
	}
}

func TestVerifyFactorPassesWhenChecksAgree(t *testing.T) {
	// Stored factor 5000; a call with base 13569 whose node penalty is the
	// per-opcode aggregate minus flooring slack (6784 - 3): derived factor
	// 10000*20350/13569-10000 = 4997, within tolerance of 5000;
	// prediction 6784, actual 6781, within tolerance.
	v := verifyFactor(
		&tx.DynamicEnergy{Factor: 5000, Usage: 6000000000, UpdateCycle: 42},
		&tx.Estimate{Energy: 20350, Penalty: 6781},
	)
	if !v.pass || !v.live {
		t.Fatalf("agreeing reads must pass: %+v", v)
	}
}

func TestVerifyFactorMismatchOnDerivedAboveStored(t *testing.T) {
	// Penalty implies a factor above the stored one: contradiction.
	v := verifyFactor(&tx.DynamicEnergy{}, &tx.Estimate{Energy: 200, Penalty: 100})
	if v.pass {
		t.Fatalf("contradictory reads must not pass: %+v", v)
	}
	if v.live {
		t.Fatalf("zero stored factor must not count as live: %+v", v)
	}
	if !strings.Contains(v.detail, "exceeds stored factor") {
		t.Errorf("detail %q should name the disagreement", v.detail)
	}
}

func TestVerifyFactorMismatchOnPredictionBreach(t *testing.T) {
	// Actual penalty one unit above the aggregate upper bound, on a base
	// large enough that the derived factor still matches the stored one
	// (5000): the formula does not explain the node.
	v := verifyFactor(&tx.DynamicEnergy{Factor: 5000}, &tx.Estimate{Energy: 1500001, Penalty: 500001})
	if v.pass {
		t.Fatalf("prediction breach must not pass: %+v", v)
	}
	if !v.live {
		t.Fatalf("positive stored factor must count as live: %+v", v)
	}
	if !strings.Contains(v.detail, "upper bound") {
		t.Errorf("detail %q should name the breached bound", v.detail)
	}
}

func TestVerifyFactorNeedsBiggerCallWhenFlooredToZero(t *testing.T) {
	// Factor 10 on a 651-energy call floors to zero penalty: uninformative,
	// not a disagreement.
	v := verifyFactor(&tx.DynamicEnergy{Factor: 10}, &tx.Estimate{Energy: 651})
	if v.pass {
		t.Fatalf("floored-to-zero must not pass: %+v", v)
	}
	if !strings.Contains(v.detail, "simulate a larger call") {
		t.Errorf("detail %q should ask for a larger call", v.detail)
	}
}

func TestVerifyFactorDegenerateEstimate(t *testing.T) {
	v := verifyFactor(&tx.DynamicEnergy{Factor: 5000}, &tx.Estimate{Energy: 100, Penalty: 100})
	if v.pass {
		t.Fatalf("degenerate estimate must not pass: %+v", v)
	}
	if !strings.Contains(v.detail, "degenerate") {
		t.Errorf("detail %q should name the degenerate input", v.detail)
	}
}

func TestVerifyFactorNilInputs(t *testing.T) {
	if v := verifyFactor(nil, &tx.Estimate{Energy: 1}); v.pass {
		t.Fatalf("nil state must not pass: %+v", v)
	}
	if v := verifyFactor(&tx.DynamicEnergy{}, nil); v.pass {
		t.Fatalf("nil estimate must not pass: %+v", v)
	}
}

func TestRandomOwnerIsWellFormed(t *testing.T) {
	a, err := randomOwner()
	if err != nil {
		t.Fatalf("randomOwner: %v", err)
	}
	if a.IsZero() {
		t.Fatal("random owner must not be the zero address")
	}
	b, err := randomOwner()
	if err != nil {
		t.Fatalf("randomOwner: %v", err)
	}
	if a == b {
		t.Fatal("two random owners must differ")
	}
}
