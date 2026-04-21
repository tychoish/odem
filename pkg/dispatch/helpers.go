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
func toOp(in int) MinutesOperation             { return MinutesOperation(in) }
func toString[T fmt.Stringer](in T) string     { return in.String() }

func resolver[T any, I ~func(context.Context, *db.Connection, T) error](reg MinutesOpRegistration, op I) I {
	return func(ctx context.Context, conn *db.Connection, arg T) error {
		if op == nil {
			return cmp.Or(reg.err, reg.unavailable())
		}
		return op(ctx, conn, arg)
	}
}

func fuzzySelectOperation(arg string) MinutesOperation {
	arg = strings.ReplaceAll(arg, " ", "-")
	grip.Debug(grip.MPrintln("selecting operation to dispatch", arg))

	operation := NewMinutesAppOperation(arg)

	if !operation.Ok() {
		var err error
		operation, err = infra.NewFuzzySearch[MinutesOperation](AllMinutesAppOps()).Prompt("odem operation").FindOne()
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

func toFzfCmdr(mao MinutesOperation) *cmdr.Commander {
	info := mao.GetInfo()
	return cmdr.MakeCommander().SetName(info.Key).SetUsage(info.Value).With(odemcli.DBOperationSpec(mao.ReportDispatcher().Op))
}

type aliasFilter func(string, MinutesOperation) (string, MinutesOperation)

func kebabTo(k string, sep string) string                                 { return strings.ReplaceAll(k, "-", sep) }
func joinKebabs(k string, v MinutesOperation) (string, MinutesOperation)  { return kebabTo(k, ""), v }
func kebab2Space(k string, v MinutesOperation) (string, MinutesOperation) { return kebabTo(k, " "), v }
func kebab2Dots(k string, v MinutesOperation) (string, MinutesOperation)  { return kebabTo(k, "."), v }

func toReportCmdr(mao MinutesOperation) *cmdr.Commander {
	i := mao.GetInfo()
	return cmdr.MakeCommander().SetName(i.Key).SetUsage(i.Value).With(ReportOperationSpec(mao.ReportDispatcher()))
}

func AllFuzzyMinutesAppCmdrs() iter.Seq[*cmdr.Commander] {
	return func(yield func(*cmdr.Commander) bool) {
		for op := range AllMinutesAppOps() {
			if !op.Registry().HasReporter() {
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
