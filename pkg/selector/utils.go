package selector

import (
	"iter"
	"time"

	"github.com/tychoish/fun/irt"
)

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
