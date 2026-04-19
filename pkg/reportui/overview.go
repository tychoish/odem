package reportui

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/mdwn"
	"github.com/tychoish/fun/stw"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/models"
	"github.com/tychoish/odem/pkg/release"
	"github.com/tychoish/odem/pkg/selector"
)

func Leader(ctx context.Context, conn *db.Connection, in Params) (err error) {
	singer, err := selector.Leader(ctx, conn, in.Search())
	if err != nil {
		return err
	}

	w, err := in.getWriter(singer.Name)
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(w.Close()) }()

	// ---------------- THE FOLD ----------------
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H1(singer.Name)

	if share, err := conn.LeaderShareOfLeads(ctx, singer.Name, 16); !ec.PushOk(err) {
		mb.KV("Share of all Leads", fmt.Sprintf("%.4f%%", stw.DerefZ(share)*100))
	}
	if v, err := conn.GetSingerConnectedness(ctx, &singer.Name); !ec.PushOk(err) {
		mb.KV("Connectedness", fmt.Sprintf("%.2f%%", v*100))
	}
	if leaderInfo, err := conn.GetLeader(ctx, &singer.Name); !ec.PushOk(err) {
		mb.KV("Number of Top 20 Leads", strconv.Itoa(int(leaderInfo.Top20Count)))
		mb.KV("Lesson Count", strconv.Itoa(int(leaderInfo.LessonCount)))
	}
	if majorKey, err := conn.GetLeaderTopMajorKey(ctx, singer.Name); !ec.PushOk(err) {
		mb.KV("Top Major Key", fmt.Sprintf("%s (%d)", majorKey.TopKey, majorKey.LeadCount))
	}
	if minorKey, err := conn.GetLeaderTopMinorKey(ctx, singer.Name); !ec.PushOk(err) {
		mb.KV("Top Minor Key", fmt.Sprintf("%s (%d)", minorKey.TopKey, minorKey.LeadCount))
	}
	if keyCounts, err := conn.GetLeaderMajorMinorCounts(ctx, singer.Name); !ec.PushOk(err) && keyCounts.MinorCount > 0 {
		mb.KV("Major/Minor Ratio", fmt.Sprintf("%.2f:1", float64(keyCounts.MajorCount)/float64(keyCounts.MinorCount)))
	}
	if topBuddy, err := conn.GetLeaderTopSingingBuddy(ctx, singer.Name); !ec.PushOk(err) {
		mb.KV("Top Singing Buddy", fmt.Sprintf("%s (%d singings)", topBuddy.BuddyName, topBuddy.SingingCount))
	}
	if activeYears, err := conn.GetLeaderActiveYears(ctx, singer.Name); !ec.PushOk(err) {
		mb.KV("Years Singing", fmt.Sprintf("%d (%d–%d)", activeYears.YearsActive, activeYears.FirstYear, activeYears.LastYear))
		mb.KV("Active (last 5 years)", strconv.FormatBool(activeYears.IsActive != 0))
	}
	if topState, err := conn.GetLeaderTopState(ctx, singer.Name); !ec.PushOk(err) {
		mb.KV("State with Most Leads", fmt.Sprintf("%s (%d)", topState.State, topState.LeadCount))
	}

	mb.Line()

	mb.H2("Most Led Songs")
	models.WriteTable(&mb, erc.HandleAll(conn.MostLedSongs(ctx, singer.Name, 24), ec.Push))
	mb.H2("Favorite Keys")
	models.WriteTable(&mb, erc.HandleAll(conn.LeaderFavoriteKey(ctx, singer.Name, 100), ec.Push))
	mb.Line()

	mb.H2("Popular Songs, as Observed")
	mb.Paragraph("Most frequently led songs at singings ", singer.Name, " attended.")
	models.WriteTable(&mb, erc.HandleAll(conn.PopularAsObserved(ctx, singer.Name, 12), ec.Push))

	mb.H2("Singing Buddies")
	mb.Paragraph("The people that have been the most singings that ", singer.Name, " was at.")
	models.WriteTable(&mb, erc.HandleAll(conn.SingingBuddies(ctx, singer.Name, 24), ec.Push))
	mb.Line()

	mb.H2("Singing Strangers")
	mb.Paragraph("People that ", singer.Name, " has never sung with who share many connections.")
	models.WriteTable(&mb, erc.HandleAll(conn.SingingStrangers(ctx, singer.Name, 24), ec.Push))
	mb.Line()

	mb.H2("Singing Role Models")
	mb.Paragraph("The top leaders of all of ", singer.Name, "'s top songs!")
	models.WriteTable(&mb, erc.HandleAll(conn.LeaderFootsteps(ctx, singer.Name, 20), ec.Push))

	mb.H2("Unfamiliar Hits")
	mb.Paragraph("Othewise popular songs that are under represented at singings ", singer.Name, " has been at.")
	models.WriteTable(&mb, erc.HandleAll(conn.TheUnfamilarHits(ctx, singer.Name, 20), ec.Push))

	mb.H2("Never Led")
	mb.Paragraph("Songs from the 2025 book that ", singer.Name, " has never led, by global popularity.")
	models.WriteTable(&mb, erc.HandleAll(irt.Limit2(conn.NeverLed(ctx, singer.Name, 20), 12), ec.Push))

	mb.H2("Never Sung")
	mb.Paragraph("Songs that have not been called at a singing ", singer.Name, " attended, by global popularity.")
	models.WriteTable(&mb, erc.HandleAll(irt.Limit2(conn.NeverSung(ctx, singer.Name), 12), ec.Push))

	mb.Line()
	mb.KV("Generated", time.Now().Format(time.DateOnly))
	mb.KV("Version", release.Version.Resolve().String())

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}
