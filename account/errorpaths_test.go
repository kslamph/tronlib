package account

// Error-path tests for the account handles. CODING_STANDARDS.md 6.5 says a
// package whose error branches are untested is not done, and every branch here
// is reachable: either the node answers with an RPC error, or it answers with a
// protobuf that decodes but carries a value the API contract does not allow (a
// vote whose witness address is not an address, an unstake with no expiry, a
// resource code outside the curated set).
//
// The shape these tests defend is consistent error handling: a corrupt field
// must fail loudly and name the field, never be silently coerced into a zero
// value that a caller would read as real money or a real witness.

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/tron"
	"github.com/kslamph/tronlib/v2/tx"
)

// notAnAddress is 2 bytes where a 0x41-prefixed 21-byte address is required.
var notAnAddress = []byte{0x01, 0x02}

// errNodeDown is the failure injected into a single RPC handler.
var errNodeDown = errors.New("node down")

// asTronError asserts err is a *tron.Error and returns it, so tests can read
// Code/Op/Hint. Error() renders only "op: code" (the Hint is deliberately not
// duplicated into it), so naming assertions must go through the fields.
func asTronError(t *testing.T, err error) *tron.Error {
	t.Helper()
	require.Error(t, err)
	var te *tron.Error
	require.True(t, errors.As(err, &te), "err = %v, want *tron.Error", err)
	return te
}

// requireNodeFailure asserts the shape an injected RPC error takes by the time
// it reaches the caller: rpc.method_failed naming the operation. Go error
// identity is deliberately not asserted - the failure crosses the gRPC boundary,
// where the transport re-encodes it as a status error, so the original value is
// unreachable by errors.Is and asserting on it would test the transport rather
// than this package.
func requireNodeFailure(t *testing.T, err error, operation string) {
	t.Helper()
	te := asTronError(t, err)
	assert.Equal(t, tron.CodeRPCMethodFailed, te.Code, "err = %v", err)
	assert.Contains(t, te.Op, operation, "the error must name the RPC that failed")
}

func TestStateErrorPaths(t *testing.T) {
	t.Run("node failure propagates", func(t *testing.T) {
		f := &fakeWalletServer{}
		f.failWith("GetAccount", errNodeDown)
		_, err := newTestHandle(t, f).State(t.Context())
		requireNodeFailure(t, err, "get account")
	})

	t.Run("vote with a non-address witness is refused", func(t *testing.T) {
		f := &fakeWalletServer{account: &core.Account{
			Address: testFrom.Bytes(),
			Votes:   []*core.Vote{{VoteAddress: notAnAddress, VoteCount: 7}},
		}}
		_, err := newTestHandle(t, f).State(t.Context())
		require.Error(t, err, "a corrupt vote must not be dropped silently")
		assert.True(t, tron.HasCode(err, tron.CodeAddressInvalid), "err = %v", err)
		assert.Contains(t, asTronError(t, err).Hint, "witness address",
			"the hint must say which field broke")
	})

	t.Run("an unstake decoded from the account read keeps its resource", func(t *testing.T) {
		f := &fakeWalletServer{account: &core.Account{
			Address: testFrom.Bytes(),
			UnfrozenV2: []*core.Account_UnFreezeV2{
				{Type: core.ResourceCode_BANDWIDTH, UnfreezeAmount: 900,
					UnfreezeExpireTime: time.Now().Add(24 * time.Hour).UnixMilli()},
			},
		}}
		st, err := newTestHandle(t, f).State(t.Context())
		require.NoError(t, err)
		require.Len(t, st.Unstakes, 1)
		assert.Equal(t, tx.ResourceBandwidth, st.Unstakes[0].Resource)
		assert.False(t, st.Unstakes[0].ExpiresAt.IsZero())
	})

	t.Run("unknown resource code is not coerced to a real resource", func(t *testing.T) {
		// TRON_POWER is on the wire but is not in the curated tx.Resource set.
		// Mapping it to Bandwidth (the zero value) would present an account's
		// TRON_POWER balance as staked Bandwidth.
		f := &fakeWalletServer{account: &core.Account{
			Address: testFrom.Bytes(),
			FrozenV2: []*core.Account_FreezeV2{
				{Type: core.ResourceCode_TRON_POWER, Amount: 1_000},
			},
		}}
		st, err := newTestHandle(t, f).State(t.Context())
		require.NoError(t, err)
		require.Len(t, st.Stakes, 1)
		assert.Equal(t, tx.Resource(-1), st.Stakes[0].Resource,
			"an unmapped resource must stay distinguishable from Bandwidth and Energy")
		assert.Equal(t, "UNKNOWN", st.Stakes[0].Resource.String())
	})

	t.Run("unstake without an expiry reads as the zero time", func(t *testing.T) {
		f := &fakeWalletServer{account: &core.Account{
			Address: testFrom.Bytes(),
			UnfrozenV2: []*core.Account_UnFreezeV2{
				{Type: core.ResourceCode_ENERGY, UnfreezeAmount: 500, UnfreezeExpireTime: 0},
			},
		}}
		st, err := newTestHandle(t, f).State(t.Context())
		require.NoError(t, err)
		require.Len(t, st.Unstakes, 1)
		assert.True(t, st.Unstakes[0].ExpiresAt.IsZero(),
			"a missing expiry must not become the unix epoch, which reads as already matured")
		assert.Equal(t, tron.SUN(500), st.Unstakes[0].Amount)
	})
}

