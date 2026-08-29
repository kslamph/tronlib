package main

import (
	"context"
	"fmt"
	"math/big"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/client/lowlevel"
	"github.com/kslamph/tronlib/pkg/types"
)

// The four transaction-building flows. Each one is the same five steps:
//
//	1. read scalingFactor from the contract so amounts are in the right space
//	2. assemble the private parameters (notes to spend, notes to create)
//	3. CreateShieldedContractParameters -> trigger_contract_input + zk proof
//	4. TriggerContract with selector || trigger_contract_input -> raw transaction
//	5. sign locally, broadcast
//
// The node returns a fully formed proof, so step 3 is where all the
// cryptography happens. Steps 4 and 5 are an ordinary TRON smart-contract
// transaction with nothing shielded about them.

// Contract method signatures. The selector is derived from the signature by
// keccak256 rather than hardcoded, so a contract with a different ABI fails
// loudly instead of sending funds somewhere unexpected.
const (
	sigMint     = "mint(uint256,bytes32[9],bytes32[2],bytes32[21])"
	sigTransfer = "transfer(bytes32[10][],bytes32[2][],bytes32[9][],bytes32[2],bytes32[21][])"
	sigBurn     = "burn(bytes32[10],bytes32[2],uint256,bytes32[2],address,bytes32[3],bytes32[9][],bytes32[21][])"
)

// ---------------------------------------------------------------------------
// mint: transparent TRC-20 -> shielded note
// ---------------------------------------------------------------------------

func runMint(ctx context.Context, s *session) error {
	if err := s.needAccount("mint (it pays the fee and holds the TRC-20)"); err != nil {
		return err
	}
	token, err := types.NewAddressFromBase58(s.cfg.token)
	if err != nil {
		return fmt.Errorf("parse token address %q: %w", s.cfg.token, err)
	}
	amount, err := s.cfg.parseAmount()
	if err != nil {
		return err
	}

	keys, err := loadKeys(s.cfg.keyFile)
	if err != nil {
		return err
	}
	kb, err := keys.decode()
	if err != nil {
		return err
	}

	sf, err := scalingFactor(ctx, s, s.owner(), s.contract)
	if err != nil {
		return err
	}
	scaled, err := toScaled(amount, sf)
	if err != nil {
		return err
	}
	fmt.Printf("scalingFactor %s: from_amount %s creates a note worth %s scaled units\n", sf, amount, scaled)

	// The shielded contract pulls the tokens with transferFrom, so it needs an
	// allowance first. This is the TRC-20 prerequisite, not part of the
	// shielded flow itself, and it costs a second transaction when the
	// allowance is not already there.
	if err := ensureAllowance(ctx, s, token, amount); err != nil {
		return err
	}

	note, err := receiveNote(ctx, s, keys.PaymentAddress, scaled, "mint")
	if err != nil {
		return err
	}

	// mint has no spends, so it needs neither ask nor nsk: only ovk, to let the
	// sender recover this note later.
	resp, err := lowlevel.CreateShieldedContractParameters(s.cli, ctx, &api.PrivateShieldedTRC20Parameters{
		Ovk:                           kb.ovk,
		FromAmount:                    amount.String(),
		ShieldedReceives:              []*api.ReceiveNote{note},
		Shielded_TRC20ContractAddress: s.contract.Bytes(),
	})
	if err != nil {
		return fmt.Errorf("createshieldedcontractparameters (mint): %w", err)
	}
	if got := resp.GetParameterType(); got != "mint" {
		return fmt.Errorf("node returned parameter_type %q, expected mint", got)
	}

	return s.submit(ctx, sigMint, resp.GetTriggerContractInput(), "mint")
}

