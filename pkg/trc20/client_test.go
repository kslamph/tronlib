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

package trc20

import (
	"math/big"
	"testing"

	"github.com/shopspring/decimal"
)

func TestToWeiFromWei_RoundTrip_Decimals18(t *testing.T) {
	amount := decimal.RequireFromString("1.234567890123456789")
	wei, err := ToWei(amount, 18)
	if err != nil {
		t.Fatalf("ToWei error: %v", err)
	}
	wantWei := new(big.Int)
	wantWei.SetString("1234567890123456789", 10)
	if wei.Cmp(wantWei) != 0 {
		t.Fatalf("unexpected wei: got %s want %s", wei.String(), wantWei.String())
	}
	back, err := FromWei(wei, 18)
	if err != nil {
		t.Fatalf("FromWei error: %v", err)
	}
	if back.String() != amount.String() {
		t.Fatalf("roundtrip mismatch: got %s want %s", back.String(), amount.String())
	}
}

func TestToWeiFromWei_RoundTrip_Decimals6(t *testing.T) {
	amount := decimal.RequireFromString("123.456789")
	wei, err := ToWei(amount, 6)
	if err != nil {
		t.Fatalf("ToWei error: %v", err)
	}
	wantWei := new(big.Int)
	wantWei.SetString("123456789", 10)
	if wei.Cmp(wantWei) != 0 {
		t.Fatalf("unexpected wei: got %s want %s", wei.String(), wantWei.String())
	}
	back, err := FromWei(wei, 6)
	if err != nil {
		t.Fatalf("FromWei error: %v", err)
	}
	if back.String() != amount.String() {
		t.Fatalf("roundtrip mismatch: got %s want %s", back.String(), amount.String())
	}
}

func TestToWei_InvalidFraction_TooManyPlaces(t *testing.T) {
	amount := decimal.RequireFromString("0.1234567") // 7 places
	if _, err := ToWei(amount, 6); err == nil {
		t.Fatalf("expected error for too many fractional places")
	}
}

func TestToWei_InvalidDecimals(t *testing.T) {
	amount := decimal.RequireFromString("1")
	if _, err := ToWei(amount, 19); err == nil {
		t.Fatalf("expected error for decimals > 18")
	}
}

func TestFromWei_InvalidDecimals(t *testing.T) {
	val := big.NewInt(1)
	if _, err := FromWei(val, 19); err == nil {
		t.Fatalf("expected error for decimals > 18")
	}
}
