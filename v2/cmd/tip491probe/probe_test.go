package main

import (
	"strings"
	"testing"
)

func TestCheckPenaltyAcceptsPositive(t *testing.T) {
	for _, p := range []int64{1, 9492} {
		if err := checkPenalty(p); err != nil {
			t.Errorf("checkPenalty(%d) = %v, want nil", p, err)
		}
	}
}

func TestCheckPenaltyInconclusiveZero(t *testing.T) {
	err := checkPenalty(0)
	if err == nil {
		t.Fatal("checkPenalty(0) = nil, want an inconclusive result — a zero factor is not a pass")
	}
	msg := err.Error()
	for _, want := range []string{"inconclusive", "getDynamicEnergyThreshold", "5e9"} {
		if !strings.Contains(msg, want) {
			t.Errorf("checkPenalty(0) message %q does not contain %q", msg, want)
		}
	}
}

func TestCheckPenaltyNegativeIsError(t *testing.T) {
	if err := checkPenalty(-1); err == nil {
		t.Error("checkPenalty(-1) = nil, want an error (a negative penalty is not a pass)")
	}
}
