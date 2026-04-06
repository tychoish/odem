package reportui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tychoish/fun/ers"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/jasper/util"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/fzfui"
	"github.com/tychoish/odem/pkg/models"
)

func itoa(in int) string                                  { return strconv.Itoa(in) }
func sumLens(s []string) (l int)                          { irt.ForEach(irt.Slice(s), func(s string) { l += len(s) }); return }
func flush(wr io.Writer, payload io.WriterTo) (err error) { _, err = payload.WriteTo(wr); return }
func intValToStr(key string, value int) (string, string)  { return key, strconv.Itoa(value) }
func fmtPercentKVs(k string, v float64) (string, string)  { return k, fmt.Sprintf("%.4f%%", v*100) }
func asRows(lsr models.LeaderSongRank) []string           { return (&lsr).StringFields() }

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

func (p Params) SelectLeader(ctx context.Context, conn *db.Connection) (string, error) {
	if p.SuppressInteractivity {
		out, err := SelectLeader(ctx, conn, p.Name)
		if err != nil {
			return "", err
		}
		return out.Name, nil
	}

	return fzfui.SelectLeader(ctx, conn, p.Name)
}

func (p Params) SelectSong(ctx context.Context, conn *db.Connection) (*models.SongDetail, error) {
	if p.SuppressInteractivity {
		out, err := SelectSong(ctx, conn, p.Name)
		if err != nil {
			return nil, err
		}
		return out, nil
	}

	return fzfui.SelectSong(ctx, conn, p.Name)
}

func (p Params) SelectSiging(ctx context.Context, conn *db.Connection) (*models.SingingInfo, error) {
	if p.SuppressInteractivity {
		out, err := SelectSinging(ctx, conn, p.Name)
		if err != nil {
			return nil, err
		}
		return out, nil
	}

	return fzfui.SelectSinging(ctx, conn, p.Name)
}

func (p Params) SelectYears(ctx context.Context, conn *db.Connection) ([]int, error) {
	if p.SuppressInteractivity {
		if len(p.Params.Years) > 0 {
			return p.Params.Years, nil
		}

		return []int{time.Now().Year() - 1}, nil
	}

	return fzfui.SelectYears(p.Name)
}

func (p Params) SelectLocality(ctx context.Context, conn *db.Connection) ([]string, error) {
	if len(p.Name) > 0 {
		return strings.Split(p.Name, ","), nil
	}
	if p.SuppressInteractivity {
		return nil, errors.New("headless locality selection not implemented")
	}

	return nil, ers.New("not implemented")
}

func (p Params) SelectKey(ctx context.Context, conn *db.Connection) (string, error) {
	if p.SuppressInteractivity {
		if len(p.Name) > 0 {
			return p.Name, nil
		}
		return "", errors.New("headless key selection not implemented")
	}

	return fzfui.SelectKey(ctx, conn, p.Name)
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

	if len(params.PathPrefix) != 0 && !util.FileExists(params.PathPrefix) {
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
