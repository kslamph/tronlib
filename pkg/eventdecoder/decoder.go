// Copyright (c) 2025 github.com/kslamph
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package eventdecoder

import (
	"encoding/hex"
	"fmt"
	"strings"
	"sync"

	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/utils"
	"golang.org/x/crypto/sha3"
)

// ParamDef is a compact representation of an event parameter definition
type ParamDef struct {
	Type    string
	Indexed bool
	Name    string
}

// EventDef is a compact representation of an event definition
type EventDef struct {
	Name   string
	Inputs []ParamDef
}

var (
	mu   sync.RWMutex
	sig4 = make(map[[4]byte]*EventDef)
)

// RegisterABIJSON registers all event entries from a JSON ABI string
func RegisterABIJSON(abiJSON string) error {
	proc := utils.NewABIProcessor(&core.SmartContract_ABI{})
	parsed, err := proc.ParseABI(abiJSON)
	if err != nil {
		return err
	}
	return RegisterABIObject(parsed)
}

// RegisterABIObject registers all event entries from a SmartContract_ABI object
func RegisterABIObject(abi *core.SmartContract_ABI) error {
	if abi == nil {
		return fmt.Errorf("nil ABI")
	}
	return RegisterABIEntries(abi.Entrys)
}

// RegisterABIEntries registers all event entries from the provided list (non-event entries are ignored)
func RegisterABIEntries(entries []*core.SmartContract_ABI_Entry) error {
	if len(entries) == 0 {
		return nil
	}

	local := make(map[[4]byte]*EventDef)

	for _, entry := range entries {
		if entry == nil || entry.Type != core.SmartContract_ABI_Entry_Event {
			continue
		}
		// Build canonical signature string: Name(types...)
		inputs := make([]string, len(entry.Inputs))
		compactInputs := make([]ParamDef, len(entry.Inputs))
		for i, in := range entry.Inputs {
			if in == nil {
				continue
			}
			inputs[i] = in.Type
			compactInputs[i] = ParamDef{Type: in.Type, Indexed: in.Indexed, Name: in.Name}
		}
		sigStr := fmt.Sprintf("%s(%s)", entry.Name, strings.Join(inputs, ","))

		// Compute 4-byte signature key
		hasher := sha3.NewLegacyKeccak256()
		hasher.Write([]byte(sigStr))
		sum := hasher.Sum(nil)

		var key [4]byte
		copy(key[:], sum[:4])

		local[key] = &EventDef{
			Name:   entry.Name,
			Inputs: compactInputs,
		}
	}

	if len(local) == 0 {
		return nil
	}

	mu.Lock()
	for k, v := range local {
		sig4[k] = v // overwrite by design
	}
	mu.Unlock()
	return nil
}

// DecodeEventSignature returns the canonical event signature string for the given 4-byte signature if known
// The boolean indicates whether the signature was found in the registry
func DecodeEventSignature(sig []byte) (string, bool) {
	if len(sig) < 4 {
		return "", false
	}
	var key [4]byte
	copy(key[:], sig[:4])

	mu.RLock()
	def, ok := sig4[key]
	mu.RUnlock()
	if !ok || def == nil {
		return "", false
	}
	types := make([]string, len(def.Inputs))
	for i, in := range def.Inputs {
		types[i] = in.Type
	}
	return fmt.Sprintf("%s(%s)", def.Name, strings.Join(types, ",")), true
}

// DecodeLog decodes a single log using the global 4-byte signature registry
func DecodeLog(topics [][]byte, data []byte) (*utils.DecodedEvent, error) {
	if len(topics) == 0 {
		return nil, fmt.Errorf("no topics provided")
	}
	sigTopic := topics[0]
	if len(sigTopic) < 4 {
		return nil, fmt.Errorf("first topic too short for signature: %d", len(sigTopic))
	}

	// Lookup by first 4 bytes
	var key [4]byte
	copy(key[:], sigTopic[:4])

	mu.RLock()
	def := sig4[key]
	mu.RUnlock()

	if def == nil {
		return &utils.DecodedEvent{
			EventName:  fmt.Sprintf("unknown_event(0x%s)", hex.EncodeToString(sigTopic[:4])),
			Parameters: []utils.DecodedEventParameter{},
		}, nil
	}

	// Build minimal ABI with just this one event entry
	abi := &core.SmartContract_ABI{Entrys: []*core.SmartContract_ABI_Entry{eventEntryFromDef(def)}}
	proc := utils.NewABIProcessor(abi)
	return proc.DecodeEventLog(topics, data)
}

// DecodeLogs decodes a slice of logs using the global 4-byte signature registry
func DecodeLogs(logs []*core.TransactionInfo_Log) ([]*utils.DecodedEvent, error) {
	if len(logs) == 0 {
		return []*utils.DecodedEvent{}, nil
	}
	result := make([]*utils.DecodedEvent, 0, len(logs))
	for _, lg := range logs {
		if lg == nil {
			continue
		}
		ev, err := DecodeLog(lg.GetTopics(), lg.GetData())
		if err != nil {
			return nil, err
		}
		result = append(result, ev)
	}
	return result, nil
}

func eventEntryFromDef(def *EventDef) *core.SmartContract_ABI_Entry {
	inputs := make([]*core.SmartContract_ABI_Entry_Param, len(def.Inputs))
	for i, in := range def.Inputs {
		inputs[i] = &core.SmartContract_ABI_Entry_Param{
			Indexed: in.Indexed,
			Name:    in.Name,
			Type:    in.Type,
		}
	}
	return &core.SmartContract_ABI_Entry{
		Anonymous:       false,
		Constant:        false,
		Name:            def.Name,
		Inputs:          inputs,
		Outputs:         nil,
		Type:            core.SmartContract_ABI_Entry_Event,
		Payable:         false,
		StateMutability: core.SmartContract_ABI_Entry_UnknownMutabilityType,
	}
}
