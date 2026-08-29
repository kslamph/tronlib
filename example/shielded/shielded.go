package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client/lowlevel"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/kslamph/tronlib/pkg/utils"
)

// This file holds the on-chain plumbing: reading contract state, building call
// data, scanning for notes and choosing which notes to spend. Everything here
// goes through the low-level wrappers in pkg/client/lowlevel.

// ---------------------------------------------------------------------------
// chain head
// ---------------------------------------------------------------------------

func currentBlock(ctx context.Context, s *session) (int64, error) {
	blk, err := lowlevel.GetNowBlock2(s.cli, ctx, &api.EmptyMessage{})
	if err != nil {
		return 0, fmt.Errorf("getnowblock2: %w", err)
	}
	return blk.GetBlockHeader().GetRawData().GetNumber(), nil
}

// ---------------------------------------------------------------------------
// call data
// ---------------------------------------------------------------------------

// calldata assembles the `data` field of a TriggerSmartContract: the 4-byte
// method selector followed by already ABI-encoded arguments.
//
// The HTTP API spells the same bytes as function_selector + parameter; the
// gRPC API only has data. The node's trigger_contract_input is the parameter
// half, so building a transaction is selector + trigger_contract_input.
func calldata(signature string, args ...[]byte) []byte {
	data := utils.EncodeMethodSignature(signature)
	for _, a := range args {
		data = append(data, a...)
	}
	return data
}

// revertSelectors emitted by solc for bare reverts and panics.
var (
	errorSelector = []byte{0x08, 0xc3, 0x79, 0xa0} // Error(string)
	panicSelector = []byte{0x4e, 0x48, 0x7b, 0x71} // Panic(uint256)
)

// decodeHex decodes a hex blob returned by the node, tolerating a 0x prefix.
func decodeHex(s, what string) ([]byte, error) {
	raw, err := hex.DecodeString(strings.TrimPrefix(strings.TrimSpace(s), "0x"))
	if err != nil {
		return nil, fmt.Errorf("%s is not valid hex: %w", what, err)
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("%s is empty", what)
	}
	return raw, nil
}

// abiUint256 left-pads an integer into a single 32-byte ABI word. A value that
// does not fit is an error rather than a truncation: silently dropping the
// amount word from approve calldata would authorise something other than what
// was asked for.
func abiUint256(what string, v *big.Int) ([]byte, error) {
	b := v.Bytes()
	if len(b) > 32 {
		return nil, fmt.Errorf("%s (%s) does not fit in 32 bytes", what, v)
	}
	out := make([]byte, 32)
	copy(out[32-len(b):], b)
	return out, nil
}

// callConstant executes a read-only method and returns its raw ABI return data.
//
// A revert has to be detected two ways. java-tron sometimes reports
// result.result == false with a message, and sometimes returns the Solidity
// revert payload in constant_result while leaving result.result true. The
// second case is the dangerous one: an Error(string) blob is 132 bytes, so a
// caller expecting 32 bytes would happily read the selector as the value.
func callConstant(ctx context.Context, s *session, owner, contract *types.Address, data []byte) ([]byte, error) {
	res, err := lowlevel.TriggerConstantContract(s.cli, ctx, &core.TriggerSmartContract{
		OwnerAddress:    owner.Bytes(),
		ContractAddress: contract.Bytes(),
		Data:            data,
		CallValue:       0,
	})
	if err != nil {
		return nil, err
	}
	if !res.GetResult().GetResult() {
		return nil, fmt.Errorf("reverted: %s", bytes.TrimSpace(res.GetResult().GetMessage()))
	}

	var out []byte
	for _, r := range res.GetConstantResult() {
		out = append(out, r...)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("empty constant result")
	}
	if reason, ok := revertReason(out); ok {
		return nil, fmt.Errorf("reverted: %s", reason)
	}
	return out, nil
}

