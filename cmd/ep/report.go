package ep

import (
	"context"
	"fmt"
	"io"
	"iter"
	"os"
	"strconv"
	"time"

	"github.com/tychoish/cmdr"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/strut"
	"github.com/tychoish/fun/stw"
	"github.com/tychoish/grip"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/infra"
	"github.com/tychoish/odem/pkg/mdwn"
	"github.com/tychoish/odem/pkg/models"
)

func Report() *cmdr.Commander {
	return cmdr.MakeCommander().
		SetName("report").
		Aliases("rpt").
		SetUsage("generate a markdown report for a singer").
		With(infra.DBOperationSpec(reportAction).Add)
}

func reportAction(ctx context.Context, conn *db.Connection, singer string) (err error) {
	if singer == "" {
		var ec erc.Collector
		names := irt.Collect(erc.HandleAll(conn.AllLeaderNames(ctx), ec.Push))
		if err := ec.Resolve(); err != nil {
			return err
		}
		singer, err = infra.NewFuzzySearch[string](names).FindOne("singer")
		if err != nil {
			return err
		}
	}

	mut := strut.MakeMutable(len(singer))
	defer mut.Release()
	mut.PushString(singer)
	mut.ReplaceAllString(" ", "-")
	mut.ReplaceAllString("’", "-")
	mut.ToLower()
	mut.PushString(".md")

	f, err := os.Create(mut.String())
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(f.Close()) }()

	grip.Infof("writing report for %q to %s", singer, mut)
	if err := writeReport(ctx, conn, f, singer); err != nil {
		return err
	}
	grip.Infof("report written to %s", mut)
	return nil
}

func writeReport(ctx context.Context, conn *db.Connection, w io.Writer, singer string) error {
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H1(singer)
	mb.KV("Generated", time.Now().Format(time.DateOnly))
	mb.Line()

	v, err := conn.SingersConnectedness(ctx, singer)
	ec.Push(err)

	mb.KV("Connectedness", fmt.Sprintf("%.2f%%", stw.DerefZ(v)*100))
	mb.Line()

	mb.H2("Most Led Songs")
	writeSongTable(&mb, erc.HandleAll(conn.MostLeadSongs(ctx, singer, 24), ec.Push))

	mb.H2("Songs in Your Experience")
	mb.Paragraph("Most frequently led songs at singings ", singer, " attended.")
	writeSongTable(&mb, erc.HandleAll(conn.PopularSongsInOnesExperience(ctx, singer, 12), ec.Push))

	mb.H2("Singing Buddies")
	mb.Paragraph("The people that have been the most singings that ", singer, " was at.")
	mb.KVTable(irt.MakeKV("Name", "Shared Singings"),
		irt.Convert2(irt.KVsplit(erc.HandleAll(conn.SingingBuddies(ctx, singer, 24), ec.Push)), intValToStr),
	)
	mb.Line()

	mb.H2("Singing Strangers")
	mb.Paragraph("People that ", singer, " has never sung with who share many connections.")
	mb.KVTable(irt.MakeKV("Name", "Mutual Connections"),
		irt.Convert2(irt.KVsplit(erc.HandleAll(conn.SingingStrangers(ctx, singer, 24), ec.Push)), intValToStr),
	)
	mb.Line()

	mb.H2("Singing Idols")
	mb.Paragraph("The top leaders of all of ", singer, "'s top songs!")
	writeLeaderFootstepTable(&mb, erc.HandleAll(conn.LeaderFootsteps(ctx, singer, 20), ec.Push))

	mb.H2("Unfamiliar Hits")
	mb.Paragraph("Othewise popular songs that are under represented at singings ", singer, " has been at.")
	writeSongTable(&mb, erc.HandleAll(conn.TheUnfamilarHits(ctx, singer, 20), ec.Push))

	mb.H2("Never Led")
	mb.Paragraph("Songs from the 2025 book that ", singer, " has never led, by global popularity.")
	writeSongTable(&mb, erc.HandleAll(irt.Limit2(conn.NeverLed(ctx, singer), 12), ec.Push))

	mb.H2("Never Sung")
	mb.Paragraph("Songs that have not been called at a singing ", singer, " attended, by global popularity.")
	writeSongTable(&mb, erc.HandleAll(irt.Limit2(conn.NeverSung(ctx, singer), 12), ec.Push))

	_, err = mb.WriteTo(w)
	ec.Push(err)
	return ec.Resolve()
}

func intValToStr(key string, value int) (string, string) { return key, strconv.Itoa(value) }
func asRows(lsr models.LeaderSongRank) []string          { return (&lsr).StringFields() }
func writeSongTable(mb *mdwn.Builder, seq iter.Seq[models.LeaderSongRank]) {
	mb.NewTable(
		mdwn.Column{Name: "Count", RightAlign: true},
		mdwn.Column{Name: "Page"},
		mdwn.Column{Name: "Title"},
		mdwn.Column{Name: "Key"},
	).Extend(irt.Convert(seq, asRows)).Build()

	mb.Line()
}

func writeLeaderFootstepTable(mb *mdwn.Builder, seq iter.Seq[models.LeaderFootstep]) {
	mb.NewTable(
		mdwn.Column{Name: "Song"},
		mdwn.Column{Name: "Page"},
		mdwn.Column{Name: "Key"},
		mdwn.Column{Name: "Top Leader"},
		mdwn.Column{Name: "Their Leads", RightAlign: true},
		mdwn.Column{Name: "Self Leads", RightAlign: true},
	).Extend(irt.Convert(seq, func(row models.LeaderFootstep) []string {
		return []string{row.SongTitle, row.SongPage, row.SongKeys, row.LeaderName, strconv.Itoa(row.TheirLeadCount), strconv.Itoa(row.SelfLeadCount)}
	})).Build()

	mb.Line()
}
