package tron_test

import (
	"fmt"

	"github.com/kslamph/tronlib/v2/tron"
)

// ErrorOf reads the fields Error() leaves out. Error() prints only "op: code",
// so the Hint — the part that names the fix — is not in the message: printing
// err alone loses it.
func ExampleErrorOf() {
	err := fmt.Errorf("submitting: %w", &tron.Error{
		Code: tron.CodeChainTimeout,
		Op:   "Broadcast",
		Hint: "the node did not answer; poll Wait(txid) before resending",
	})

	fmt.Println("message:", err)

	if te := tron.ErrorOf(err); te != nil {
		fmt.Println("hint:", te.Hint)
		fmt.Println("next:", te.Action())
	}
	// Output:
	// message: submitting: Broadcast: chain.timeout
	// hint: the node did not answer; poll Wait(txid) before resending
	// next: retry
}

// HasCode recovers a code through arbitrary %w wrapping — the sanctioned
// detection verb. errors.Is(err, tron.CodeAmountTooManyDecimals) would
// compile and silently return false, because Code is a string.
func ExampleHasCode() {
	inner := &tron.Error{Code: tron.CodeAmountTooManyDecimals, Op: "example"}
	wrapped := fmt.Errorf("building transfer: %w", inner)

	if tron.HasCode(wrapped, tron.CodeAmountTooManyDecimals) {
		fmt.Println("round to 6 decimal places")
	}
	// Output: round to 6 decimal places
}
