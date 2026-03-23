// Package ep (entrypoint) holds the implementation of the CLI
// handling/dispatching glue for all the commands.
package ep

import (
	"context"

	"github.com/tychoish/cmdr"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/odem/pkg/db"
	"github.com/urfave/cli/v3"
)

func Version() *cmdr.Commander {
	return cmdr.MakeCommander().
		SetName("version").
		Aliases("v").
		SetUsage("returns the version and build information of the binary").
		SetAction(func(ctx context.Context, cc *cli.Command) error {
			grip.Log(grip.Sender().Priority(), message.Fields{
				"name":    cc.Name,
				"version": cc.Version,
			})

			return nil
		})
}

func Hacking() *cmdr.Commander {
	return cmdr.MakeCommander().
		SetName("hacking").
		Aliases("hack").
		SetUsage("hacking and testing").
		SetAction(func(ctx context.Context, cc *cli.Command) error {
			conn, err := db.Connect(ctx)
			grip.EmergencyPanic(err)
			var ec erc.Collector

			for name := range erc.HandleAll(conn.AllLeaderNames(ctx), ec.Push) {
				grip.Info(name)
			}

			return ec.Resolve()
		})
}
