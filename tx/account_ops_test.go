package tx

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/tron"
)

// --- Stake 2.0 lifecycle ---

func TestBuildFreezeBalanceV2SendsStakeRequest(t *testing.T) {
	var got *core.FreezeBalanceV2Contract
	f := &fakeWalletServer{FreezeV2: func(_ context.Context, in *core.FreezeBalanceV2Contract) (*api.TransactionExtention, error) {
		got = in
		return nativeOpExt(core.Transaction_Contract_FreezeBalanceV2Contract, in), nil
	}}
	cp := newTxTestClient(t, f)
	tx, err := BuildFreezeBalanceV2(t.Context(), cp, testFrom, ResourceEnergy, tron.TRX(100))
	if err != nil {
		t.Fatalf("BuildFreezeBalanceV2: %v", err)
	}
	if got.GetFrozenBalance() != 100_000_000 || got.GetResource() != core.ResourceCode_ENERGY {
		t.Fatalf("request = %+v, want 100 TRX for ENERGY", got)
	}
	if tx.Kind() != KindNative {
		t.Fatalf("kind = %v, want native", tx.Kind())
	}
	if ct, ok := contractTypeOf(tx); !ok || ct != core.Transaction_Contract_FreezeBalanceV2Contract {
		t.Fatalf("wrapped contract = %v, want FreezeBalanceV2Contract", ct)
	}
}

func TestBuildFreezeBalanceV2Validates(t *testing.T) {
	cp := newTxTestClient(t, &fakeWalletServer{})
	cases := map[string]struct {
		owner  tron.Address
		res    Resource
		amount tron.SUN
		code   tron.Code
	}{
		"zero address": {tron.Address{}, ResourceEnergy, tron.TRX(1), tron.CodeAddressInvalid},
		"bad resource": {testFrom, Resource(7), tron.TRX(1), tron.CodeTxInvalidArgument},
		"zero amount":  {testFrom, ResourceEnergy, 0, tron.CodeAmountInvalid},
		"negative":     {testFrom, ResourceEnergy, -1, tron.CodeAmountNegative},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := BuildFreezeBalanceV2(t.Context(), cp, c.owner, c.res, c.amount); !tron.HasCode(err, c.code) {
				t.Errorf("err = %v, want %v", err, c.code)
			}
		})
	}
	// Unfreeze validates the same way.
	if _, err := BuildUnfreezeBalanceV2(t.Context(), cp, testFrom, Resource(9), tron.TRX(1)); !tron.HasCode(err, tron.CodeTxInvalidArgument) {
		t.Errorf("UnfreezeBalanceV2 bad resource err = %v, want tx.invalid_argument", err)
	}
}

func TestBuildUnfreezeAndWithdrawAndCancel(t *testing.T) {
	cp := newTxTestClient(t, &fakeWalletServer{})
	unfreeze, err := BuildUnfreezeBalanceV2(t.Context(), cp, testFrom, ResourceBandwidth, tron.TRX(5))
	if err != nil {
		t.Fatalf("BuildUnfreezeBalanceV2: %v", err)
	}
	if ct, _ := contractTypeOf(unfreeze); ct != core.Transaction_Contract_UnfreezeBalanceV2Contract {
		t.Errorf("contract = %v, want UnfreezeBalanceV2Contract", ct)
	}
	withdraw, err := BuildWithdrawExpireUnfreeze(t.Context(), cp, testFrom)
	if err != nil {
		t.Fatalf("BuildWithdrawExpireUnfreeze: %v", err)
	}
	if ct, _ := contractTypeOf(withdraw); ct != core.Transaction_Contract_WithdrawExpireUnfreezeContract {
		t.Errorf("contract = %v, want WithdrawExpireUnfreezeContract", ct)
	}
	cancel, err := BuildCancelAllUnfreezeV2(t.Context(), cp, testFrom)
	if err != nil {
		t.Fatalf("BuildCancelAllUnfreezeV2: %v", err)
	}
	if ct, _ := contractTypeOf(cancel); ct != core.Transaction_Contract_CancelAllUnfreezeV2Contract {
		t.Errorf("contract = %v, want CancelAllUnfreezeV2Contract", ct)
	}
	if _, err := BuildWithdrawExpireUnfreeze(t.Context(), cp, tron.Address{}); !tron.HasCode(err, tron.CodeAddressInvalid) {
		t.Errorf("zero owner err = %v, want address.invalid", err)
	}
}

