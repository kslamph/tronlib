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
// appear here (TestAllCodesHaveActionAndDoc enforces it).
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

// Action returns the default remediation for this code class (§8.4 of the
// design spec). Task 7's docgen regenerates this switch from AllCodes; the
// default returns ActionBug so an unhandled code is loud rather than silent.
func (c Code) Action() Action {
	switch c {
	case CodeAmountInvalid, CodeAmountTooManyDecimals, CodeAmountOverflow, CodeAmountNegative, CodeAmountDecimalsMismatch,
		CodeAddressInvalid, CodeAddressWrongPrefix,
		CodeTxAlreadySigned,
		CodeTxInvalidArgument, CodeTxUnknownContract,
		CodeChainNetworkMismatch,
		CodeContractNotFound, CodeContractNoABI, CodeContractBadABI, CodeContractBadMetadata, CodeContractMethodUnknown, CodeContractArgMismatch, CodeContractResultTypeMismatch,
		CodeAccountPermissionDenied,
		CodeKeyInvalid, CodeKeyMnemonicInvalid:
		return ActionFixCall
	case CodeTxNoSigner, CodeTxExpired, CodeTxFeeLimitTooLow, CodeTxTaposInvalid, CodeTxTooLarge,
		CodeReceiptReverted, CodeReceiptOutOfEnergy, CodeReceiptFailed:
		return ActionFixTransaction
	case CodeChainConnection, CodeChainTimeout, CodeChainClosed, CodeChainUnavailable, CodeRPCMethodFailed:
		return ActionRetry
	case CodeChainUnconfirmed:
		return ActionWait
	case CodeAccountInsufficientBalance, CodeAccountInsufficientEnergy, CodeAccountInsufficientBandwidth:
		return ActionFund
	default: // CodeTxDuplicate and any future code not yet classified
		return ActionBug
	}
}

// Doc returns a one-line, factual description of the code class. It explains
// what happened, not what to do about it — remediation lives in Error.Hint.
// Task 7's docgen regenerates this switch from AllCodes; the default is loud
// so an unhandled code surfaces immediately.
func (c Code) Doc() string {
	switch c {
	case CodeAmountInvalid:
		return "an amount argument failed validation"
	case CodeAmountTooManyDecimals:
		return "the amount has more decimal places than the token allows"
	case CodeAmountOverflow:
		return "the amount overflows the integer representation"
	case CodeAmountNegative:
		return "a negative amount was supplied"
	case CodeAmountDecimalsMismatch:
		return "the operands have different decimals"
	case CodeAddressInvalid:
		return "the value is not a valid TRON address"
	case CodeAddressWrongPrefix:
		return "the address's network prefix is not TRON's (0x41)"
	case CodeTxNoSigner:
		return "the transaction has no signer attached"
	case CodeTxAlreadySigned:
		return "the transaction is already signed"
	case CodeTxExpired:
		return "the transaction's expiration has passed"
	case CodeTxDuplicate:
		return "the node reports the transaction already exists"
	case CodeTxFeeLimitTooLow:
		return "the fee limit is below the node's minimum"
	case CodeTxInvalidArgument:
		return "a transaction parameter failed validation"
	case CodeTxUnknownContract:
		return "the referenced contract type is unknown to the SDK"
	case CodeTxTaposInvalid:
		return "the node rejected the transaction's reference block (TAPOS_ERROR)"
	case CodeTxTooLarge:
		return "the transaction exceeds the node's size limit (TOO_BIG_TRANSACTION_ERROR)"
	case CodeChainConnection:
		return "the node could not be reached"
	case CodeChainTimeout:
		return "the node did not respond in time"
	case CodeChainClosed:
		return "the connection to the node is closed"
	case CodeChainUnavailable:
		return "the node is unavailable (server error or maintenance)"
	case CodeChainUnconfirmed:
		return "the transaction was broadcast but is not yet confirmed"
	case CodeChainNetworkMismatch:
		return "the endpoint is on a different TRON network than expected"
	case CodeReceiptReverted:
		return "the contract execution reverted"
	case CodeReceiptOutOfEnergy:
		return "the contract execution ran out of energy"
	case CodeReceiptFailed:
		return "the transaction was processed but failed"
	case CodeContractNotFound:
		return "no contract exists at the given address"
	case CodeContractNoABI:
		return "no ABI was provided for the contract"
	case CodeContractBadABI:
		return "the provided ABI could not be parsed"
	case CodeContractBadMetadata:
		return "the contract's metadata violates the expected layout"
	case CodeContractMethodUnknown:
		return "the method name is not present in the contract ABI"
	case CodeContractArgMismatch:
		return "the arguments do not match the method's ABI signature"
	case CodeContractResultTypeMismatch:
		return "the result accessor does not match the ABI return type"
	case CodeAccountInsufficientBalance:
		return "the account lacks the required balance"
	case CodeAccountInsufficientEnergy:
		return "the account lacks the energy required for this call"
	case CodeAccountInsufficientBandwidth:
		return "the account lacks the bandwidth required for this transaction"
	case CodeAccountPermissionDenied:
		return "the signer lacks permission for this operation"
	case CodeKeyInvalid:
		return "the private key is not a valid secp256k1 key"
	case CodeKeyMnemonicInvalid:
		return "the mnemonic phrase is invalid"
	case CodeRPCMethodFailed:
		return "the node returned an error for the RPC method"
	default:
		return "unknown code: " + string(c)
	}
}
