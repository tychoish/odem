package reportui

import (
	"io"
	"os"
	"strconv"

	"github.com/tychoish/fun/ers"
	"github.com/tychoish/odem/pkg/models"
)

func atoi(in string) (n int)                              { n, _ = strconv.Atoi(in); return }
func flush(wr io.Writer, payload io.WriterTo) (err error) { _, err = payload.WriteTo(wr); return }
func intValToStr(key string, value int) (string, string)  { return key, strconv.Itoa(value) }
func asRows(lsr models.LeaderSongRank) []string           { return (&lsr).StringFields() }

func sumLens(in []string) (total int) {
	for _, s := range in {
		total += len(s)
	}
	return
}

type Params struct {
	Name       string
	Years      []int
	PathPrefix string
	Limit      int
	ToStdout   bool
}

// getWriter returns an io.Writer (stdout or a new file) plus a cleanup func.
// The caller must call cleanup() when done. For stdout, cleanup is a no-op.
func (params Params) getWriter(tags ...string) (io.WriteCloser, error) {
	if params.ToStdout {
		return wstdout{File: os.Stdout}, nil
	}
	if len(tags) == 0 {
		return nil, ers.New("must specify a file name for the report")
	}
	f, err := getFile(tags...)
	if err != nil {
		return nil, err
	}

	return &loggingCloser{reportName: tags[0], f: f}, nil
}
