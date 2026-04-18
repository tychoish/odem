package selector

import (
	"context"
	"iter"
	"strconv"
	"strings"

	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/grip"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/infra"
	"github.com/tychoish/odem/pkg/models"
)

func noop[T any](in T) T { return in }

func Leader(ctx context.Context, dbconn *db.Connection, sp *infra.SearchParams) (*models.LeaderProfile, error) {
	profiles, err := erc.FromIteratorAll(dbconn.AllLeaderProfiles(ctx))
	if err != nil {
		return nil, err
	}

	res, err := infra.FuzzySearchWithFallback(
		profiles,
		models.MenuFormat,
		sp.WithPrompt("leader").UseFirstResult(),
		noop,
	)
	if err != nil {
		return nil, err
	}
	match := erc.MustOk(irt.Initial(irt.Ptrs(res)))

	grip.Debug(grip.MPrintln("resolved leader:", match))

	return match, nil
}

func Singing(ctx context.Context, conn *db.Connection, sp *infra.SearchParams) (*models.SingingInfo, error) {
	singings, err := erc.FromIteratorAll(conn.AllSingings(ctx))
	if err != nil {
		return nil, err
	}

	res, err := infra.FuzzySearchWithFallback(
		singings,
		models.MenuFormat,
		sp.WithPrompt("singing").UseFirstResult(),
		noop,
	)
	if err != nil {
		return nil, err
	}
	match := erc.MustOk(irt.Initial(irt.Ptrs(res)))

	grip.Debug(grip.MPrintln("resolved singing:", match.MenuFormat()))

	return match, nil
}

func Song(ctx context.Context, dbconn *db.Connection, sp *infra.SearchParams) (*models.SongDetail, error) {
	details, err := erc.FromIteratorAll(dbconn.AllSongDetails(ctx))
	if err != nil {
		return nil, err
	}

	res, err := infra.FuzzySearchWithFallback(
		details,
		models.MenuFormat,
		sp.WithPrompt("song").UseFirstResult(),
		noop,
	)
	if err != nil {
		return nil, err
	}
	match := erc.MustOk(irt.Initial(irt.Ptrs(res)))

	grip.Debug(grip.MPrintln("resolved song:", match.MenuFormat()))

	return match, nil
}

func Key(ctx context.Context, conn *db.Connection, sp *infra.SearchParams) (string, error) {
	keys, err := erc.FromIteratorAll(conn.AllKeys(ctx))
	if err != nil {
		return "", err
	}

	match, err := infra.FuzzySearchWithFallback(
		keys,
		noop,
		sp.WithPrompt("key").UseFirstResult(),
		noop,
	)
	if err != nil {
		return "", err
	}

	grip.Debug(grip.MPrintln("selected key", match))

	return erc.MustOk(irt.Initial(match)), nil
}

func Years(sp *infra.SearchParams) ([]int, error) {
	years, err := infra.FuzzySearchWithFallback(
		irt.Collect(infra.YearSelectorRange(1995)),
		strconv.Itoa,
		sp.WithPrompt("years (0 = all)").WithMulti(),
		noop[int],
	)
	if err != nil {
		return nil, err
	}
	return irt.Collect(years), nil
}

func Locality(sp *infra.SearchParams) (models.SingingLocality, error) {
	match, err := infra.FuzzySearchWithFallback(
		models.AllLocalities(),
		func(l models.SingingLocality) string { return string(l) },
		sp.WithPrompt("locality").UseFirstResult(),
		noop[models.SingingLocality],
	)
	if err != nil {
		return "", err
	}
	return erc.MustOk(irt.Initial(match)), nil
}

// Concordance provides a selector for a unique list of allwords (less some amount of stemming.)
func Concordance(ctx context.Context, conn *db.Connection, sp *infra.SearchParams) (string, error) {
	songTexts, err := erc.FromIteratorAll(conn.AllSongTexts(ctx))
	if err != nil {
		return "", err
	}
	lyrics := irt.Unique(irt.Chain(irt.Convert(irt.Slice(songTexts), func(sl models.SongLyrics) iter.Seq[string] { return strings.SplitSeq(sl.Text, "\n") })))
	lyrics = irt.Unique(irt.Chain(irt.Convert(lyrics, func(in string) iter.Seq[string] { return strings.SplitSeq(in, " ") })))
	lyrics = irt.Convert(lyrics, func(in string) string { return strings.Trim(in, " \t\n'\",‘’“”!?.;:[]{}/\\") })
	lyrics = irt.Convert(irt.RemoveZeros(lyrics), strings.ToLower)

	lyrics = irt.Keep(lyrics, func(in string) bool {
		switch in {
		case "a", "s":
			return false
		case "as", "or", "it", "by", "my", "on", "is", "in", "of", "i", "an":
			return false
		case "the", "has", "you", "for":
			return false
		case "from", "then", "there", "mine":
			return false
		case ";", ",", ".", "-", "?", "!":
			return false
		default:
			return true
		}
	})

	res, err := infra.FuzzySearchWithFallback(
		irt.Collect(irt.Unique(lyrics)),
		noop,
		sp.WithPrompt("concordance"),
		noop)
	if err != nil {
		return "", err
	}
	return erc.MustOk(irt.Initial(res)), nil
}
