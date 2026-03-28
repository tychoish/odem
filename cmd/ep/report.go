package ep

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"iter"
	"os"
	"strconv"
	"strings"
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
	"github.com/urfave/cli/v3"
)

type reportInput struct {
	Arg      string
	ToStdout bool
}

func reportOperationSpec() *cmdr.OperationSpec[*infra.WithInput[reportInput]] {
	return infra.DBOperationSpecWith(
		func(cc *cli.Command) reportInput {
			return reportInput{
				Arg:      cmdr.GetFlagOrFirstArg[string](cc, "name"),
				ToStdout: cmdr.GetFlag[bool](cc, "stdout"),
			}
		},
		func(ctx context.Context, conn *db.Connection, in reportInput) error {
			return reportAction(ctx, conn, in)
		},
	)
}

func reportSubcmd(op fzfui.MinutesAppOperation) *cmdr.Commander {
	info := op.GetInfo()
	return cmdr.MakeCommander().
		SetName(info.Key).
		SetUsage(info.Value).
		With(infra.DBOperationSpec(func(ctx context.Context, conn *db.Connection, arg string) error {
			return dispatcher(op).Report(ctx, conn, os.Stdout, reportParams{Arg: arg})
		}).Add)
}

func Report() *cmdr.Commander {
	return cmdr.MakeCommander().
		SetName("report").
		Aliases("rpt").
		SetUsage("generate a markdown report for a singer").
		Flags(cmdr.FlagBuilder(false).
			SetName("stdout", "o").
			SetUsage("write report to stdout instead of a file").
			Flag()).
		With(reportOperationSpec().Add).
		Subcommanders(irt.Collect(irt.Convert(fzfui.AllMinutesAppOperations(), reportSubcmd))...)
}

func reportAction(ctx context.Context, conn *db.Connection, in reportInput) (err error) {
	singer := in.Arg
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

	w, cleanup, err := getWriter(in.ToStdout, singer)
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(cleanup()) }()
	if !in.ToStdout {
		grip.Infof("writing report for %q to file", singer)
	}

	if err := fullReport(ctx, conn, w, singer); err != nil {
		return err
	}

	return nil
}

// getWriter returns an io.Writer (stdout or a new file) plus a cleanup func.
// The caller must call cleanup() when done. For stdout, cleanup is a no-op.
func getWriter(stdout bool, singer string, tags ...string) (io.Writer, func() error, error) {
	if stdout {
		return os.Stdout, func() error { return nil }, nil
	}
	f, err := getFile(singer, tags...)
	if err != nil {
		return nil, nil, err
	}
	return f, f.Close, nil
}

