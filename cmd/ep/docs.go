package ep

import (
	"context"
	"io"
	"os"

	"github.com/tychoish/cmdr"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/fnx"
	"github.com/tychoish/odem"
	"github.com/tychoish/odem/pkg/infra"
)

func Docs() *cmdr.Commander {
	return cmdr.MakeCommander().
		SetName("docs").
		SetUsage("access to 'odem' package documentation").
		With(infra.RootHelpAction).
		Subcommanders(
			cmdr.MakeCommander().SetName("readme").Aliases("README").With(infra.WorkerAction(printDocs("README.md"))),
			cmdr.MakeCommander().SetName("licenses").Aliases("LICENSES").With(infra.WorkerAction(printDocs("LICENSE"))),
		)
}

func printDocs(path string) fnx.Worker {
	return func(ctx context.Context) (err error) {
		var ec erc.Collector
		defer func() { err = ec.Resolve() }()
		defer ec.Recover()

		f, err := odem.GetFile(path)
		if !ec.PushOk(err) {
			return
		}
		defer func() { ec.Push(f.Close()) }()

		_, err = io.Copy(os.Stderr, f)
		return err
	}
}
