package infra

import (
	"cmp"
	"fmt"
	"iter"
	"reflect"

	"github.com/koki-develop/go-fzf"
	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/ers"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/stw"
	"github.com/tychoish/odem/pkg/infra"
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
	prompt        string
	caseSensitive bool
	selections    []int
	prefix        string
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
	case stw.Slice[T]:
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
		erc.Invariant(ers.ErrInvalidRuntimeType, "cannot build a fuzzy search list from type", func() error { return fmt.Errorf("%T", in) })
		return nil
	}
}

func (fsi *FuzzySearchItems[T]) ItemString(i int) string {
	if fsi.toString == nil {
		return fsi.defaultString(fsi.Index(i))
	}
	return fsi.toString(fsi.Index(i))
}

func (fsi *FuzzySearchItems[T]) Prompt(prompt string) *FuzzySearchItems[T] {
	fsi.prompt = prompt
	return fsi
}

func (fsi *FuzzySearchItems[T]) CaseSensitive(v bool) *FuzzySearchItems[T] {
	fsi.caseSensitive = v
	return fsi
}
func joinstr(args ...string) string                   { return irt.JoinStrings(irt.Slice(args)) }
func (fsi *FuzzySearchItems[T]) Len() int             { return fsi.Slice.Len() }
func (fsi *FuzzySearchItems[T]) toItem(m fzf.Match) T { return fsi.Index(m.Index) }
func (fsi *FuzzySearchItems[T]) Search(searchFor string) iter.Seq[T] {
	return irt.Convert(irt.Slice(fzf.Search(fsi, searchFor, fzf.WithSearchCaseSensitive(fsi.caseSensitive))), fsi.toItem)
}

func (fsi *FuzzySearchItems[T]) WithToString(in func(T) string) *FuzzySearchItems[T] {
	fsi.toString = in
	return fsi
}

func (FuzzySearchItems[T]) defaultString(in T) string { return fmt.Sprint(in) }
func (FuzzySearchItems[T]) zero() (z T)               { return z }
func (FuzzySearchItems[T]) noError(T) error           { return nil }
func (fsi *FuzzySearchItems[T]) WithSelectedPrefix(pre string) *FuzzySearchItems[T] {
	fsi.prefix = pre
	return fsi
}

func (fsi *FuzzySearchItems[T]) WithSelections(idxs []int) *FuzzySearchItems[T] {
	fsi.selections = idxs
	return fsi
}

func (fsi *FuzzySearchItems[T]) Find() iter.Seq2[T, error] {
	args := []fzf.Option{
		fzf.WithPrompt(joinstr(cmp.Or(fsi.prompt, "find (many)"), " => ")),
		fzf.WithCaseSensitive(fsi.caseSensitive),
	}

	if fsi.prefix != "" {
		args = append(args, fzf.WithSelectedPrefix(fsi.prefix))
	}

	ff, err := fzf.New(args...)
	if err != nil {
		return irt.Two(fsi.zero(), err)
	}
	idxs, err := ff.Find(fsi.Slice, fsi.ItemString)
	if err != nil {
		return irt.Two(fsi.zero(), err)
	}

	return irt.With(irt.Convert(irt.Slice(idxs), fsi.Index), fsi.noError)
}

func (fsi *FuzzySearchItems[T]) FindOne() (T, error) {
	fsi.Prompt(cmp.Or(fsi.prompt, "find (one)"))
	for v, err := range fsi.Find() {
		return v, err
	}
	return fsi.zero(), ers.New("not found")
}

type SearchParams struct {
	Prompt string
	Input  string
}

func (sp *SearchParams) ClearInput() *SearchParams              { return sp.With("") }
func (sp *SearchParams) With(input string) *SearchParams        { sp.Input = input; return sp }
func (sp *SearchParams) WithPrompt(prompt string) *SearchParams { sp.Prompt = prompt; return sp }

func FuzzySearchWithFallback[A, B any, S ~[]A](options S, toString func(A) string, sp *SearchParams, resolver func(A) B) B {
	if len(options) == 1 {
		return resolver(options[0])
	}

	if sp.Input != "" {
		narrowed := irt.Collect(
			infra.NewFuzzySearch[A](options).
				WithToString(toString).
				Search(sp.Input))
		if len(narrowed) == 1 {
			return resolver(narrowed[0])
		}
		if len(narrowed) > 1 {
			return FuzzySearchWithFallback(narrowed, toString, sp.ClearInput(), resolver)
		}
	}

	res, err := infra.NewFuzzySearch[A](options).
		WithToString(toString).
		Prompt(sp.Prompt).
		FindOne()
	if err != nil {
		panic(err)
	}

	return resolver(res)
}
