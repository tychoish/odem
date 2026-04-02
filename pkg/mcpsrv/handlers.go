package mcpsrv

import (
	"cmp"
	"context"
	"strings"

	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/stw"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/models"
	"github.com/tychoish/odem/pkg/reportui"
)

type ContextualSequence[C, T any] struct {
	Context C
	Results []T
}

func NeverSung(ctx context.Context, conn *db.Connection, p models.Params) (*ContextualSequence[string, models.LeaderSongRank], error) {
	leader, err := reportui.SelectLeader(ctx, conn, p.Name)
	if err != nil {
		return nil, err
	}

	lessons, err := erc.FromIteratorUntil(irt.Limit2(conn.NeverSung(ctx, leader.Name), cmp.Or(p.Limit, 20)))
	if err != nil {
		return nil, err
	}

	return &ContextualSequence[string, models.LeaderSongRank]{
		Results: lessons,
		Context: leader.Name,
	}, nil
}

func NeverLed(ctx context.Context, conn *db.Connection, p models.Params) (*ContextualSequence[string, models.LeaderSongRank], error) {
	leader, err := reportui.SelectLeader(ctx, conn, p.Name)
	if err != nil {
		return nil, err
	}

	results, err := erc.FromIteratorUntil(irt.Limit2(conn.NeverLed(ctx, leader.Name), cmp.Or(p.Limit, 20)))
	if err != nil {
		return nil, err
	}

	return &ContextualSequence[string, models.LeaderSongRank]{
		Results: results,
		Context: leader.Name,
	}, nil
}

func MostLeadSongs(ctx context.Context, conn *db.Connection, p models.Params) (*ContextualSequence[string, models.LeaderSongRank], error) {
	leader, err := reportui.SelectLeader(ctx, conn, p.Name)
	if err != nil {
		return nil, err
	}

	results, err := erc.FromIteratorUntil(conn.MostLeadSongs(ctx, leader.Name, cmp.Or(p.Limit, 24)))
	if err != nil {
		return nil, err
	}

	return &ContextualSequence[string, models.LeaderSongRank]{
		Results: results,
		Context: leader.Name,
	}, nil
}

func LeaderLeadHistory(ctx context.Context, conn *db.Connection, p models.Params) (*ContextualSequence[string, models.LessonInfo], error) {
	leader, err := reportui.SelectLeader(ctx, conn, p.Name)
	if err != nil {
		return nil, err
	}

	results, err := erc.FromIteratorUntil(irt.Limit2(conn.LeaderLeadHistory(ctx, leader.Name), cmp.Or(p.Limit, 100)))
	if err != nil {
		return nil, err
	}

	return &ContextualSequence[string, models.LessonInfo]{
		Results: results,
		Context: leader.Name,
	}, nil
}

func LeaderSingings(ctx context.Context, conn *db.Connection, p models.Params) (*ContextualSequence[string, models.LeaderSingingAttendance], error) {
	leader, err := reportui.SelectLeader(ctx, conn, p.Name)
	if err != nil {
		return nil, err
	}

	results, err := erc.FromIteratorUntil(conn.LeaderSingingsAttended(ctx, leader.Name, cmp.Or(p.Limit, 0)))
	if err != nil {
		return nil, err
	}

	return &ContextualSequence[string, models.LeaderSingingAttendance]{
		Results: results,
		Context: leader.Name,
	}, nil
}

func Songs(ctx context.Context, conn *db.Connection, p models.Params) (*ContextualSequence[models.SongDetail, models.LeaderOfSongInfo], error) {
	sg, err := reportui.SelectSong(ctx, conn, p.Name)
	if err != nil {
		return nil, err
	}

	leaders, err := erc.FromIteratorUntil(conn.TopLeadersOfSong(ctx, sg.PageNum, cmp.Or(p.Limit, 20)))
	if err != nil {
		return nil, err
	}

	return &ContextualSequence[models.SongDetail, models.LeaderOfSongInfo]{
		Context: stw.Deref(sg),
		Results: leaders,
	}, nil
}

func Singings(ctx context.Context, conn *db.Connection, p models.Params) (*ContextualSequence[models.SingingInfo, models.SingingLessionInfo], error) {
	info, err := reportui.SelectSinging(ctx, conn, p.Name)
	if err != nil {
		return nil, err
	}

	lessons, err := erc.FromIteratorUntil(conn.SingingLessons(ctx, info.SingingName))
	if err != nil {
		return nil, err
	}

	return &ContextualSequence[models.SingingInfo, models.SingingLessionInfo]{
		Context: *info,
		Results: lessons,
	}, nil
}

func Buddies(ctx context.Context, conn *db.Connection, p models.Params) (*ContextualSequence[string, irt.KV[string, int]], error) {
	leader, err := reportui.SelectLeader(ctx, conn, p.Name)
	if err != nil {
		return nil, err
	}

	results, err := erc.FromIteratorUntil(conn.SingingBuddies(ctx, leader.Name, cmp.Or(p.Limit, 24)))
	if err != nil {
		return nil, err
	}

	return &ContextualSequence[string, irt.KV[string, int]]{
		Results: results,
		Context: leader.Name,
	}, nil
}