// revertReason reports whether ABI return data is actually a Solidity revert
// blob: Error(string) or Panic(uint256).
func revertReason(data []byte) (string, bool) {
	if len(data) < 4 {
		return "", false
	}
	switch {
	case bytes.HasPrefix(data, errorSelector):
		// Error(string): offset, length, then the bytes. The length word is an
		// untrusted uint256 from the node, so bound it against the buffer before
		// converting; a truncated conversion would slice past the end of data.
		if len(data) < 68 {
			return "unspecified contract error", true
		}
		raw := new(big.Int).SetBytes(data[36:68])
		limit := big.NewInt(int64(len(data) - 68))
		if raw.Sign() <= 0 || raw.Cmp(limit) > 0 {
			return "unspecified contract error", true
		}
		// Safe: raw is between 1 and len(data)-68, so it fits an int exactly.
		n := raw.Int64()
		return string(data[68 : 68+n]), true
	case bytes.HasPrefix(data, panicSelector):
		if len(data) >= 36 {
			return fmt.Sprintf("panic code %s", new(big.Int).SetBytes(data[4:36])), true
		}
		return "contract panic", true
	}
	return "", false
}

// scalingFactor reads the shielded contract's scalingFactor, which is fixed at
// deployment as 10**scalingFactorExponent.
//
// It is a public getter, so there is no reason to hardcode it: getting it wrong
// silently misprices every note. The contract enforces
// rawValue % scalingFactor == 0 in rawValueToValue.
func scalingFactor(ctx context.Context, s *session, owner, contract *types.Address) (*big.Int, error) {
	data, err := callConstant(ctx, s, owner, contract, calldata("scalingFactor()"))
	if err != nil {
		return nil, fmt.Errorf("scalingFactor(): %w", err)
	}
	if len(data) < 32 {
		return nil, fmt.Errorf("scalingFactor(): got %d bytes, want 32", len(data))
	}
	sf := new(big.Int).SetBytes(data[:32])
	if sf.Sign() <= 0 {
		return nil, fmt.Errorf("scalingFactor(): got %s, want a positive value", sf)
	}
	return sf, nil
}

// getPath returns the merkle anchor and the 32 sibling hashes for a leaf.
//
// The contract requires position < leafCount, so a failure here is the real
// "this note is not spendable yet" signal: the leaf has not been appended to
// the tree. There is no point guessing with a confirmation count.
func getPath(ctx context.Context, s *session, owner, contract *types.Address, position int64) (root, path []byte, err error) {
	arg, err := abiUint256("position", big.NewInt(position))
	if err != nil {
		return nil, nil, err
	}
	data, err := callConstant(ctx, s, owner, contract, calldata("getPath(uint256)", arg))
	if err != nil {
		return nil, nil, fmt.Errorf("getPath(%d): %w", position, err)
	}
	// The return type is (bytes32, bytes32[32]), both statically sized, so the
	// encoding is 32 + 1024 inline bytes with no offset head.
	const want = 32 + 32*32
	if len(data) != want {
		return nil, nil, fmt.Errorf("getPath(%d): got %d return bytes, want %d", position, len(data), want)
	}
	return data[:32], data[32:], nil
}

// randomScalar fetches a fresh 32-byte field element.
//
// The node API reuses GetRcm for two different roles: rcm, the randomness that
// blinds a note commitment, and alpha, the randomness behind a spend's rk and
// nullifier.
func randomScalar(ctx context.Context, s *session, what string) ([]byte, error) {
	resp, err := lowlevel.GetRcm(s.cli, ctx, &api.EmptyMessage{})
	if err != nil {
		return nil, fmt.Errorf("getrcm for %s: %w", what, err)
	}
	if len(resp.GetValue()) != 32 {
		return nil, fmt.Errorf("getrcm for %s returned %d bytes, want 32", what, len(resp.GetValue()))
	}
	return resp.GetValue(), nil
}

// ---------------------------------------------------------------------------
// scanning
// ---------------------------------------------------------------------------

