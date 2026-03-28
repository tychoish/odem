package reportui

import (
	"os"

	"github.com/tychoish/fun/strut"
	"github.com/tychoish/grip"
)

func getFile(args ...string) (*os.File, error) {
	mut := strut.MakeMutable(sumLens(args) + 3)
	defer mut.Release()
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

type wstdout struct {
	*os.File
}

func (wstdout) Close() error { return nil }

type loggingCloser struct {
	reportName string
	f          *os.File
}

func (f *loggingCloser) Write(in []byte) (int, error) { return f.f.Write(in) }
func (f *loggingCloser) Close() error {
	grip.Infof("wrote report %s to %s", f.reportName, f.f.Name())
	return f.f.Close()
}
