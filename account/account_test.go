package account

import (
	"testing"
	"time"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/tron"
	"github.com/kslamph/tronlib/v2/tx"
)

// transferBitmap is the operations bitmap the canned active permission
// carries: TRX transfers and contract calls, nothing else.
var transferBitmap = func() []byte {
	b, err := tx.OperationsBitmap(tx.TypeTransfer, tx.TypeTriggerSmartContract)
	if err != nil {
		panic(err)
	}
	return b
}()

// richAccount is the canned GetAccount answer the decode tests read.
func richAccount() *core.Account {
	expired := time.Now().Add(-time.Hour).UnixMilli()
	return &core.Account{
		Address:     testFrom.Bytes(),
		Balance:     5_000_000,
		AccountName: []byte("treasury"),
		CreateTime:  1_700_000_000_000,
		Votes:       []*core.Vote{{VoteAddress: testTo.Bytes(), VoteCount: 100}},
		FrozenV2: []*core.Account_FreezeV2{
			{Type: core.ResourceCode_BANDWIDTH, Amount: 2_000_000},
			{Type: core.ResourceCode_ENERGY, Amount: 3_000_000},
		},
		UnfrozenV2: []*core.Account_UnFreezeV2{
			{Type: core.ResourceCode_ENERGY, UnfreezeAmount: 1_000_000, UnfreezeExpireTime: expired},
		},
		DelegatedFrozenV2BalanceForBandwidth:         500_000,
		AcquiredDelegatedFrozenV2BalanceForBandwidth: 250_000,
		AccountResource: &core.Account_AccountResource{
			DelegatedFrozenV2BalanceForEnergy:         400_000,
			AcquiredDelegatedFrozenV2BalanceForEnergy: 300_000,
		},
		OwnerPermission: &core.Permission{
			Id: 0, PermissionName: "owner", Threshold: 2,
			Keys: []*core.Key{{Address: testFrom.Bytes(), Weight: 1}, {Address: testTo.Bytes(), Weight: 1}},
		},
		WitnessPermission: &core.Permission{
			Id: 1, PermissionName: "witness", Threshold: 1,
			Keys: []*core.Key{{Address: testFrom.Bytes(), Weight: 1}},
		},
		ActivePermission: []*core.Permission{{
			Id: 2, PermissionName: "active0", Threshold: 1,
			Keys:       []*core.Key{{Address: testHot.Bytes(), Weight: 1}},
			Operations: transferBitmap,
		}},
	}
}

func TestStateDecodesAccount(t *testing.T) {
	acc := richAccount()
	h := newTestHandle(t, &fakeWalletServer{account: acc})
	st, err := h.State(t.Context())
	if err != nil {
		t.Fatalf("State: %v", err)
	}
	if !st.Exists || st.Balance != 5_000_000 || st.Name != "treasury" {
		t.Fatalf("state = %+v", st)
	}
	if st.CreatedAt.UnixMilli() != 1_700_000_000_000 {
		t.Errorf("CreatedAt = %v", st.CreatedAt)
	}
	if len(st.Votes) != 1 || st.Votes[0].Witness != testTo || st.Votes[0].Count != 100 {
		t.Errorf("Votes = %+v", st.Votes)
	}
	if len(st.Stakes) != 2 || st.Stakes[1].Resource != tx.ResourceEnergy || st.Stakes[1].Amount != 3_000_000 {
		t.Errorf("Stakes = %+v", st.Stakes)
	}
	if len(st.Unstakes) != 1 || st.Unstakes[0].Amount != 1_000_000 ||
		st.Unstakes[0].ExpiresAt.UnixMilli() != acc.UnfrozenV2[0].GetUnfreezeExpireTime() {
		t.Errorf("Unstakes = %+v", st.Unstakes)
	}
	if st.DelegatedOutBandwidth != 500_000 || st.DelegatedOutEnergy != 400_000 {
		t.Errorf("delegated out = %d/%d", st.DelegatedOutBandwidth, st.DelegatedOutEnergy)
	}
	if st.DelegatedInBandwidth != 250_000 || st.DelegatedInEnergy != 300_000 {
		t.Errorf("delegated in = %d/%d", st.DelegatedInBandwidth, st.DelegatedInEnergy)
	}
}

func TestStateUnknownAccountIsZeroNotError(t *testing.T) {
	// The fake's default answer is an account shell with no address — the
	// "not created yet" shape. A send to such an address creates it.
	h := newTestHandle(t, &fakeWalletServer{})
	st, err := h.State(t.Context())
	if err != nil {
		t.Fatalf("State: %v", err)
	}
	if st.Exists || st.Balance != 0 {
		t.Fatalf("state = %+v, want non-existent zero account", st)
	}
}

