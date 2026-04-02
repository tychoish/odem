package odem

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/jasper/util"
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
	}
	Services struct {
		Address string `bson:"addr" json:"addr" yaml:"addr"`
		Port    int    `bson:"port" json:"port" yaml:"port"`
	} `bson:"services" json:"services" yaml:"services"`
}

func ReadConfiguration() (*Configuration, error) {
	pwd := erc.Must(os.Getwd())
	home := util.GetHomedir()
	var ec erc.Collector
	for path := range irt.Args(
		filepath.Join(pwd, ".odem.yml"),
		filepath.Join(pwd, ".odem.yaml"),
		filepath.Join(pwd, ".odem.json"),
		filepath.Join(home, ".odem.yml"),
		filepath.Join(home, ".odem.yaml"),
		filepath.Join(home, ".odem.json"),
		filepath.Join(home, ".config", "odem", "conf.yml"),
		filepath.Join(home, ".config", "odem", "conf.yaml"),
		filepath.Join(home, ".config", "odem", "conf.json"),
	) {
		if util.FileExists(path) {
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
