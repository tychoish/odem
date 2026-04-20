package infra

import "github.com/tychoish/fun/fnx"

func WrapPassthroughSecond[A, B, C any](op func(A) B) func(A, C) (B, C) {
	return func(a A, c C) (B, C) { return op(a), c }
}

func PassErrorThroughConverter[A, B any](op func(A) B) func(A, error) (B, error) {
	return WrapPassthroughSecond[A, B, error](op)
}
func ErrWorker(err error) fnx.Worker           { return fnx.MakeWorker(func() error { return err }) }
func NoopWorker() fnx.Worker                   { return fnx.MakeWorker(func() error { return nil }) }
func WorkerJoin(wfns ...fnx.Worker) fnx.Worker { return NoopWorker().Join(wfns...) }