func TestPermissionsCurrentDecodesAllSlots(t *testing.T) {
	h := newTestHandle(t, &fakeWalletServer{account: richAccount()})
	set, err := h.Permissions().Current(t.Context())
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if set.Owner.Threshold != 2 || len(set.Owner.Keys) != 2 || set.Owner.ID != 0 {
		t.Fatalf("owner = %+v", set.Owner)
	}
	if set.Witness == nil || len(set.Witness.Keys) != 1 {
		t.Fatalf("witness = %+v", set.Witness)
	}
	if len(set.Actives) != 1 || set.Actives[0].ID != 2 || len(set.Actives[0].Operations) != 32 {
		t.Fatalf("actives = %+v", set.Actives)
	}
	// The decoded set must be re-submittable: the round trip through the
	// builder is the documented editing pattern.
	if _, err := h.Permissions().Update(t.Context(), set); err != nil {
		t.Fatalf("Update(current): %v", err)
	}
}

func TestPermissionsCurrentRejectsAccountWithoutPermissions(t *testing.T) {
	// A non-existent account has no permission configuration: reading it must
	// say so rather than return an all-zero set that would wipe permissions
	// if submitted.
	h := newTestHandle(t, &fakeWalletServer{})
	if _, err := h.Permissions().Current(t.Context()); !tron.HasCode(err, tron.CodeContractBadMetadata) {
		t.Errorf("Current on a permission-less account err = %v, want contract.bad_metadata", err)
	}
}

func TestSignWeightDecodesThresholdAndEnough(t *testing.T) {
	f := &fakeWalletServer{signWeight: &api.TransactionSignWeight{
		Permission: &core.Permission{
			Id: 0, Threshold: 2,
			Keys: []*core.Key{{Address: testFrom.Bytes(), Weight: 1}, {Address: testTo.Bytes(), Weight: 1}},
		},
		ApprovedList:  [][]byte{testFrom.Bytes()},
		CurrentWeight: 1,
		Result: &api.TransactionSignWeight_Result{
			Code: api.TransactionSignWeight_Result_NOT_ENOUGH_PERMISSION, Message: "not enough",
		},
	}}
	h := newTestHandle(t, f)
	native, err := h.TransferTRX(t.Context(), testTo, tron.TRX(1))
	if err != nil {
		t.Fatalf("TransferTRX: %v", err)
	}
	status, err := h.Permissions().SignWeight(t.Context(), native)
	if err != nil {
		t.Fatalf("SignWeight: %v", err)
	}
	if status.Enough {
		t.Error("Enough must be false at weight 1 of 2")
	}
	if status.Weight != 1 || status.Threshold != 2 || len(status.Approved) != 1 || status.Approved[0] != testFrom {
		t.Fatalf("status = %+v", status)
	}
	if status.Permission == nil || len(status.Permission.Keys) != 2 {
		t.Fatalf("status.Permission = %+v", status.Permission)
	}

	// The sufficient case flips Enough, and the threshold agreement is
	// re-checked locally so a node that reports ENOUGH below threshold is not
	// trusted blindly.
	f.signWeight = &api.TransactionSignWeight{
		Permission:    &core.Permission{Threshold: 2},
		ApprovedList:  [][]byte{testFrom.Bytes(), testTo.Bytes()},
		CurrentWeight: 2,
		Result:        &api.TransactionSignWeight_Result{Code: api.TransactionSignWeight_Result_ENOUGH_PERMISSION},
	}
	status, err = h.Permissions().SignWeight(t.Context(), native)
	if err != nil {
		t.Fatalf("SignWeight(enough): %v", err)
	}
	if !status.Enough || status.Weight != 2 {
		t.Fatalf("status = %+v", status)
	}
	if status.String() == "" {
		t.Error("String() must render")
	}
}

func TestApprovalsSurfacesSignatureErrors(t *testing.T) {
	f := &fakeWalletServer{approved: &api.TransactionApprovedList{
		Result: &api.TransactionApprovedList_Result{
			Code: api.TransactionApprovedList_Result_SIGNATURE_FORMAT_ERROR, Message: "bad signature",
		},
	}}
	h := newTestHandle(t, f)
	native, err := h.TransferTRX(t.Context(), testTo, tron.TRX(1))
	if err != nil {
		t.Fatalf("TransferTRX: %v", err)
	}
	if _, err := h.Permissions().Approvals(t.Context(), native); !tron.HasCode(err, tron.CodeKeyInvalid) {
		t.Errorf("Approvals with a bad signature err = %v, want key.invalid", err)
	}

	f.approved = &api.TransactionApprovedList{
		ApprovedList: [][]byte{testFrom.Bytes()},
		Result:       &api.TransactionApprovedList_Result{Code: api.TransactionApprovedList_Result_SUCCESS},
	}
	got, err := h.Permissions().Approvals(t.Context(), native)
	if err != nil || len(got) != 1 || got[0] != testFrom {
		t.Fatalf("Approvals = %v, %v", got, err)
	}
}

