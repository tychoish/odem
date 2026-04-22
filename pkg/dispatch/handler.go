package dispatch

import (
	"context"
	"strings"

	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/models"
	"github.com/tychoish/odem/pkg/reportui"
)

// Eventually, there should be a "super-fzf" interface that would render output using that would
// jump _through_ successive fzf menus "leader -> [which query] -> [fzf songs rendered with
// LineItem] -> song selection -> [top leaders, other queries we can look at] -> "<views>", however
// implementation of that CAN ONLY begin after begin the current report/fzf interface is
// rationalized.
//
// This will need to be implemented in it's own package, as a state machine (an enum/type for each
// state,) operations all map to functions which return functions that take care of the next state.

// Reporter is the common handler type for all operations: it receives a
// db connection and a Params struct and writes output to the writer
// specified in Params (file, stdout, or an io.Writer).
type Reporter func(context.Context, *db.Connection, reportui.Params) error

func (r Reporter) Report(ctx context.Context, conn *db.Connection, params reportui.Params) error {
	return r(ctx, conn, params)
}

// Op calls the reporter with fzf defaults (ToStdout=true) and
// joins args into Params.Name. This is the entry point used by the
// fzf CLI path.
func (r Reporter) Op(ctx context.Context, conn *db.Connection, args []string) error {
	return r(ctx, conn, reportui.Params{
		Params:                models.Params{Name: strings.Join(args, " ")},
		ToStdout:              true,
		SuppressInteractivity: false,
	})
}

// Handle calls Op with variadic args for convenience.
func (r Reporter) Handle(ctx context.Context, conn *db.Connection, args ...string) error {
	return r.Op(ctx, conn, args)
}