// ensureAllowance grants the shielded contract permission to pull amount tokens.
func ensureAllowance(ctx context.Context, s *session, token *types.Address, amount *big.Int) error {

	// allowance(owner, spender) is a plain TRC-20 view call.
	argOwner := abiAddressWord(s.owner())
	argSpender := abiAddressWord(s.contract)
	res, err := callConstant(ctx, s, s.owner(), token,
		calldata("allowance(address,address)", argOwner, argSpender))
	if err != nil {
		return fmt.Errorf("read allowance: %w", err)
	}
	if len(res) < 32 {
		return fmt.Errorf("allowance(): got %d bytes, want 32", len(res))
	}
	allowed := new(big.Int).SetBytes(res[:32])
	fmt.Printf("current allowance for the shielded contract: %s\n", allowed)

	if allowed.Cmp(amount) >= 0 {
		return nil
	}

	fmt.Printf("approving the shielded contract for %s tokens...\n", amount)
	amtWord, err := abiUint256("amount", amount)
	if err != nil {
		return err
	}
	data := calldata("approve(address,uint256)", argSpender, amtWord)
	return s.submitRaw(ctx, token, data, "approve")
}

// ---------------------------------------------------------------------------
// scan: find owned notes
// ---------------------------------------------------------------------------

func runScan(ctx context.Context, s *session) error {
	keys, err := loadKeys(s.cfg.keyFile)
	if err != nil {
		return err
	}
	kb, err := keys.decode()
	if err != nil {
		return err
	}
	begin, end, err := scanRange(ctx, s, keys)
	if err != nil {
		return err
	}

	var notes []*api.DecryptNotesTRC20_NoteTx
	switch s.cfg.scanBy {
	case "ivk":
		fmt.Printf("scanning blocks %d-%d by ivk (notes received)\n", begin, end)
		notes, err = scanNotesByIvk(ctx, s, s.contract, kb, begin, end)
	case "ovk":
		fmt.Printf("scanning blocks %d-%d by ovk (notes sent, including change)\n", begin, end)
		notes, err = scanNotesByOvk(ctx, s, s.contract, kb, begin, end)
	default:
		return fmt.Errorf("-by must be ivk or ovk, got %q", s.cfg.scanBy)
	}
	if err != nil {
		return err
	}

	fmt.Printf("\n%d note(s)\n", len(notes))
	for i, n := range notes {
		if n.GetNote() == nil {
			// An OVK scan also reports the transparent half of a burn: it has a
			// to_amount and a transparent_to_address but no note.
			fmt.Printf("  %d. transparent output: txid %x, to_amount %s, position %d\n",
				i+1, n.GetTxid(), n.GetToAmount(), n.GetPosition())
			continue
		}
		fmt.Printf("  %d. value %d  position %d  spent=%-5v  txid %x\n",
			i+1, n.GetNote().GetValue(), n.GetPosition(), n.GetIsSpent(), n.GetTxid())
		fmt.Printf("     payment address %s\n", n.GetNote().GetPaymentAddress())
	}

	// Only an IVK scan finds notes this address owns. An OVK scan finds notes it
	// sent, most of which belong to other people, so reporting a "spendable
	// total" or re-checking spend status over those would be meaningless.
	if s.cfg.scanBy != "ivk" {
		if s.cfg.checkSpent {
			fmt.Println("\n-check-spent is ignored for an ovk scan: only an ivk scan finds notes this address owns")
		}
		return nil
	}

	unspent := spendableNotes(notes)
	total := new(big.Int)
	for _, n := range unspent {
		total.Add(total, big.NewInt(n.GetNote().GetValue()))
	}
	fmt.Printf("\n%d unspent note(s), %s scaled units total\n", len(unspent), total)

	// The scan response already carries is_spent. -check-spent re-derives each
	// nullifier from ak, nk and the position and asks the contract directly, via
	// IsShieldedTRC20ContractNoteSpent. That is the targeted check for one note
	// and the way to confirm a scan result without rescanning.
	if s.cfg.checkSpent {
		fmt.Println("\nre-checking spend status against the nullifier table:")
		for _, n := range unspent {
			spent, err := noteIsSpent(ctx, s, s.contract, n.GetNote(), kb, n.GetPosition())
			if err != nil {
				return err
			}
			fmt.Printf("  position %d: is_spent=%v\n", n.GetPosition(), spent)
		}
	}

	return nil
}

