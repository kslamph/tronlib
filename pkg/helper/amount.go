package helper

import (
	"fmt"
	"strings"
)

func SunToTrx(amount int64) int64 {
	return amount / 1_000_000
}

func TrxToSun(amount int64) int64 {
	return amount * 1_000_000
}

func SunToTrxString(sun int64) string {
	negative := sun < 0
	if negative {
		sun = -sun
		return fmt.Sprintf("-%d.%06d", sun/1000000, sun%1000000)
	}
	return fmt.Sprintf("%d.%06d", sun/1000000, sun%1000000)
}

func SunToTrxStringCommas(sun int64) string {
	negative := sun < 0
	if negative {
		sun = -sun
	}
	whole := sun / 1_000_000
	decimal := sun % 1_000_000
	wholeStr := formatWholeNumberWithCommas(fmt.Sprintf("%d", whole))
	decimalStr := fmt.Sprintf("%06d", decimal)
	decimalStr = strings.TrimRight(decimalStr, "0")
	if negative {
		return fmt.Sprintf("-%s.%s", wholeStr, decimalStr)
	}
	return fmt.Sprintf("%s.%s", wholeStr, decimalStr)
}

func formatWholeNumberWithCommas(wholeNumStr string) string {
	n := len(wholeNumStr)
	if n <= 3 {
		return wholeNumStr
	}

	var result strings.Builder
	firstChunkLen := n % 3
	if firstChunkLen == 0 { // If length is a multiple of 3, first chunk is 3 digits
		firstChunkLen = 3
	}

	// Write the first chunk (1, 2, or 3 digits)
	result.WriteString(wholeNumStr[:firstChunkLen])

	// Write the remaining chunks, prepended by a comma
	for i := firstChunkLen; i < n; i += 3 {
		result.WriteString(",")
		result.WriteString(wholeNumStr[i : i+3])
	}
	return result.String()
}