func TestResourcesSummaryFoldsBothReads(t *testing.T) {
	expired := time.Now().Add(-time.Hour).UnixMilli()
	acc := richAccount()
	acc.UnfrozenV2 = append(acc.UnfrozenV2, &core.Account_UnFreezeV2{
		Type: core.ResourceCode_BANDWIDTH, UnfreezeAmount: 2_000_000, UnfreezeExpireTime: expired,
	})
	f := &fakeWalletServer{
		account: acc,
		resources: &api.AccountResourceMessage{
			EnergyLimit: 10_000, EnergyUsed: 4_000,
			NetLimit: 5_000, NetUsed: 1_000, FreeNetLimit: 600, FreeNetUsed: 200,
			TronPowerLimit: 9_000, TronPowerUsed: 3_000,
		},
		canWithdraw: 4_000_000,
		unfreezeCnt: 30,
	}
	h := newTestHandle(t, f)
	s, err := h.Resources().Summary(t.Context())
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if s.StakedByResource[tx.ResourceEnergy] != 3_000_000 || s.StakedByResource[tx.ResourceBandwidth] != 2_000_000 {
		t.Fatalf("StakedByResource = %+v", s.StakedByResource)
	}
	if s.UnstakePending != 3_000_000 {
		t.Errorf("UnstakePending = %d, want 3_000_000", s.UnstakePending)
	}
	// The node's figure (4 TRX) is larger than the locally computed matured
	// total (3 TRX), so the summary prefers the node's — it accounts for
	// unstakes the account read may not list.
	if s.UnstakeWithdrawable != 4_000_000 {
		t.Errorf("UnstakeWithdrawable = %d, want the node's 4_000_000", s.UnstakeWithdrawable)
	}
	if s.UnstakeSlots != 30 || s.TronPowerAvailable != 6_000 {
		t.Errorf("slots/tp = %d/%d", s.UnstakeSlots, s.TronPowerAvailable)
	}
	if s.DelegatedOut != 900_000 || s.DelegatedIn != 550_000 {
		t.Errorf("delegated = %d/%d", s.DelegatedOut, s.DelegatedIn)
	}
}

func TestResourcesReadsAndLimits(t *testing.T) {
	f := &fakeWalletServer{
		resources: &api.AccountResourceMessage{
			EnergyLimit: 10_000, EnergyUsed: 4_000,
			NetLimit: 5_000, NetUsed: 1_000, FreeNetLimit: 600, FreeNetUsed: 200,
			TronPowerLimit: 9_000, TronPowerUsed: 3_000,
		},
		canDelegate: 7_000_000,
		unfreezeCnt: 31,
		canWithdraw: 4_000_000,
	}
	h := newTestHandle(t, f)
	res := h.Resources()

	state, err := res.State(t.Context())
	if err != nil {
		t.Fatalf("State: %v", err)
	}
	if state.EnergyAvailable() != 6_000 || state.BandwidthAvailable() != 4_400 || state.TronPowerAvailable() != 6_000 {
		t.Fatalf("resource state = %+v", state)
	}
	if got, err := res.Delegatable(t.Context(), tx.ResourceEnergy); err != nil || got != 7_000_000 {
		t.Fatalf("Delegatable = %v, %v", got, err)
	}
	if got, err := res.UnstakeSlots(t.Context()); err != nil || got != 31 {
		t.Fatalf("UnstakeSlots = %v, %v", got, err)
	}
	if got, err := res.Withdrawable(t.Context()); err != nil || got != 4_000_000 {
		t.Fatalf("Withdrawable = %v, %v", got, err)
	}
}

