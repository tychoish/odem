package dispatch

import (
	"cmp"
	"context"
	"fmt"
	"iter"
	"strings"

	"github.com/tychoish/cmdr"
	"github.com/tychoish/grip"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/infra"
	"github.com/tychoish/odem/pkg/models"
	"github.com/tychoish/odem/pkg/odemcli"
	"github.com/tychoish/odem/pkg/reportui"
	"github.com/urfave/cli/v3"
)

func isOk[T interface{ Ok() bool }](in T) bool { return in.Ok() }
func toOp(in int) MinutesAppOperation          { return MinutesAppOperation(in) }
func toString[T fmt.Stringer](in T) string     { return in.String() }

func resolver[T any, I ~func(context.Context, *db.Connection, T) error](reg MinutesAppRegistration, op I) I {
	return func(ctx context.Context, conn *db.Connection, arg T) error {
		if op == nil {
			return cmp.Or(reg.err, reg.unavailable())
		}
		return op(ctx, conn, arg)
	}
}

func fuzzySelectOperation(arg string) MinutesAppOperation {
	// this needs to be in the dispatcher package to avoid a circular dependency, even though it
	// feels like it wants to be in the fzfui package.
	arg = strings.ReplaceAll(arg, " ", "-")
	grip.Debug(grip.MPrintln("selecting operation to dispatch", arg))

	operation := NewMinutesAppOperation(arg)

	if !operation.Ok() {
		var err error
		operation, err = infra.NewFuzzySearch[MinutesAppOperation](AllMinutesAppFuzzOps()).Prompt("odem operation").FindOne()
		if operation.Ok() {
			return operation
		}
		if newop := NewMinutesAppOperation(operation.String()); newop.Ok() {
			grip.Debug(grip.MPrintln("succeeded to identify %s on fallback", newop))
			return newop
		}

		if err != nil {
			grip.Warning(grip.MPrintf("operation %q is not valid, %v, retrying", operation.String(), err))
			return MinutesAppOpRetry
		}
	}

	grip.Debug(grip.MPrintln("selected", operation.String()))
	return operation
}

func toFzfCmdr(mao MinutesAppOperation) *cmdr.Commander {
	info := mao.GetInfo()
	return cmdr.MakeCommander().SetName(info.Key).SetUsage(info.Value).With(odemcli.DBOperationSpec(mao.FuzzyDispatcher().Op))
}

type aliasFilter func(string, MinutesAppOperation) (string, MinutesAppOperation)

func joinKebabs(k string, v MinutesAppOperation) (string, MinutesAppOperation) {
	return strings.ReplaceAll(k, "-", ""), v
}

func replaceKebabsWithSpace(k string, v MinutesAppOperation) (string, MinutesAppOperation) {
	return strings.ReplaceAll(k, "-", " "), v
}

func toReportCmdr(mao MinutesAppOperation) *cmdr.Commander {
	i := mao.GetInfo()
	return cmdr.MakeCommander().SetName(i.Key).SetUsage(i.Value).With(ReportOperationSpec(mao.ReportDispatcher()))
}

func AllFuzzyMinutesAppCmdrs() iter.Seq[*cmdr.Commander] {
	return func(yield func(*cmdr.Commander) bool) {
		for op := range AllMinutesAppOps() {
			if !op.Registry().HasFuzz() {
				continue
			}
			if !yield(toFzfCmdr(op)) {
				return
			}
		}
	}
}

func AllReportMinutesAppCmdrs() iter.Seq[*cmdr.Commander] {
	return func(yield func(*cmdr.Commander) bool) {
		for op := range AllMinutesAppOps() {
			if !op.Registry().HasReporter() {
				continue
			}
			if !yield(toReportCmdr(op)) {
				return
			}
		}
	}
}

func ReportOperationSpec(rptr Reporter) func(*cmdr.Commander) {
	return func(cc *cmdr.Commander) {
		cc.With(odemcli.DBOperationSpecWith(
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
