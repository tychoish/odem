package release

import (
	"bytes"
	"cmp"

	"github.com/tychoish/fun/strut"
	"github.com/tychoish/jasper"
	"github.com/tychoish/jasper/util"
)

var Version string

func init() {
	// TODO this needs to be injected using -ldflags into the build rather than reassed at runtime, but can use the same code.

	mut := strut.MakeMutable(32)
	defer mut.Release()
	buf := util.NewLocalBuffer(bytes.NewBuffer(mut.Bytes()))
	jasper.NewCommand().AppendArgs("git", "describe").SetOutputWriter(buf).Worker().Ignore().Wait()
	Version = cmp.Or(buf.String(), "<pending>")
}