// scanRange resolves the block window, defaulting to "since this address was
// created" up to the current head.
//
// The default is deliberately honest rather than fast: it covers every block
// that could hold a note for this address, which on a long-lived testnet
// address is thousands of chunked RPCs. Pass -begin to bound a working session.
func scanRange(ctx context.Context, s *session, keys *shieldedKeys) (int64, int64, error) {
	end := s.cfg.end
	if end == 0 {
		head, err := currentBlock(ctx, s)
		if err != nil {
			return 0, 0, err
		}
		end = head
	}

	begin := s.cfg.begin
	if begin == 0 {
		begin = keys.StartBlock
	}
	if begin == 0 {
		return 0, 0, fmt.Errorf("no scan start block: pass -begin <height> explicitly, or delete the key " +
			"file and run -mode=walletgen to start a fresh address whose creation height is recorded " +
			"(note that a new address cannot spend the old address's notes)")
	}
	if begin > end {
		return 0, 0, fmt.Errorf("-begin %d is after end block %d", begin, end)
	}
	return begin, end, nil
}

// ---------------------------------------------------------------------------
// transfer: shielded note -> another ztron address
// ---------------------------------------------------------------------------

func runTransfer(ctx context.Context, s *session) error {
	if err := s.needAccount("transfer (it pays the fee)"); err != nil {
		return err
	}
	amount, err := s.cfg.parseAmount()
	if err != nil {
		return err
	}
	if s.cfg.to == "" {
		return fmt.Errorf("-to is required for transfer (a ztron1... payment address)")
	}

	keys, kb, notes, scaled, err := openSpend(ctx, s, amount)
	if err != nil {
		return err
	}

	plan, err := planTransfer(notes, scaled)
	if err != nil {
		return err
	}
	fmt.Println(plan.describe("transfer", "pay", scaled))

	spends, err := buildSpends(ctx, s, plan.spends)
	if err != nil {
		return err
	}

	// Outputs: the payment, then change back to our own address if any.
	// transfer allows at most two outputs, so payment + change is the ceiling.
	payment, err := receiveNote(ctx, s, s.cfg.to, scaled, "payment")
	if err != nil {
		return err
	}
	receives := []*api.ReceiveNote{payment}
	change, err := changeNote(ctx, s, keys, plan)
	if err != nil {
		return err
	}
	if change != nil {
		receives = append(receives, change)
	}

	resp, err := lowlevel.CreateShieldedContractParameters(s.cli, ctx, &api.PrivateShieldedTRC20Parameters{
		Ask:                           kb.ask,
		Nsk:                           kb.nsk,
		Ovk:                           kb.ovk,
		ShieldedSpends:                spends,
		ShieldedReceives:              receives,
		Shielded_TRC20ContractAddress: s.contract.Bytes(),
	})
	if err != nil {
		return fmt.Errorf("createshieldedcontractparameters (transfer): %w", err)
	}
	if got := resp.GetParameterType(); got != "transfer" {
		return fmt.Errorf("node returned parameter_type %q, expected transfer", got)
	}

	return s.submit(ctx, sigTransfer, resp.GetTriggerContractInput(), "transfer")
}

// ---------------------------------------------------------------------------
// burn: shielded note -> transparent TRC-20
// ---------------------------------------------------------------------------