// --- delegation ---

func TestBuildDelegateResourceLockSemantics(t *testing.T) {
	var got *core.DelegateResourceContract
	f := &fakeWalletServer{DelegateFn: func(_ context.Context, in *core.DelegateResourceContract) (*api.TransactionExtention, error) {
		got = in
		return nativeOpExt(core.Transaction_Contract_DelegateResourceContract, in), nil
	}}
	cp := newTxTestClient(t, f)
	if _, err := BuildDelegateResource(t.Context(), cp, testFrom, testTo, ResourceEnergy, tron.TRX(10), DelegateOptions{Lock: true, LockBlocks: 28800}); err != nil {
		t.Fatalf("BuildDelegateResource: %v", err)
	}
	if !got.GetLock() || got.GetLockPeriod() != 28800 {
		t.Fatalf("request = %+v, want lock with 28800 blocks", got)
	}
	if got.GetBalance() != 10_000_000 || got.GetResource() != core.ResourceCode_ENERGY {
		t.Fatalf("request = %+v, want 10 TRX of ENERGY stake", got)
	}

	// The protocol's minimum delegation is 1 TRX of stake.
	if _, err := BuildDelegateResource(t.Context(), cp, testFrom, testTo, ResourceEnergy, 999_999, DelegateOptions{}); !tron.HasCode(err, tron.CodeTxInvalidArgument) {
		t.Errorf("sub-1-TRX delegation err = %v, want tx.invalid_argument", err)
	}
	// Self-delegation, a lock period without lock, and a negative lock all fail.
	if _, err := BuildDelegateResource(t.Context(), cp, testFrom, testFrom, ResourceEnergy, tron.TRX(1), DelegateOptions{}); !tron.HasCode(err, tron.CodeTxInvalidArgument) {
		t.Errorf("self-delegation err = %v, want tx.invalid_argument", err)
	}
	if _, err := BuildDelegateResource(t.Context(), cp, testFrom, testTo, ResourceEnergy, tron.TRX(1), DelegateOptions{LockBlocks: 100}); !tron.HasCode(err, tron.CodeTxInvalidArgument) {
		t.Errorf("lock period without lock err = %v, want tx.invalid_argument", err)
	}
	if _, err := BuildDelegateResource(t.Context(), cp, testFrom, testTo, ResourceEnergy, tron.TRX(1), DelegateOptions{Lock: true, LockBlocks: -1}); !tron.HasCode(err, tron.CodeTxInvalidArgument) {
		t.Errorf("negative lock err = %v, want tx.invalid_argument", err)
	}
	if _, err := BuildUnDelegateResource(t.Context(), cp, testFrom, testFrom, ResourceEnergy, tron.TRX(1)); !tron.HasCode(err, tron.CodeTxInvalidArgument) {
		t.Errorf("self-undelegation err = %v, want tx.invalid_argument", err)
	}
}

func TestDelegationReads(t *testing.T) {
	f := &fakeWalletServer{
		DelegatedRes: func(_ context.Context, in *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
			return &api.DelegatedResourceList{DelegatedResource: []*core.DelegatedResource{{
				FrozenBalanceForBandwidth: 1_000_000,
				FrozenBalanceForEnergy:    2_000_000,
				ExpireTimeForEnergy:       1_800_000_000_000,
			}}}, nil
		},
		DelegationIndex: func(_ context.Context, in *api.BytesMessage) (*core.DelegatedResourceAccountIndex, error) {
			return &core.DelegatedResourceAccountIndex{
				FromAccounts: [][]byte{testFrom.Bytes()},
				ToAccounts:   [][]byte{testTo.Bytes()},
			}, nil
		},
	}
	cp := newTxTestClient(t, f)
	got, err := DelegationsOf(t.Context(), cp, testFrom, testTo)
	if err != nil {
		t.Fatalf("DelegationsOf: %v", err)
	}
	if len(got) != 1 || got[0].Bandwidth != 1_000_000 || got[0].Energy != 2_000_000 {
		t.Fatalf("delegations = %+v", got)
	}
	if got[0].BandwidthExpiresAt.IsZero() != true {
		t.Errorf("unlocked bandwidth must map to the zero time, got %v", got[0].BandwidthExpiresAt)
	}
	if got[0].EnergyExpiresAt.IsZero() {
		t.Error("locked energy delegation must carry its expiry")
	}

	idx, err := DelegationIndexOf(t.Context(), cp, testFrom)
	if err != nil {
		t.Fatalf("DelegationIndexOf: %v", err)
	}
	if len(idx.From) != 1 || idx.From[0] != testFrom || len(idx.To) != 1 || idx.To[0] != testTo {
		t.Fatalf("index = %+v", idx)
	}
}

