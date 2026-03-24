package infra

import (
	"fmt"
	"iter"
	"reflect"

	"github.com/koki-develop/go-fzf"
	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/ers"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/stw"
)

func IterStruct(foo any) iter.Seq2[string, any] {
	return func(yield func(string, any) bool) {
		v := reflect.ValueOf(foo)
		typ := v.Type()
		if typ.Kind() == reflect.Pointer {
			v = v.Elem()
			typ = v.Type()
		}

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

type FuzzySearchItems[T any] struct {
	stw.Slice[T]
	toString func(in T) string
}

func NewFuzzySearch[
	T any,
	S iter.Seq[T] | ~[]T | func() iter.Seq[T] | *dt.List[T] | dt.List[T] | *dt.Stack[T] | dt.Stack[T],
](in S) *FuzzySearchItems[T] {
	switch inner := any(in).(type) {
	case iter.Seq[T]:
		return &FuzzySearchItems[T]{Slice: irt.Collect(inner)}
	case []T:
		return &FuzzySearchItems[T]{Slice: inner}
	case func() iter.Seq[T]:
		return &FuzzySearchItems[T]{Slice: irt.Collect(inner())}
	case *dt.Stack[T]:
		return &FuzzySearchItems[T]{Slice: irt.Collect(inner.Iterator())}
	case *dt.List[T]:
		return &FuzzySearchItems[T]{Slice: irt.Collect(inner.IteratorFront())}
	case dt.Stack[T]:
		return &FuzzySearchItems[T]{Slice: irt.Collect(inner.Iterator())}
	case dt.List[T]:
		return &FuzzySearchItems[T]{Slice: irt.Collect(inner.IteratorFront())}
	default:
		erc.Invariant(ers.ErrInvalidRuntimeType, "cannot build a fuzzy search list from type")
		return nil
	}
}

func (fsi *FuzzySearchItems[T]) ItemString(i int) string {
	if fsi.toString == nil {
		return fsi.defaultString(fsi.Index(i))
	}
	return fsi.toString(fsi.Index(i))
}
func (fsi *FuzzySearchItems[T]) Len() int             { return fsi.Slice.Len() }
func (fsi *FuzzySearchItems[T]) toItem(m fzf.Match) T { return fsi.Index(m.Index) }
func (fsi *FuzzySearchItems[T]) Search(searchFor string) iter.Seq[T] {
	return irt.Convert(irt.Slice(fzf.Search(fsi, searchFor, fzf.WithSearchCaseSensitive(true))), fsi.toItem)
}

func (fsi *FuzzySearchItems[T]) WithToString(in func(T) string) *FuzzySearchItems[T] {
	fsi.toString = in
	return fsi
}

func (FuzzySearchItems[T]) defaultString(in T) string { return fmt.Sprint(in) }
func (FuzzySearchItems[T]) zero() (z T)               { return z }
func (FuzzySearchItems[T]) noError(T) error           { return nil }
func (fsi *FuzzySearchItems[T]) Find(prompt string) iter.Seq2[T, error] {
	ff, err := fzf.New(
		fzf.WithPrompt(fmt.Sprintf("%s => ", prompt)),
		fzf.WithCaseSensitive(false),
	)
	if err != nil {
		return irt.Two(fsi.zero(), err)
	}
	idxs, err := ff.Find(fsi.Slice, fsi.ItemString)
	if err != nil {
		return irt.Two(fsi.zero(), err)
	}

	return irt.With(irt.Convert(irt.Slice(idxs), fsi.Index), fsi.noError)
}

func (fsi *FuzzySearchItems[T]) FindOne(prompt string) (T, error) {
	for v, err := range fsi.Find(prompt) {
		return v, err
	}
	return fsi.zero(), ers.New("not found")
}
