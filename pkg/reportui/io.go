package reportui

import (
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

type wstdout struct{ adt.Once[*os.File] }

func (w *wstdout) stdout() *os.File            { return w.Do(w.init) }
func (w *wstdout) init() *os.File              { return os.Stdout }
func (w *wstdout) Write(b []byte) (int, error) { return w.stdout().Write(b) }
func (*wstdout) Close() error                  { return nil }

type wrcloselog struct {
	name string
	f    *os.File
}

const wrclTmpl = "wrote %q to %s"

func (f *wrcloselog) Write(in []byte) (int, error) { return f.f.Write(in) }
func (f *wrcloselog) Close() error                 { grip.Infof(wrclTmpl, f.name, f.f.Name()); return f.f.Close() }
