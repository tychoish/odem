package mcpsrv

import (
	"cmp"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/stw"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/infra"
	"github.com/tychoish/odem/pkg/models"
	"github.com/tychoish/odem/pkg/selector"
)

type ContextualSequence[C, T any] struct {
	Context C
	Results []T
}

func searchParams(p models.Params) *infra.SearchParams {
	return new(infra.SearchParams).With(p.Name).WithoutInteractive().UseFirstResult()
}

func NeverSung(ctx context.Context, conn *db.Connection, p models.Params) (*ContextualSequence[string, models.LeaderSongRank], error) {
	leader, err := selector.Leader(ctx, conn, searchParams(p))
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
	leader, err := selector.Leader(ctx, conn, searchParams(p))
	if err != nil {
		return nil, err
	}

	results, err := erc.FromIteratorUntil(conn.NeverLed(ctx, leader.Name, cmp.Or(p.Limit, 20)))
	if err != nil {
		return nil, err
	}

	return &ContextualSequence[string, models.LeaderSongRank]{
		Results: results,
		Context: leader.Name,
	}, nil
}

func MostLeadSongs(ctx context.Context, conn *db.Connection, p models.Params) (*ContextualSequence[string, models.LeaderSongRank], error) {
	leader, err := selector.Leader(ctx, conn, searchParams(p))
	if err != nil {
		return nil, err
	}

	results, err := erc.FromIteratorUntil(conn.MostLedSongs(ctx, leader.Name, cmp.Or(p.Limit, 24)))
	if err != nil {
		return nil, err
	}

	return &ContextualSequence[string, models.LeaderSongRank]{
		Results: results,
		Context: leader.Name,
	}, nil
}

func LeaderLeadHistory(ctx context.Context, conn *db.Connection, p models.Params) (*ContextualSequence[string, models.LessonInfo], error) {
	leader, err := selector.Leader(ctx, conn, searchParams(p))
	if err != nil {
		return nil, err
	}

	results, err := erc.FromIteratorUntil(conn.LeaderLeadHistory(ctx, leader.Name, cmp.Or(p.Limit, 100)))
	if err != nil {
		return nil, err
	}

	return &ContextualSequence[string, models.LessonInfo]{
		Results: results,
		Context: leader.Name,
	}, nil
}

