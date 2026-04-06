package reportui

import (
	"context"

	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/ers"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/infra"
	"github.com/tychoish/odem/pkg/models"
)

func SelectLeader(ctx context.Context, dbconn *db.Connection, input string) (*models.LeaderProfile, error) {
	var ec erc.Collector
	profiles := erc.HandleAll(dbconn.AllLeaderProfiles(ctx), ec.Push)
	if !ec.Ok() {
		return nil, ec.Resolve()
	}

	matches := irt.Collect(infra.NewFuzzySearch[models.LeaderProfile](profiles).
		WithToString(models.MenuFormat).
		Search(input))

	if len(matches) == 0 {
		return nil, ers.Error("leader not found")
	}

	grip.Debug(message.NewKV().
		KV("outcome", "resolved leader").
		KV("leader", matches[0].Name).
		KV("matches", len(matches)))

	return &matches[0], nil
}

func SelectSinging(ctx context.Context, conn *db.Connection, name string) (*models.SingingInfo, error) {
	var ec erc.Collector
	singings := irt.Collect(erc.HandleAll(conn.AllSingings(ctx), ec.Push))
	if !ec.Ok() {
		return nil, ec.Resolve()
	}

	matches := irt.Collect(infra.NewFuzzySearch[models.SingingInfo](singings).
		WithToString(models.MenuFormat).
		Search(name))

	if len(matches) == 0 {
		return nil, ers.Error("singing not found")
	}

	grip.Debug(message.NewKV().
		KV("outcome", "resolved leader").
		KV("singing", matches[0].SingingName).
		KV("matches", len(matches)))

	return &matches[0], nil
}

func SelectSong(ctx context.Context, conn *db.Connection, name string) (*models.SongDetail, error) {
	songs, err := erc.FromIteratorAll(conn.AllSongDetails(ctx))
	if err != nil {
		return nil, err
	}

	matches := irt.Collect(infra.NewFuzzySearch[models.SongDetail](songs).
		WithToString(models.MenuFormat).Search(name))

	if len(matches) == 0 {
		return nil, ers.Error("song not found")
	}
	return &matches[0], nil
}
