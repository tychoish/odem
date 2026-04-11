package fzfui

import (
	"io"
	"iter"
	"os"

	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/mdwn"
	"github.com/tychoish/odem/pkg/models"
)

func flush(wr io.Writer, payload io.WriterTo) (err error) { _, err = payload.WriteTo(wr); return }

func writeLeaderCountTable(mb *mdwn.Builder, seq iter.Seq[models.LeaderSongRank]) {
	models.WriteTable(mb, irt.Convert(seq, models.WrapLeaderSongRank("Count")))
}

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
