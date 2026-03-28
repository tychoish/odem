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

func selectLeader(ctx context.Context, dbconn *db.Connection) (string, error) {
	var ec erc.Collector

	names := irt.Collect(erc.HandleAll(dbconn.AllLeaderNames(ctx), ec.Push))
	if !ec.Ok() {
		return "", ec.Resolve()
	}

	leader, err := infra.NewFuzzySearch[string](names).FindOne("leaders")
	if !ec.PushOk(err) {
		return "", ec.Resolve()
	}

	grip.Debugln("selected leader", leader)
	return leader, nil
}

func selectSinging(ctx context.Context, dbconn *db.Connection) (*models.SingingInfo, error) {
	var ec erc.Collector

	singings := irt.Collect(erc.HandleAll(dbconn.AllSingings(ctx), ec.Push))
	singing, err := infra.NewFuzzySearch[models.SingingInfo](singings).
		WithToString(func(info models.SingingInfo) string {
			return fmt.Sprintf("%s -- %s (%s)", info.SingingDate.Time().Format("2006-01-02"), strings.Split(info.SingingName, "\\n")[0], info.SingingLocation)
		}).
		FindOne("leaders")

	if !ec.PushOk(err) || !ec.Ok() {
		return nil, ec.Resolve()
	}
	grip.Debugln("selected singing", singing.SingingName)
	return &singing, nil
}

func interactivelyResolveSingerName(ctx context.Context, conn *db.Connection, singer string) (string, error) {
	if singer != "" {
		return singer, nil
	}

	singer, err := selectLeader(ctx, conn)
	if err != nil {
		return "", err
	}
	if singer == "" {
		return "", ers.New("not found")
	}

	return singer, nil
}

// selectYears parses years from userInput (comma-separated integers) or
// prompts the user with a fuzzy selector. Selecting or passing 0 means
// "all years" and returns nil, nil. An empty selection also returns nil, nil.
func selectYears(userInput string) ([]int, error) {
	if userInput != "" {
		years, err := erc.FromIteratorAll(
			irt.With2(
				irt.Slice(strings.Split(userInput, ",")),
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
	).Find("years (0 = all)"))

	if err != nil {
		return nil, err
	}
	return filterYears(years), nil
}

// filterYears removes 0 from the slice (0 = all years sentinel).
// Returns nil if the result would be empty or contained only zeros.
func filterYears(years []int) []int {
	var out []int
	for _, y := range years {
		if y != 0 {
			out = append(out, y)
		}
	}
	return out
}