func TestResourcesDelegationReadsOrientAccounts(t *testing.T) {
	f := &fakeWalletServer{}
	f.delegIndex = &core.DelegatedResourceAccountIndex{
		FromAccounts: [][]byte{testTo.Bytes()},
		ToAccounts:   [][]byte{testHot.Bytes()},
	}
	h := newTestHandle(t, f)
	res := h.Resources()

	if _, err := res.DelegationsGrantedTo(t.Context(), testTo); err != nil {
		t.Fatalf("DelegationsGrantedTo: %v", err)
	}
	if _, err := res.DelegationsReceivedFrom(t.Context(), testTo); err != nil {
		t.Fatalf("DelegationsReceivedFrom: %v", err)
	}
	idx, err := res.DelegationIndex(t.Context())
	if err != nil {
		t.Fatalf("DelegationIndex: %v", err)
	}
	if len(idx.From) != 1 || idx.From[0] != testTo || len(idx.To) != 1 || idx.To[0] != testHot {
		t.Fatalf("index = %+v", idx)
	}
}

func TestVotingReadsAndBuilds(t *testing.T) {
	f := &fakeWalletServer{account: richAccount(), nextMaint: 1_700_000_000_000}
	h := newTestHandle(t, f)

	votes, err := h.Voting().Votes(t.Context())
	if err != nil || len(votes) != 1 || votes[0].Witness != testTo {
		t.Fatalf("Votes = %v, %v", votes, err)
	}
	if got, err := h.Voting().Rewards(t.Context()); err != nil || got != 42 {
		t.Fatalf("Rewards = %v, %v", got, err)
	}
	tally, err := h.Voting().NextTally(t.Context())
	if err != nil || tally.UnixMilli() != 1_700_000_000_000 {
		t.Fatalf("NextTally = %v, %v", tally, err)
	}
	if _, err := h.Voting().SetVotes(t.Context(), []tx.Vote{{Witness: testTo, Count: 10}}); err != nil {
		t.Fatalf("SetVotes: %v", err)
	}
	if f.gotVote == nil || len(f.gotVote.GetVotes()) != 1 {
		t.Fatalf("vote request = %+v", f.gotVote)
	}
	if _, err := h.Voting().ClaimRewards(t.Context()); err != nil {
		t.Fatalf("ClaimRewards: %v", err)
	}
}

func TestHandleBuildersReachTheNode(t *testing.T) {
	f := &fakeWalletServer{}
	h := newTestHandle(t, f)

	if _, err := h.TransferTRX(t.Context(), testTo, tron.TRX(1)); err != nil {
		t.Fatalf("TransferTRX: %v", err)
	}
	if f.gotCreate.GetAmount() != 1_000_000 {
		t.Errorf("transfer request = %+v", f.gotCreate)
	}
	if _, err := h.TransferToken(t.Context(), testTo, "1000001", 5); err != nil {
		t.Fatalf("TransferToken: %v", err)
	}
	if string(f.gotAsset.GetAssetName()) != "1000001" {
		t.Errorf("asset request = %+v", f.gotAsset)
	}
	if _, err := h.Deploy(t.Context(), tx.DeployParams{Bytecode: []byte{0x60}}); err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if len(f.gotDeploy.GetNewContract().GetBytecode()) == 0 {
		t.Errorf("deploy request = %+v", f.gotDeploy)
	}
	if _, err := h.Resources().Stake(t.Context(), tx.ResourceEnergy, tron.TRX(5)); err != nil {
		t.Fatalf("Stake: %v", err)
	}
	if f.gotFreeze.GetFrozenBalance() != 5_000_000 || f.gotFreeze.GetResource() != core.ResourceCode_ENERGY {
		t.Errorf("stake request = %+v", f.gotFreeze)
	}
}

func TestCostPreviewAndTotalCostUseTheBoundOwner(t *testing.T) {
	f := &fakeWalletServer{account: richAccount(), resources: &api.AccountResourceMessage{EnergyLimit: 0, EnergyUsed: 0}}
	h := newTestHandle(t, f)

	call, err := tx.BuildTriggerSmartContract(t.Context(), mustClient(t, f), testFrom, testTo, nil, 0)
	if err != nil {
		t.Fatalf("BuildTriggerSmartContract: %v", err)
	}
	preview, err := h.CostPreview(t.Context(), call)
	if err != nil {
		t.Fatalf("CostPreview: %v", err)
	}
	if preview.EnergyNeeded != 5000 || preview.TronToBurn != 5000*420 {
		t.Fatalf("preview = %+v", preview)
	}
	if f.gotTrigger == nil || string(f.gotTrigger.GetOwnerAddress()) != string(testFrom.Bytes()) {
		t.Error("CostPreview must simulate as the bound owner")
	}

	signed, err := tx.Sign(call, mustSigner(t, testKeyHex))
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	cost, err := h.TotalCost(t.Context(), signed)
	if err != nil {
		t.Fatalf("TotalCost: %v", err)
	}
	if cost.Total == 0 {
		t.Fatal("TotalCost must produce a non-zero total")
	}
}
