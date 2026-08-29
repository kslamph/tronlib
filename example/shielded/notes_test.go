package main

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kslamph/tronlib/pb/api"
)

// note builds one scanned note for the selection tests.
func note(position, value int64, spent bool) *api.DecryptNotesTRC20_NoteTx {
	return &api.DecryptNotesTRC20_NoteTx{
		Position: position,
		IsSpent:  spent,
		Note:     &api.Note{Value: value, PaymentAddress: "ztron1test"},
	}
}

func TestSpendableNotes(t *testing.T) {
	tests := []struct {
		name    string
		in      []*api.DecryptNotesTRC20_NoteTx
		wantPos []int64
	}{
		{"empty", nil, nil},
		{"drops spent", []*api.DecryptNotesTRC20_NoteTx{note(1, 10, true), note(2, 20, false)}, []int64{2}},
		{"drops noteless transparent output", []*api.DecryptNotesTRC20_NoteTx{{Position: 3, ToAmount: "10"}}, nil},
		{"sorts oldest leaf first", []*api.DecryptNotesTRC20_NoteTx{note(9, 10, false), note(4, 20, false), note(7, 30, false)},
			[]int64{4, 7, 9}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := spendableNotes(tt.in)
			require.Len(t, got, len(tt.wantPos))
			for i, want := range tt.wantPos {
				assert.Equal(t, want, got[i].GetPosition(), "note %d", i)
			}
		})
	}
}

func TestPlanBurn(t *testing.T) {
	notes := []*api.DecryptNotesTRC20_NoteTx{
		note(1, 5_000_000, false),
		note(2, 10_000_000, false),
		note(3, 6_000_000, false),
		note(4, 8_000_000, true), // spent, must never be selected
	}

	tests := []struct {
		name    string
		need    int64
		wantPos int64
		wantVal int64
		wantChg int64
		wantErr string
	}{
		{"exact note leaves no change", 5_000_000, 1, 5_000_000, 0, ""},
		{"smallest sufficient, not first", 5_500_000, 3, 6_000_000, 500_000, ""},
		{"skips spent notes", 8_000_000, 2, 10_000_000, 2_000_000, ""},
		{"nothing covers the withdrawal", 11_000_000, 0, 0, 0, "no single unspent note covers"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := planBurn(notes, big.NewInt(tt.need))
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Contains(t, err.Error(), "-mode=transfer", "the error should point at consolidating")
				return
			}
			require.NoError(t, err)
			require.Len(t, plan.spends, 1, "burn takes exactly one input")
			assert.Equal(t, tt.wantPos, plan.spends[0].GetPosition())
			assert.Equal(t, tt.wantChg, plan.change.Int64())

			// The equation the contract proves over this plan.
			spend := new(big.Int).Mul(big.NewInt(tt.wantVal), big.NewInt(1))
			lhs := spend
			rhs := new(big.Int).Add(new(big.Int).Mul(plan.change, big.NewInt(1)), big.NewInt(tt.need))
			assert.Zero(t, lhs.Cmp(rhs), "spend.value must equal to_amount + change")
		})
	}
}