// scanChunk is the block window submitted per RPC. Nodes cap the range a single
// scan may cover, so longer histories are walked in windows and merged.
const scanChunk = 1000

// scanNotesByIvk decrypts every note payable to ivk over [begin, end].
//
// IVK is the right key for finding notes you can spend: it covers payments from
// anyone, including change you sent yourself.
func scanNotesByIvk(ctx context.Context, s *session, contract *types.Address, kb *keyBundle, begin, end int64) ([]*api.DecryptNotesTRC20_NoteTx, error) {
	var found []*api.DecryptNotesTRC20_NoteTx
	for _, batch := range blockWindows(begin, end) {
		resp, err := lowlevel.ScanShieldedTRC20NotesByIvk(s.cli, ctx, &api.IvkDecryptTRC20Parameters{
			StartBlockIndex:               batch[0],
			EndBlockIndex:                 batch[1],
			Shielded_TRC20ContractAddress: contract.Bytes(),
			Ivk:                           kb.ivk,
			Ak:                            kb.ak,
			Nk:                            kb.nk,
		})
		if err != nil {
			return nil, fmt.Errorf("scanshieldedtrc20notesbyivk %d-%d: %w", batch[0], batch[1], err)
		}
		found = append(found, resp.GetNoteTxs()...)
	}
	return found, nil
}

// scanNotesByOvk decrypts notes you sent out of your own address, including the
// change outputs of your transactions.
//
// OVK sees a different set than IVK: it recovers outgoing notes even when the
// recipient is someone else. It is not a substitute for IVK when looking for
// notes to spend.
func scanNotesByOvk(ctx context.Context, s *session, contract *types.Address, kb *keyBundle, begin, end int64) ([]*api.DecryptNotesTRC20_NoteTx, error) {
	var found []*api.DecryptNotesTRC20_NoteTx
	for _, batch := range blockWindows(begin, end) {
		resp, err := lowlevel.ScanShieldedTRC20NotesByOvk(s.cli, ctx, &api.OvkDecryptTRC20Parameters{
			StartBlockIndex:               batch[0],
			EndBlockIndex:                 batch[1],
			Shielded_TRC20ContractAddress: contract.Bytes(),
			Ovk:                           kb.ovk,
		})
		if err != nil {
			return nil, fmt.Errorf("scanshieldedtrc20notesbyovk %d-%d: %w", batch[0], batch[1], err)
		}
		found = append(found, resp.GetNoteTxs()...)
	}
	return found, nil
}

// blockWindows splits the inclusive range [begin, end] into non-overlapping
// windows of at most scanChunk blocks.
//
// The windows must not overlap. A note lives in exactly one block, so scanning
// a boundary block twice would report the same note twice, and a duplicated note
// could then be selected twice into one transaction and revert on a duplicate
// nullifier.
func blockWindows(begin, end int64) [][2]int64 {
	var windows [][2]int64
	for start := begin; start <= end; start += scanChunk {
		stop := start + scanChunk - 1
		if stop > end {
			stop = end
		}
		windows = append(windows, [2]int64{start, stop})
	}
	return windows
}

// noteIsSpent asks the node whether this note's nullifier is already on chain.
//
// The scan response carries an is_spent flag; this is the targeted check for a
// single note, and it recomputes the nullifier from ak, nk and the position.
func noteIsSpent(ctx context.Context, s *session, contract *types.Address, note *api.Note, kb *keyBundle, position int64) (bool, error) {
	resp, err := lowlevel.IsShieldedTRC20ContractNoteSpent(s.cli, ctx, &api.NfTRC20Parameters{
		Note:                          note,
		Ak:                            kb.ak,
		Nk:                            kb.nk,
		Position:                      position,
		Shielded_TRC20ContractAddress: contract.Bytes(),
	})
	if err != nil {
		return false, fmt.Errorf("isshieldedtrc20contractnotespent at position %d: %w", position, err)
	}
	return resp.GetIsSpent(), nil
}

