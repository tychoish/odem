package infra

import (
	"iter"

	"github.com/tychoish/fun/irt"
)

func ReverseMapping[A, B any](seq iter.Seq2[A, []B]) iter.Seq2[B, A] {
	return func(yield func(B, A) bool) {
		for value, keys := range seq {
			for key := range irt.Slice(keys) {
				if !yield(key, value) {
					return
				}
			}
		}
	}
}
