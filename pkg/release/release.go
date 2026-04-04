package release

import (
	"bytes"
	"cmp"
	"context"
	"time"

	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/ers"
	"github.com/tychoish/grip"
	"github.com/tychoish/jasper"
	"github.com/tychoish/jasper/util"
)

var (
	version   string
	buildTime string
)

func init() {
	// TODO this needs to be injected using -ldflags into the build rather than reassed at runtime, but can use the same code.
}

func GetVersion() string      { return version }
func GetBuildTime() time.Time { return erc.Must(time.Parse(time.RFC3339, buildTime)) }
func GitDescribe(ctx context.Context) string {
	b := new(bytes.Buffer)
	buf := util.NewLocalBuffer(b)

	err := jasper.Context(ctx).CreateCommand(ctx).AppendArgs("git", "describe").SetOutputWriter(buf).Run(ctx)
	grip.Warning(ers.Wrap(err, "git describe for release versioning"))

	return cmp.Or(string(bytes.TrimSpace(b.Bytes())), "<UNKNOWN>")
}
