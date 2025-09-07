package types_test

import (
	"fmt"

	"github.com/kslamph/tronlib/pkg/types"
)

// ExampleNewAddress demonstrates constructing an Address from base58.
func ExampleNewAddress() {
	addr, _ := types.NewAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
	fmt.Println(len(addr.Bytes()) > 0)
	// Output:
	// true
}
