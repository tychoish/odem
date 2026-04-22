package reportui

import (
	"io"
	"os"
	"strconv"
	"time"

	"github.com/tychoish/fun/ers"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/odem/pkg/infra"
	"github.com/tychoish/odem/pkg/models"
	"github.com/tychoish/odem/pkg/selector"
)

func itoa(in int) string                                  { return strconv.Itoa(in) }
func sumLens(s []string) (l int)                          { irt.ForEach(irt.Slice(s), func(s string) { l += len(s) }); return }
func flush(wr io.Writer, payload io.WriterTo) (err error) { _, err = payload.WriteTo(wr); return }

// Params is the collection of arguments for generating a
type Params struct {
	models.Params // query parameters
	// Prefix (directory, etc.) of the path to write a report to.
	PathPrefix string
	// ToStdout write report to standard out.
	ToStdout              bool      // for reportUI only
	ToWriter              io.Writer // for tgbot
	SuppressInteractivity bool      // when true do not fall back to interactive fuzzy search
}

// WithoutInteraction returns a params struct that tells the
// implementation to avoid interaction.
func (p Params) WithoutInteraction() Params { p.SuppressInteractivity = true; return p }

func (p Params) Search() *infra.SearchParams {
	return new(infra.SearchParams).With(p.Name).Interaction(!p.SuppressInteractivity)
}

func (p Params) selectYears() ([]int, error) {
	if p.SuppressInteractivity {
		if len(p.Years) > 0 {
			return p.Years, nil
		}

		return []int{time.Now().Year() - 1}, nil
	}
	if len(p.Years) > 0 {
		p.Name = irt.JoinStringsWith(irt.Chain(irt.Args(
			irt.Args(p.Name),
			irt.Convert(irt.Slice(p.Years), strconv.Itoa),
		)), " ")
	}

	return selector.Years(p.Search())
}

// getWriter returns an io.Writer (stdout or a new file) plus a cleanup func.
// The caller must call cleanup() when done. For stdout, cleanup is a no-op.
func (params Params) getWriter(tags ...string) (io.WriteCloser, error) {
	switch {
	case params.ToStdout:
		return wrapWriter(os.Stdout), nil
	case params.ToWriter != nil:
		return wrapWriter(params.ToWriter), nil
	case len(tags) == 0:
		return nil, ers.New("must specify a file name for the report")
	}

	if len(params.PathPrefix) != 0 && !infra.FileExists(params.PathPrefix) {
		if err := os.MkdirAll(params.PathPrefix, 0o755); err != nil {
			return nil, err
		}
	}

	f, err := getFile(params.PathPrefix, tags...)
	if err != nil {
		return nil, err
	}

	return &wrcloselog{name: tags[0], f: f}, nil
}
