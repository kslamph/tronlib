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
// when the decoder was told it: Decode only sees topics and data and leaves
// Address zero, DecodeFor fills it in.
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
// returned Log is zero; callers that know the emitting contract prefer
// DecodeFor, which also consults that contract's own registrations.
//
// Errors are *tron.Error:
//   - no topics, or a first topic that is not exactly 32 bytes →
//     tron.CodeContractArgMismatch (malformed log, not an unknown event)
//   - no registered definition matches the signature, or the signature is
//     ambiguous (two different layouts registered for it) →
//     tron.CodeEventUnknown (register the emitting contract's ABI)
//   - a known event whose topics/data don't match its ABI (missing topic,
//     empty data, or data that fails ABI decoding) →
//     tron.CodeContractArgMismatch
func Decode(topics [][]byte, data []byte) (*Log, error) {
	return decodeAt(tron.Address{}, topics, data, "event.Decode")
}

// DecodeFor decodes a log emitted by addr, consulting the definitions
// registered for that contract first (RegisterABIJSONForAddress) and the
// global registry for signatures it does not define. This is the correct entry
// point for any caller that knows the emitting contract — a log's signature
// topic alone cannot distinguish two contracts that use the same event name
// with different indexed layouts, which is the ambiguity Decode refuses to
// guess. The returned Log carries Address = addr.
//
// Errors are the same as Decode's, scoped to addr's registry first.
func DecodeFor(addr tron.Address, topics [][]byte, data []byte) (*Log, error) {
	return decodeAt(addr, topics, data, "event.DecodeFor")
}

// decodeAt resolves topics[0] against addr's scope (global when addr is
// unset), then decodes. op names the entry point in returned errors.
func decodeAt(addr tron.Address, topics [][]byte, data []byte, op string) (*Log, error) {
	if len(topics) == 0 {
		return nil, &tron.Error{Code: tron.CodeContractArgMismatch, Op: op, Hint: "log has no topics; the first topic must be the event signature hash"}
	}
	sigTopic := topics[0]
	if len(sigTopic) != len(sigKey{}) {
		return nil, &tron.Error{Code: tron.CodeContractArgMismatch, Op: op, Hint: fmt.Sprintf("first topic is %d bytes, want %d (the whole event signature hash)", len(sigTopic), len(sigKey{}))}
	}

	var key sigKey
	copy(key[:], sigTopic)

	def, ambiguous := lookup(addr, key)
	if ambiguous {
		return nil, &tron.Error{
			Code: tron.CodeEventUnknown,
			Op:   op,
			Hint: fmt.Sprintf("two different definitions are registered for signature 0x%s and they disagree about which parameters are indexed, so neither can be decoded safely; register the emitting contract's ABI with event.RegisterABIJSONForAddress and decode with event.DecodeFor", hex.EncodeToString(key[:])),
		}
	}
	if def == nil {
		return nil, &tron.Error{
			Code: tron.CodeEventUnknown,
			Op:   op,
			Hint: fmt.Sprintf("no registered event definition matches signature 0x%s; register the emitting contract's ABI (event.RegisterABIJSON, or event.RegisterABIJSONForAddress for this contract alone) or call event.BuiltinTRC20", hex.EncodeToString(key[:])),
		}
	}

	log, err := decodeEvent(def, topics, data, op)
	if err != nil {
		return nil, err
	}
	log.Address = addr
	return log, nil
}

// DecodeLenient decodes a log like Decode, but materializes logs whose
// signature matches no registered definition instead of dropping them:
// on an unknown or ambiguous signature it returns
// &Log{Topics: topics, Data: data, EventName: ""}, nil — callers classify
// by EventName == "" and still get the raw bytes. Receipt.Logs and
// Events() use this so unknown logs are materialized with EventName == ""
// instead of dropped.
//
// Known events behave exactly like Decode: a successful decode returns the
// decoded Log; a known event whose topics/data don't match its ABI (or a
// malformed log shape — no topics, wrong-size signature topic) returns the
// same *tron.Error as Decode. Corrupt data on a known event is corrupt: it is
// an error, not a lenient pass-through.
func DecodeLenient(topics [][]byte, data []byte) (*Log, error) {
	return decodeLenientAt(tron.Address{}, topics, data, "event.Decode")
}

// DecodeLenientFor is DecodeLenient scoped to an emitting contract: it decodes
// through DecodeFor, so a contract with its own registered ABI resolves even
// when the global registry holds a conflicting layout for the same signature.
// The materialized fallback keeps Address = addr.
func DecodeLenientFor(addr tron.Address, topics [][]byte, data []byte) (*Log, error) {
	return decodeLenientAt(addr, topics, data, "event.DecodeFor")
}