func runBurn(ctx context.Context, s *session) error {
	if err := s.needAccount("burn (it receives the transparent tokens)"); err != nil {
		return err
	}
	amount, err := s.cfg.parseAmount()
	if err != nil {
		return err
	}

	keys, kb, notes, scaled, err := openSpend(ctx, s, amount)
	if err != nil {
		return err
	}

	plan, err := planBurn(notes, scaled)
	if err != nil {
		return err
	}
	fmt.Println(plan.describe("burn", "to_amount", scaled))

	spends, err := buildSpends(ctx, s, plan.spends)
	if err != nil {
		return err
	}

	// The change note is what makes a partial burn possible. Omitting it when
	// the spent note is larger than the withdrawal breaks the equation the
	// proof is built over:
	//
	//	spend.value * sf == change.value * sf + to_amount
	//
	// so the transaction either fails or destroys the difference.
	receives := []*api.ReceiveNote{}
	change, err := changeNote(ctx, s, keys, plan)
	if err != nil {
		return err
	}
	if change != nil {
		receives = append(receives, change)
		fmt.Printf("returning %s scaled units of change to %s\n", plan.change, keys.PaymentAddress)
	}

	// The transparent half always comes back to the -private-key account, which
	// is why burn refuses to start without one.
	resp, err := lowlevel.CreateShieldedContractParameters(s.cli, ctx, &api.PrivateShieldedTRC20Parameters{
		Ask:                           kb.ask,
		Nsk:                           kb.nsk,
		Ovk:                           kb.ovk,
		ShieldedSpends:                spends,
		ShieldedReceives:              receives,
		TransparentToAddress:          s.owner().Bytes(),
		ToAmount:                      amount.String(),
		Shielded_TRC20ContractAddress: s.contract.Bytes(),
	})
	if err != nil {
		return fmt.Errorf("createshieldedcontractparameters (burn): %w", err)
	}
	if got := resp.GetParameterType(); got != "burn" {
		return fmt.Errorf("node returned parameter_type %q, expected burn", got)
	}

	return s.submit(ctx, sigBurn, resp.GetTriggerContractInput(), "burn")
}

// ---------------------------------------------------------------------------
// shared plumbing
// ---------------------------------------------------------------------------

// openSpend gathers everything a spend needs: the decoded keys and the notes
// this address owns in the scan window, with the amount converted into note
// space.
//
// transfer and burn were byte-identical up to this point before it was
// extracted.
func openSpend(ctx context.Context, s *session, amount *big.Int) (*shieldedKeys, *keyBundle,
	[]*api.DecryptNotesTRC20_NoteTx, *big.Int, error) {

	keys, err := loadKeys(s.cfg.keyFile)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	kb, err := keys.decode()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	sf, err := scalingFactor(ctx, s, s.owner(), s.contract)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	scaled, err := toScaled(amount, sf)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	begin, end, err := scanRange(ctx, s, keys)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	notes, err := scanNotesByIvk(ctx, s, s.contract, kb, begin, end)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return keys, kb, notes, scaled, nil
}

// receiveNote builds one output note with fresh randomness from the node.
func receiveNote(ctx context.Context, s *session, paymentAddress string, value *big.Int,
	what string) (*api.ReceiveNote, error) {

	v, err := noteValue(what, value)
	if err != nil {
		return nil, err
	}
	rcm, err := randomScalar(ctx, s, what)
	if err != nil {
		return nil, err
	}
	return &api.ReceiveNote{Note: &api.Note{Value: v, PaymentAddress: paymentAddress, Rcm: rcm}}, nil
}

// changeNote returns the plan's change output, or nil when the plan leaves none.
func changeNote(ctx context.Context, s *session, keys *shieldedKeys, plan notePlan) (*api.ReceiveNote, error) {
	if plan.change.Sign() == 0 {
		return nil, nil
	}
	return receiveNote(ctx, s, keys.PaymentAddress, plan.change, "change")
}

