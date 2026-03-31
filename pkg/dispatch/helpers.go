package dispatch

import (
	"iter"

	"github.com/tychoish/cmdr"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/odem/pkg/infra"
	"github.com/tychoish/odem/pkg/models"
	"github.com/tychoish/odem/pkg/reportui"
	"github.com/urfave/cli/v3"
)

func isOk[T interface{ Ok() bool }](in T) bool { return in.Ok() }
func toOp(in int) MinutesAppOperation          { return MinutesAppOperation(in) }
func idxorz[T any, S ~[]T](s S, idx int) (z T) {
	if len(s) > idx {
		return s[idx]
	}
	return z
}

func toFzfCmdr(mao MinutesAppOperation) *cmdr.Commander {
	info := mao.GetInfo()
	return cmdr.MakeCommander().SetName(info.Key).SetUsage(info.Value).With(infra.DBOperationSpec(mao.FuzzyDispatcher().Op).Add)
}

func toReportCmdr(mao MinutesAppOperation) *cmdr.Commander {
	i := mao.GetInfo()
	return cmdr.MakeCommander().SetName(i.Key).SetUsage(i.Value).With(ReportOperationSpec(mao.ReportDispatcher()).Add)
}

func AllFuzzyMinutesAppCmdrs() iter.Seq[*cmdr.Commander] {
	return irt.Convert(AllMinutesAppOps(), toFzfCmdr)
}

func AllReportMinutesAppCmdrs() iter.Seq[*cmdr.Commander] {
	return irt.Convert(AllMinutesAppOps(), toReportCmdr)
}

func ReportOperationSpec(rptr Reporter) *cmdr.OperationSpec[*infra.WithInput[reportui.Params]] {
	return infra.DBOperationSpecWith(
		func(cc *cli.Command) reportui.Params {
			return reportui.Params{
				Params: models.Params{
					Name:  cmdr.GetFlagOrFirstArg[string](cc, "name"),
					Limit: cmdr.GetFlag[int](cc, "limit"),
					Years: cmdr.GetFlag[[]int](cc, "year"),
				},
				ToStdout:   cmdr.GetFlag[bool](cc, "stdout"),
				PathPrefix: cmdr.GetFlag[string](cc, "prefix"),
			}
		},
		rptr.Report,
	)
}
