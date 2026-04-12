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

	share, err := conn.LeaderShareOfLeads(ctx, singer.Name, 16)
	ec.Push(err)
	v, err := conn.GetSingerConnectedness(ctx, &singer.Name)
	ec.Push(err)
	leaderInfo, err := conn.GetLeader(ctx, &singer.Name)
	ec.Push(err)

	mb.KV("Generated", time.Now().Format(time.DateOnly))
	mb.KV("Share of all Leads", fmt.Sprintf("%.4f%%", stw.DerefZ(share)*100))
	mb.KV("Connectedness", fmt.Sprintf("%.2f%%", v*100))
	mb.KV("Number of Top 20 Leads", strconv.Itoa(int(leaderInfo.Top20Count)))
	mb.KV("Lesson Count", strconv.Itoa(int(leaderInfo.LessonCount)))
	// TODO add: Most frequently led Major Key: <key>, (count)
	// TODO add: Most frequently led Minor Key: <key>, (count)
	// TODO add: Top Singing Buddy: <name> (cont)
	// TODO add: Major/Minor Ratio: <ratio>
	// TODO add: Number of Years Singing: <n>
	// TODO add: Active (last 5 years): yes/no
	// TODO add: State with the Most Leads:

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

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}
