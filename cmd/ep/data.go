package ep

import (
	"context"

	"github.com/tychoish/cmdr"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/odem"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/dispatch"
	"github.com/tychoish/odem/pkg/infra"
	"github.com/tychoish/odem/pkg/mcpsrv"
)

func Fuzzy() *cmdr.Commander {
	return cmdr.MakeCommander().
		SetName("fuzzy").
		Aliases("fzf").
		SetUsage("fuzzy cli UI to minutes data").
		With(infra.DBOperationSpec(dispatch.MinutesAppOpRetry.FuzzyDispatcher().Op).Add).
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
		With(dispatch.ReportOperationSpec(dispatch.MinutesAppOpRetry.ReportDispatcher()).Add).
		Subcommanders(irt.Collect(dispatch.AllReportMinutesAppCmdrs())...)
}

func MCP() *cmdr.Commander {
	return cmdr.MakeCommander().
		SetName("mcp").
		SetUsage("run an MCP server (stdio) that provides access to Sacred Harp Minutes Data and analysis.").
		Flags(cmdr.FlagBuilder(false).SetName("http").SetUsage("call to start use the http service").Flag()).
		Flags(cmdr.FlagBuilder("127.0.0.1").SetName("addr").SetUsage("address/interface to listen for requests").Flag()).
		Flags(cmdr.FlagBuilder(1844).SetName("port").SetUsage("set the port to run the http service on").Flag()).
		With(odem.AttachConfiguration).
		With(infra.SimpleDBOperationSpec(func(ctx context.Context, conn *db.Connection) error {
			return mcpsrv.New(odem.GetConfiguration(ctx), conn, dispatch.AllMinutesAppMCPHandlers()).Run(ctx)
		}).Add)
}