func TestBalancePropagatesNodeFailure(t *testing.T) {
	f := &fakeWalletServer{}
	f.failWith("GetAccount", errNodeDown)
	bal, err := newTestHandle(t, f).Balance(t.Context())
	require.Error(t, err)
	assert.Equal(t, tron.SUN(0), bal, "a failed read must not report a balance")
}

// badKeyPermission is a permission whose first key is not an address; field
// names the slot the decoder is expected to report.
func badKeyPermission() *core.Permission {
	return &core.Permission{
		Id:             0,
		PermissionName: "owner",
		Threshold:      1,
		Keys:           []*core.Key{{Address: notAnAddress, Weight: 1}},
	}
}

func goodPermission(name string) *core.Permission {
	return &core.Permission{
		PermissionName: name,
		Threshold:      1,
		Keys:           []*core.Key{{Address: testFrom.Bytes(), Weight: 1}},
	}
}

func TestPermissionsCurrentErrorPaths(t *testing.T) {
	t.Run("node failure propagates", func(t *testing.T) {
		f := &fakeWalletServer{}
		f.failWith("GetAccount", errNodeDown)
		_, err := newTestHandle(t, f).Permissions().Current(t.Context())
		requireNodeFailure(t, err, "get account")
	})

	for _, tt := range []struct {
		name  string
		acc   *core.Account
		field string
	}{
		{"owner key", &core.Account{OwnerPermission: badKeyPermission()}, "owner.keys[0]"},
		{"witness key", &core.Account{
			OwnerPermission:   goodPermission("owner"),
			WitnessPermission: badKeyPermission(),
		}, "witness.keys[0]"},
		{"active key", &core.Account{
			OwnerPermission:  goodPermission("owner"),
			ActivePermission: []*core.Permission{badKeyPermission()},
		}, "actives[0].keys[0]"},
	} {
		t.Run(tt.name+" names its slot", func(t *testing.T) {
			// A permission whose keys cannot be decoded must fail: reading it
			// back as an empty key list would make an unusable configuration
			// look valid, and Update would write it back to the chain.
			f := &fakeWalletServer{account: tt.acc}
			_, err := newTestHandle(t, f).Permissions().Current(t.Context())
			require.Error(t, err, "%s must not decode", tt.field)
			assert.True(t, tron.HasCode(err, tron.CodeAddressInvalid), "err = %v", err)
			assert.Contains(t, asTronError(t, err).Hint, tt.field,
				"the error must name the offending slot")
		})
	}
}

func TestSignWeightErrorPaths(t *testing.T) {
	f := &fakeWalletServer{}
	h := newTestHandle(t, f)
	native, err := h.TransferTRX(t.Context(), testTo, tron.TRX(1))
	require.NoError(t, err)

	t.Run("nil transaction", func(t *testing.T) {
		st, err := h.Permissions().SignWeight(t.Context(), nil)
		require.Error(t, err)
		assert.Nil(t, st)
		assert.True(t, tron.HasCode(err, tron.CodeTxInvalidArgument), "err = %v", err)
	})

	t.Run("node failure propagates", func(t *testing.T) {
		f.failWith("GetTransactionSignWeight", errNodeDown)
		defer func() { f.failWith("GetTransactionSignWeight", nil) }()
		_, err := h.Permissions().SignWeight(t.Context(), native)
		requireNodeFailure(t, err, "get transaction sign weight")
	})

	t.Run("approved address that is not an address", func(t *testing.T) {
		f.signWeight = &api.TransactionSignWeight{
			Permission:   goodPermission("owner"),
			ApprovedList: [][]byte{notAnAddress},
		}
		_, err := h.Permissions().SignWeight(t.Context(), native)
		require.Error(t, err, "a signer the API cannot name must not be dropped")
		assert.True(t, tron.HasCode(err, tron.CodeAddressInvalid), "err = %v", err)
	})

	t.Run("permission with an undecodable key", func(t *testing.T) {
		f.signWeight = &api.TransactionSignWeight{
			Permission:   badKeyPermission(),
			ApprovedList: [][]byte{testFrom.Bytes()},
		}
		_, err := h.Permissions().SignWeight(t.Context(), native)
		require.Error(t, err)
		assert.True(t, tron.HasCode(err, tron.CodeAddressInvalid), "err = %v", err)
	})
}