func Strangers(ctx context.Context, conn *db.Connection, p models.Params) (*ContextualSequence[string, irt.KV[string, int]], error) {
	leader, err := reportui.SelectLeader(ctx, conn, p.Name)
	if err != nil {
		return nil, err
	}

	results, err := erc.FromIteratorUntil(conn.SingingStrangers(ctx, leader.Name, cmp.Or(p.Limit, 24)))
	if err != nil {
		return nil, err
	}

	return &ContextualSequence[string, irt.KV[string, int]]{
		Results: results,
		Context: leader.Name,
	}, nil
}

func PopularInOnesExperience(ctx context.Context, conn *db.Connection, p models.Params) (*ContextualSequence[string, models.LeaderSongRank], error) {
	leader, err := reportui.SelectLeader(ctx, conn, p.Name)
	if err != nil {
		return nil, err
	}

	results, err := erc.FromIteratorUntil(conn.PopularSongsInOnesExperience(ctx, leader.Name, cmp.Or(p.Limit, 25)))
	if err != nil {
		return nil, err
	}

	return &ContextualSequence[string, models.LeaderSongRank]{
		Results: results,
		Context: leader.Name,
	}, nil
}

func PopularInYears(ctx context.Context, conn *db.Connection, p models.Params) (*ContextualSequence[string, models.LeaderSongRank], error) {
	results, err := erc.FromIteratorUntil(irt.Limit2(conn.GloballyPopularForYears(ctx, p.Years...), cmp.Or(p.Limit, 25)))
	if err != nil {
		return nil, err
	}

	return &ContextualSequence[string, models.LeaderSongRank]{
		Results: results,
	}, nil
}

func LocallyPopular(ctx context.Context, conn *db.Connection, p models.Params) (*ContextualSequence[string, models.LeaderSongRank], error) {
	var localities []models.SingingLocality
	for part := range strings.SplitSeq(p.Name, ",") {
		localities = append(localities, models.NewSingingLocality(strings.TrimSpace(part)))
	}

	results, err := erc.FromIteratorUntil(conn.LocallyPopular(ctx, cmp.Or(p.Limit, 32), localities...))
	if err != nil {
		return nil, err
	}

	return &ContextualSequence[string, models.LeaderSongRank]{
		Results: results,
		Context: p.Name,
	}, nil
}

func UnfamilarHits(ctx context.Context, conn *db.Connection, p models.Params) (*ContextualSequence[string, models.LeaderSongRank], error) {
	leader, err := reportui.SelectLeader(ctx, conn, p.Name)
	if err != nil {
		return nil, err
	}

	results, err := erc.FromIteratorUntil(conn.TheUnfamilarHits(ctx, leader.Name, cmp.Or(p.Limit, 20)))
	if err != nil {
		return nil, err
	}

	return &ContextualSequence[string, models.LeaderSongRank]{
		Results: results,
		Context: leader.Name,
	}, nil
}

func LeaderFavoriteKey(ctx context.Context, conn *db.Connection, p models.Params) (*ContextualSequence[string, irt.KV[string, int]], error) {
	leader, err := reportui.SelectLeader(ctx, conn, p.Name)
	if err != nil {
		return nil, err
	}

	results, err := erc.FromIteratorUntil(conn.LeaderFavoriteKey(ctx, leader.Name, cmp.Or(p.Limit, 20)))
	if err != nil {
		return nil, err
	}

	return &ContextualSequence[string, irt.KV[string, int]]{
		Results: results,
		Context: leader.Name,
	}, nil
}

func Connectedness(ctx context.Context, conn *db.Connection, p models.Params) (*ContextualSequence[string, irt.KV[string, float64]], error) {
	results, err := erc.FromIteratorUntil(conn.AllLeaderConnectedness(ctx, cmp.Or(p.Limit, 40)))
	if err != nil {
		return nil, err
	}

	return &ContextualSequence[string, irt.KV[string, float64]]{
		Results: results,
	}, nil
}

func LeaderFootsteps(ctx context.Context, conn *db.Connection, p models.Params) (*ContextualSequence[string, models.LeaderFootstep], error) {
	leader, err := reportui.SelectLeader(ctx, conn, p.Name)
	if err != nil {
		return nil, err
	}

	results, err := erc.FromIteratorUntil(conn.LeaderFootsteps(ctx, leader.Name, cmp.Or(p.Limit, 20)))
	if err != nil {
		return nil, err
	}

	return &ContextualSequence[string, models.LeaderFootstep]{
		Results: results,
		Context: leader.Name,
	}, nil
}

func TopLeaders(ctx context.Context, conn *db.Connection, p models.Params) (*ContextualSequence[string, models.LeaderLeadCount], error) {
	results, err := erc.FromIteratorUntil(conn.TopLeadersByLeads(ctx, cmp.Or(p.Limit, 40), p.Years...))
	if err != nil {
		return nil, err
	}

	return &ContextualSequence[string, models.LeaderLeadCount]{
		Results: results,
	}, nil
}

// LeaderShareOutput is the result of a leader's share-of-leads query.
type LeaderShareOutput struct {
	Leader string
	Years  []int
	Share  float64
}

func LeaderShare(ctx context.Context, conn *db.Connection, p models.Params) (*LeaderShareOutput, error) {
	leader, err := reportui.SelectLeader(ctx, conn, p.Name)
	if err != nil {
		return nil, err
	}

	share, err := conn.LeaderShareOfLeads(ctx, leader.Name, p.Years...)
	if err != nil {
		return nil, err
	}

	v := stw.DerefZ(share)

	return &LeaderShareOutput{
		Leader: leader.Name,
		Years:  p.Years,
		Share:  v,
	}, nil
}
