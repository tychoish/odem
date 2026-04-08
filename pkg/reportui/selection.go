package reportui

import (
	"context"
	"strconv"

	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/grip"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/infra"
	"github.com/tychoish/odem/pkg/models"
)

func SelectLeader(ctx context.Context, dbconn *db.Connection, sp *infra.SearchParams) (*models.LeaderProfile, error) {
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

	grip.Debugln("resolved leader:", match)

	return match, nil
}

func SelectSinging(ctx context.Context, conn *db.Connection, sp *infra.SearchParams) (*models.SingingInfo, error) {
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

	grip.Debugln("resolved singing:", match.MenuFormat())

	return match, nil
}

func SelectSong(ctx context.Context, dbconn *db.Connection, sp *infra.SearchParams) (*models.SongDetail, error) {
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

	grip.Debugln("resolved song:", match.MenuFormat())

	return match, nil
}

func SelectKey(ctx context.Context, conn *db.Connection, sp *infra.SearchParams) (string, error) {
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

	grip.Debugln("selected key", match)

	return erc.MustOk(irt.Initial(match)), nil
}

func SelectYears(sp *infra.SearchParams) ([]int, error) {
	return erc.FromIteratorAll(infra.FuzzySearchWithFallback(
		irt.Collect(infra.YearSelectorRange(1995)),
		strconv.Itoa,
		new(infra.SearchParams).With(sp.Input).WithPrompt("years (0 = all)").WithMulti(),
		noop[int],
	))
}
