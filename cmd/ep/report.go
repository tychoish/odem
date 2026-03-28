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
	"github.com/tychoish/fun/ers"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/strut"
	"github.com/tychoish/fun/stw"
	"github.com/tychoish/grip"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/fzfui"
	"github.com/tychoish/odem/pkg/infra"
	"github.com/tychoish/odem/pkg/mdwn"
	"github.com/tychoish/odem/pkg/models"
)

/*
TODO implement larger plans and
- [ ] strict mode where missing or invalid input for fzfui and reports result in an error. should be envvar setable
- [ ] attempt to use the fzf search api wiht input from the user to avoid needing to ask someone if there is only one match. if there are multiple matches and not in strict mode, the user can start narrowed. this could be applied to input for some other queries, so providing an isolated implementation.
- [ ] should move some of the core dispatching code from fzfui to a new package (cmdln) [this would include the enum, and core methods, but nothing else Actions would still be in fzfui, and reports would be in a reportui package]
- [ ] complete a report UI for  (as a prototype in this directory).
- [x] add a query/fzfui/report for leader ordered by number of leads, potentially allow filtering by year (in the way of the song popularity)
- [x] add a query (etc.) for "number of leaders who led N% of songs," also filtered by year.
*/
func Report() *cmdr.Commander {
	return cmdr.MakeCommander().
		SetName("report").
		Aliases("rpt").
		SetUsage("generate a markdown report for a singer").
		// TODO add a flag for
		//   - write to standard output
		With(infra.DBOperationSpec(reportAction).Add).
		Subcommanders(
		// ... subcommand for commander for every indivual report
		)
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

	f, err := getFile(singer)
	if err != nil {
		return err
	}

	defer func() { err = erc.Join(f.Close()) }()

	grip.Infof("writing report for %q to %s", singer, f.Name())

	if err := fullReport(ctx, conn, f, singer); err != nil {
		return err
	}

	grip.Infof("report written to %s", f.Name())
	return nil
}

func getFile(singer string, tags ...string) (*os.File, error) {
	mut := strut.MakeMutable(len(singer))
	defer mut.Release()
	mut.PushString(singer)
	mut.ReplaceAllString(" ", "-")
	mut.ReplaceAllString("’", "-")
	mut.ReplaceAllString(".", "")
	mut.ToLower()
	mut.JoinStrings(tags, "-")
	mut.PushString(".md")

	f, err := os.Create(mut.String())
	if err != nil {
		return nil, err
	}

	return f, nil
}

func fullReport(ctx context.Context, conn *db.Connection, w io.Writer, singer string) error {
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H1(singer)
	mb.KV("Generated", time.Now().Format(time.DateOnly))
	mb.Line()

	share, err := conn.LeaderShareOfLeads(ctx, singer)
	ec.Push(err)
	v, err := conn.SingersConnectedness(ctx, singer)
	ec.Push(err)

	mb.KV("Share of All Leads", fmt.Sprintf("%.4f%%", stw.DerefZ(share)*100))
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

type reporterFunc func(context.Context, *db.Connection, io.Writer, string) error

func dispatcher(op fzfui.MinutesAppOperation) reporterFunc {
	// TODO generate a function for each of the reports mirroring that render a report using the
	// same query as the fzfui operations that are dispatched for these operation labels. If the inital argument (e.g. for a )
	switch op {
	case fzfui.MinutesAppOpLeaders:
		return fullReport
	case fzfui.MinutesAppOpSongs:
	case fzfui.MinutesAppOpSingings:
	case fzfui.MinutesAppOpBuddies:
	case fzfui.MinutesAppOpStrangers:
	case fzfui.MinutesAppOpPopularInOnesExperience:
	case fzfui.MinutesAppOpPopularInYears:
	case fzfui.MinutesAppOpLocallyPopular:
	case fzfui.MinutesAppOpRetry:
	case fzfui.MinutesAppOpNeverSung:
	case fzfui.MinutesAppOpNeverLed:
	case fzfui.MinutesAppOpUnfamilarHits:
	case fzfui.MinutesAppOpConnectedness:
	case fzfui.MinutesAppOpTopLeaders:
	case fzfui.MinutesAppOpLeaderShare:
	case fzfui.MinutesAppOpLeaderFootsteps:
	case fzfui.MinutesAppOpExit:
		grip.Info("goodbye!")
		return nil
	case fzfui.MinutesAppOpInvalid:
		erc.Invariant(ers.New("explicitly invalid"))
		return nil
	case fzfui.MinutesAppOpUnknown:
		erc.Invariant(ers.New("unknown"))
		return nil
	default:
		erc.Invariant(ers.New("implicitly invalid"))
		return nil
	}
	panic(erc.NewInvariantError("unreachable"))
}
