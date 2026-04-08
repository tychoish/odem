package fzfui

import (
	"io"
	"iter"
	"os"

	"github.com/tychoish/fun/erc"
	"github.com/tychoish/odem/pkg/mdwn"
)

func noop[T any](in T) T                                  { return in }
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

func idxorz[T any, S ~[]T](sl S, idx int) (z T) {
	if len(sl) < idx {
		return z
	}
	return sl[idx]
}
