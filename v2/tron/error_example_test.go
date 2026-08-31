package tron_test

import (
	"fmt"

	"github.com/kslamph/tronlib/v2/tron"
)

// HasCode recovers a code through arbitrary %w wrapping — the sanctioned
// detection verb. errors.Is(err, tron.CodeAmountTooManyDecimals) would
// compile and silently return false, because Code is a string.
func ExampleHasCode() {
	inner := &tron.Error{Code: tron.CodeAmountTooManyDecimals, Op: "ParseTRX"}
	wrapped := fmt.Errorf("building transfer: %w", inner)

	if tron.HasCode(wrapped, tron.CodeAmountTooManyDecimals) {
		fmt.Println("round to 6 decimal places")
	}
	// Output: round to 6 decimal places
}
