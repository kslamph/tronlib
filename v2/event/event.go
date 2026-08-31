package event

import (
	"encoding/hex"
	"fmt"
	"math/big"

	eABI "github.com/ethereum/go-ethereum/accounts/abi"
	eCommon "github.com/ethereum/go-ethereum/common"
	"github.com/kslamph/tronlib/v2/tron"
)

// Log is a decoded TRON log entry. Address carries the emitting contract
// when the caller knows it (Decode itself only sees topics and data, so it
// leaves Address zero; tx receipt decoding populates it).
type Log struct {
	Address    tron.Address // emitting contract
	Topics     [][]byte     // raw topics; Topics[0] is the event signature hash
	Data       []byte       // raw non-indexed parameter data
	EventName  string       // matched definition's event name
	Parameters []Param      // decoded parameters in the event's declared input order
}

// Param is one decoded event parameter.
//
// Value is the decoded ABI value; its dynamic type depends on the ABI type
// (bool, string, *big.Int, tron.Address, []byte, etc.). This is the one
// deliberate any in v2: decoded contract data is genuinely dynamic — the
// value's shape is fixed by the ABI type string, not by the caller, and a
// closed sum type over every valid Solidity ABI type (including nested
// arrays and tuples) would cost more than it buys. It is not the banned
// "any in signatures" pattern: no v2 function takes or returns any as part
// of its own contract; this field merely holds contract-owned data.
type Param struct {
	Name  string
	Value any
}

// Decode decodes a single log's topics and data into a Log, looking the
// event signature up in the global registry. The Address field of the
// returned Log is zero; callers that know the emitting contract set it
// themselves.
//
// Errors are *tron.Error:
//   - no topics, or a first topic too short to hold a signature →
//     tron.CodeContractArgMismatch (malformed log, not an unknown event)
//   - no registered definition matches the signature →
//     tron.CodeEventUnknown (register the emitting contract's ABI)
//   - a known event whose topics/data don't match its ABI (missing topic,
//     empty data, or data that fails ABI decoding) →
//     tron.CodeContractArgMismatch
func Decode(topics [][]byte, data []byte) (*Log, error) {
	if len(topics) == 0 {
		return nil, &tron.Error{Code: tron.CodeContractArgMismatch, Op: "event.Decode", Hint: "log has no topics; the first topic must be the event signature hash"}
	}
	sigTopic := topics[0]
	if len(sigTopic) < 4 {
		return nil, &tron.Error{Code: tron.CodeContractArgMismatch, Op: "event.Decode", Hint: fmt.Sprintf("first topic is %d bytes, want at least 4 (the event signature)", len(sigTopic))}
	}

	var key [4]byte
	copy(key[:], sigTopic[:4])

	mu.RLock()
	def := sig4[key]
	mu.RUnlock()

	if def == nil {
		return nil, &tron.Error{
			Code: tron.CodeEventUnknown,
			Op:   "event.Decode",
			Hint: fmt.Sprintf("no registered event definition matches signature 0x%s; register the emitting contract's ABI (event.RegisterABIJSON) or call event.BuiltinTRC20", hex.EncodeToString(sigTopic[:4])),
		}
	}

	return decodeEvent(def, topics, data)
}

// decodeEvent decodes a matched event definition against raw topics/data.
// Ported from v1 decodeEventInternal, with decoded ABI values instead of
// display strings.
func decodeEvent(def *EventDef, topics [][]byte, data []byte) (*Log, error) {
	var indexedParams, nonIndexedParams []ParamDef
	for _, input := range def.Inputs {
		if input.Indexed {
			indexedParams = append(indexedParams, input)
		} else {
			nonIndexedParams = append(nonIndexedParams, input)
		}
	}

	// Indexed parameters come from topics[1:].
	indexedValues := make([]Param, 0, len(indexedParams))
	for i, param := range indexedParams {
		if i+1 >= len(topics) {
			return nil, &tron.Error{Code: tron.CodeContractArgMismatch, Op: "event.Decode", Hint: fmt.Sprintf("event %s: missing topic %d for indexed parameter %q", def.Name, i+1, param.Name)}
		}
		indexedValues = append(indexedValues, Param{
			Name:  param.Name,
			Value: decodeTopicValue(topics[i+1], param.Type),
		})
	}

	// Non-indexed parameters come from data. Empty data with declared
	// non-indexed parameters is malformed, not decodable-to-nothing
	// (v1 semantics).
	if len(nonIndexedParams) > 0 && len(data) == 0 {
		return nil, &tron.Error{Code: tron.CodeContractArgMismatch, Op: "event.Decode", Hint: fmt.Sprintf("event %s: empty data for %d non-indexed parameters", def.Name, len(nonIndexedParams))}
	}
	var nonIndexedValues []Param
	if len(nonIndexedParams) > 0 {
		decoded, err := decodeEventData(data, nonIndexedParams)
		if err != nil {
			return nil, &tron.Error{Code: tron.CodeContractArgMismatch, Op: "event.Decode", Hint: fmt.Sprintf("event %s: data does not match the registered ABI types", def.Name), Cause: err}
		}
		nonIndexedValues = decoded
	}

	// Combine all parameters in original declared order.
	allParams := make([]Param, 0, len(def.Inputs))
	for _, input := range def.Inputs {
		if input.Indexed {
			for _, p := range indexedValues {
				if p.Name == input.Name {
					allParams = append(allParams, p)
					break
				}
			}
		} else {
			for _, p := range nonIndexedValues {
				if p.Name == input.Name {
					allParams = append(allParams, p)
					break
				}
			}
		}
	}

	return &Log{
		Topics:     topics,
		Data:       data,
		EventName:  def.Name,
		Parameters: allParams,
	}, nil
}

