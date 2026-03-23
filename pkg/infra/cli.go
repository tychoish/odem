package infra

import (
	"context"
	"encoding/json"
	"os/signal"
	"syscall"

	"github.com/goccy/go-yaml"
	"github.com/tychoish/cmdr"
	"github.com/tychoish/fun/erc"
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
func DBOperationSpec(action func(context.Context, *db.Connection, string) error) *cmdr.OperationSpec[*WithInput[string]] {
	return MakeDBOperationSpec("name", action)
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
	).SetAction(func(ctx context.Context, in *WithInput[T]) error {
		return action(ctx, in.DB, in.Args)
	})
}

func MainCLI(name string, cmdrs ...*cmdr.Commander) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	logger.SetupDefault()

	cmdr.Main(ctx, cmdr.MakeRootCommander().
		SetName(name).
		Middleware(logger.WithPlain).
		EnableCompletionCmd().
		SetAction(func(ctx context.Context, cc *cli.Command) error {
			return cli.ShowAppHelp(cc)
		}).
		Subcommanders(cmdrs...),
	)
}

type JSON[T any] struct{ inner T }

func NewJSON[T any](in T) JSON[T] { return JSON[T]{inner: in} }

func (j JSON[T]) String() string { return string(erc.Must(json.Marshal(j.inner))) }

type YAML[T any] struct{ inner T }

func NewYAML[T any](in T) YAML[T] { return YAML[T]{inner: in} }

func (j YAML[T]) String() string { return string(erc.Must(yaml.Marshal(j.inner))) }
