package dispatch

import (
	"context"
	"strings"

	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/reportui"
)

// TODO unify FuzzHandler and Reporter: both should take reportui.Params which should move to models
// (for now, and wrap/combine with the existing models.Params). The underlying packages could also
// be combined: fzf is just reports with interactivity and with output written to stdout. both
// _could_ take arguments on the command line to pre-populate Params structs.
//
// Eventually, there should be a "super-fzf" interface that would render output using that would
// jump _through_ successive fzf menus "leader -> [which query] -> [fzf songs rendered with
// LineItem] -> song selection -> [top leaders, other queries we can look at] -> "<views>", however
// implementation of that CAN ONLY begin after begin the current report/fzf interface is rationalized.
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