func getFile(singer string, tags ...string) (*os.File, error) {
	mut := strut.MakeMutable(len(singer))
	defer mut.Release()
	mut.PushString(singer)
	mut.ReplaceAllString(" ", "-")
	mut.ReplaceAllString("'", "-")
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

func fullReport(ctx context.Context, conn *db.Connection, w io.Writer, params reportParams) error {
	var ec erc.Collector
	var mb mdwn.Builder
	var singer string

	mb.H1(params.Arg)
	mb.KV("Generated", time.Now().Format(time.DateOnly))
	mb.Line()

	share, err := conn.LeaderShareOfLeads(ctx, params.Arg)
	ec.Push(err)
	v, err := conn.Params.ArgsConnectedness(ctx, params.Arg)
	ec.Push(err)

	mb.KV("Share of All Leads", fmt.Sprintf("%.4f%%", stw.DerefZ(share)*100))
	mb.KV("Connectedness", fmt.Sprintf("%.2f%%", stw.DerefZ(v)*100))
	mb.Line()

	mb.H2("Most Led Songs")
	writeSongTable(&mb, erc.HandleAll(conn.MostLeadSongs(ctx, params.Arg, 24), ec.Push))

	mb.H2("Songs in Your Experience")
	mb.Paragraph("Most frequently led songs at singings ", params.Arg, " attended.")
	writeSongTable(&mb, erc.HandleAll(conn.PopularSongsInOnesExperience(ctx, params.Arg, 12), ec.Push))

	mb.H2("Singing Buddies")
	mb.Paragraph("The people that have been the most singings that ", params.Arg, " was at.")
	mb.KVTable(irt.MakeKV("Name", "Shared Singings"),
		irt.Convert2(irt.KVsplit(erc.HandleAll(conn.SingingBuddies(ctx, params.Arg, 24), ec.Push)), intValToStr),
	)
	mb.Line()

	mb.H2("Singing Strangers")
	mb.Paragraph("People that ", params.Arg, " has never sung with who share many connections.")
	mb.KVTable(irt.MakeKV("Name", "Mutual Connections"),
		irt.Convert2(irt.KVsplit(erc.HandleAll(conn.SingingStrangers(ctx, params.Arg, 24), ec.Push)), intValToStr),
	)
	mb.Line()

	mb.H2("Singing Idols")
	mb.Paragraph("The top leaders of all of ", params.Arg, "'s top songs!")
	writeLeaderFootstepTable(&mb, erc.HandleAll(conn.LeaderFootsteps(ctx, params.Arg, 20), ec.Push))

	mb.H2("Unfamiliar Hits")
	mb.Paragraph("Othewise popular songs that are under represented at singings ", params.Arg, " has been at.")
	writeSongTable(&mb, erc.HandleAll(conn.TheUnfamilarHits(ctx, params.Arg, 20), ec.Push))

	mb.H2("Never Led")
	mb.Paragraph("Songs from the 2025 book that ", params.Arg, " has never led, by global popularity.")
	writeSongTable(&mb, erc.HandleAll(irt.Limit2(conn.NeverLed(ctx, params.Arg), 12), ec.Push))

	mb.H2("Never Sung")
	mb.Paragraph("Songs that have not been called at a singing ", params.Arg, " attended, by global popularity.")
	writeSongTable(&mb, erc.HandleAll(irt.Limit2(conn.NeverSung(ctx, params.Arg), 12), ec.Push))

	_, err = mb.WriteTo(w)
	ec.Push(err)
	return ec.Resolve()
}

func flush(wr io.Writer, payload io.WriterTo) (err error) { _, err = payload.WriteTo(wr); return }
func intValToStr(key string, value int) (string, string)  { return key, strconv.Itoa(value) }
func asRows(lsr models.LeaderSongRank) []string           { return (&lsr).StringFields() }
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
		mdwn.Column{Name: "Last Year", RightAlign: true},
		mdwn.Column{Name: "Self Leads", RightAlign: true},
	).Extend(irt.Convert(seq, func(row models.LeaderFootstep) []string {
		return []string{
			row.SongTitle,
			row.SongPage,
			row.SongKeys,
			row.LeaderName,
			strconv.Itoa(row.TheirLeadCount),
			strconv.Itoa(row.TheirLastLeadYear),
			strconv.Itoa(row.SelfLeadCount),
		}
	})).Build()

	mb.Line()
}

type reportParams struct {
	Arg   string
	Limit int // 0 means use the function's built-in default
}

type reporterFunc func(context.Context, *db.Connection, io.Writer, reportParams) error

func (r reporterFunc) Report(ctx context.Context, conn *db.Connection, w io.Writer, p reportParams) error {
	return r(ctx, conn, w, p)
}

const defaultN = 25

