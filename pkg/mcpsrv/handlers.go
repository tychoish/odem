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

type NeverSungOutput struct {
	Leader  string
	Lessons []models.LeaderSongRank
}

func NeverSung(ctx context.Context, conn *db.Connection, p models.Params) (*NeverSungOutput, error) {
	leader, err := reportui.SelectLeader(ctx, conn, p.Name)
	if err != nil {
		return nil, err
	}

	lessons, err := erc.FromIteratorUntil(irt.Limit2(conn.NeverSung(ctx, leader.Name), cmp.Or(p.Limit, 20)))
	if err != nil {
		return nil, err
	}

	return &NeverSungOutput{Lessons: lessons, Leader: p.Name}, nil
}