// tronAddressFromEVM converts a 20-byte EVM address into a tron.Address by
// prepending TRON's 0x41 network prefix.
func tronAddressFromEVM(b []byte) (tron.Address, error) {
	full := make([]byte, 21)
	full[0] = 0x41
	copy(full[1:], b)
	return tron.AddressFromBytes(full)
}

// decodeTopicValue decodes an indexed parameter from its 32-byte topic.
// Ported from v1 decodeTopicValue, returning decoded values instead of
// strings.
func decodeTopicValue(topic []byte, paramType string) any {
	switch paramType {
	case "address":
		addr, err := tronAddressFromEVM(eCommon.BytesToAddress(topic).Bytes())
		if err != nil {
			return hex.EncodeToString(topic)
		}
		return addr
	case "uint256", "uint128", "uint64", "uint32", "uint16", "uint8",
		"int256", "int128", "int64", "int32", "int16", "int8":
		return new(big.Int).SetBytes(topic)
	case "bool":
		// ABI encodes bool as a 32-byte word: true = 0x00...01.
		for _, b := range topic {
			if b != 0 {
				return true
			}
		}
		return false
	case "bytes32", "bytes31", "bytes30", "bytes29", "bytes28", "bytes27",
		"bytes26", "bytes25", "bytes24", "bytes23", "bytes22", "bytes21",
		"bytes20", "bytes19", "bytes18", "bytes17", "bytes16", "bytes15",
		"bytes14", "bytes13", "bytes12", "bytes11", "bytes10", "bytes9",
		"bytes8", "bytes7", "bytes6", "bytes5", "bytes4", "bytes3", "bytes2",
		"bytes1":
		out := make([]byte, len(topic))
		copy(out, topic)
		return out
	default:
		return hex.EncodeToString(topic)
	}
}

// decodeEventData decodes non-indexed parameters from the data blob via
// standard ABI unpacking. Ported from v1 decodeEventData/formatEventValue,
// returning decoded values instead of display strings.
func decodeEventData(data []byte, params []ParamDef) ([]Param, error) {
	args := make(eABI.Arguments, len(params))
	for i, param := range params {
		abiType, err := eABI.NewType(param.Type, "", nil)
		if err != nil {
			return nil, fmt.Errorf("invalid ABI type %q: %w", param.Type, err)
		}
		args[i] = eABI.Argument{Name: param.Name, Type: abiType}
	}

	values, err := args.Unpack(data)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack event data: %w", err)
	}

	result := make([]Param, len(params))
	for i, param := range params {
		var value any
		if i < len(values) {
			value = convertABIValue(values[i], param.Type)
		}
		result[i] = Param{Name: param.Name, Value: value}
	}
	return result, nil
}

// convertABIValue maps geth's decoded Go values onto v2's value types:
// EVM addresses become tron.Address; everything else (integers as
// *big.Int, bool, string, []byte, arrays, tuples) passes through as geth
// decoded it.
func convertABIValue(value any, paramType string) any {
	if addr, ok := value.(eCommon.Address); ok {
		tronAddr, err := tronAddressFromEVM(addr.Bytes())
		if err != nil {
			return value
		}
		return tronAddr
	}
	// geth decodes fixed byte arrays as [N]byte, which is fine to pass
	// through, but normalize to []byte for bytesN for consistency with
	// indexed decoding.
	if paramType == "bytes32" {
		if arr, ok := value.([32]byte); ok {
			return arr[:]
		}
	}
	return value
}
