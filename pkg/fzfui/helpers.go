package fzfui

import (
	"io"
	"iter"
	"os"

	"github.com/tychoish/fun/erc"
	"github.com/tychoish/odem/pkg/mdwn"
)

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

func renderTable[T any](write func(*mdwn.Builder, iter.Seq[T]), seq iter.Seq2[T, error]) error {
	var ec erc.Collector
	var mb mdwn.Builder
	write(&mb, erc.HandleAll(seq, ec.Push))
	if ec.Ok() {
		_, err := mb.WriteTo(os.Stdout)
		ec.Push(err)
	}
	return ec.Resolve()
}
