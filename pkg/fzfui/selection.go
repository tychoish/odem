package fzfui

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

func SelectSong(ctx context.Context, dbconn *db.Connection, input string) (*models.SongDetail, error) {
	details, err := erc.FromIteratorAll(dbconn.AllSongDetails(ctx))
	if err != nil {
		return nil, err
	}

	res, err := infra.FuzzySearchWithFallback(
		details,
		models.MenuFormat,
		new(infra.SearchParams).With(input).WithPrompt("song"),
		noop,
	)
	if err != nil {
		return nil, err
	}
	match := erc.MustOk(irt.Initial(res))

	grip.Debugln("resolved song:", match.MenuFormat())

	return &match, nil
}

func SelectLeader(ctx context.Context, dbconn *db.Connection, input string) (*models.LeaderProfile, error) {
	profiles, err := erc.FromIteratorAll(dbconn.AllLeaderProfiles(ctx))
	if err != nil {
		return nil, err
	}

	res, err := infra.FuzzySearchWithFallback(
		profiles,
		models.MenuFormat,
		new(infra.SearchParams).With(input).WithPrompt("leader"),
		noop,
	)
	if err != nil {
		return nil, err
	}
	match := erc.MustOk(irt.Initial(res))

	grip.Debugln("resolved leader:", match)
	return &match, nil
}

func SelectSinging(ctx context.Context, dbconn *db.Connection, input string) (*models.SingingInfo, error) {
	options, err := erc.FromIteratorAll(dbconn.AllSingings(ctx))
	if err != nil {
		return nil, err
	}

	res, err := infra.FuzzySearchWithFallback(
		options,
		models.MenuFormat,
		new(infra.SearchParams).With(input).WithPrompt("leader"),
		noop,
	)
	if err != nil {
		return nil, err
	}
	match := erc.MustOk(irt.Initial(res))

	grip.Debugln("selected singing", match.SingingName)
	return &match, nil
}

func SelectKey(ctx context.Context, conn *db.Connection, input string) (string, error) {
	keys, err := erc.FromIteratorAll(conn.AllKeys(ctx))
	if err != nil {
		return "", err
	}

	match, err := infra.FuzzySearchWithFallback(
		keys,
		noop,
		new(infra.SearchParams).With(input).WithPrompt("key"),
		noop,
	)
	if err != nil {
		return "", err
	}

	grip.Debugln("selected key", match)

	return erc.MustOk(irt.Initial(match)), nil
}

// SelectYears parses years from userInput (space-separated integers) or
// prompts the user with a fuzzy multi-selector. Selecting or passing 0 means
// "all years" and returns nil, nil. An empty selection also returns nil, nil.
func SelectYears(userInput []string) ([]int, error) {
	if len(userInput) > 1 {
		return erc.FromIteratorAll(irt.With2(irt.Slice(userInput), strconv.Atoi))
	}

	return erc.FromIteratorAll(infra.FuzzySearchWithFallback(
		irt.Collect(infra.YearSelectorRange(1995)),
		strconv.Itoa,
		new(infra.SearchParams).With(idxorz(userInput, 0)).WithPrompt("years (0 = all)").WithMulti(),
		noop[int],
	))
}
