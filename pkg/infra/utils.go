package infra

import (
	"iter"
	"time"

	"github.com/tychoish/fun/fnx"
	"github.com/tychoish/fun/irt"
)

func ErrWorker(err error) fnx.Worker           { return fnx.MakeWorker(func() error { return err }) }
func NoopWorker() fnx.Worker                   { return fnx.MakeWorker(func() error { return nil }) }
func WorkerJoin(wfns ...fnx.Worker) fnx.Worker { return NoopWorker().Join(wfns...) }

func YearSelectorRange(earliest int) iter.Seq[int] {
	currentYear := time.Now().Year()

	return irt.Chain(
		irt.Args(
			irt.Args(0), // 0 = all years (no filter)
			irt.While(irt.MonotonicFrom(earliest), func(v int) bool { return v < currentYear }),
			irt.While(irt.MonotonicFrom(-1*currentYear), func(v int) bool { return v < -earliest }),
		),
	)
}