func TestApprovalsErrorPaths(t *testing.T) {
	f := &fakeWalletServer{}
	h := newTestHandle(t, f)
	native, err := h.TransferTRX(t.Context(), testTo, tron.TRX(1))
	require.NoError(t, err)

	t.Run("nil transaction", func(t *testing.T) {
		got, err := h.Permissions().Approvals(t.Context(), nil)
		require.Error(t, err)
		assert.Nil(t, got)
		assert.True(t, tron.HasCode(err, tron.CodeTxInvalidArgument), "err = %v", err)
	})

	t.Run("node failure propagates", func(t *testing.T) {
		f.failWith("GetTransactionApprovedList", errNodeDown)
		defer func() { f.failWith("GetTransactionApprovedList", nil) }()
		_, err := h.Permissions().Approvals(t.Context(), native)
		requireNodeFailure(t, err, "get transaction approved list")
	})

	t.Run("approved address that is not an address", func(t *testing.T) {
		f.approved = &api.TransactionApprovedList{ApprovedList: [][]byte{notAnAddress}}
		_, err := h.Permissions().Approvals(t.Context(), native)
		require.Error(t, err)
		assert.True(t, tron.HasCode(err, tron.CodeAddressInvalid), "err = %v", err)
	})
}

// TestSummaryFailsOnEachRead: Summary folds three reads, so each one's failure
// must abort before any partial summary is returned.
func TestSummaryFailsOnEachRead(t *testing.T) {
	for _, rpc := range []string{"GetAccount", "GetAccountResource", "GetAvailableUnfreezeCount"} {
		t.Run(rpc, func(t *testing.T) {
			f := &fakeWalletServer{account: richAccount()}
			f.failWith(rpc, errNodeDown)
			s, err := newTestHandle(t, f).Resources().Summary(t.Context())
			require.Error(t, err, "%s failure must abort Summary", rpc)
			assert.Nil(t, s, "no partial summary on a failed read")
			assert.True(t, tron.HasCode(err, tron.CodeRPCMethodFailed), "err = %v", err)
		})
	}
}

func TestVotingErrorPaths(t *testing.T) {
	t.Run("Votes propagates the account read failure", func(t *testing.T) {
		f := &fakeWalletServer{}
		f.failWith("GetAccount", errNodeDown)
		votes, err := newTestHandle(t, f).Voting().Votes(t.Context())
		require.Error(t, err)
		assert.Nil(t, votes)
	})

	t.Run("NextTally propagates the node failure", func(t *testing.T) {
		f := &fakeWalletServer{}
		f.failWith("GetNextMaintenanceTime", errNodeDown)
		when, err := newTestHandle(t, f).Voting().NextTally(t.Context())
		require.Error(t, err)
		assert.True(t, when.IsZero())
	})

	t.Run("NextTally refuses a nonsensical time", func(t *testing.T) {
		// A node answering 0 means "unknown"; reading it as the unix epoch
		// would tell a voter their votes take effect in 1970.
		f := &fakeWalletServer{nextMaint: 0}
		when, err := newTestHandle(t, f).Voting().NextTally(t.Context())
		require.Error(t, err, "a zero maintenance time must be an error, not 1970")
		assert.True(t, when.IsZero(), "no time may be reported alongside the error")
		assert.True(t, tron.HasCode(err, tron.CodeContractBadMetadata), "err = %v", err)
	})

	t.Run("Rewards propagates the node failure", func(t *testing.T) {
		f := &fakeWalletServer{}
		f.failWith("GetRewardInfo", errNodeDown)
		_, err := newTestHandle(t, f).Voting().Rewards(t.Context())
		require.Error(t, err)
	})
}

// TestResourceReadFailures covers the single-RPC read helpers that pass the
// node's error straight through: each must report the failure rather than a
// zero, because a zero Withdrawable or UnstakeSlots reads as "nothing to do".
func TestResourceReadFailures(t *testing.T) {
	for _, tt := range []struct {
		name string
		rpc  string
		op   string
		call func(*Handle) error
	}{
		{"Resources.State", "GetAccountResource", "get account resource", func(h *Handle) error { _, e := h.Resources().State(t.Context()); return e }},
		{"Withdrawable", "GetCanWithdrawUnfreezeAmount", "get can withdraw unfreeze amount", func(h *Handle) error { _, e := h.Resources().Withdrawable(t.Context()); return e }},
		{"UnstakeSlots", "GetAvailableUnfreezeCount", "get available unfreeze count", func(h *Handle) error { _, e := h.Resources().UnstakeSlots(t.Context()); return e }},
		{"Delegatable", "GetCanDelegatedMaxSize", "get can delegated max size", func(h *Handle) error { _, e := h.Resources().Delegatable(t.Context(), tx.ResourceEnergy); return e }},
		{"DelegationsGrantedTo", "GetDelegatedResourceV2", "get delegated resource v2", func(h *Handle) error {
			_, e := h.Resources().DelegationsGrantedTo(t.Context(), testTo)
			return e
		}},
		{"DelegationIndex", "GetDelegatedResourceAccountIndexV2", "get delegated resource account index v2", func(h *Handle) error {
			_, e := h.Resources().DelegationIndex(t.Context())
			return e
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeWalletServer{account: richAccount()}
			h := newTestHandle(t, f)
			f.failWith(tt.rpc, errNodeDown)
			err := tt.call(h)
			require.Error(t, err, "%s must surface the node error", tt.name)
			requireNodeFailure(t, err, tt.op)
		})
	}
}