func TestResourceAndLimitReads(t *testing.T) {
	f := &fakeWalletServer{
		AccountResource: func(_ context.Context, in *core.Account) (*api.AccountResourceMessage, error) {
			return &api.AccountResourceMessage{
				EnergyLimit: 1000, EnergyUsed: 400,
				NetLimit: 500, NetUsed: 100, FreeNetLimit: 600, FreeNetUsed: 200,
				TronPowerLimit: 900, TronPowerUsed: 300,
			}, nil
		},
		CanDelegateMax: func(_ context.Context, in *api.CanDelegatedMaxSizeRequestMessage) (*api.CanDelegatedMaxSizeResponseMessage, error) {
			return &api.CanDelegatedMaxSizeResponseMessage{MaxSize: 7_000_000}, nil
		},
		UnfreezeCount: func(_ context.Context, in *api.GetAvailableUnfreezeCountRequestMessage) (*api.GetAvailableUnfreezeCountResponseMessage, error) {
			return &api.GetAvailableUnfreezeCountResponseMessage{Count: 30}, nil
		},
		CanWithdraw: func(_ context.Context, in *api.CanWithdrawUnfreezeAmountRequestMessage) (*api.CanWithdrawUnfreezeAmountResponseMessage, error) {
			return &api.CanWithdrawUnfreezeAmountResponseMessage{Amount: 4_000_000}, nil
		},
	}
	cp := newTxTestClient(t, f)
	state, err := ResourceStateOf(t.Context(), cp, testFrom)
	if err != nil {
		t.Fatalf("ResourceStateOf: %v", err)
	}
	if state.EnergyAvailable() != 600 || state.BandwidthAvailable() != 800 || state.TronPowerAvailable() != 600 {
		t.Fatalf("state = %+v (avail energy %d bandwidth %d tp %d)", state, state.EnergyAvailable(), state.BandwidthAvailable(), state.TronPowerAvailable())
	}
	if state.String() == "" {
		t.Error("String() must render")
	}
	if got, err := DelegatableOf(t.Context(), cp, testFrom, ResourceEnergy); err != nil || got != 7_000_000 {
		t.Fatalf("DelegatableOf = %v, %v", got, err)
	}
	if got, err := UnfreezeSlotsOf(t.Context(), cp, testFrom); err != nil || got != 30 {
		t.Fatalf("UnfreezeSlotsOf = %v, %v", got, err)
	}
	if got, err := WithdrawableUnfreezeOf(t.Context(), cp, testFrom); err != nil || got != 4_000_000 {
		t.Fatalf("WithdrawableUnfreezeOf = %v, %v", got, err)
	}
}

// --- voting ---

