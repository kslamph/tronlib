package event

// Simple ABI parser: converts a standard Solidity ABI JSON string into a
// *core.SmartContract_ABI. Ported verbatim from v1's SimpleABIParser
// (pkg/eventdecoder/decoder.go); RegisterABIJSON routes through it so both
// registration entry points share one code path.

import (
	"encoding/json"
	"fmt"

	"github.com/kslamph/tronlib/v2/pb/core"
)

type simpleABIParser struct{}

func newSimpleABIParser() *simpleABIParser { return &simpleABIParser{} }

// jsonABIEntry mirrors the standard Solidity ABI JSON format.
type jsonABIEntry struct {
	Anonymous bool           `json:"anonymous"`
	Inputs    []jsonABIParam `json:"inputs"`
	Name      string         `json:"name"`
	Outputs   []jsonABIParam `json:"outputs"`
	Type      string         `json:"type"`
}

type jsonABIParam struct {
	Indexed bool   `json:"indexed"`
	Name    string `json:"name"`
	Type    string `json:"type"`
}

// entryTypeMap maps JSON ABI "type" strings to protobuf enum values.
var entryTypeMap = map[string]core.SmartContract_ABI_Entry_EntryType{
	"constructor": core.SmartContract_ABI_Entry_Constructor,
	"function":    core.SmartContract_ABI_Entry_Function,
	"event":       core.SmartContract_ABI_Entry_Event,
	"fallback":    core.SmartContract_ABI_Entry_Fallback,
	"receive":     core.SmartContract_ABI_Entry_Receive,
	"error":       core.SmartContract_ABI_Entry_Error,
}

// parseABI parses a standard Solidity ABI JSON string into a SmartContract_ABI.
func (p *simpleABIParser) parseABI(abiJSON string) (*core.SmartContract_ABI, error) {
	var entries []jsonABIEntry
	if err := json.Unmarshal([]byte(abiJSON), &entries); err != nil {
		return nil, fmt.Errorf("failed to parse ABI JSON: %w", err)
	}

	result := &core.SmartContract_ABI{
		Entrys: make([]*core.SmartContract_ABI_Entry, 0, len(entries)),
	}

	for _, e := range entries {
		entryType, ok := entryTypeMap[e.Type]
		if !ok {
			continue
		}

		entry := &core.SmartContract_ABI_Entry{
			Anonymous: e.Anonymous,
			Name:      e.Name,
			Type:      entryType,
			Inputs:    convertABIParams(e.Inputs),
			Outputs:   convertABIParams(e.Outputs),
		}
		result.Entrys = append(result.Entrys, entry)
	}

	return result, nil
}

func convertABIParams(params []jsonABIParam) []*core.SmartContract_ABI_Entry_Param {
	if len(params) == 0 {
		return nil
	}
	result := make([]*core.SmartContract_ABI_Entry_Param, len(params))
	for i, p := range params {
		result[i] = &core.SmartContract_ABI_Entry_Param{
			Indexed: p.Indexed,
			Name:    p.Name,
			Type:    p.Type,
		}
	}
	return result
}
