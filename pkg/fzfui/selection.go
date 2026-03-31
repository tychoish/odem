package fzfui

import (
	"context"
	"fmt"
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
		WithToString(func(in models.SongDetail) string {
			return fmt.Sprintf("pg %s -- %s", in.PageNum, in.SongTitle)
		}).Search(strings.Join(args, " "))
	// TODO skip this is we didnt get ny results and go back to normal
	sdIdx := map[models.SongDetail]int{}
	for i, v := range songDetails {
		sdIdx[v] = i
	}
	preselction := []int{}
	for detail := range sg {
		if sidx, ok := sdIdx[detail]; ok == true {
			preselction = append(preselction, sidx)
		}
	}
	res, err := infra.NewFuzzySearch[models.SongDetail](songDetails).
		WithSelections(preselction).
		WithToString(func(in models.SongDetail) string {
			return fmt.Sprintf("pg %s -- %s", in.PageNum, in.SongTitle)
		}).FindOne()
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

	toString := func(l models.LeaderProfile) string {
		return fmt.Sprintf("%s (%d-%d) -- %d lesson(s) [%d unique] at %d singing(s)",
			l.Name, l.FirstYear, l.LastYear, l.LessonCount, l.UniqueLessonCount, l.SingingCount,
		)
	}

	if input != "" {
		matches := irt.Collect(infra.NewFuzzySearch[models.LeaderProfile](profiles).
			WithToString(toString).
			Search(input))
		if len(matches) == 1 {
			grip.Debugln("resolved leader", matches[0].Name)
			return matches[0].Name, nil
		}
	}

	leader, err := infra.NewFuzzySearch[models.LeaderProfile](profiles).
		WithToString(toString).
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
		WithToString(func(info models.SingingInfo) string {
			return fmt.Sprintf("%s -- %s (%s)",
				info.SingingDate.Time().Format("2006-01-02"),
				strings.ReplaceAll(info.SingingName, "\\n", "; "),
				info.SingingLocation,
			)
		}).
		Prompt("leaders").
		FindOne()

	if !ec.PushOk(err) || !ec.Ok() {
		return nil, ec.Resolve()
	}
	grip.Debugln("selected singing", singing.SingingName)
	return &singing, nil
}

// TODO implement exported/reusable SelectSong handler

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