func TestBuildVoteWitnessValidates(t *testing.T) {
	var got *core.VoteWitnessContract
	f := &fakeWalletServer{VoteWitnessFn: func(_ context.Context, in *core.VoteWitnessContract) (*api.TransactionExtention, error) {
		got = in
		return nativeOpExt(core.Transaction_Contract_VoteWitnessContract, in), nil
	}}
	cp := newTxTestClient(t, f)
	other := mustAddr(0x33)
	if _, err := BuildVoteWitness(t.Context(), cp, testFrom, []Vote{{Witness: testTo, Count: 10}, {Witness: other, Count: 5}}); err != nil {
		t.Fatalf("BuildVoteWitness: %v", err)
	}
	if len(got.GetVotes()) != 2 || got.GetVotes()[0].GetVoteCount() != 10 {
		t.Fatalf("request = %+v", got)
	}
	cases := map[string]struct {
		votes []Vote
		code  tron.Code
	}{
		"empty":          {nil, tron.CodeTxInvalidArgument},
		"zero witness":   {[]Vote{{Witness: tron.Address{}, Count: 1}}, tron.CodeAddressInvalid},
		"zero count":     {[]Vote{{Witness: testTo, Count: 0}}, tron.CodeTxInvalidArgument},
		"negative count": {[]Vote{{Witness: testTo, Count: -1}}, tron.CodeTxInvalidArgument},
		"duplicate":      {[]Vote{{Witness: testTo, Count: 1}, {Witness: testTo, Count: 2}}, tron.CodeTxInvalidArgument},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := BuildVoteWitness(t.Context(), cp, testFrom, c.votes); !tron.HasCode(err, c.code) {
				t.Errorf("err = %v, want %v", err, c.code)
			}
		})
	}
}

func TestWithdrawRewardsAndRead(t *testing.T) {
	f := &fakeWalletServer{RewardInfo: func(_ context.Context, in *api.BytesMessage) (*api.NumberMessage, error) {
		return &api.NumberMessage{Num: 12_345}, nil
	}}
	cp := newTxTestClient(t, f)
	tx, err := BuildWithdrawRewards(t.Context(), cp, testFrom)
	if err != nil {
		t.Fatalf("BuildWithdrawRewards: %v", err)
	}
	if ct, _ := contractTypeOf(tx); ct != core.Transaction_Contract_WithdrawBalanceContract {
		t.Errorf("contract = %v, want WithdrawBalanceContract", ct)
	}
	got, err := VotingRewardsOf(t.Context(), cp, testFrom)
	if err != nil || got != 12_345 {
		t.Fatalf("VotingRewardsOf = %v, %v", got, err)
	}
}

// --- permission bitmap ---

func TestOperationsBitmapRoundTrip(t *testing.T) {
	bitmap, err := OperationsBitmap(TypeTransfer, TypeVoteWitness, TypeFreezeBalanceV2)
	if err != nil {
		t.Fatalf("OperationsBitmap: %v", err)
	}
	if len(bitmap) != 32 {
		t.Fatalf("bitmap length = %d, want 32", len(bitmap))
	}
	// Contract id n is bit (n%8) of byte (n/8): each requested id must be set.
	for _, id := range []int{1, 4, 54} {
		if bitmap[id/8]&(1<<(id%8)) == 0 {
			t.Fatalf("bitmap %x does not enable contract type %d", bitmap, id)
		}
	}
	list, err := OperationsList(bitmap)
	if err != nil {
		t.Fatalf("OperationsList: %v", err)
	}
	want := []ContractType{TypeTransfer, TypeVoteWitness, TypeFreezeBalanceV2}
	if len(list) != len(want) {
		t.Fatalf("decoded %v, want %v", list, want)
	}
	for i := range want {
		if list[i] != want[i] {
			t.Fatalf("decoded %v, want %v", list, want)
		}
	}
	if TypeFreezeBalanceV2.String() != "FreezeBalanceV2Contract" {
		t.Errorf("String() = %q", TypeFreezeBalanceV2.String())
	}
	if ContractType(200).String() != "ContractType(200)" {
		t.Errorf("unknown String() = %q", ContractType(200).String())
	}
	if _, err := OperationsBitmap(ContractType(300)); !tron.HasCode(err, tron.CodeTxInvalidArgument) {
		t.Errorf("out-of-range id err = %v, want tx.invalid_argument", err)
	}
	if _, err := OperationsList([]byte{1, 2}); !tron.HasCode(err, tron.CodeTxInvalidArgument) {
		t.Errorf("short bitmap err = %v, want tx.invalid_argument", err)
	}
}

// --- permission set ---

