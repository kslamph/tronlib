package contract

import (
	"encoding/json"
	"fmt"
	"strings"

	eABI "github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/kslamph/tronlib/v2/pb/core"
)

// ABI plumbing. The on-chain shape of a TRON ABI is the protobuf
// SmartContract_ABI; the parsing/packing shape is go-ethereum's
// accounts/abi (the v1 dependency, reused — not reimplemented). The bridge
// is JSON: the pb entries render to a Solidity JSON ABI string, which
// geth parses. The JSON string is also what ABI() reports and what
// UseABI accepts, so every path funnels through one parser.

// jsonABIParam is the Solidity JSON ABI parameter shape (no components —
// the pb ABI carries no component metadata, so tuples cannot be expressed
// here; that is a protocol limitation of the on-chain ABI, not of this
// bridge).
type jsonABIParam struct {
	Name    string `json:"name,omitempty"`
	Type    string `json:"type"`
	Indexed bool   `json:"indexed,omitempty"`
}

type jsonABIEntry struct {
	Type            string         `json:"type"`
	Name            string         `json:"name,omitempty"`
	Inputs          []jsonABIParam `json:"inputs,omitempty"`
	Outputs         []jsonABIParam `json:"outputs,omitempty"`
	Anonymous       bool           `json:"anonymous,omitempty"`
	StateMutability string         `json:"stateMutability,omitempty"`
}

// pbABIToJSON renders a protobuf SmartContract_ABI into the Solidity JSON
// ABI form geth parses. The pb entries DO carry mutability — StateMutability
// (field 8, enum StateMutabilityType) plus the legacy Payable (field 7) and
// Constant (field 2) bools — and it must survive the rendering: geth
// rejects a "receive" entry without "stateMutability":"payable", which
// would fail the lazy ABI load for every Solidity >=0.6 payable contract.
// The mapping mirrors v1's pkg/utils/abi_parse.go. Fallback entries parse
// fine without a mutability, so none is invented for them.
func pbABIToJSON(abi *core.SmartContract_ABI) (string, error) {
	if abi == nil || len(abi.GetEntrys()) == 0 {
		return "", fmt.Errorf("ABI has no entries")
	}
	entries := make([]jsonABIEntry, 0, len(abi.Entrys))
	for _, e := range abi.Entrys {
		if e == nil {
			continue
		}
		// The pb enum String() is capitalized ("Function", "Event") while
		// the Solidity JSON ABI geth parses uses lowercase ("function",
		// "event") — normalize here so the on-chain form actually parses.
		entry := jsonABIEntry{
			Type:            strings.ToLower(e.GetType().String()),
			Name:            e.GetName(),
			Inputs:          paramsToJSON(e.GetInputs()),
			Outputs:         paramsToJSON(e.GetOutputs()),
			StateMutability: mutabilityToJSON(e),
		}
		// geth requires "the statemutability of receive can only be
		// payable" — force it when the node left the entry's mutability
		// unset, or the whole lazy ABI load would fail.
		if entry.Type == "receive" && entry.StateMutability == "" {
			entry.StateMutability = "payable"
		}
		entries = append(entries, entry)
	}
	if len(entries) == 0 {
		return "", fmt.Errorf("ABI has no non-empty entries")
	}
	b, err := json.Marshal(entries)
	if err != nil {
		return "", fmt.Errorf("ABI re-encode failed: %w", err)
	}
	return string(b), nil
}

// mutabilityToJSON maps a pb entry's mutability to its Solidity JSON ABI
// spelling (the inverse of v1's pkg/utils/abi_parse.go mapping: pure, view,
// nonpayable, payable). When the enum is unset/Unknown the legacy
// Payable/Constant bools take over: Payable -> payable, Constant -> view.
// Anything else renders without a mutability (geth reads that as
// nonpayable).
func mutabilityToJSON(e *core.SmartContract_ABI_Entry) string {
	switch e.GetStateMutability() {
	case core.SmartContract_ABI_Entry_Pure:
		return "pure"
	case core.SmartContract_ABI_Entry_View:
		return "view"
	case core.SmartContract_ABI_Entry_Nonpayable:
		return "nonpayable"
	case core.SmartContract_ABI_Entry_Payable:
		return "payable"
	default: // UnknownMutabilityType / unset — fall back to the legacy bools
		if e.GetPayable() {
			return "payable"
		}
		if e.GetConstant() {
			return "view"
		}
		return ""
	}
}

func paramsToJSON(params []*core.SmartContract_ABI_Entry_Param) []jsonABIParam {
	if len(params) == 0 {
		return nil
	}
	out := make([]jsonABIParam, 0, len(params))
	for _, p := range params {
		if p == nil {
			continue
		}
		out = append(out, jsonABIParam{Name: p.GetName(), Type: p.GetType(), Indexed: p.GetIndexed()})
	}
	return out
}

// parseABIJSON parses a Solidity JSON ABI string with geth's parser.
// Everything that can be wrong with a caller-supplied ABI lands here as
// one error the callers classify as contract.bad_abi.
func parseABIJSON(abiJSON string) (*eABI.ABI, error) {
	parsed := &eABI.ABI{}
	if err := parsed.UnmarshalJSON([]byte(abiJSON)); err != nil {
		return nil, err
	}
	return parsed, nil
}