func TestPlanTransfer(t *testing.T) {
	tests := []struct {
		name      string
		notes     []*api.DecryptNotesTRC20_NoteTx
		need      int64
		wantPos   []int64
		wantChg   int64
		wantErrRe string
	}{
		{
			name:    "single note preferred over a pair",
			notes:   []*api.DecryptNotesTRC20_NoteTx{note(1, 6_000_000, false), note(2, 4_000_000, false), note(3, 5_000_000, false)},
			need:    4_000_000,
			wantPos: []int64{2}, wantChg: 0,
		},
		{
			name:    "single note minimises change",
			notes:   []*api.DecryptNotesTRC20_NoteTx{note(1, 9_000_000, false), note(2, 7_000_000, false)},
			need:    6_000_000,
			wantPos: []int64{2}, wantChg: 1_000_000,
		},
		{
			// No single note covers 6.5M, so the pair with the smallest remainder
			// wins: 3M+4M leaves 500k, where 3M+5M leaves 1.5M and 4M+5M leaves 2.5M.
			name:    "falls back to the least-wasteful pair",
			notes:   []*api.DecryptNotesTRC20_NoteTx{note(1, 3_000_000, false), note(2, 4_000_000, false), note(3, 5_000_000, false)},
			need:    6_500_000,
			wantPos: []int64{1, 2}, wantChg: 500_000,
		},
		{
			name:    "a single sufficient note still beats a pair",
			notes:   []*api.DecryptNotesTRC20_NoteTx{note(1, 3_000_000, false), note(2, 4_000_000, false), note(3, 9_000_000, false)},
			need:    6_500_000,
			wantPos: []int64{3}, wantChg: 2_500_000,
		},
		{
			name:      "two notes cannot cover it",
			notes:     []*api.DecryptNotesTRC20_NoteTx{note(1, 1_000_000, false), note(2, 2_000_000, false), note(3, 3_000_000, false)},
			need:      10_000_000,
			wantErrRe: "at most two notes",
		},
		{
			name:      "no unspent notes at all",
			notes:     []*api.DecryptNotesTRC20_NoteTx{note(1, 1_000_000, true)},
			need:      1,
			wantErrRe: "insufficient shielded balance",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := planTransfer(tt.notes, big.NewInt(tt.need))
			if tt.wantErrRe != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrRe)
				return
			}
			require.NoError(t, err)
			require.LessOrEqual(t, len(plan.spends), 2, "transfer takes at most two inputs")
			got := make([]int64, 0, len(plan.spends))
			for _, n := range plan.spends {
				got = append(got, n.GetPosition())
			}
			assert.Equal(t, tt.wantPos, got)
			assert.Equal(t, tt.wantChg, plan.change.Int64())

			sum := new(big.Int)
			for _, n := range plan.spends {
				sum.Add(sum, big.NewInt(n.GetNote().GetValue()))
			}
			back := new(big.Int).Add(big.NewInt(tt.need), plan.change)
			assert.Zero(t, sum.Cmp(back), "spend.value must equal payment + change")
		})
	}
}

func TestNotePlanDescribe(t *testing.T) {
	plan := notePlan{
		spends: []*api.DecryptNotesTRC20_NoteTx{note(16, 6_000_000, false)},
		change: big.NewInt(2_000_000),
	}
	assert.Equal(t,
		"burn plan: spend position 16 (value 6000000), to_amount 4000000 scaled, change 2000000 scaled",
		plan.describe("burn", "to_amount", big.NewInt(4_000_000)))

	plan.spends = []*api.DecryptNotesTRC20_NoteTx{note(4, 3_000_000, false), note(9, 5_000_000, false)}
	assert.Equal(t,
		"transfer plan: spend position 4+9 (value 3000000+5000000), pay 6000000 scaled, change 2000000 scaled",
		plan.describe("transfer", "pay", big.NewInt(6_000_000)))
}

func TestToScaled(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		sf      string
		want    string
		wantErr string
	}{
		{"scaling factor one is the identity", "10000000", "1", "10000000", ""},
		{"exact multiple divides", "10000000", "1000000", "10", ""},
		{"remainder is refused, not truncated", "1500000", "1000000", "", "not a multiple"},
		{"zero scales to zero", "0", "7", "0", ""},
		{"above the note ceiling is refused", "9223372036854775807", "1", "", "note ceiling"},
		{"beyond int64 is refused", "99999999999999999999999999", "1", "", "note ceiling"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toScaled(bi(tt.raw), bi(tt.sf))
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.String())
		})
	}
}

// TestNoteValue_RejectsOverflowingChange is the regression guard for the change
// note built from two notes minted elsewhere: each is a legal int64, their sum
// is not, and wrapping it silently would mint a negative-value note.
func TestNoteValue_RejectsOverflowingChange(t *testing.T) {
	tests := []struct {
		name    string
		value   *big.Int
		want    int64
		wantErr bool
	}{
		{"zero", big.NewInt(0), 0, false},
		{"just under the ceiling", big.NewInt(maxNoteValue - 1), maxNoteValue - 1, false},
		{"at the ceiling", big.NewInt(maxNoteValue), 0, true},
		{"negative", big.NewInt(-1), 0, true},
		{"beyond int64", new(big.Int).Lsh(big.NewInt(1), 70), 0, true},
		{"the wrapped pair sum from the bug report", new(big.Int).Add(big.NewInt(1<<62), big.NewInt(1<<62)), 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := noteValue("change", tt.value)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "does not fit a note value")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func bi(s string) *big.Int {
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		panic("bad test literal: " + s)
	}
	return v
}
