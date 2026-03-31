package dispatch

import (
	"context"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/reportui"
)

type MinutesAppOperationHandler func(context.Context, *db.Connection, ...string) error

func FuzzyySingleHandler(op func(context.Context, *db.Connection, string) error) MinutesAppOperationHandler {
	return func(c context.Context, d *db.Connection, a ...string) error { return op(c, d, idxorz(a, 0)) }
}

func FuzzyyHandler(op func(context.Context, *db.Connection) error) MinutesAppOperationHandler {
	return func(c context.Context, d *db.Connection, a ...string) error { return op(c, d) }
}

func FuzzyHandlerWithJoinArgs(op func(context.Context, *db.Connection, string) error) MinutesAppOperationHandler {
	return func(c context.Context, d *db.Connection, a ...string) error { return op(c, d, strings.Join(a, " ")) }
}

func (maoh MinutesAppOperationHandler) Handle(ctx context.Context, conn *db.Connection, args ...string) error {
	return maoh.Op(ctx, conn, args)
}

func (maoh MinutesAppOperationHandler) Op(ctx context.Context, conn *db.Connection, args []string) error {
	return maoh(ctx, conn, args...)
}

type Reporter func(context.Context, *db.Connection, reportui.Params) error

func (r Reporter) Report(ctx context.Context, conn *db.Connection, params reportui.Params) error {
	return r(ctx, conn, params)
}

type ToolOperation[IN, OUT any] func(ctx context.Context, input IN) (OUT, error)

func NewMCPTool[A, B any](op func(context.Context, A) (B, error)) ToolOperation[A, B] { return op }

func (to ToolOperation[IN, OUT]) Resolve() mcp.ToolHandlerFor[IN, OUT] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in IN) (*mcp.CallToolResult, OUT, error) {
		out, err := to(ctx, in)
		return nil, out, err
	}
}
