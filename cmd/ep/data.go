package ep

import (
	"context"
	"os"
	"path/filepath"
	"slices"

	"github.com/tychoish/cmdr"
	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/fnx"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/wpa"
	"github.com/tychoish/odem"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/dispatch"
	"github.com/tychoish/odem/pkg/infra"
	"github.com/tychoish/odem/pkg/mcpsrv"
	"github.com/tychoish/odem/pkg/reportui"
	"github.com/tychoish/odem/pkg/tgbot"
)

func MCP() *cmdr.Commander {
	return cmdr.MakeCommander().
		SetName("mcp").
		SetUsage("run an MCP server (stdio) that provides access to Sacred Harp Minutes Data and analysis.").
		Flags(cmdr.FlagBuilder(false).SetName("http").SetUsage("call to start use the http service").Flag()).
		Flags(cmdr.FlagBuilder("127.0.0.1").SetName("addr").SetUsage("address/interface to listen for requests").Flag()).
		Flags(cmdr.FlagBuilder(1844).SetName("port").SetUsage("set the port to run the http service on").Flag()).
		With(infra.SimpleDBOperationSpec(func(ctx context.Context, conn *db.Connection) error {
			return mcpsrv.New(odem.GetConfiguration(ctx), conn, dispatch.AllMinutesAppMCPHandlers()).Run(ctx)
		}))
}

func Telegram() *cmdr.Commander {
	return cmdr.MakeCommander().
		SetName("telegram").Aliases("tg").
		SetUsage("telegram chat bot service").
		With(infra.SimpleDBOperationSpec(func(ctx context.Context, conn *db.Connection) error {
			return tgbot.NewService(ctx, odem.GetConfiguration(ctx), conn).Start(ctx)
		}))
}

func Fuzzy() *cmdr.Commander {
	return cmdr.MakeCommander().
		SetName("fuzzy").
		Aliases("fzf").
		SetUsage("fuzzy cli UI to minutes data").
		With(infra.DBOperationSpec(dispatch.MinutesAppOpRetry.FuzzyDispatcher().Op)).
		Subcommanders(irt.Collect(dispatch.AllFuzzyMinutesAppCmdrs())...)
}

func Report() *cmdr.Commander {
	return cmdr.MakeCommander().
		SetName("report").
		Aliases("rpt").
		SetUsage("generate a markdown report for a singer").
		Flags(cmdr.FlagBuilder(false).
			SetName("stdout", "o").
			SetUsage("write report to stdout instead of a file").
			Flag()).
		With(dispatch.ReportOperationSpec(dispatch.MinutesAppOpRetry.ReportDispatcher())).
		Subcommanders(irt.Collect(dispatch.AllReportMinutesAppCmdrs())...).
		Subcommanders(
			cmdr.MakeCommander().
				SetName("generate").
				SetUsage("render a report for a specific user, with interactive user selection").
				With(dispatch.ReportOperationSpec(reportui.Leader)),
			cmdr.MakeCommander().
				SetName("batch").
				SetUsage("render all configured reports").
				With(infra.SimpleDBOperationSpec(func(ctx context.Context, conn *db.Connection) error {
					conf := odem.GetConfiguration(ctx)
					path := filepath.Join(erc.Must(os.Getwd()), conf.Reports.BasePath)

					var ec erc.Collector
					var jobs dt.List[fnx.Worker]

					// There's only one batch right now, and there's no benefit to splitting it up rn.
					for batch := range slices.Values(conf.Reports.Batches) {
						jobs.Extend(reportui.LeaderJobs(conn, filepath.Join(batch.Name, path), batch.Leaders))
					}

					ec.Push(wpa.RunWithPool(jobs.IteratorFront(), wpa.WorkerGroupConfDefaults()).Run(ctx))

					return ec.Resolve()
				})),
		)
}
