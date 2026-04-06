package infra

import (
	"context"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/tychoish/cmdr"
	"github.com/tychoish/fun/fnx"
	"github.com/tychoish/fun/srv"
	"github.com/tychoish/jasper"
	"github.com/tychoish/odem"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/logger"
	"github.com/tychoish/odem/pkg/release"
	"github.com/urfave/cli/v3"
)

type WithInput[T any] struct {
	DB   *db.Connection
	Conf *odem.Configuration
	Args T
}

// DBOperationSpec builds a cmdr.OperationSpec[WithInput[string]] whose
// constructor connects to the database and captures the first positional
// CLI argument, then calls action(ctx, conn, query) as its operation.
func DBOperationSpec[T cmdr.FlagTypes](action func(context.Context, *db.Connection, T) error) func(*cmdr.Commander) {
	return func(cc *cmdr.Commander) { cc.With(MakeDBOperationSpec("name", action)) }
}

func SimpleDBOperationSpec(action func(context.Context, *db.Connection) error) func(*cmdr.Commander) {
	return func(cc *cmdr.Commander) {
		cc.With(MakeDBOperationSpec("name", func(ctx context.Context, db *db.Connection, _ string) error { return action(ctx, db) }))
	}
}

// DBOperationSpecWith builds a cmdr.OperationSpec that connects to the database
// and extracts arbitrary input from the CLI command. Unlike DBOperationSpec, T
// is unconstrained so callers can bundle multiple flags into a struct.
func DBOperationSpecWith[T any](
	extract func(*cli.Command) T,
	action func(context.Context, *db.Connection, T) error,
) func(*cmdr.Commander) {
	return func(cc *cmdr.Commander) {
		cc.With(AttachConfiguration).With(cmdr.SpecBuilder(
			func(ctx context.Context, cc *cli.Command) (*WithInput[T], error) {
				conn, err := db.Connect(ctx)
				if err != nil {
					return nil, err
				}
				return &WithInput[T]{Conf: odem.GetConfiguration(ctx), DB: conn, Args: extract(cc)}, nil
			},
		).SetAction(func(ctx context.Context, in *WithInput[T]) error { return action(ctx, in.DB, in.Args) }).Add)
	}
}

func MakeDBOperationSpec[T cmdr.FlagTypes](argName string, action func(context.Context, *db.Connection, T) error) func(cc *cmdr.Commander) {
	return func(cc *cmdr.Commander) {
		cc.With(AttachConfiguration).With(cmdr.SpecBuilder(
			func(ctx context.Context, cc *cli.Command) (*WithInput[T], error) {
				conn, err := db.Connect(ctx)
				if err != nil {
					return nil, err
				}

				return &WithInput[T]{Conf: odem.GetConfiguration(ctx), DB: conn, Args: cmdr.GetFlagOrFirstArg[T](cc, argName)}, nil
			},
		).SetAction(func(ctx context.Context, in *WithInput[T]) error { return action(ctx, in.DB, in.Args) }).Add)
	}
}

type cancelCtxKey struct{}

func withCanceler(ctx context.Context) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	return context.WithValue(ctx, cancelCtxKey{}, cancel)
}

func GetCanceler(ctx context.Context) context.CancelFunc {
	return ctx.Value(cancelCtxKey{}).(context.CancelFunc)
}

func MainCLI(name string, cmdrs ...*cmdr.Commander) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	cmdr.Main(ctx, cmdr.MakeRootCommander().
		SetName(name).
		SetAppOptions(cmdr.AppOptions{Name: name, Usage: "🚩 🌞 🔲 stats application", Version: release.Version.Resolve().String()}).
		Middleware(logger.Setup).
		Middleware(withCanceler).
		Middleware(JasperSetup).
		EnableCompletionCmd().
		With(RootHelpAction).
		Subcommanders(cmdrs...),
	)
}

func JasperSetup(ctx context.Context) context.Context {
	jpm := jasper.NewManager(jasper.ManagerOptionDefaults(), jasper.ManagerOptionMaxProcs(runtime.NumCPU()))
	srv.AddCleanup(ctx, jpm.Close)
	return jasper.WithManager(ctx, jpm)
}

func RootHelpAction(cmd *cmdr.Commander) {
	cmd.SetAction(func(ctx context.Context, cc *cli.Command) error { return cli.DefaultShowRootCommandHelp(cc) })
}

func CommandHelpAction(cmd *cmdr.Commander) {
	cmd.SetAction(func(ctx context.Context, cc *cli.Command) error { return cli.DefaultShowSubcommandHelp(cc) })
}

func WorkerAction(op fnx.Worker) func(cmd *cmdr.Commander) {
	return func(cmd *cmdr.Commander) {
		cmd.SetAction(func(ctx context.Context, cc *cli.Command) error { return op(ctx) })
	}
}
