// Package format holds display-only string helpers shared by the tron and
// token packages. It is internal: display formatting is not part of the
// public API, and both packages need the identical grouping rule.
package format

// Thousands inserts ',' every three digits from the right ("1234567" ->
// "1,234,567"). digits must be an unsigned decimal integer with no sign and
// no separators. Display only: the result is not parseable as a number by the
// amount parsers, which reject ',' outright.
func Thousands(digits string) string {
	n := len(digits)
	if n <= 3 {
		return digits
	}
	var b []byte
	head := n % 3
	if head > 0 {
		b = append(b, digits[:head]...)
		b = append(b, ',')
	}
	for i := head; i < n; i += 3 {
		b = append(b, digits[i:i+3]...)
		if i+3 < n {
			b = append(b, ',')
		}
	}
	return string(b)
}