func decodeLenientAt(addr tron.Address, topics [][]byte, data []byte, op string) (*Log, error) {
	log, err := decodeAt(addr, topics, data, op)
	if err != nil && tron.HasCode(err, tron.CodeEventUnknown) {
		return &Log{Address: addr, Topics: topics, Data: data, EventName: ""}, nil
	}
	return log, err
}

// DecodeEventSignature returns the canonical event signature string
// ("Transfer(address,address,uint256)") for a full 32-byte signature topic,
// without decoding a log. The boolean reports whether the topic matches a
// registered definition in the global registry (built-ins included).
//
// Unlike Decode this answers for the global registry only — it takes no
// emitting address — and it still answers for an ambiguous signature: the two
// competing definitions hash to the same topic precisely because their
// signature strings agree, so the name is known even when the layout is not.
func DecodeEventSignature(sig []byte) (string, bool) {
	if len(sig) != len(sigKey{}) {
		return "", false
	}
	var key sigKey
	copy(key[:], sig)

	def := globalDef(key)
	if def == nil {
		return "", false
	}
	return def.signature(), true
}

// decodeEvent decodes a matched event definition against raw topics/data,
// producing decoded ABI values rather than display strings. Parameters are
// merged positionally rather than by name — see the combine step below.
func decodeEvent(def *Definition, topics [][]byte, data []byte, op string) (*Log, error) {
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
			return nil, &tron.Error{Code: tron.CodeContractArgMismatch, Op: op, Hint: fmt.Sprintf("event %s: missing topic %d for indexed parameter %q", def.Name, i+1, param.Name)}
		}
		indexedValues = append(indexedValues, Param{
			Name:  param.Name,
			Value: decodeTopicValue(topics[i+1], param.Type),
		})
	}

	// Non-indexed parameters come from data. Empty data with declared
	// non-indexed parameters is malformed, not decodable-to-nothing.
	if len(nonIndexedParams) > 0 && len(data) == 0 {
		return nil, &tron.Error{Code: tron.CodeContractArgMismatch, Op: op, Hint: fmt.Sprintf("event %s: empty data for %d non-indexed parameters", def.Name, len(nonIndexedParams))}
	}
	var nonIndexedValues []Param
	if len(nonIndexedParams) > 0 {
		decoded, err := decodeEventData(data, nonIndexedParams)
		if err != nil {
			return nil, &tron.Error{Code: tron.CodeContractArgMismatch, Op: op, Hint: fmt.Sprintf("event %s: data does not match the registered ABI types", def.Name), Cause: err}
		}
		nonIndexedValues = decoded
	}

	// Combine all parameters in original declared order. indexedValues and
	// nonIndexedValues are built above by walking def.Inputs in declared
	// order and appending one value per input, so each slice is already
	// positionally aligned with the indexed / non-indexed subset of Inputs.
	// Two cursors therefore reassemble the exact values without needing a
	// key. Matching by name (inherited from v1) is not a valid key: Solidity
	// emits "" for unnamed parameters, the parse path does not validate
	// names, and duplicate names are legal ABI JSON — first-match semantics
	// then return the same value for every repeated name and silently drop
	// the values after it.
	allParams := make([]Param, 0, len(def.Inputs))
	var nextIndexed, nextNonIndexed int
	for _, input := range def.Inputs {
		if input.Indexed {
			allParams = append(allParams, indexedValues[nextIndexed])
			nextIndexed++
		} else {
			allParams = append(allParams, nonIndexedValues[nextNonIndexed])
			nextNonIndexed++
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

// decodeTopicValue decodes an indexed parameter from its 32-byte topic,
// returning decoded values instead of strings. Signed int types are
// interpreted two's-complement (NOT via big.Int.SetBytes, which is unsigned,
// so a negative indexed int256 topic
// decoded as a huge positive (see TestDecodeIndexedNegativeInt).
func decodeTopicValue(topic []byte, paramType string) any {
	switch paramType {
	case "address":
		addr, err := tronAddressFromEVM(eCommon.BytesToAddress(topic).Bytes())
		if err != nil {
			return hex.EncodeToString(topic)
		}
		return addr
	case "uint256", "uint128", "uint64", "uint32", "uint16", "uint8":
		return new(big.Int).SetBytes(topic)
	case "int256", "int128", "int64", "int32", "int16", "int8":
		// Two's complement: a 32-byte ABI word with the top bit set is
		// negative. big.Int.SetBytes is unsigned, so negate the
		// two's-complement of the magnitude in that case.
		v := new(big.Int).SetBytes(topic)
		if len(topic) > 0 && topic[0]&0x80 != 0 {
			// twoPowWidth is 2^(8·len(topic)); naming it anything but max
			// keeps the builtin visible in this scope.
			twoPowWidth := new(big.Int).Lsh(big.NewInt(1), uint(len(topic)*8))
			v.Sub(v, twoPowWidth)
		}
		return v
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
// standard ABI unpacking, returning decoded values instead of display strings.
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
