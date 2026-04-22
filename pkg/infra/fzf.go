package infra

import (
	"cmp"
	"fmt"
	"iter"

	"github.com/koki-develop/go-fzf"
	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/ers"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/stw"
	"github.com/tychoish/grip"
)

const ErrEmptyResults = ers.Error("no results")

type FuzzySearchItems[T any] struct {
	prompt        string
	caseSensitive bool
	noLimit       bool
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

func (FuzzySearchItems[T]) defaultString(in T) string {
	if s, ok := any(in).(string); ok {
		return s
	}
	return fmt.Sprint(in)
}
func (FuzzySearchItems[T]) zero() (z T)               { return z }
func (FuzzySearchItems[T]) noError(T) error           { return nil }
func (fsi *FuzzySearchItems[T]) WithNoLimit() *FuzzySearchItems[T] {
	fsi.noLimit = true
	return fsi
}

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

	if fsi.noLimit {
		args = append(args, fzf.WithNoLimit(true))
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
	DisableInteraction  bool
	SelectFirstWhenMany bool
	Prompt              string
	Input               string
	Multi               bool
}

func (sp *SearchParams) ClearInput() *SearchParams              { return sp.With("") }
func (sp *SearchParams) WithoutInteractive() *SearchParams      { return sp.Interaction(false) }
func (sp *SearchParams) Interaction(arg bool) *SearchParams     { sp.DisableInteraction = !arg; return sp }
func (sp *SearchParams) UseFirstResult() *SearchParams          { sp.SelectFirstWhenMany = true; return sp }
func (sp *SearchParams) With(input string) *SearchParams        { sp.Input = input; return sp }
func (sp *SearchParams) WithPrompt(prompt string) *SearchParams { sp.Prompt = prompt; return sp }
func (sp *SearchParams) WithMulti() *SearchParams               { sp.Multi = true; return sp }

func FuzzySearchWithFallback[A, B any, S ~[]A](options S, toString func(A) string, sp *SearchParams, resolver func(A) B) (iter.Seq[B], error) {
	if len(options) == 1 {
		return irt.One(resolver(options[0])), nil
	}

	if sp.Input != "" {
		narrowed := irt.Collect(
			NewFuzzySearch[A](options).
				WithToString(toString).
				Search(sp.Input))
		switch {
		case len(narrowed) == 1:
			return irt.One(resolver(narrowed[0])), nil
		case len(narrowed) > 1 && sp.SelectFirstWhenMany:
			return irt.One(resolver(narrowed[0])), nil
		case len(narrowed) > 1 && sp.Multi:
			return irt.Convert(irt.Slice(narrowed), resolver), nil
		}

		if len(narrowed) > 1 && !sp.Multi {
			return FuzzySearchWithFallback(narrowed, toString, sp.ClearInput(), resolver)
		}
		if sp.DisableInteraction {
			grip.Error(grip.
				KV("op", "fuzzy-search").
				KV("interactivity", !sp.DisableInteraction).
				KV("input", sp.Input).
				KV("options", len(options)),
			)

			return irt.Zero[B](), ers.ErrNotFound
		}
	}
	if sp.DisableInteraction {
		grip.Warning("interactivity disabled without input")
		return irt.Zero[B](), ErrEmptyResults
	}

	search := NewFuzzySearch[A](options).
		WithToString(toString).
		Prompt(sp.Prompt)

	if sp.Multi {
		narrowed, err := erc.FromIteratorAll(search.WithNoLimit().Find())
		if err != nil {
			return irt.Zero[B](), err
		}

		return irt.Convert(irt.Slice(narrowed), resolver), nil
	}

	res, err := search.FindOne()
	if err != nil {
		grip.Error(grip.KV("op", "fuzzy-search").KV("err", err))
		return irt.Zero[B](), err
	}

	return irt.One(resolver(res)), nil
}
