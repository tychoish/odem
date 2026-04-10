package dispatch

import (
	"context"
	"fmt"
	"iter"
	"strings"

	"github.com/tychoish/cmdr"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/grip"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/infra"
	"github.com/tychoish/odem/pkg/models"
	"github.com/tychoish/odem/pkg/reportui"
	"github.com/urfave/cli/v3"
)

func isOk[T interface{ Ok() bool }](in T) bool { return in.Ok() }
func toOp(in int) MinutesAppOperation          { return MinutesAppOperation(in) }
func toString[T fmt.Stringer](in T) string     { return in.String() }

func resolver[T any, I ~func(context.Context, *db.Connection, T) error](op I, err error) I {
	return func(ctx context.Context, conn *db.Connection, arg T) error {
		if op == nil {
			return err
		}
		return op(ctx, conn, arg)
	}
}

func fuzzySelectOperation(arg string) MinutesAppOperation {
	// this needs to be in the dispatcher package to avoid a circular dependency, even though it
	// feels like it wants to be in the fzfui package.
	arg = strings.ReplaceAll(arg, " ", "-")
	grip.Debugln("selecting operation to dispatch", arg)

	operation := NewMinutesAppOperation(arg)

	if !operation.Ok() {
		var err error
		operation, err = infra.NewFuzzySearch[MinutesAppOperation](AllMinutesAppOps()).Prompt("odem operation").FindOne()
		if operation.Ok() {
			return operation
		}
		if newop := NewMinutesAppOperation(operation.String()); newop.Ok() {
			grip.Debugln("succeeded to identify %s on fallback", newop)
			return newop
		}

		if err != nil {
			grip.Warningf("operation %q is not valid, %v, retrying", operation.String(), err)
			return MinutesAppOpRetry
		}
	}

	grip.Debugln("selected", operation)
	return operation
}

func toFzfCmdr(mao MinutesAppOperation) *cmdr.Commander {
	info := mao.GetInfo()
	return cmdr.MakeCommander().SetName(info.Key).SetUsage(info.Value).With(infra.DBOperationSpec(mao.FuzzyDispatcher().Op))
}

func toReportCmdr(mao MinutesAppOperation) *cmdr.Commander {
	i := mao.GetInfo()
	return cmdr.MakeCommander().SetName(i.Key).SetUsage(i.Value).With(ReportOperationSpec(mao.ReportDispatcher()))
}

func AllFuzzyMinutesAppCmdrs() iter.Seq[*cmdr.Commander] {
	return irt.Convert(AllMinutesAppOps(), toFzfCmdr)
}

func AllReportMinutesAppCmdrs() iter.Seq[*cmdr.Commander] {
	return irt.Convert(AllMinutesAppOps(), toReportCmdr)
}

func ReportOperationSpec(rptr Reporter) func(*cmdr.Commander) {
	return func(cc *cmdr.Commander) {
		cc.With(infra.DBOperationSpecWith(
			func(cc *cli.Command) reportui.Params {
				return reportui.Params{
					Params: models.Params{
						Name:  cmdr.GetFlagOrFirstArg[string](cc, "name"),
						Limit: cmdr.GetFlag[int](cc, "limit"),
						Years: cmdr.GetFlag[[]int](cc, "year"),
					},
					ToStdout:              cmdr.GetFlag[bool](cc, "stdout"),
					PathPrefix:            cmdr.GetFlag[string](cc, "prefix"),
					SuppressInteractivity: cmdr.GetFlag[bool](cc, "no-ask"),
				}
			},
			rptr.Report,
		))
	}
}
