package dispatch

import (
	"context"

	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/reportui"
)

type MinutesAppOperationHandler func(context.Context, *db.Connection, ...string) error

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
