package odem

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/goccy/go-yaml"
	"github.com/tychoish/cmdr"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/jasper/util"
	"github.com/urfave/cli/v3"
)

type Configuration struct {
	Settings struct {
		Level    level.Priority `bson:"log_level" json:"log_level" yaml:"log_level"`
		ReloadDB string         `bson:"reload_db" json:"reload_db" yaml:"reload_db"`
	} `bson:"settings" json:"settings" yaml:"settings"`
	Reports struct {
		BasePath string `bson:"base_path" json:"base_path" yaml:"base_path"`
		Batches  []struct {
			Name    string   `bson:"name" json:"name" yaml:"name"`
			Leaders []string `bson:"leaders" json:"leaders" yaml:"leaders"`
		} `bson:"batches" json:"batches" yaml:"batches"`
	} `bson:"reports" json:"reports" yaml:"reports"`
	Services struct {
		Address string `bson:"addr" json:"addr" yaml:"addr"`
		Port    int    `bson:"port" json:"port" yaml:"port"`
	} `bson:"services" json:"services" yaml:"services"`
	Build struct {
		Path    string `bson:"path" json:"path" yaml:"path"`
		Targets []struct {
			GOOS   string `bson:"GOOS" json:"GOOS" yaml:"GOOS"`
			GOARCH string `bson:"GOARCH" json:"GOARCH" yaml:"GOARCH"`
		} `bson:"targets" json:"targets" yaml:"targets"`
		Version            string `bson:"version" json:"version" yaml:"version"`
		DisableCompression bool   `bson:"disable_compression" json:"disable_compression" yaml:"disable_compression"`
	} `bson:"build" json:"build" yaml:"build"`
	Runtime struct {
		DryRun    bool
		RemoteMCP bool
	} `bson:"-" json:"-" yaml:"-"`
}

func AttachConfiguration(c *cmdr.Commander) {
	c.Flags(
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
	).With(cmdr.SpecBuilder(func(ctx context.Context, cc *cli.Command) (*Configuration, error) {
		conf, err := ReadConfiguration(util.TryExpandHomedir(cmdr.GetFlag[string](cc, "conf")))
		if err != nil {
			return nil, err
		}

		conf.Runtime.RemoteMCP = cmdr.GetFlag[bool](cc, "http")
		conf.Settings.Level = cmp.Or(level.FromString(cmdr.GetFlag[string](cc, "level")), conf.Settings.Level, level.Info)
		conf.Reports.BasePath = cmp.Or(conf.Reports.BasePath, filepath.Join(erc.Must(os.Getwd()), "build"))
		conf.Services.Port = cmp.Or(cmdr.GetFlag[int](cc, "port"), conf.Services.Port, 1844)
		conf.Services.Address = cmp.Or(cmdr.GetFlag[string](cc, "addr"), conf.Services.Address, "127.0.0.1")
		conf.Build.Path = cmp.Or(conf.Build.Path, "build")

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
	}).SetMiddleware(WithConfiguration).Add)
}

func ReadConfiguration(paths ...string) (*Configuration, error) {
	pwd := erc.Must(os.Getwd())
	home := util.GetHomedir()
	var ec erc.Collector

	for path := range irt.Keep(irt.Chain(irt.Args(
		irt.Slice(paths),
		irt.Args(filepath.Join(pwd, ".odem.yml"),
			filepath.Join(pwd, ".odem.yaml"),
			filepath.Join(pwd, ".odem.json"),
			filepath.Join(home, ".odem.yml"),
			filepath.Join(home, ".odem.yaml"),
			filepath.Join(home, ".odem.json"),
			filepath.Join(home, ".config", "odem", "conf.yml"),
			filepath.Join(home, ".config", "odem", "conf.yaml"),
			filepath.Join(home, ".config", "odem", "conf.json")),
	)), util.FileExists) {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		newDecoder := newDecoderForFile(path)
		dec := newDecoder(f)

		var conf Configuration
		if err := dec.Decode(&conf); err != nil {
			ec.Wrap(err, path)
			continue
		}
		return &conf, nil
	}
	if ec.Ok() {
		return &Configuration{}, nil
	}

	return nil, ec.Resolve()
}

type confCtxKey struct{}

func WithConfiguration(ctx context.Context, conf *Configuration) context.Context {
	return context.WithValue(ctx, confCtxKey{}, conf)
}

func GetConfiguration(ctx context.Context) *Configuration {
	conf, ok := ctx.Value(confCtxKey{}).(*Configuration)
	if ok {
		return conf
	}
	return nil
}

func newDecoderForFile(path string) func(io.Reader) interface{ Decode(any) error } {
	switch filepath.Ext(path) {
	case ".json":
		return func(in io.Reader) interface{ Decode(any) error } { return json.NewDecoder(in) }
	case ".yaml", ".yml":
		return func(in io.Reader) interface{ Decode(any) error } { return yaml.NewDecoder(in) }
	default:
		panic(erc.NewInvariantError("cannot resolve decoder for", path))
	}
}
