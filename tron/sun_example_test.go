package tron_test

import (
	"fmt"

	"github.com/kslamph/tronlib/v2/tron"
)

func ExampleTRX() {
	// SUN implements Stringer (canonical TRX text), so print the raw sun
	// count via Int64 to show the 1e6 scaling.
	fmt.Println(tron.TRX(1).Int64())
	// Output: 1000000
}

func ExampleParseTRX() {
	v, err := tron.ParseTRX("1.6")
	if err != nil {
		fmt.Println("err:", err)
		return
	}
	fmt.Println(v.String())
	// Output: 1.6
}
