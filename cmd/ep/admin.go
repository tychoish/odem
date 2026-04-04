// Package ep (entrypoint) holds the implementation of the CLI
// handling/dispatching glue for all the commands.
package ep

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tychoish/cmdr"
	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/fnx"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/wpa"
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/jasper"
	"github.com/tychoish/jasper/util"
	"github.com/tychoish/odem"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/infra"
	"github.com/tychoish/odem/pkg/logger"
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
				"name":    cc.Name,
				"version": cc.Version,
			})

			return nil
		})
}

func Setup() *cmdr.Commander {
	return cmdr.MakeCommander().
		SetName("setup").
		SetUsage("initialize the cached/local database").
		SetAction(func(ctx context.Context, cc *cli.Command) error {
			return db.Init()
		}).
		Subcommanders(
			cmdr.MakeCommander().
				SetName("reset").
				SetUsage("remove the cached/local database").
				SetAction(func(ctx context.Context, cc *cli.Command) error {
					return db.Reset()
				}),
		)
}

func Hacking() *cmdr.Commander {
	return cmdr.MakeCommander().
		SetName("hacking").
		Aliases("hack").
		SetUsage("hacking and testing").
		Flags(cmdr.FlagBuilder(false).SetName("http").SetUsage("call to start use the http service").Flag()).
		With(odem.AttachConfiguration).
		SetAction(func(ctx context.Context, cc *cli.Command) error {
			grip.Infoln("🤖 🎶", release.GetVersion())
			for k, v := range infra.IterStruct(odem.GetConfiguration(ctx)) {
				grip.Infoln(k, "->", fmt.Sprintf("%+v", v))
			}
			return nil
		})
}

const wpaName = "odem.wpa"

func Release() *cmdr.Commander {
	return cmdr.MakeCommander().
		SetName("release").
		SetUsage("build artifacts for odem relases; must run inside of the odem git repositry").
		Flags(cmdr.FlagBuilder(false).SetName("dry-run", "n").SetUsage("disables all (most?) imput").Flag()).
		With(odem.AttachConfiguration).
		SetAction(func(ctx context.Context, cc *cli.Command) error {
			versionString := release.GitDescribe(ctx)
			grip.Infoln("🤖 🎶", versionString)
			conf := odem.GetConfiguration(ctx)
			const ldFlagTmpl = `-ldflags=-s -w -X github.com/tychoish/odem/pkg/release.version=%s -X github.com/tychoish/odem.buildTime=%s`
			ldFlag := fmt.Sprintf(ldFlagTmpl, versionString, time.Now().Round(time.Millisecond).Format(time.RFC3339))
			var ec erc.Collector
			dryRun := cmdr.GetFlag[bool](cc, "dry-run")
			jpm := jasper.Context(ctx)
			grip.Sender().SetPriority(level.Trace)

			var jobs dt.List[fnx.Worker]

			for build := range irt.Slice(conf.Build.Targets) {
				binaryPath := filepath.Join(conf.Build.Path, versionString, joindot(build.GOOS, build.GOARCH))
				if !dryRun {
					ec.Push(mkdirdashp(binaryPath))
				}

				var binaryName string
				if build.GOOS == "windows" {
					binaryName = "odem.exe"
				} else {
					binaryName = "odem"
				}

				cmd := jpm.CreateCommand(ctx).
					ID(binaryPath).
					AppendArgs("go", "build", ldFlag, "-o", filepath.Join(binaryPath, binaryName), "./cmd/odem.go").
					AddEnv("GOOS", build.GOOS).
					AddEnv("GOARCH", build.GOARCH).
					RedirectOutputToError(true).
					SetOutputSender(level.Debug, logger.Plain(ctx).Sender()).
					SetErrorSender(level.Error, logger.Plain(ctx).Sender())

				if !conf.Build.DisableCompression && build.GOOS != "darwin" {
					zpath := joindot(binaryPath, "lzma")
					if !dryRun {
						ec.Push(mkdirdashp(zpath))
					}

					cmd.AppendArgs("upx", "-q", "--lzma",
						filepath.Join(binaryPath, binaryName),
						"-o", filepath.Join(zpath, binaryName))
				}

				if dryRun {
					grip.Info(cmd.String())
					continue
				}

				jobs.PushBack(cmd.Worker())
			}

			ec.Push(wpa.RunWithPool(jobs.IteratorFront(), wpa.WorkerGroupConfDefaults()).Run(ctx))

			return ec.Resolve()
		})
}

func joindot(s ...string) string { return strings.Join(s, ".") }

func mkdirdashp(path string) error {
	if util.FileExists(path) {
		return nil
	}
	grip.Infof("making directory %q", path)
	return os.MkdirAll(path, 0o766)
}
