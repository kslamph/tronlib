package tron

// Code identifies a failure class machine-parseably.
// Dotted, prefixed, and extensible. Prefixes carry a default Action.
type Code string

// Action is the remediation a caller (human or agent) should take.
// Zero is the unset sentinel: Error.Next == 0 means "derive from Code", so
// no Action member may have the value 0 — otherwise an explicit
// Next: ActionRetry could not be distinguished from "unset".
type Action int

const (
	ActionRetry Action = iota + 1 // 0 is the unset sentinel: Error.Next == 0 means "derive from Code"
	ActionWait                    // the call may have landed; poll, do NOT resend
	ActionFixCall
	ActionFixTransaction
	ActionFund
	ActionBug
)

// String returns the lower-case, human-readable name of the Action, or
// "unset" for the zero value and any out-of-range Action. It never claims a
// real member for a value outside the enum.
func (a Action) String() string {
	switch a {
	case ActionRetry:
		return "retry"
	case ActionWait:
		return "wait"
	case ActionFixCall:
		return "fix_call"
	case ActionFixTransaction:
		return "fix_transaction"
	case ActionFund:
		return "fund"
	case ActionBug:
		return "bug"
	default:
		return "unset"
	}
}

const (
	CodeAmountInvalid          Code = "amount.invalid"
	CodeAmountTooManyDecimals  Code = "amount.too_many_decimals"
	CodeAmountOverflow         Code = "amount.overflow"
	CodeAmountNegative         Code = "amount.negative"
	CodeAmountDecimalsMismatch Code = "amount.decimals_mismatch"

	CodeAddressInvalid     Code = "address.invalid"
	CodeAddressWrongPrefix Code = "address.wrong_prefix"

	CodeTxNoSigner        Code = "tx.no_signer"
	CodeTxAlreadySigned   Code = "tx.already_signed"
	CodeTxExpired         Code = "tx.expired"
	CodeTxDuplicate       Code = "tx.duplicate"
	CodeTxFeeLimitTooLow  Code = "tx.fee_limit_too_low"
	CodeTxInvalidArgument Code = "tx.invalid_argument"
	CodeTxUnknownContract Code = "tx.unknown_contract"
	CodeTxTaposInvalid    Code = "tx.tapos_invalid"
	CodeTxTooLarge        Code = "tx.too_large"

	CodeChainConnection      Code = "chain.connection"
	CodeChainTimeout         Code = "chain.timeout"
	CodeChainClosed          Code = "chain.closed"
	CodeChainUnavailable     Code = "chain.unavailable"
	CodeChainUnconfirmed     Code = "chain.unconfirmed"
	CodeChainNetworkMismatch Code = "chain.network_mismatch"

	CodeReceiptReverted    Code = "receipt.reverted"
	CodeReceiptOutOfEnergy Code = "receipt.out_of_energy"
	CodeReceiptFailed      Code = "receipt.failed"

	CodeContractNotFound           Code = "contract.not_found"
	CodeContractNoABI              Code = "contract.no_abi"
	CodeContractBadABI             Code = "contract.bad_abi"
	CodeContractBadMetadata        Code = "contract.bad_metadata"
	CodeContractMethodUnknown      Code = "contract.method_unknown"
	CodeContractArgMismatch        Code = "contract.arg_mismatch"
	CodeContractResultTypeMismatch Code = "contract.result_type_mismatch"

	CodeAccountInsufficientBalance   Code = "account.insufficient_balance"
	CodeAccountInsufficientEnergy    Code = "account.insufficient_energy"
	CodeAccountInsufficientBandwidth Code = "account.insufficient_bandwidth"
	CodeAccountPermissionDenied      Code = "account.permission_denied"

	CodeKeyInvalid         Code = "key.invalid"
	CodeKeyMnemonicInvalid Code = "key.mnemonic_invalid"

	CodeRPCMethodFailed Code = "rpc.method_failed"
)

// AllCodes is the authoritative code set. docgen regenerates the
// Action/Doc tables from this slice; hand-editing those tables is the
// drift this package exists to prevent. Every exported Code constant MUST
// appear here (docgen's parity check enforces it; see cmd/docgen).
var AllCodes = []Code{
	// amount
	CodeAmountInvalid,
	CodeAmountTooManyDecimals,
	CodeAmountOverflow,
	CodeAmountNegative,
	CodeAmountDecimalsMismatch,

	// address
	CodeAddressInvalid,
	CodeAddressWrongPrefix,

	// tx
	CodeTxNoSigner,
	CodeTxAlreadySigned,
	CodeTxExpired,
	CodeTxDuplicate,
	CodeTxFeeLimitTooLow,
	CodeTxInvalidArgument,
	CodeTxUnknownContract,
	CodeTxTaposInvalid,
	CodeTxTooLarge,

	// chain
	CodeChainConnection,
	CodeChainTimeout,
	CodeChainClosed,
	CodeChainUnavailable,
	CodeChainUnconfirmed,
	CodeChainNetworkMismatch,

	// receipt
	CodeReceiptReverted,
	CodeReceiptOutOfEnergy,
	CodeReceiptFailed,

	// contract
	CodeContractNotFound,
	CodeContractNoABI,
	CodeContractBadABI,
	CodeContractBadMetadata,
	CodeContractMethodUnknown,
	CodeContractArgMismatch,
	CodeContractResultTypeMismatch,

	// account
	CodeAccountInsufficientBalance,
	CodeAccountInsufficientEnergy,
	CodeAccountInsufficientBandwidth,
	CodeAccountPermissionDenied,

	// key
	CodeKeyInvalid,
	CodeKeyMnemonicInvalid,

	// rpc
	CodeRPCMethodFailed,
}

// Three layers, three jobs:
//   - Code.Doc(): the agent-facing ADVICE table, rendered by docgen.
//     Remediation language is correct and expected here.
//   - Error.Hint: call-site-specific remediation. Never printed by Error().
//   - Error(): fact only ("op: code").
