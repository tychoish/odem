package infra

import (
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"text/tabwriter"

	"github.com/goccy/go-yaml"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/strut"
)

type (
	YAML[T any] struct{ inner T }
	JSON[T any] struct{ inner T }
)

func NewYAML[T any](in T) YAML[T] { return YAML[T]{inner: in} }
func NewJSON[T any](in T) JSON[T] { return JSON[T]{inner: in} }
func (j YAML[T]) String() string  { return string(erc.Must(yaml.Marshal(j.inner))) }
func (j JSON[T]) String() string  { return string(erc.Must(json.Marshal(j.inner))) }

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

func Write(wr io.Writer, in []byte) error {
	num, err := wr.Write(in)
	if err != nil {
		return err
	}
	if len(in) != num {
		return fmt.Errorf("wrote %d bytes of %d", num, len(in))
	}
	return nil
}

func WriteString(wr io.Writer, in string) error {
	num, err := io.WriteString(wr, in)
	if err != nil {
		return err
	}
	if len(in) != num {
		return fmt.Errorf("wrote %d bytes of %d", num, len(in))
	}
	return nil
}
