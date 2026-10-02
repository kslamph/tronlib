package tron

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorFields(t *testing.T) {
	cause := errors.New("connection refused")
	e := &Error{Code: CodeChainConnection, Op: "Dial", Hint: "check the endpoint host and port", Cause: cause}
	assert.Contains(t, e.Error(), "chain.connection")
	assert.Contains(t, e.Error(), "Dial")
	assert.ErrorIs(t, e, cause) // Unwrap chain intact
	assert.True(t, HasCode(e, CodeChainConnection))
}

func TestHasCodeThroughWrapping(t *testing.T) {
	inner := &Error{Code: CodeAmountOverflow, Op: "ParseTRX"}
	wrapped := fmt.Errorf("building transfer: %w", inner)
	assert.True(t, HasCode(wrapped, CodeAmountOverflow))
	assert.False(t, HasCode(wrapped, CodeAddressInvalid))
	assert.False(t, HasCode(nil, CodeAmountOverflow))
	assert.False(t, HasCode(errors.New("plain"), CodeAmountOverflow))
}

// TestErrorOfExposesWhatErrorHides pins the accessor that makes Op, Hint, TxID
// and Next reachable. Error() prints only "op: code" by design (see
// TestHintIsNotInMessage), so Hint — the remediation README.md and
// docs/errors.md promise, "a Hint that names the fix" — is otherwise
// unreachable without importing errors and type-asserting *Error at every call
// site.
func TestErrorOfExposesWhatErrorHides(t *testing.T) {
	inner := &Error{
		Code: CodeAddressInvalid,
		Op:   "account.Permissions.Current",
		Hint: "owner.keys[0] is not a 0x41-prefixed 21-byte address",
		Next: ActionFixCall,
	}

	// Through a wrapping chain, which is how a caller actually meets it.
	te := ErrorOf(fmt.Errorf("reading permissions: %w", inner))
	if te == nil {
		t.Fatal("ErrorOf must find the *Error through a wrapping chain")
	}
	assert.Equal(t, CodeAddressInvalid, te.Code, "the fact")
	assert.Equal(t, "account.Permissions.Current", te.Op)
	assert.Contains(t, te.Hint, "owner.keys[0]", "the actionable half must survive")
	assert.Equal(t, ActionFixCall, te.Action())

	// Directly.
	assert.Same(t, inner, ErrorOf(inner))

	// And the cases where there is nothing to return.
	assert.Nil(t, ErrorOf(nil))
	assert.Nil(t, ErrorOf(errors.New("plain")), "a non-tronlib error has no fields to read")
}

func TestActionDerivedFromCode(t *testing.T) {
	assert.Equal(t, ActionRetry, CodeChainConnection.Action())
	assert.Equal(t, ActionRetry, CodeChainTimeout.Action())
	assert.Equal(t, ActionWait, CodeChainUnconfirmed.Action())
	assert.Equal(t, ActionWait, CodeTxDuplicate.Action())
	assert.Equal(t, ActionFixCall, CodeAmountInvalid.Action())
	assert.Equal(t, ActionFund, CodeAccountInsufficientEnergy.Action())
	assert.Equal(t, ActionFixTransaction, CodeReceiptReverted.Action())
}

func TestNextOverridesDefault(t *testing.T) {
	e := &Error{Code: CodeChainTimeout, Next: ActionWait, TxID: "abc"}
	assert.Equal(t, ActionWait, e.Next)
	assert.Equal(t, "abc", e.TxID)
	// Zero must remain the unset sentinel even for ActionRetry: an explicit
	// Next: ActionRetry must not be swallowed into "derive from Code",
	// or the §8.2 override mechanism could never force a retry.
	assert.Equal(t, ActionRetry, (&Error{Code: CodeChainConnection, Next: ActionRetry}).Action())
}

func TestActionZeroIsUnset(t *testing.T) {
	// Zero is the unset sentinel, not a member of the enum.
	assert.Equal(t, "unset", Action(0).String())
	assert.Equal(t, "retry", ActionRetry.String())
	assert.Equal(t, "wait", ActionWait.String())
	assert.Equal(t, "fix_call", ActionFixCall.String())
	assert.Equal(t, "fix_transaction", ActionFixTransaction.String())
	assert.Equal(t, "fund", ActionFund.String())
	assert.Equal(t, "bug", ActionBug.String())
}

func TestHintIsNotInMessage(t *testing.T) {
	// The reason Hint exists: v1 fused fact and advice into one string.
	e := &Error{Code: CodeAddressInvalid, Op: "ParseAddress", Hint: "pass a 34-character base58 address"}
	msg := e.Error()
	assert.Contains(t, msg, "address.invalid")
	assert.NotContains(t, msg, "34-character", "Hint must not leak into Error(): machine and human needs diverge")
}

func TestErrorIsMatchesCode(t *testing.T) {
	err := &Error{Code: CodeAddressInvalid}
	assert.True(t, errors.Is(err, &Error{Code: CodeAddressInvalid}))
	assert.False(t, errors.Is(err, &Error{Code: CodeAmountOverflow}))
}

func TestAllCodesHaveActionAndDoc(t *testing.T) {
	// Every code exported must have a non-empty Doc and a valid Action.
	// This is the test that prevents v1's "advertised but never returned" problem.
	for _, c := range AllCodes {
		assert.NotEmpty(t, c.Doc(), "code %s has no Doc", c)
		assert.NotEqual(t, Action(0), c.Action(), "code %s has no Action", c)
	}
}
