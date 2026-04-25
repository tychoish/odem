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
	"github.com/tychoish/odem/pkg/home"
)

//go:generate sqlc generate

type Configuration struct {
	Settings struct {
		Level          level.Priority `bson:"log_level" json:"log_level" yaml:"log_level"`
		ManualReloadDB bool           `bson:"manual_reload_db" json:"manual_reload_db" yaml:"manual_reload_db"`
	} `bson:"settings" json:"settings" yaml:"settings"`
	Telegram struct {
		BotToken             string `bson:"bot_token" json:"bot_token" yaml:"bot_token"`
		Quiet                bool   `bson:"quiet" json:"quiet" yaml:"quiet"`
		MaxSelectionAttempts int    `bson:"max_selection_attempts" json:"max_selection_attempts" yaml:"max_selection_attempts"`
		Webhook              struct {
			Enabled bool   `bson:"enabled" json:"enabled" yaml:"enabled"`
			URL     string `bson:"url" json:"url" yaml:"url"`
			Listen  string `bson:"listen" json:"listen" yaml:"listen"`
		} `bson:"webhook" json:"webhook" yaml:"webhook"`
	} `bson:"telegram" json:"telegram" yaml:"telegram"`
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
		Tag     string `bson:"-" json:"-" yaml:"-"`
		Path    string `bson:"path" json:"path" yaml:"path"`
		Targets []struct {
			GOOS   string `bson:"GOOS" json:"GOOS" yaml:"GOOS"`
			GOARCH string `bson:"GOARCH" json:"GOARCH" yaml:"GOARCH"`
		} `bson:"targets" json:"targets" yaml:"targets"`
		Version            string `bson:"version" json:"version" yaml:"version"`
		DisableCompression bool   `bson:"disable_compression" json:"disable_compression" yaml:"disable_compression"`
		LocalRepoPath      string `bson:"local_repo_path" json:"local_repo_path" yaml:"local_repo_path"`
		Deploy             struct {
			Remote        string `bson:"remote" json:"remote" yaml:"remote"`
			GlobalService bool   `bson:"global" json:"global" yaml:"global"`
			Intstance     string `bson:"-" json:"-" yaml:"-"`
			Target        string `bson:"-" json:"-" yaml:"-"`
		} `bson:"deploy" json:"deploy" yaml:"deploy"`
		BinaryLink string `bson:"binary_link" json:"binary_link" yaml:"binary_link"`
	} `bson:"build" json:"build" yaml:"build"`
	Runtime struct {
		DryRun    bool
		RemoteMCP bool
		Hostname  string
	} `bson:"-" json:"-" yaml:"-"`
}

func ReadConfiguration(paths ...string) (_ *Configuration, err error) {
	pwd := erc.Must(os.Getwd())
	home := home.GetDirectory()
	var ec erc.Collector
	defer func() { err = ec.Resolve() }()

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
	)), fileExists) {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer ec.Check(f.Close)

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

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
