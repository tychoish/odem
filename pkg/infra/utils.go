package infra

import (
	"iter"
	"time"

	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/fnx"
	"github.com/tychoish/fun/irt"
)

func ErrWorker(err error) fnx.Worker { return fnx.MakeWorker(func() error { return err }) }

func YearSelectorRange(earliest int) iter.Seq[int] {
	currentYear := time.Now().Year()

	erc.InvariantOk(earliest < 0 || earliest > currentYear, "year range must be greater than 0 and less than the current year:", earliest)

	return irt.Chain(
		irt.Args(
			irt.Args(0), // 0 = all years (no filter)
			irt.While(irt.MonotonicFrom(earliest), func(v int) bool { return v < currentYear }),
			irt.While(irt.MonotonicFrom(-1*currentYear), func(v int) bool { return v < -earliest }),
		),
	)
}
