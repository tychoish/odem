// Package ep (entrypoint) holds the implementation of the CLI
// handling/dispatching glue for all the commands.
package ep

import (
	"context"

	"github.com/tychoish/cmdr"
	"github.com/tychoish/fun/fnx"
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/odem"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/infra"
	"github.com/tychoish/odem/pkg/release"
	"github.com/urfave/cli/v3"
)

func Version() *cmdr.Commander {
	return cmdr.MakeCommander().
		SetName("version").
		Aliases("v").
		SetUsage("returns the version and build information of the binary").
		SetAction(func(ctx context.Context, cc *cli.Command) error {
			grip.Log(grip.Sender().Priority(), message.Fields{
				"name":       cc.Name,
				"version":    cc.Version,
				"release":    release.Version.Resolve().String(),
				"build_time": release.BuildTime.Resolve(),
			})

			return nil
		})
}

func Setup() *cmdr.Commander {
	return cmdr.MakeCommander().
		SetName("setup").
		SetUsage("initialize the cached/local database").
		With(infra.AttachConfiguration).
		With(infra.WorkerAction(db.Init)).
		Subcommanders(
			cmdr.MakeCommander().
				SetName("reset").
				SetUsage("remove the cached/local database").
				With(infra.WorkerAction(fnx.MakeWorker(db.Reset))),
		)
}

func Hacking() *cmdr.Commander {
	return cmdr.MakeCommander().
		SetName("hacking").
		Aliases("hack").
		SetHidden(true).
		SetUsage("hacking and testing").
		With(infra.AttachConfiguration).
		SetAction(func(ctx context.Context, cc *cli.Command) error {
			grip.Info(grip.MPrintln("🤖 🎶", release.Version.Resolve()))
			for k, v := range infra.IterStruct(odem.GetConfiguration(ctx)) {
				grip.Info(grip.MPrintf("%s -> %+v", k, v))
			}
			return nil
		})
}