// buildSpends turns scanned notes into the SpendNoteTRC20 the node expects, by
// fetching each note's merkle anchor and path from the contract.
//
// A note whose leaf is not in the tree yet fails here, at getPath. That is the
// honest version of "wait for confirmations".
func buildSpends(ctx context.Context, s *session,
	notes []*api.DecryptNotesTRC20_NoteTx) ([]*api.SpendNoteTRC20, error) {

	spends := make([]*api.SpendNoteTRC20, 0, len(notes))
	for _, n := range notes {
		root, path, err := getPath(ctx, s, s.owner(), s.contract, n.GetPosition())
		if err != nil {
			return nil, fmt.Errorf("note at position %d is not spendable yet: %w", n.GetPosition(), err)
		}
		alpha, err := randomScalar(ctx, s, "alpha")
		if err != nil {
			return nil, err
		}
		fmt.Printf("spend: position %d, value %d, anchor %x\n", n.GetPosition(), n.GetNote().GetValue(), root)
		spends = append(spends, &api.SpendNoteTRC20{
			Note:  n.GetNote(),
			Alpha: alpha,
			Root:  root,
			Path:  path,
			Pos:   n.GetPosition(),
		})
	}
	return spends, nil
}

// submit triggers a shielded method with the node-built proof and broadcasts it.
func (s *session) submit(ctx context.Context, signature, triggerInput, what string) error {
	if triggerInput == "" {
		return fmt.Errorf("node returned an empty trigger_contract_input for %s", what)
	}
	input, err := decodeHex(triggerInput, "trigger_contract_input")
	if err != nil {
		return err
	}
	return s.submitRaw(ctx, s.contract, calldata(signature, input), what)
}

// submitRaw builds a TriggerContract transaction from raw call data, signs it
// locally and broadcasts it. This is the ordinary half of a shielded flow.
func (s *session) submitRaw(ctx context.Context, contract *types.Address, data []byte, what string) error {
	if err := s.needAccount(what); err != nil {
		return err
	}

	tx, err := lowlevel.TriggerContract(s.cli, ctx, &core.TriggerSmartContract{
		OwnerAddress:    s.owner().Bytes(),
		ContractAddress: contract.Bytes(),
		Data:            data,
		CallValue:       0,
	})
	if err != nil {
		return fmt.Errorf("triggersmartcontract (%s): %w", what, err)
	}
	fmt.Printf("%s: transaction built, %d bytes of call data\n", what, len(data))

	opts := client.DefaultBroadcastOptions()
	opts.FeeLimit = s.cfg.feeLimit
	opts.WaitForReceipt = true
	// The default WaitTimeout is 15s, shorter than a shielded call often takes
	// to solidify. On a timeout the broadcaster hands back the broadcast ack with
	// zero usage, so without this window the example would print "confirmed" for
	// a transaction that has not been mined yet; see the receipt check below.
	opts.WaitTimeout = s.cfg.timeout

	result, err := s.cli.SignAndBroadcast(ctx, tx, opts, s.acct)
	if err != nil {
		return fmt.Errorf("sign and broadcast %s: %w", what, err)
	}
	if !result.Success {
		return fmt.Errorf("%s failed: code=%v message=%s", what, result.Code, result.Message)
	}
	if result.EnergyUsage == 0 && result.NetUsage == 0 && len(result.Logs) == 0 {
		// A mined contract call always reports energy, so all-zero means the
		// receipt poll gave up, not that the transaction failed.
		return fmt.Errorf("%s: broadcast accepted (txid %s) but no receipt within the deadline; "+
			"it may still solidify, verify with -mode=scan", what, result.TxID)
	}
	fmt.Printf("%s: confirmed, txid %s (energy %d, net %d)\n", what, result.TxID, result.EnergyUsage, result.NetUsage)
	return nil
}

// abiAddressWord places an address into a 32-byte ABI word. Solidity encodes
// `address` as the 20-byte EVM form right-aligned, so the 0x41-prefixed 21-byte
// TRON representation must not be used here.
func abiAddressWord(a *types.Address) []byte {
	b := a.BytesEVM()
	out := make([]byte, 32)
	copy(out[32-len(b):], b)
	return out
}
