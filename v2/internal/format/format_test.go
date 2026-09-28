package format

import "testing"

func TestThousands(t *testing.T) {
	cases := map[string]string{
		"0":       "0",
		"12":      "12",
		"123":     "123",
		"1234":    "1,234",
		"1234567": "1,234,567",
	}
	for in, want := range cases {
		if got := Thousands(in); got != want {
			t.Errorf("Thousands(%q) = %q, want %q", in, got, want)
		}
	}
}
