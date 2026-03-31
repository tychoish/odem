package mcpsrv

import (
	"context"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tychoish/fun/fnx"
	"github.com/tychoish/grip/send"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/dispatch"
	"github.com/tychoish/odem/pkg/logger"
)

func New(conn *db.Connection) fnx.Worker {
	srv := mcp.NewServer(
		&mcp.Implementation{
			Name:    "odem",
			Title:   "Fasola Minutes Data",
			Version: "v0.1.0",
		}, nil)

	for info := range dispatch.AllMinutesAppOperations() {
		switch {
		case info == dispatch.MinutesAppOpRetry:
		case info == dispatch.MinutesAppOpExit:
		case info.Ok():
			mcp.AddTool(srv, &mcp.Tool{
				Name:        info.GetInfo().Key,
				Description: info.GetInfo().Value,
			}, dispatch.NewMCPTool(func(ctx context.Context, singer string) (string, error) {
				return "", errors.New("tool not implemented (yet!)")
			}).Resolve())
		}
	}

	return func(ctx context.Context) error {
		return srv.Run(ctx, &mcp.LoggingTransport{
			Writer:    send.MakeWriterSender(logger.Plain(ctx).Sender()),
			Transport: &mcp.StdioTransport{},
		})
	}
}
