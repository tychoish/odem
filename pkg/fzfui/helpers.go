package fzfui

import "io"

// filterYears removes 0 from the slice (0 = all years sentinel).
// Returns nil if the result would be empty or contained only zeros.
func filterYears(years []int) []int {
	var out []int
	for _, y := range years {
		if y != 0 {
			out = append(out, y)
		}
	}
	return out
}

func flush(wr io.Writer, payload io.WriterTo) (err error) { _, err = payload.WriteTo(wr); return }