// twoOfThree returns a valid 2-of-3 owner set with one active permission,
// the shape the protocol documentation uses.
func twoOfThree(t *testing.T, bitmap []byte) PermissionSet {
	t.Helper()
	return PermissionSet{
		Owner: Permission{
			Name:      "owner",
			Threshold: 2,
			Keys: []PermissionKey{
				{Address: testFrom, Weight: 1},
				{Address: testTo, Weight: 1},
				{Address: mustAddr(0x33), Weight: 1},
			},
		},
		Actives: []Permission{{
			Name:       "active0",
			Threshold:  1,
			Keys:       []PermissionKey{{Address: testFrom, Weight: 1}},
			Operations: bitmap,
		}},
	}
}

func TestBuildAccountPermissionUpdateHappyPath(t *testing.T) {
	var got *core.AccountPermissionUpdateContract
	f := &fakeWalletServer{PermissionUpd: func(_ context.Context, in *core.AccountPermissionUpdateContract) (*api.TransactionExtention, error) {
		got = in
		return nativeOpExt(core.Transaction_Contract_AccountPermissionUpdateContract, in), nil
	}}
	cp := newTxTestClient(t, f)
	bitmap, err := OperationsBitmap(TypeTransfer, TypeTriggerSmartContract)
	if err != nil {
		t.Fatalf("OperationsBitmap: %v", err)
	}
	tx, err := BuildAccountPermissionUpdate(t.Context(), cp, testFrom, twoOfThree(t, bitmap))
	if err != nil {
		t.Fatalf("BuildAccountPermissionUpdate: %v", err)
	}
	if ct, _ := contractTypeOf(tx); ct != core.Transaction_Contract_AccountPermissionUpdateContract {
		t.Fatalf("contract = %v", ct)
	}
	if got.GetOwner().GetThreshold() != 2 || len(got.GetOwner().GetKeys()) != 3 {
		t.Fatalf("owner = %+v", got.GetOwner())
	}
	if len(got.GetActives()) != 1 || got.GetActives()[0].GetId() != 2 {
		t.Fatalf("actives = %+v", got.GetActives())
	}
	if !bytes.Equal(got.GetActives()[0].GetOperations(), bitmap) {
		t.Error("active operations bitmap was not forwarded")
	}
	if got.GetWitness() != nil {
		t.Error("a non-SR update must not carry a witness permission")
	}
}

func TestBuildAccountPermissionUpdateValidates(t *testing.T) {
	cp := newTxTestClient(t, &fakeWalletServer{})
	bitmap, _ := OperationsBitmap(TypeTransfer)

	base := func() PermissionSet { return twoOfThree(t, bitmap) }
	cases := map[string]func(s *PermissionSet){
		"no actives": func(s *PermissionSet) { s.Actives = nil },
		"nine actives": func(s *PermissionSet) {
			for len(s.Actives) < 9 {
				s.Actives = append(s.Actives, s.Actives[0])
			}
		},
		"threshold > sum": func(s *PermissionSet) { s.Owner.Threshold = 10 },
		"zero weight":     func(s *PermissionSet) { s.Owner.Keys[0].Weight = 0 },
		"duplicate key":   func(s *PermissionSet) { s.Owner.Keys[1].Address = s.Owner.Keys[0].Address },
		"too many keys":   func(s *PermissionSet) { s.Owner.Keys = append(s.Owner.Keys, s.Owner.Keys[0], s.Owner.Keys[0]) },
		"zero threshold":  func(s *PermissionSet) { s.Owner.Threshold = 0 },
		"empty keys":      func(s *PermissionSet) { s.Owner.Keys = nil },
		"long name":       func(s *PermissionSet) { s.Owner.Name = string(make([]byte, 33)) },
		"wrong active id": func(s *PermissionSet) { s.Actives[0].ID = 5 },
		"short bitmap":    func(s *PermissionSet) { s.Actives[0].Operations = []byte{1} },
		"missing bitmap":  func(s *PermissionSet) { s.Actives[0].Operations = nil },
		"two-key witness": func(s *PermissionSet) {
			s.Witness = &Permission{Threshold: 1, Keys: []PermissionKey{{Address: testFrom, Weight: 1}, {Address: testTo, Weight: 1}}}
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			set := base()
			mutate(&set)
			if _, err := BuildAccountPermissionUpdate(t.Context(), cp, testFrom, set); !tron.HasCode(err, tron.CodeTxInvalidArgument) {
				t.Errorf("err = %v, want tx.invalid_argument", err)
			}
		})
	}
	if _, err := BuildAccountPermissionUpdate(t.Context(), cp, tron.Address{}, base()); !tron.HasCode(err, tron.CodeAddressInvalid) {
		t.Errorf("zero owner err = %v, want address.invalid", err)
	}
	// A one-key witness permission is accepted.
	set := base()
	set.Witness = &Permission{Name: "witness", Threshold: 1, Keys: []PermissionKey{{Address: testFrom, Weight: 1}}}
	if _, err := BuildAccountPermissionUpdate(t.Context(), cp, testFrom, set); err != nil {
		t.Errorf("valid witness permission rejected: %v", err)
	}
}

