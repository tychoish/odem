// Package ep (entrypoint) holds the implementation of the CLI
// handling/dispatching glue for all the commands.
package ep

import (
	"context"
	"fmt"

	"github.com/tychoish/cmdr"
	"github.com/tychoish/fun/fnx"
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/jasper"
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
		With(infra.WorkerAction(fnx.MakeWorker(db.Init))).
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
		SetUsage("hacking and testing").
		Flags(cmdr.FlagBuilder(false).
			SetName("http").
			SetUsage("call to start use the http service").
			Flag()).
		With(infra.AttachConfiguration).
		SetAction(func(ctx context.Context, cc *cli.Command) error {
			grip.Infoln("🤖 🎶", release.Version.Resolve())
			for k, v := range infra.IterStruct(odem.GetConfiguration(ctx)) {
				grip.Infoln(k, "->", fmt.Sprintf("%+v", v))
			}
			return nil
		})
}

func Release() *cmdr.Commander {
	return cmdr.MakeCommander().
		SetName("release").
		SetUsage("build and release automation").
		With(infra.HelpAction).
		Subcommanders(
			cmdr.MakeCommander().
				SetName("tag").
				SetUsage("shortcut create release tag").
				Flags(cmdr.FlagBuilder("v0.0.0").
					SetName("tag").
					SetRequired(true).
					SetUsage("name of new tag, should start with a V").Flag()).
				SetAction(func(ctx context.Context, cc *cli.Command) error {
					return jasper.Context(ctx).
						CreateCommand(ctx).
						AppendArgs("git", "tag", "--annotate", "--edit", cmdr.GetFlag[string](cc, "tag")).
						Run(ctx)
				}),
			cmdr.MakeCommander().
				SetName("build").
				SetUsage("build artifacts for odem release; must run inside of the odem git repository").
				Flags(cmdr.FlagBuilder(false).
					SetName("dry-run", "n").
					SetUsage("disables all (most?) write operations for some (admin) operations").
					Flag()).
				With(infra.AttachConfiguration).
				With(infra.WorkerAction(release.BuildArtifacts)),
			cmdr.MakeCommander().
				SetName("upload").
				SetUsage("upload built artifacts for the given release tag to GitHub").
				Flags(cmdr.FlagBuilder("").
					SetName("tag").
					SetUsage("git tag / version string to upload (e.g. v1.2.3)").
					SetRequired(true).
					SetValidate(release.ValidateVersion).
					Flag()).
				With(infra.AttachConfiguration).
				With(infra.Operation(release.UploadArtifacts)),
		)
}
