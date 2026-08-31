package contract

import (
	"encoding/json"
	"fmt"
	"strings"

	eABI "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/kslamph/tronlib/pb/core"
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
// ABI form geth parses. The pb param carries no stateMutability, so
// function entries render without it — geth treats that as nonpayable,
// which matches how the entries are used here (the node, not the ABI,
// decides what a call may spend).
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
			Type:    strings.ToLower(e.GetType().String()),
			Name:    e.GetName(),
			Inputs:  paramsToJSON(e.GetInputs()),
			Outputs: paramsToJSON(e.GetOutputs()),
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
