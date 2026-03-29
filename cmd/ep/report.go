package ep

import (
	"github.com/tychoish/cmdr"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/odem/pkg/dispatch"
)

func Report() *cmdr.Commander {
	return cmdr.MakeCommander().
		SetName("report").
		Aliases("rpt").
		SetUsage("generate a markdown report for a singer").
		Flags(cmdr.FlagBuilder(false).
			SetName("stdout", "o").
			SetUsage("write report to stdout instead of a file").
			Flag()).
		With(dispatch.ReportOperationSpec(dispatch.MinutesAppOpRetry.ReportDispatcher()).Add).
		Subcommanders(irt.Collect(dispatch.AllReportMinutesAppCmdrs())...)
}
