package infra

import (
	"fmt"
	"io"
	"iter"
	"reflect"
	"text/tabwriter"

	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/strut"
)

func IterStruct(foo any) iter.Seq2[string, any] {
	return func(yield func(string, any) bool) {
		v := reflect.ValueOf(foo)
		typ := v.Type()
		for idx := range v.NumField() {
			val := v.Field(idx)
			if val.Type().Kind() == reflect.Pointer {
				val = val.Elem()
			}
			if !yield(typ.Field(idx).Name, val.Interface()) {
				return
			}
		}
	}
}

func WriteTabbedKVs[T any](wr io.Writer, seq iter.Seq2[string, T]) error {
	tw := tabwriter.NewWriter(wr, 2, 4, 2, ' ', 0)
	buf := strut.MakeMutable(32)

	buf.ExtendStringsJoin(irt.Args("Key", "Value"), "\t")
	buf.Line()
	buf.ExtendStringsJoin(irt.Args("-----", "-----"), "\t")
	buf.Line()

	if _, err := tw.Write(buf.Bytes()); err != nil {
		return err
	}

	buf.Release()

	for k, v := range seq {
		if _, err := fmt.Fprintf(tw, "%s\t%v\n", k, v); err != nil {
			return err
		}
	}
	return tw.Flush()
}
