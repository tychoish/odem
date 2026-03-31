package ep

import (
	"context"

	"github.com/tychoish/cmdr"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/dispatch"
	"github.com/tychoish/odem/pkg/infra"
	"github.com/tychoish/odem/pkg/mcpsrv"
)

func MCP() *cmdr.Commander {
	return cmdr.MakeCommander().
		SetName("mcp").
		SetUsage("run an MCP server (stdio) that provides access to Sacred Harp Minutes Data and analysis.").
		With(infra.SimpleDBOperationSpec(func(ctx context.Context, conn *db.Connection) error {
			return mcpsrv.New(conn, dispatch.AllMinutesAppMCPHandlers()).Run(ctx)
		}).Add)
}
