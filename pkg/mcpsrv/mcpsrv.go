package mcpsrv

import (
	"context"
	"fmt"
	"iter"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tychoish/fun/fnx"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/srv"
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/send"
	"github.com/tychoish/odem"
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

type RegistrationFunc func(srv *mcp.Server, dbconn *db.Connection, info irt.KV[string, string])

func (tool ToolOperation[IN, OUT]) Register(srv *mcp.Server, dbconn *db.Connection, info irt.KV[string, string]) {
	mcp.AddTool(srv, &mcp.Tool{
		Name:        info.Key,
		Description: info.Value,
	}, tool.Resolve(dbconn))
}

func New(conf *odem.Configuration, conn *db.Connection, seq iter.Seq2[irt.KV[string, string], RegistrationFunc]) fnx.Worker {
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "odem",
			Title:   "Fasola Minutes Data",
			Version: "v0.1.0",
		}, nil)

	for info, reg := range seq {
		reg(server, conn, info)
	}

	if conf.Runtime.RemoteMCP {
		grip.Info("creating http service...")
		return srv.HTTP("odem-mcp",
			time.Minute,
			&http.Server{
				Addr: fmt.Sprintf("%s:%d", conf.Services.Address, conf.Services.Port),
				Handler: mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
					return server
				}, nil),
			}).Worker()
	}

	return func(ctx context.Context) error {
		grip.Info("starting stdio service...")
		return server.Run(ctx, &mcp.LoggingTransport{
			Writer:    send.MakeWriterSender(logger.Plain(ctx).Sender()),
			Transport: &mcp.StdioTransport{},
		})
	}
}