func LeaderSingings(ctx context.Context, conn *db.Connection, p models.Params) (*ContextualSequence[string, models.LeaderSingingAttendance], error) {
	leader, err := selector.Leader(ctx, conn, searchParams(p))
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

func SongsByWord(ctx context.Context, conn *db.Connection, p models.Params) (*ContextualSequence[string, models.SongWordMatch], error) {
	results, err := erc.FromIteratorUntil(conn.SongsByWord(ctx, p.Name, cmp.Or(p.Limit, 50)))
	if err != nil {
		return nil, err
	}
	return &ContextualSequence[string, models.SongWordMatch]{
		Context: p.Name,
		Results: results,
	}, nil
}

func SongLyrics(ctx context.Context, conn *db.Connection, p models.Params) (*ContextualSequence[models.SongLyrics, string], error) {
	sg, err := selector.Song(ctx, conn, searchParams(p))
	if err != nil {
		return nil, err
	}

	sl, err := conn.SongLyrics(ctx, sg.PageNum)
	if err != nil {
		return nil, err
	}

	return &ContextualSequence[models.SongLyrics, string]{
		Context: sl,
	}, nil
}

func Songs(ctx context.Context, conn *db.Connection, p models.Params) (*ContextualSequence[models.SongDetail, models.LeaderOfSongInfo], error) {
	sg, err := selector.Song(ctx, conn, searchParams(p))
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
	info, err := selector.Singing(ctx, conn, searchParams(p))
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

func Buddies(ctx context.Context, conn *db.Connection, p models.Params) (*ContextualSequence[string, models.SingingBuddy], error) {
	leader, err := selector.Leader(ctx, conn, searchParams(p))
	if err != nil {
		return nil, err
	}

	results, err := erc.FromIteratorUntil(conn.SingingBuddies(ctx, leader.Name, cmp.Or(p.Limit, 24)))
	if err != nil {
		return nil, err
	}

	return &ContextualSequence[string, models.SingingBuddy]{
		Results: results,
		Context: leader.Name,
	}, nil
}

func Strangers(ctx context.Context, conn *db.Connection, p models.Params) (*ContextualSequence[string, models.SingingStranger], error) {
	leader, err := selector.Leader(ctx, conn, searchParams(p))
	if err != nil {
		return nil, err
	}

	results, err := erc.FromIteratorUntil(conn.SingingStrangers(ctx, leader.Name, cmp.Or(p.Limit, 24)))
	if err != nil {
		return nil, err
	}

	return &ContextualSequence[string, models.SingingStranger]{
		Results: results,
		Context: leader.Name,
	}, nil
}

func PopularInOnesExperience(ctx context.Context, conn *db.Connection, p models.Params) (*ContextualSequence[string, models.LeaderSongRank], error) {
	leader, err := selector.Leader(ctx, conn, searchParams(p))
	if err != nil {
		return nil, err
	}

	results, err := erc.FromIteratorUntil(conn.PopularAsObserved(ctx, leader.Name, cmp.Or(p.Limit, 25)))
	if err != nil {
		return nil, err
	}

	return &ContextualSequence[string, models.LeaderSongRank]{
		Results: results,
		Context: leader.Name,
	}, nil
}

func PopularInYears(ctx context.Context, conn *db.Connection, p models.Params) (*ContextualSequence[string, models.LeaderSongRank], error) {
	results, err := erc.FromIteratorUntil(irt.Limit2(conn.GloballyPopularForYears(ctx, 20, p.Years...), cmp.Or(p.Limit, 25)))
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
	leader, err := selector.Leader(ctx, conn, searchParams(p))
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

func LeaderFavoriteKey(ctx context.Context, conn *db.Connection, p models.Params) (*ContextualSequence[string, models.LeaderKeyCount], error) {
	leader, err := selector.Leader(ctx, conn, searchParams(p))
	if err != nil {
		return nil, err
	}

	results, err := erc.FromIteratorUntil(conn.LeaderFavoriteKey(ctx, leader.Name, cmp.Or(p.Limit, 20)))
	if err != nil {
		return nil, err
	}

	return &ContextualSequence[string, models.LeaderKeyCount]{
		Results: results,
		Context: leader.Name,
	}, nil
}

func Connectedness(ctx context.Context, conn *db.Connection, p models.Params) (*ContextualSequence[string, models.LeaderConnectedness], error) {
	results, err := erc.FromIteratorUntil(conn.AllLeaderConnectedness(ctx, cmp.Or(p.Limit, 40)))
	if err != nil {
		return nil, err
	}

	return &ContextualSequence[string, models.LeaderConnectedness]{
		Results: results,
	}, nil
}

func LeaderSingingsPerYear(ctx context.Context, conn *db.Connection, p models.Params) (*ContextualSequence[string, models.LeaderSingingsInYear], error) {
	leader, err := selector.Leader(ctx, conn, searchParams(p))
	if err != nil {
		return nil, err
	}

	results, err := erc.FromIteratorUntil(conn.LeaderSingingsPerYear(ctx, leader.Name))
	if err != nil {
		return nil, err
	}

	return &ContextualSequence[string, models.LeaderSingingsInYear]{
		Results: results,
		Context: leader.Name,
	}, nil
}

func LeaderFootsteps(ctx context.Context, conn *db.Connection, p models.Params) (*ContextualSequence[string, models.LeaderFootstep], error) {
	leader, err := selector.Leader(ctx, conn, searchParams(p))
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

func NewLeadersByYear(ctx context.Context, conn *db.Connection, p models.Params) (*ContextualSequence[int, models.LeaderSongRank], error) {
	year := 0
	if len(p.Years) > 0 {
		year = p.Years[0]
	}
	if year == 0 {
		year = cmp.Or(year, time.Now().Year())
	}

	results, err := erc.FromIteratorUntil(conn.NewLeadersByYear(ctx, year, cmp.Or(p.Limit, 40)))
	if err != nil {
		return nil, err
	}

	return &ContextualSequence[int, models.LeaderSongRank]{
		Context: year,
		Results: results,
	}, nil
}

func SongsByKey(ctx context.Context, conn *db.Connection, p models.Params) (*ContextualSequence[string, models.LeaderSongRank], error) {
	results, err := erc.FromIteratorUntil(conn.SongsByKey(ctx, p.Years...))
	if err != nil {
		return nil, err
	}
	label := "all time"
	if len(p.Years) > 0 {
		var sb strings.Builder
		for i, y := range p.Years {
			if i > 0 {
				sb.WriteString(", ")
			}
			fmt.Fprintf(&sb, "%d", y)
		}
		label = sb.String()
	}

	return &ContextualSequence[string, models.LeaderSongRank]{
		Context: label,
		Results: results,
	}, nil
}

// LeadersByKey returns the top N leaders with regards to the number of times they are a top20
// leader for a song.
func LeadersByTop20Leads(ctx context.Context, conn *db.Connection, p models.Params) (*ContextualSequence[string, models.LeaderSongRank], error) {
	results, err := erc.FromIteratorUntil(conn.LeadersByTop20Leads(ctx, cmp.Or(p.Limit, 40)))
	if err != nil {
		return nil, err
	}

	return &ContextualSequence[string, models.LeaderSongRank]{
		Results: results,
	}, nil
}

// LeadersByKey returns every key, and the top leader of songs in that key.
func LeadersByKey(ctx context.Context, conn *db.Connection, p models.Params) (*ContextualSequence[string, models.LeaderSongRank], error) {
	results, err := erc.FromIteratorUntil(conn.LeadersByKey(ctx, p.Name, cmp.Or(p.Limit, 40)))
	if err != nil {
		return nil, err
	}

	return &ContextualSequence[string, models.LeaderSongRank]{
		Context: p.Name,
		Results: results,
	}, nil
}

func PopularSongsByKey(ctx context.Context, conn *db.Connection, p models.Params) (*ContextualSequence[string, models.LeaderSongRank], error) {
	results, err := erc.FromIteratorUntil(conn.PopularSongsByKey(ctx, p.Name, cmp.Or(p.Limit, 40)))
	if err != nil {
		return nil, err
	}

	return &ContextualSequence[string, models.LeaderSongRank]{
		Context: p.Name,
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
	leader, err := selector.Leader(ctx, conn, searchParams(p))
	if err != nil {
		return nil, err
	}

	share, err := conn.LeaderShareOfLeads(ctx, leader.Name, 32, p.Years...)
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