// TestNativeBuildersArePortable: every new native builder must round-trip
// through the portable envelope, which is what makes offline signing of a
// stake or a permission update possible.
func TestNativeBuildersArePortable(t *testing.T) {
	cp := newTxTestClient(t, &fakeWalletServer{})
	bitmap, _ := OperationsBitmap(TypeTransfer)
	other := mustAddr(0x33)

	builds := map[string]func() (Tx, error){
		"freeze": func() (Tx, error) {
			return BuildFreezeBalanceV2(t.Context(), cp, testFrom, ResourceEnergy, tron.TRX(1))
		},
		"unfreeze": func() (Tx, error) {
			return BuildUnfreezeBalanceV2(t.Context(), cp, testFrom, ResourceEnergy, tron.TRX(1))
		},
		"withdraw expire": func() (Tx, error) { return BuildWithdrawExpireUnfreeze(t.Context(), cp, testFrom) },
		"cancel unfreeze": func() (Tx, error) { return BuildCancelAllUnfreezeV2(t.Context(), cp, testFrom) },
		"delegate": func() (Tx, error) {
			return BuildDelegateResource(t.Context(), cp, testFrom, other, ResourceEnergy, tron.TRX(1), DelegateOptions{})
		},
		"undelegate": func() (Tx, error) {
			return BuildUnDelegateResource(t.Context(), cp, testFrom, other, ResourceEnergy, tron.TRX(1))
		},
		"vote": func() (Tx, error) {
			return BuildVoteWitness(t.Context(), cp, testFrom, []Vote{{Witness: testTo, Count: 1}})
		},
		"withdraw rewards": func() (Tx, error) { return BuildWithdrawRewards(t.Context(), cp, testFrom) },
		"permission update": func() (Tx, error) {
			return BuildAccountPermissionUpdate(t.Context(), cp, testFrom, twoOfThree(t, bitmap))
		},
	}
	for name, build := range builds {
		t.Run(name, func(t *testing.T) {
			orig, err := build()
			if err != nil {
				t.Fatalf("build: %v", err)
			}
			blob, err := Encode(orig)
			if err != nil {
				t.Fatalf("Encode: %v", err)
			}
			back, err := Decode(blob)
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
			if back.ID() != orig.ID() || back.Kind() != KindNative {
				t.Fatalf("round trip: id %s/%s kind %v", back.ID(), orig.ID(), back.Kind())
			}
		})
	}
}

// TestErrorActionIsFixCallForBuilderValidation pins the recovery contract:
// a builder validation error is fix_call, so an agent does not retry it.
func TestErrorActionIsFixCallForBuilderValidation(t *testing.T) {
	cp := newTxTestClient(t, &fakeWalletServer{})
	_, err := BuildFreezeBalanceV2(t.Context(), cp, testFrom, ResourceEnergy, 0)
	var te *tron.Error
	if !errors.As(err, &te) {
		t.Fatalf("want *tron.Error, got %T", err)
	}
	if te.Action() != tron.ActionFixCall {
		t.Errorf("Action() = %v, want fix_call", te.Action())
	}
}
