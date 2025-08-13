// Copyright (c) 2025 github.com/kslamph
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package eventdecoder_test

import (
	"encoding/hex"
	"fmt"

	"github.com/kslamph/tronlib/pkg/eventdecoder"
)

// Example is a package-level example showing DecodeLog with built-in TRC20 signatures.
func Example() {
	// Synthetic TRC20 Transfer event
	sigTopic, _ := hex.DecodeString("ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
	fromTopic, _ := hex.DecodeString("000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")
	toTopic, _ := hex.DecodeString("0000000000000000000000004e83362442b8d1bec281594cea3050c8eb01311c")
	amountData, _ := hex.DecodeString("00000000000000000000000000000000000000000000000000000000000003e8") // 1000

	ev, _ := eventdecoder.DecodeLog([][]byte{sigTopic, fromTopic, toTopic}, amountData)
	fmt.Println(ev.EventName)
	// Output:
	// Transfer
}

// ExampleRegisterABIJSON demonstrates extending the decoder with a custom ABI.
func ExampleRegisterABIJSON() {
	abiJSON := `{"entrys":[{"type":"event","name":"Custom","inputs":[{"name":"x","type":"uint256","indexed":true}]}]}`
	_ = eventdecoder.RegisterABIJSON(abiJSON)
}
