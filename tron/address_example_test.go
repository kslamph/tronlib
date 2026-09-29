package tron_test

import (
	"fmt"

	"github.com/kslamph/tronlib/v2/tron"
)

func ExampleParseAddress() {
	a, err := tron.ParseAddress("TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb")
	if err != nil {
		fmt.Println("err:", err)
		return
	}
	fmt.Println(a.String())
	// Output: TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb
}
