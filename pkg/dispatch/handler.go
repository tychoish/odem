package dispatch

import (
	"context"
	"strings"

	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/reportui"
)

type FuzzHandler func(context.Context, *db.Connection, string) error

func SimpleFuzzyHandler(op func(context.Context, *db.Connection) error) FuzzHandler {
	return func(c context.Context, d *db.Connection, _ string) error { return op(c, d) }
}

func (maoh FuzzHandler) Handle(ctx context.Context, conn *db.Connection, args ...string) error {
	return maoh.Op(ctx, conn, args)
}

func (maoh FuzzHandler) Op(ctx context.Context, conn *db.Connection, args []string) error {
	return maoh(ctx, conn, strings.Join(args, " "))
}

type Reporter func(context.Context, *db.Connection, reportui.Params) error

func (r Reporter) Report(ctx context.Context, conn *db.Connection, params reportui.Params) error {
	return r(ctx, conn, params)
}
