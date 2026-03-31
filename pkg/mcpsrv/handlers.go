package mcpsrv

import (
	"cmp"
	"context"

	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/models"
	"github.com/tychoish/odem/pkg/reportui"
)

type ContextToolOutput[T any] struct {
	Context string
	Results []T
}

func NeverSung(ctx context.Context, conn *db.Connection, p models.Params) (*ContextToolOutput[models.LeaderSongRank], error) {
	leader, err := reportui.SelectLeader(ctx, conn, p.Name)
	if err != nil {
		return nil, err
	}

	lessons, err := erc.FromIteratorUntil(irt.Limit2(conn.NeverSung(ctx, leader.Name), cmp.Or(p.Limit, 20)))
	if err != nil {
		return nil, err
	}

	return &ContextToolOutput[models.LeaderSongRank]{
		Results: lessons,
		Context: leader.Name,
	}, nil
}
