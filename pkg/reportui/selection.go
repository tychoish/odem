package reportui

import (
	"context"
	"fmt"

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

	toString := func(l models.LeaderProfile) string {
		return fmt.Sprintf("%s (%d-%d) -- %d lesson(s) [%d unique] at %d singing(s)",
			l.Name, l.FirstYear, l.LastYear, l.LessonCount, l.UniqueLessonCount, l.SingingCount,
		)
	}

	matches := irt.Collect(infra.NewFuzzySearch[models.LeaderProfile](profiles).
		WithToString(toString).
		Search(input))

	if len(matches) == 0 {
		return nil, ers.Error("not found")
	}

	grip.Debug(message.NewKV().
		KV("outcome", "resolved leader").
		KV("leader", matches[0].Name).
		KV("matches", len(matches)))

	return &matches[0], nil
}
