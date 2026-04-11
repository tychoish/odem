package models

import (
	"iter"

	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/mdwn"
)

type TableRender interface {
	ColumnNames() []mdwn.Column
	RowValues() []string
}

func getRow[T TableRender](t T) []string           { return t.RowValues() }
func getColumnNames[T TableRender]() []mdwn.Column { var z T; return z.ColumnNames() }

func WriteTable[T TableRender](mb *mdwn.Builder, seq iter.Seq[T]) {
	mb.NewTableWithColumns(getColumnNames[T]()).Extend(irt.Convert(seq, getRow)).Build()
	mb.Line()
}
