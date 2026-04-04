package infra

import (
	"cmp"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/tychoish/cmdr"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/jasper/util"
	"github.com/tychoish/odem"
	"github.com/urfave/cli/v3"
)

func AttachConfiguration(c *cmdr.Commander) { confCmdrFlags(c).With(confCmdr().Add) }

func Operation(op func(context.Context, *odem.Configuration) error) func(*cmdr.Commander) {
	return func(cc *cmdr.Commander) { confCmdrFlags(cc).With(confCmdr().SetAction(op).Add) }
}

func confCmdrFlags(c *cmdr.Commander) *cmdr.Commander {
	return c.Flags(
		cmdr.FlagBuilder("info").
			SetName("level").
			SetUsage("specify logging threshold: emergency|alert|critical|error|warning|notice|info|debug").
			SetValidate(func(val string) error {
				priority := level.FromString(val)
				if priority == level.Invalid {
					return fmt.Errorf("%q is not a valid logging level", val)
				}
				return nil
			}).Flag(),
		cmdr.FlagBuilder("~/.odem.yaml").
			SetName("conf").
			SetUsage("Set the path to override the default config file path").
			Flag(),
	)
}

func confCmdr() *cmdr.OperationSpec[*odem.Configuration] {
	return cmdr.SpecBuilder(func(ctx context.Context, cc *cli.Command) (*odem.Configuration, error) {
		conf, err := odem.ReadConfiguration(util.TryExpandHomedir(cmdr.GetFlag[string](cc, "conf")))
		if err != nil {
			return nil, err
		}

		conf.Runtime.RemoteMCP = cmdr.GetFlag[bool](cc, "http")
		conf.Runtime.DryRun = cmdr.GetFlag[bool](cc, "dry-run")
		conf.Settings.Level = cmp.Or(level.FromString(cmdr.GetFlag[string](cc, "level")), conf.Settings.Level, level.Info)
		conf.Reports.BasePath = cmp.Or(conf.Reports.BasePath, filepath.Join(erc.Must(os.Getwd()), "build"))
		conf.Services.Port = cmp.Or(cmdr.GetFlag[int](cc, "port"), conf.Services.Port, 1844)
		conf.Services.Address = cmp.Or(cmdr.GetFlag[string](cc, "addr"), conf.Services.Address, "127.0.0.1")
		conf.Build.Path = cmp.Or(conf.Build.Path, "build")
		conf.Build.Tag = cmdr.GetFlag[string](cc, "tag")

		if len(conf.Build.Targets) == 0 {
			conf.Build.Targets = append(conf.Build.Targets, struct {
				GOOS   string `bson:"GOOS" json:"GOOS" yaml:"GOOS"`
				GOARCH string `bson:"GOARCH" json:"GOARCH" yaml:"GOARCH"`
			}{
				GOOS:   runtime.GOOS,
				GOARCH: runtime.GOARCH,
			})
		}

		grip.Sender().SetPriority(conf.Settings.Level)
		return conf, nil
	}).SetMiddleware(odem.WithConfiguration)
}