// ---------------------------------------------------------------------------
// note selection
// ---------------------------------------------------------------------------

// spendableNotes returns the notes a transaction could actually spend: the
// unspent ones, oldest leaf first, since those are the notes most likely
// already appended to the merkle tree.
func spendableNotes(notes []*api.DecryptNotesTRC20_NoteTx) []*api.DecryptNotesTRC20_NoteTx {
	var out []*api.DecryptNotesTRC20_NoteTx
	for _, n := range notes {
		if n.GetIsSpent() || n.GetNote() == nil {
			continue
		}
		out = append(out, n)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].GetPosition() < out[j].GetPosition() })
	return out
}

// notePlan is the outcome of note selection: which notes to spend and the value
// of the change note they leave behind. Selection is pure and silent so it can
// be tested; callers report the plan before they broadcast anything.
type notePlan struct {
	spends []*api.DecryptNotesTRC20_NoteTx
	change *big.Int
}

// describe renders the plan as one line, e.g.
//
//	burn plan: spend position 16 (value 6000000), to_amount 4000000 scaled, change 2000000 scaled
func (p notePlan) describe(verb, needLabel string, need *big.Int) string {
	positions := make([]string, 0, len(p.spends))
	values := make([]string, 0, len(p.spends))
	for _, n := range p.spends {
		positions = append(positions, fmt.Sprintf("%d", n.GetPosition()))
		values = append(values, fmt.Sprintf("%d", n.GetNote().GetValue()))
	}
	return fmt.Sprintf("%s plan: spend position %s (value %s), %s %s scaled, change %s scaled",
		verb, strings.Join(positions, "+"), strings.Join(values, "+"), needLabel, need, p.change)
}

// smallestCover returns the single unspent note whose value covers need with the
// least remainder, so the change note is as small as possible.
//
// "Smallest remainder", not "first sufficient": taking the biggest note because
// it was minted first fragments the balance for no reason.
func smallestCover(unspent []*api.DecryptNotesTRC20_NoteTx, need *big.Int) (*api.DecryptNotesTRC20_NoteTx, *big.Int) {
	var best *api.DecryptNotesTRC20_NoteTx
	var bestChange *big.Int
	for _, n := range unspent {
		value := big.NewInt(n.GetNote().GetValue())
		if value.Cmp(need) < 0 {
			continue
		}
		change := new(big.Int).Sub(value, need)
		if bestChange == nil || change.Cmp(bestChange) < 0 {
			best, bestChange = n, change
		}
	}
	return best, bestChange
}

// bestPair covers need with exactly two notes, minimising the remainder.
func bestPair(unspent []*api.DecryptNotesTRC20_NoteTx, need *big.Int) ([]*api.DecryptNotesTRC20_NoteTx, *big.Int) {
	var pair []*api.DecryptNotesTRC20_NoteTx
	var pairChange *big.Int
	for i := 0; i < len(unspent); i++ {
		for j := i + 1; j < len(unspent); j++ {
			sum := new(big.Int).Add(big.NewInt(unspent[i].GetNote().GetValue()),
				big.NewInt(unspent[j].GetNote().GetValue()))
			if sum.Cmp(need) < 0 {
				continue
			}
			change := new(big.Int).Sub(sum, need)
			if pairChange == nil || change.Cmp(pairChange) < 0 {
				pair, pairChange = []*api.DecryptNotesTRC20_NoteTx{unspent[i], unspent[j]}, change
			}
		}
	}
	return pair, pairChange
}

