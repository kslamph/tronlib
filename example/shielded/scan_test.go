package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBlockWindows(t *testing.T) {
	tests := []struct {
		name  string
		begin int64
		end   int64
		want  [][2]int64
	}{
		{"empty range", 10, 9, nil},
		{"single block", 10, 10, [][2]int64{{10, 10}}},
		{"exactly one chunk", 1, scanChunk, [][2]int64{{1, scanChunk}}},
		{"one block spills into a second window", 1, scanChunk + 1, [][2]int64{{1, scanChunk}, {scanChunk + 1, scanChunk + 1}}},
		{"partial last window", 5_980_8727, 59_840_000, [][2]int64{
			{59_808_727, 59_809_726}, {59_809_727, 59_810_726}, {59_810_727, 59_811_726},
			{59_811_727, 59_812_726}, {59_812_727, 59_813_726}, {59_813_727, 59_814_726},
			{59_814_727, 59_815_726}, {59_815_727, 59_816_726}, {59_816_727, 59_817_726},
			{59_817_727, 59_818_726}, {59_818_727, 59_819_726}, {59_819_727, 59_820_726},
			{59_820_727, 59_821_726}, {59_821_727, 59_822_726}, {59_822_727, 59_823_726},
			{59_823_727, 59_824_726}, {59_824_727, 59_825_726}, {59_825_727, 59_826_726},
			{59_826_727, 59_827_726}, {59_827_727, 59_828_726}, {59_828_727, 59_829_726},
			{59_829_727, 59_830_726}, {59_830_727, 59_831_726}, {59_831_727, 59_832_726},
			{59_832_727, 59_833_726}, {59_833_727, 59_834_726}, {59_834_727, 59_835_726},
			{59_835_727, 59_836_726}, {59_836_727, 59_837_726}, {59_837_727, 59_838_726},
			{59_838_727, 59_839_726}, {59_839_727, 59_840_000},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := blockWindows(tt.begin, tt.end)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestBlockWindows_CoversRangeExactlyWithoutOverlap guards the property the
// scanning code depends on. Overlapping windows would report one note twice,
// and a duplicated note could then be selected twice into a single transaction
// that reverts on a duplicate nullifier.
func TestBlockWindows_CoversRangeExactlyWithoutOverlap(t *testing.T) {
	for _, tc := range []struct{ begin, end int64 }{
		{1, 1}, {1, 999}, {1, 1000}, {1, 1001}, {500, 5000}, {59_808_727, 70_501_330},
	} {
		windows := blockWindows(tc.begin, tc.end)
		if tc.begin > tc.end {
			assert.Empty(t, windows)
			continue
		}
		assert.Equal(t, tc.begin, windows[0][0], "first window starts at begin")
		assert.Equal(t, tc.end, windows[len(windows)-1][1], "last window ends at end")
		seen := 0
		for i, w := range windows {
			assert.LessOrEqual(t, w[0], w[1], "window %d is inverted", i)
			assert.LessOrEqual(t, w[1]-w[0]+1, int64(scanChunk), "window %d exceeds the chunk size", i)
			if i > 0 {
				assert.Equal(t, windows[i-1][1]+1, w[0], "window %d is not contiguous with the previous one", i)
			}
			seen += int(w[1] - w[0] + 1)
		}
		assert.Equal(t, int(tc.end-tc.begin+1), seen, "every block covered exactly once")
	}
}
