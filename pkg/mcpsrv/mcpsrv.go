package mcpsrv

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tychoish/fun/fnx"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/grip/send"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/logger"
)

func NewTool[A, B any](op func(context.Context, *db.Connection, A) (B, error)) ToolOperation[A, B] {
	return op
}

type ToolOperation[IN, OUT any] func(ctx context.Context, conn *db.Connection, input IN) (OUT, error)

func (tool ToolOperation[IN, OUT]) Resolve(conn *db.Connection) mcp.ToolHandlerFor[IN, OUT] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in IN) (*mcp.CallToolResult, OUT, error) {
		out, err := tool(ctx, conn, in)
		return nil, out, err
	}
}

func (tool ToolOperation[IN, OUT]) Register(srv *mcp.Server, dbconn *db.Connection, info irt.KV[string, string]) {
	mcp.AddTool(srv, &mcp.Tool{
		Name:        info.Key,
		Description: info.Value,
	}, tool.Resolve(dbconn))
}

func New(conn *db.Connection) fnx.Worker {
	srv := mcp.NewServer(
		&mcp.Implementation{
			Name:    "odem",
			Title:   "Fasola Minutes Data",
			Version: "v0.1.0",
		}, nil)

	return func(ctx context.Context) error {
		return srv.Run(ctx, &mcp.LoggingTransport{
			Writer:    send.MakeWriterSender(logger.Plain(ctx).Sender()),
			Transport: &mcp.StdioTransport{},
		})
	}
}