// planBurn chooses the single note to redeem.
//
// burn takes exactly one input and at most one output (ShieldedTRC20.sol:146),
// so the smallest note that covers the withdrawal is picked and the remainder
// comes back as change. The value equation the contract proves is
//
//	spend.value * sf == change.value * sf + to_amount
//
// A note that matches exactly yields zero change and no output at all.
func planBurn(notes []*api.DecryptNotesTRC20_NoteTx, toAmountScaled *big.Int) (notePlan, error) {
	unspent := spendableNotes(notes)
	note, change := smallestCover(unspent, toAmountScaled)
	if note == nil {
		return notePlan{}, fmt.Errorf(
			"no single unspent note covers %s scaled units; burn redeems one note at a time, "+
				"so mint a note of the right size or use -mode=transfer to consolidate first", toAmountScaled)
	}
	return notePlan{spends: []*api.DecryptNotesTRC20_NoteTx{note}, change: change}, nil
}

// planTransfer chooses the notes to spend for a shielded-to-shielded payment.
//
// transfer takes one or two inputs and one or two outputs
// (ShieldedTRC20.sol:87-89). A payment plus change fills both output slots, so
// at most two inputs can ever be consolidated in one transaction.
//
// Selection minimises the change returned, because change is not free: it costs
// an extra output proof now and leaves a smaller, more fragmented balance later.
func planTransfer(notes []*api.DecryptNotesTRC20_NoteTx, needScaled *big.Int) (notePlan, error) {
	unspent := spendableNotes(notes)

	// A single note is preferred: it leaves an output slot free and costs less
	// energy.
	if note, change := smallestCover(unspent, needScaled); note != nil {
		return notePlan{spends: []*api.DecryptNotesTRC20_NoteTx{note}, change: change}, nil
	}

	// No single note suffices, so combine two. Check every pair and keep the one
	// whose sum covers the payment with the least remainder.
	if pair, change := bestPair(unspent, needScaled); pair != nil {
		return notePlan{spends: pair, change: change}, nil
	}

	total := new(big.Int)
	for _, n := range unspent {
		total.Add(total, big.NewInt(n.GetNote().GetValue()))
	}
	return notePlan{}, fmt.Errorf("insufficient shielded balance: %d unspent notes totalling %s scaled units, need %s "+
		"(a single transaction can combine at most two notes)", len(unspent), total, needScaled)
}

// maxNoteValue is the largest note value this example will create, or return as
// change.
//
// The contract's own ceiling is value < INT64_MAX (ShieldedTRC20.sol:269). This
// example halves it for two reasons: transfer can combine two notes, so keeping
// each below 2**62 means their sum stays inside int64; and notes minted by
// someone else can still arrive near INT64_MAX, so noteValue re-checks every
// value before it goes into a note field rather than trusting the cap.
const maxNoteValue = int64(1) << 62

// toScaled converts a raw token amount into the note-space value the proofs
// operate on, and rejects amounts the contract would refuse.
//
// The contract requires the raw amount to be an exact multiple of
// scalingFactor, so a remainder is a hard error rather than a silent truncation.
func toScaled(rawAmount, sf *big.Int) (*big.Int, error) {
	if new(big.Int).Mod(rawAmount, sf).Sign() != 0 {
		return nil, fmt.Errorf("amount %s is not a multiple of the contract scalingFactor %s", rawAmount, sf)
	}
	scaled := new(big.Int).Div(rawAmount, sf)
	if !scaled.IsInt64() || scaled.Int64() >= maxNoteValue {
		return nil, fmt.Errorf("amount %s scales to %s, which is at or above the %s note ceiling this example allows",
			rawAmount, scaled, big.NewInt(maxNoteValue))
	}
	return scaled, nil
}

// noteValue turns a plan amount into the int64 a note field holds, refusing
// anything outside the range rather than wrapping it into a negative note.
func noteValue(what string, v *big.Int) (int64, error) {
	if !v.IsInt64() || v.Int64() < 0 || v.Int64() >= maxNoteValue {
		return 0, fmt.Errorf("%s is %s, which does not fit a note value (0 <= v < %s)",
			what, v, big.NewInt(maxNoteValue))
	}
	return v.Int64(), nil
}