func songsReport(ctx context.Context, conn *db.Connection, w io.Writer, p reportParams) error {
	var ec erc.Collector
	var mb mdwn.Builder

	sg, err := conn.GetSong(ctx, p.Arg)
	ec.Push(err)

	mb.H2(fmt.Sprintf("Song: %s — %s", sg.PageNum, sg.SongTitle))
	mb.KV("Page", sg.PageNum)
	mb.KV("Keys", sg.Keys)
	mb.KV("Meter", sg.SongMeter)
	mb.KV("Music", sg.MusicAttribution)
	mb.KV("Words", sg.WordsAttribution)
	mb.Line()

	mb.H3("Top Leaders")
	mb.NewTable(
		mdwn.Column{Name: "Name"},
		mdwn.Column{Name: "Count", RightAlign: true},
		mdwn.Column{Name: "Led Last Year"},
		mdwn.Column{Name: "Years Active", RightAlign: true},
	).Extend(irt.Convert(erc.HandleAll(conn.TopLeadersOfSong(ctx, sg.PageNum, cmp.Or(p.Limit, 20)), ec.Push), func(l models.LeaderOfSongInfo) []string {
		return []string{l.Name, strconv.Itoa(l.Count), strconv.FormatBool(l.LedInLastYear), strconv.Itoa(l.NumYears)}
	})).Build()
	mb.Line()

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func singingsReport(ctx context.Context, conn *db.Connection, w io.Writer, p reportParams) error {
	if p.Arg == "" {
		return ers.New("singing name required")
	}
	var ec erc.Collector
	var mb mdwn.Builder

	var found *models.SingingInfo
	for s, err := range conn.AllSingings(ctx) {
		if !ec.PushOk(err) {
			break
		}
		if s.SingingName == p.Arg {
			s := s
			found = &s
			break
		}
	}
	if !ec.Ok() {
		return ec.Resolve()
	}

	if found != nil {
		mb.H2(fmt.Sprintf("Singing: %s", found.SingingName))
		mb.KV("Date", found.SingingDate.Time().Format(time.DateOnly))
		mb.KV("Location", found.SingingLocation)
		mb.KV("State", found.SingingState)
		mb.KV("Lessons", strconv.FormatInt(found.NumberOfLessons, 10))
		mb.KV("Leaders", strconv.FormatInt(found.NumberOfLeaders, 10))
		mb.Line()
	} else {
		mb.H2(fmt.Sprintf("Singing: %s", p.Arg))
	}

	mb.H3("Lessons")
	mb.NewTable(
		mdwn.Column{Name: "Lesson", RightAlign: true},
		mdwn.Column{Name: "Leader"},
		mdwn.Column{Name: "Song"},
		mdwn.Column{Name: "Key"},
		mdwn.Column{Name: "Title"},
	).Extend(irt.Convert(erc.HandleAll(conn.SingingLessons(ctx, p.Arg), ec.Push), func(s models.SingingLessionInfo) []string {
		return []string{strconv.Itoa(s.LessonID), s.SingerName, s.SongPageNumber, s.SongKey, s.SongName}
	})).Build()
	mb.Line()

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func buddiesReport(ctx context.Context, conn *db.Connection, w io.Writer, p reportParams) error {
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H2(fmt.Sprintf("Singing Buddies: %s", p.Arg))
	mb.KVTable(
		irt.MakeKV("Name", "Shared Singings"),
		irt.Convert2(irt.KVsplit(erc.HandleAll(conn.SingingBuddies(ctx, p.Arg, cmp.Or(p.Limit, 24)), ec.Push)), intValToStr),
	)
	mb.Line()

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func strangersReport(ctx context.Context, conn *db.Connection, w io.Writer, p reportParams) error {
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H2(fmt.Sprintf("Singing Strangers: %s", p.Arg))
	mb.KVTable(
		irt.MakeKV("Name", "Mutual Connections"),
		irt.Convert2(irt.KVsplit(erc.HandleAll(conn.SingingStrangers(ctx, p.Arg, cmp.Or(p.Limit, 24)), ec.Push)), intValToStr),
	)
	mb.Line()

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func popularInExperienceReport(ctx context.Context, conn *db.Connection, w io.Writer, p reportParams) error {
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H2(fmt.Sprintf("Popular in %s's Experience", p.Arg))
	writeSongTable(&mb, erc.HandleAll(conn.PopularSongsInOnesExperience(ctx, p.Arg, cmp.Or(p.Limit, defaultN)), ec.Push))

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func popularInYearsReport(ctx context.Context, conn *db.Connection, w io.Writer, p reportParams) error {
	var ec erc.Collector
	var mb mdwn.Builder

	var years []int
	for part := range strings.SplitSeq(p.Arg, ",") {
		y, err := strconv.Atoi(strings.TrimSpace(part))
		if err == nil && y != 0 {
			years = append(years, y)
		}
	}

	if len(years) > 0 {
		mb.H2(fmt.Sprintf("Globally Popular (years: %v)", years))
	} else {
		mb.H2("Globally Popular")
	}
	writeSongTable(&mb, erc.HandleAll(conn.GloballyPopularForYears(ctx, years...), ec.Push))

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func locallyPopularReport(ctx context.Context, conn *db.Connection, w io.Writer, p reportParams) error {
	var ec erc.Collector
	var mb mdwn.Builder

	var localities []models.SingingLocality
	for part := range strings.SplitSeq(p.Arg, ",") {
		localities = append(localities, models.NewSingingLocality(strings.TrimSpace(part)))
	}

	mb.H2(fmt.Sprintf("Locally Popular: %s", p.Arg))
	writeSongTable(&mb, erc.HandleAll(conn.LocallyPopular(ctx, cmp.Or(p.Limit, 32), localities...), ec.Push))

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func neverSungReport(ctx context.Context, conn *db.Connection, w io.Writer, p reportParams) error {
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H2(fmt.Sprintf("Never Sung: %s", p.Arg))
	writeSongTable(&mb, erc.HandleAll(irt.Limit2(conn.NeverSung(ctx, p.Arg), cmp.Or(p.Limit, 20)), ec.Push))

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func neverLedReport(ctx context.Context, conn *db.Connection, w io.Writer, p reportParams) error {
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H2(fmt.Sprintf("Never Led: %s", p.Arg))
	writeSongTable(&mb, erc.HandleAll(irt.Limit2(conn.NeverLed(ctx, p.Arg), cmp.Or(p.Limit, 20)), ec.Push))

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func unfamilarHitsReport(ctx context.Context, conn *db.Connection, w io.Writer, p reportParams) error {
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H2(fmt.Sprintf("Unfamiliar Hits: %s", p.Arg))
	writeSongTable(&mb, erc.HandleAll(conn.TheUnfamilarHits(ctx, p.Arg, cmp.Or(p.Limit, 20)), ec.Push))

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func connectednessReport(ctx context.Context, conn *db.Connection, w io.Writer, p reportParams) error {
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H2("Leaders by Connectedness")
	mb.KVTable(
		irt.MakeKV("Name", "Connectedness"),
		irt.Convert2(irt.KVsplit(erc.HandleAll(conn.AllLeaderConnectedness(ctx, cmp.Or(p.Limit, 40)), ec.Push)), func(k string, v float64) (string, string) {
			return k, fmt.Sprintf("%.4f%%", v*100)
		}),
	)
	mb.Line()

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func topLeadersReport(ctx context.Context, conn *db.Connection, w io.Writer, p reportParams) error {
	var ec erc.Collector
	var mb mdwn.Builder

	years := irt.Collect(irt.RemoveZeros(irt.Convert(strings.SplitSeq(p.Arg, ","), atoi)))

	if len(years) > 0 {
		mb.H2(fmt.Sprintf("Top Leaders (years: %v)", years))
	} else {
		mb.H2("Top Leaders")
	}

	pos := 0
	mb.NewTable(
		mdwn.Column{Name: "#", RightAlign: true},
		mdwn.Column{Name: "Name"},
		mdwn.Column{Name: "Leads", RightAlign: true},
		mdwn.Column{Name: "Last Year", RightAlign: true},
		mdwn.Column{Name: "%", RightAlign: true},
		mdwn.Column{Name: "Running Total %", RightAlign: true},
	).Extend(irt.Convert(erc.HandleAll(conn.TopLeadersByLeads(ctx, cmp.Or(p.Limit, 40), years...), ec.Push), func(row models.LeaderLeadCount) []string {
		pos++
		return []string{strconv.Itoa(pos), row.Name, strconv.Itoa(row.Count), strconv.Itoa(row.LastLeadYear), fmt.Sprintf("%.2f%%", row.Percentage*100), fmt.Sprintf("%.2f%%", row.RunningTotal*100)}
	})).Build()
	mb.Line()

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func leaderShareReport(ctx context.Context, conn *db.Connection, w io.Writer, p reportParams) error {
	var ec erc.Collector
	var mb mdwn.Builder

	// TODO avoid parsing the p.Arg and just add another explicit arg to the structure and the
	// parsing of the command
	parts := strings.SplitN(p.Arg, ",", 2)
	singer := strings.TrimSpace(parts[0])

	var years []int
	if len(parts) > 1 {
		years = irt.Collect(irt.RemoveZeros(irt.Convert(strings.SplitSeq(parts[1], ","), atoi)))
	}

	v, err := conn.LeaderShareOfLeads(ctx, singer, years...)
	ec.Push(err)

	label := "Share of All Leads"
	if len(years) > 0 {
		label = fmt.Sprintf("Share of Leads (%v)", years)
	}

	mb.H2(fmt.Sprintf("Leader Share: %s", singer))
	mb.KV("Leader", singer)
	mb.KV(label, fmt.Sprintf("%.4f%%", stw.DerefZ(v)*100))
	mb.Line()

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func atoi(in string) (n int) { n, _ = strconv.Atoi(in); return }
func leaderFootstepsReport(ctx context.Context, conn *db.Connection, w io.Writer, p reportParams) error {
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H2(fmt.Sprintf("Leader Footsteps: %s", p.Arg))
	writeLeaderFootstepTable(&mb, erc.HandleAll(conn.LeaderFootsteps(ctx, p.Arg, cmp.Or(p.Limit, 20)), ec.Push))

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func dispatcher(op fzfui.MinutesAppOperation) reporterFunc {
	switch op {
	case fzfui.MinutesAppOpLeaders:
		return fullReport
	case fzfui.MinutesAppOpSongs:
		return songsReport
	case fzfui.MinutesAppOpSingings:
		return singingsReport
	case fzfui.MinutesAppOpBuddies:
		return buddiesReport
	case fzfui.MinutesAppOpStrangers:
		return strangersReport
	case fzfui.MinutesAppOpPopularInOnesExperience:
		return popularInExperienceReport
	case fzfui.MinutesAppOpPopularInYears:
		return popularInYearsReport
	case fzfui.MinutesAppOpLocallyPopular:
		return locallyPopularReport
	case fzfui.MinutesAppOpNeverSung:
		return neverSungReport
	case fzfui.MinutesAppOpNeverLed:
		return neverLedReport
	case fzfui.MinutesAppOpUnfamilarHits:
		return unfamilarHitsReport
	case fzfui.MinutesAppOpConnectedness:
		return connectednessReport
	case fzfui.MinutesAppOpTopLeaders:
		return topLeadersReport
	case fzfui.MinutesAppOpLeaderShare:
		return leaderShareReport
	case fzfui.MinutesAppOpLeaderFootsteps:
		return leaderFootstepsReport
	default:
		return nil
	}
}
