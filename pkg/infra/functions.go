package infra

func WrapPassthroughSecond[A, B, C any](op func(A) B) func(A, C) (B, C) {
	return func(a A, c C) (B, C) { return op(a), c }
}

func PassErrorThroughConverter[A, B any](op func(A) B) func(A, error) (B, error) {
	return WrapPassthroughSecond[A, B, error](op)
}
