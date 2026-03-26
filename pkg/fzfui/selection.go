package fzfui

import (
	"context"
	"fmt"
	"strings"

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
