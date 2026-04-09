package reportui

import (
	"context"
	"fmt"
	"time"

	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/stw"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/mdwn"
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

	mb.KV("Generated", time.Now().Format(time.DateOnly))
	mb.KV("Share of All Leads", fmt.Sprintf("%.4f%%", stw.DerefZ(share)*100))
	mb.KV("Connectedness", fmt.Sprintf("%.2f%%", v*100))
	mb.Line()

	mb.H2("Most Led Songs")
	models.WriteSongTable(&mb, erc.HandleAll(conn.MostLedSongs(ctx, singer.Name, 24), ec.Push))
	mb.H2("Favorite Keys")
	mb.KVTable(
		irt.MakeKV("Count", "Key"),
		irt.Convert2(irt.KVsplit(erc.HandleAll(conn.LeaderFavoriteKey(ctx, singer.Name, 100), ec.Push)), intValToStr),
	)
	mb.Line()

	mb.H2("Songs in Your Experience")
	mb.Paragraph("Most frequently led songs at singings ", singer.Name, " attended.")
	models.WriteSongTable(&mb, erc.HandleAll(conn.PopularSongsInOnesExperience(ctx, singer.Name, 12), ec.Push))

	mb.H2("Singing Buddies")
	mb.Paragraph("The people that have been the most singings that ", singer.Name, " was at.")
	mb.KVTable(irt.MakeKV("Name", "Shared Singings"),
		irt.Convert2(irt.KVsplit(erc.HandleAll(conn.SingingBuddies(ctx, singer.Name, 24), ec.Push)), intValToStr),
	)
	mb.Line()

	mb.H2("Singing Strangers")
	mb.Paragraph("People that ", singer.Name, " has never sung with who share many connections.")
	mb.KVTable(
		irt.MakeKV("Name", "Mutual Connections"),
		irt.Convert2(irt.KVsplit(erc.HandleAll(conn.SingingStrangers(ctx, singer.Name, 24), ec.Push)), intValToStr),
	)
	mb.Line()

	mb.H2("Singing Idols")
	mb.Paragraph("The top leaders of all of ", singer.Name, "'s top songs!")
	models.WriteLeaderFootstepTable(&mb, erc.HandleAll(conn.LeaderFootsteps(ctx, singer.Name, 20), ec.Push))

	mb.H2("Unfamiliar Hits")
	mb.Paragraph("Othewise popular songs that are under represented at singings ", singer.Name, " has been at.")
	models.WriteSongTable(&mb, erc.HandleAll(conn.TheUnfamilarHits(ctx, singer.Name, 20), ec.Push))

	mb.H2("Never Led")
	mb.Paragraph("Songs from the 2025 book that ", singer.Name, " has never led, by global popularity.")
	models.WriteSongTable(&mb, erc.HandleAll(irt.Limit2(conn.NeverLed(ctx, singer.Name, 20), 12), ec.Push))

	mb.H2("Never Sung")
	mb.Paragraph("Songs that have not been called at a singing ", singer.Name, " attended, by global popularity.")
	models.WriteSongTable(&mb, erc.HandleAll(irt.Limit2(conn.NeverSung(ctx, singer.Name), 12), ec.Push))

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}
