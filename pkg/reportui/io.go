package reportui

import (
	"io"
	"os"
	"path/filepath"

	"github.com/tychoish/fun/adt"
	"github.com/tychoish/fun/strut"
	"github.com/tychoish/grip"
)

func getFile(dir string, args ...string) (*os.File, error) {
	mut := strut.MakeMutable(len(dir) + sumLens(args) + 3)
	defer mut.Release()
	mut.PushString(dir)
	if len(dir) > 1 && !mut.HasSuffix([]byte{filepath.Separator}) {
		mut.PushBytes([]byte{filepath.Separator})
	}
	mut.JoinStrings(args, "-")
	mut.ReplaceAllString(" ", "-")
	mut.ReplaceAllString("'", "-")
	mut.ReplaceAllString(".", "")
	mut.ToLower()
	mut.PushString(".md")

	f, err := os.Create(mut.String())
	if err != nil {
		return nil, err
	}

	return f, nil
}

type wstdout[W io.Writer] struct{ adt.Once[W] }

func wrapWriter[W io.Writer](in W) *wstdout[W]    { return new(wstdout[W]).with(in) }
func (w *wstdout[W]) with(in W) *wstdout[W]       { w.Set(func() W { return in }); return w }
func (w *wstdout[W]) Write(b []byte) (int, error) { return w.Resolve().Write(b) }
func (*wstdout[w]) Close() error                  { return nil }

type wrcloselog struct {
	name string
	f    *os.File
}

const wrclTmpl = "wrote %q to %s"

func (f *wrcloselog) Write(in []byte) (int, error) { return f.f.Write(in) }
func (f *wrcloselog) Close() error                 { grip.Info(grip.MPrintf(wrclTmpl, f.name, f.f.Name())); return f.f.Close() }
