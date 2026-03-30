package infra

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/tychoish/cmdr"
	"github.com/tychoish/odem"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/logger"
	"github.com/urfave/cli/v3"
)

type WithInput[T any] struct {
	DB   *db.Connection
	Conf odem.Configuration
	Args T
}

// DBOperationSpec builds a cmdr.OperationSpec[WithInput[string]] whose
// constructor connects to the database and captures the first positional
// CLI argument, then calls action(ctx, conn, query) as its operation.
func DBOperationSpec[T cmdr.FlagTypes](action func(context.Context, *db.Connection, T) error) *cmdr.OperationSpec[*WithInput[T]] {
	return MakeDBOperationSpec("name", action)
}

func SimpleDBOperationSpec(action func(context.Context, *db.Connection) error) *cmdr.OperationSpec[*WithInput[string]] {
	return MakeDBOperationSpec("name", func(ctx context.Context, db *db.Connection, _ string) error { return action(ctx, db) })
}

// DBOperationSpecWith builds a cmdr.OperationSpec that connects to the database
// and extracts arbitrary input from the CLI command. Unlike DBOperationSpec, T
// is unconstrained so callers can bundle multiple flags into a struct.
func DBOperationSpecWith[T any](
	extract func(*cli.Command) T,
	action func(context.Context, *db.Connection, T) error,
) *cmdr.OperationSpec[*WithInput[T]] {
	return cmdr.SpecBuilder(
		func(ctx context.Context, cc *cli.Command) (*WithInput[T], error) {
			conn, err := db.Connect(ctx)
			if err != nil {
				return nil, err
			}
			return &WithInput[T]{DB: conn, Args: extract(cc)}, nil
		},
	).SetAction(func(ctx context.Context, in *WithInput[T]) error { return action(ctx, in.DB, in.Args) })
}

func MakeDBOperationSpec[T cmdr.FlagTypes](argName string, action func(context.Context, *db.Connection, T) error) *cmdr.OperationSpec[*WithInput[T]] {
	return cmdr.SpecBuilder(
		func(ctx context.Context, cc *cli.Command) (*WithInput[T], error) {
			conn, err := db.Connect(ctx)
			if err != nil {
				return nil, err
			}

			return &WithInput[T]{DB: conn, Args: cmdr.GetFlagOrFirstArg[T](cc, argName)}, nil
		},
	).SetAction(func(ctx context.Context, in *WithInput[T]) error { return action(ctx, in.DB, in.Args) })
}

func MainCLI(name string, cmdrs ...*cmdr.Commander) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	cmdr.Main(ctx, cmdr.MakeRootCommander().
		SetName(name).
		Middleware(logger.Setup).
		EnableCompletionCmd().
		SetAction(func(ctx context.Context, cc *cli.Command) error {
			return cli.ShowAppHelp(cc)
		}).
		Subcommanders(cmdrs...),
	)
}
