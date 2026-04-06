package fzfui

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/ers"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/grip"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/infra"
	"github.com/tychoish/odem/pkg/models"
)

func SelectSong(ctx context.Context, dbconn *db.Connection, args ...string) (*models.SongDetail, error) {
	songDetails, err := erc.FromIteratorAll(dbconn.AllSongDetails(ctx))
	if err != nil {
		return nil, err
	}

	sg := infra.NewFuzzySearch[models.SongDetail](songDetails).
		WithToString(models.MenuFormat).
		Search(strings.Join(args, " "))

	sdIdx := map[models.SongDetail]int{}
	for i, v := range songDetails {
		sdIdx[v] = i
	}
	preselction := []int{}
	for detail := range sg {
		if sidx, ok := sdIdx[detail]; ok {
			preselction = append(preselction, sidx)
		}
	}

	fs := infra.NewFuzzySearch[models.SongDetail](songDetails).
		WithToString(models.MenuFormat)

	if len(preselction) > 0 {
		fs = fs.WithSelections(preselction)
	}
	res, err := fs.FindOne()
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func SelectLeader(ctx context.Context, dbconn *db.Connection, input string) (string, error) {
	var ec erc.Collector
	profiles := erc.HandleAll(dbconn.AllLeaderProfiles(ctx), ec.Push)
	if !ec.Ok() {
		return "", ec.Resolve()
	}

	if input != "" {
		matches := irt.Collect(infra.NewFuzzySearch[models.LeaderProfile](profiles).
			WithToString(models.MenuFormat).
			Search(input))
		if len(matches) == 1 {
			grip.Debugln("resolved leader", matches[0].Name)
			return matches[0].Name, nil
		}
	}

	leader, err := infra.NewFuzzySearch[models.LeaderProfile](profiles).
		WithToString(models.MenuFormat).
		FindOne()
	if err != nil {
		return "", err
	}

	grip.Debugln("selected leader", leader)
	return leader.Name, nil
}

func SelectSinging(ctx context.Context, dbconn *db.Connection, args ...string) (*models.SingingInfo, error) {
	var ec erc.Collector

	singings := irt.Collect(erc.HandleAll(dbconn.AllSingings(ctx), ec.Push))
	singing, err := infra.NewFuzzySearch[models.SingingInfo](singings).
		WithToString(models.MenuFormat).
		Prompt("leaders").
		FindOne()

	if !ec.PushOk(err) || !ec.Ok() {
		return nil, ec.Resolve()
	}
	grip.Debugln("selected singing", singing.SingingName)
	return &singing, nil
}

func SelectKey(ctx context.Context, conn *db.Connection, input string) (string, error) {
	if input != "" {
		return input, nil
	}
	keys, err := erc.FromIteratorAll(conn.AllKeys(ctx))
	if err != nil {
		return "", err
	}
	return infra.NewFuzzySearch[string](keys).Prompt("key").FindOne()
}

func interactivelyResolveSingerName(ctx context.Context, conn *db.Connection, singer string) (string, error) {
	if singer != "" {
		return singer, nil
	}

	singer, err := SelectLeader(ctx, conn, singer)
	if err != nil {
		return "", err
	}
	if singer == "" {
		return "", ers.New("not found")
	}

	return singer, nil
}

// SelectYears parses years from userInput (comma-separated integers) or
// prompts the user with a fuzzy selector. Selecting or passing 0 means
// "all years" and returns nil, nil. An empty selection also returns nil, nil.
func SelectYears(userInput string) ([]int, error) {
	if userInput != "" {
		years, err := erc.FromIteratorAll(
			irt.With2(
				irt.Slice(strings.Split(userInput, " ")),
				strconv.Atoi,
			),
		)
		switch {
		case err != nil:
			return nil, err
		case len(years) != 0:
			return filterYears(years), nil
		}
	}

	currentYear := time.Now().Year()

	years, err := erc.FromIteratorAll(infra.NewFuzzySearch[int](
		irt.Chain(irt.Args(
			irt.Args(0), // 0 = all years (no filter)
			irt.While(irt.MonotonicFrom(1995), func(v int) bool { return v < currentYear }),
			irt.While(irt.MonotonicFrom(-1*currentYear), func(v int) bool { return v < -1995 }),
		)),
	).Prompt("years (0 = all)").Find())
	if err != nil {
		return nil, err
	}
	return filterYears(years), nil
}
