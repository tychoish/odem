package ep

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"iter"
	"os"
	"slices"
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

type reporterFunc func(context.Context, *db.Connection, reportParams) error

func (r reporterFunc) Report(ctx context.Context, conn *db.Connection, params reportParams) error {
	return r(ctx, conn, params)
}

type Reporter interface {
	Report(ctx context.Context, conn *db.Connection, params reportParams) error
}

const defaultN = 25

type reportParams struct {
	Name       string
	Years      []int
	PathPrefix string
	Limit      int
	ToStdout   bool
}

func reportOperationSpec(rptr reporterFunc) *cmdr.OperationSpec[*infra.WithInput[reportParams]] {
	return infra.DBOperationSpecWith(
		func(cc *cli.Command) reportParams {
			return reportParams{
				Name:       cmdr.GetFlagOrFirstArg[string](cc, "name"),
				ToStdout:   cmdr.GetFlag[bool](cc, "stdout"),
				Limit:      cmdr.GetFlag[int](cc, "limit"),
				Years:      cmdr.GetFlag[[]int](cc, "year"),
				PathPrefix: cmdr.GetFlag[string](cc, "prefix"),
			}
		},
		rptr.Report,
	)
}

func toCommand(mao fzfui.MinutesAppOperation) *cmdr.Commander {
	i := mao.GetInfo()
	return cmdr.MakeCommander().SetName(i.Key).SetUsage(i.Value).With(reportOperationSpec(dispatcher(mao)).Add)
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
		With(reportOperationSpec(reportAction).Add).
		Subcommanders(irt.Collect(irt.Convert(fzfui.AllMinutesAppOperations(), toCommand))...)
}

type wstdout struct {
	*os.File
}

func (wstdout) Close() error { return nil }

type loggingCloser struct {
	reportName string
	f          *os.File
}

func (f *loggingCloser) Write(in []byte) (int, error) { return f.f.Write(in) }
func (f *loggingCloser) Close() error {
	grip.Infof("wrote report %s to %s", f.reportName, f.f.Name())
	return f.f.Close()
}

// getWriter returns an io.Writer (stdout or a new file) plus a cleanup func.
// The caller must call cleanup() when done. For stdout, cleanup is a no-op.
func (params reportParams) getWriter(tags ...string) (io.WriteCloser, error) {
	if params.ToStdout {
		return wstdout{File: os.Stdout}, nil
	}
	if len(tags) == 0 {
		return nil, ers.New("must specify a file name for the report")
	}
	f, err := getFile(tags...)
	if err != nil {
		return nil, err
	}

	return &loggingCloser{reportName: tags[0], f: f}, nil
}

func sumLens(in []string) (total int) {
	for _, s := range in {
		total += len(s)
	}
	return
}

func getFile(args ...string) (*os.File, error) {
	mut := strut.MakeMutable(sumLens(args) + 3)
	defer mut.Release()
	mut.JoinStrings(args, "-")
	mut.ReplaceAllString(" ", "-")
	mut.ReplaceAllString("'", "-")
	mut.ReplaceAllString(".", "")
	mut.ToLower()
	mut.PushString(".md")

	f, err := os.Create(mut.String())
	if err != nil {
		return nil, err
	}

	return f, nil
}

func reportAction(ctx context.Context, conn *db.Connection, in reportParams) (err error) {
	singer, err := fzfui.SelectLeader(ctx, conn, in.Name)
	if err != nil {
		return err
	}

	w, err := in.getWriter(singer)
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(w.Close()) }()

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
	writeSongTable(&mb, erc.HandleAll(irt.Limit2(conn.NeverSung(ctx, singer), 12), ec.
		Push))

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func songsReport(ctx context.Context, conn *db.Connection, p reportParams) (err error) {
	var ec erc.Collector
	var mb mdwn.Builder

	sg, err := fzfui.SelectSong(ctx, conn, p.Name)
	ec.Push(err)

	wr, err := p.getWriter(stw.DerefZ(sg).PageNum)
	if !ec.PushOk(err) {
		return ec.Resolve()
	}
	defer func() { err = erc.Join(err, wr.Close()) }()

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

	ec.Push(flush(wr, &mb))
	return ec.Resolve()
}

func singingsReport(ctx context.Context, conn *db.Connection, p reportParams) (err error) {
	info, err := fzfui.SelectSinging(ctx, conn, p.Name)
	if err != nil {
		return err
	}
	var ec erc.Collector
	var mb mdwn.Builder

	wr, err := p.getWriter("siging", stw.DerefZ(info).SingingName)
	if !ec.PushOk(err) {
		return ec.Resolve()
	}
	defer func() { err = erc.Join(err, wr.Close()) }()

	mb.H2(fmt.Sprintf("Singing: %s", info.SingingName))
	mb.KV("Date", info.SingingDate.Time().Format(time.DateOnly))
	mb.KV("", info.SingingLocation)
	mb.KV("State", info.SingingState)
	mb.KV("Lessons", strconv.FormatInt(info.NumberOfLessons, 10))
	mb.KV("Leaders", strconv.FormatInt(info.NumberOfLeaders, 10))
	mb.Line()

	mb.H3("Lessons")
	mb.NewTable(
		mdwn.Column{Name: "Lesson", RightAlign: true},
		mdwn.Column{Name: "Leader"},
		mdwn.Column{Name: "Song"},
		mdwn.Column{Name: "Key"},
		mdwn.Column{Name: "Title"},
	).Extend(irt.Convert(erc.HandleAll(conn.SingingLessons(ctx, p.Name), ec.Push), func(s models.SingingLessionInfo) []string {
		return []string{strconv.Itoa(s.LessonID), s.SingerName, s.SongPageNumber, s.SongKey, s.SongName}
	})).Build()
	mb.Line()

	ec.Push(flush(wr, &mb))
	return ec.Resolve()
}

func buddiesReport(ctx context.Context, conn *db.Connection, p reportParams) (err error) {
	var ec erc.Collector
	var mb mdwn.Builder

	singer, err := fzfui.SelectLeader(ctx, conn, p.Name)
	if err != nil {
		return err
	}

	w, err := p.getWriter(singer)
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(w.Close()) }()

	mb.H2(fmt.Sprintf("Singing Buddies: %s", singer))
	mb.KVTable(
		irt.MakeKV("Name", "Shared Singings"),
		irt.Convert2(irt.KVsplit(erc.HandleAll(conn.SingingBuddies(ctx, singer, cmp.Or(p.Limit, 24)), ec.Push)), intValToStr),
	)
	mb.Line()

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func strangersReport(ctx context.Context, conn *db.Connection, p reportParams) (err error) {
	var ec erc.Collector
	var mb mdwn.Builder

	singer, err := fzfui.SelectLeader(ctx, conn, p.Name)
	if err != nil {
		return err
	}

	w, err := p.getWriter(singer, "strangers")
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(w.Close()) }()

	mb.H2(fmt.Sprintf("Singing Strangers: %s", singer))
	mb.KVTable(
		irt.MakeKV("Name", "Mutual Connections"),
		irt.Convert2(irt.KVsplit(erc.HandleAll(conn.SingingStrangers(ctx, singer, cmp.Or(p.Limit, 24)), ec.Push)), intValToStr),
	)
	mb.Line()

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func popularInExperienceReport(ctx context.Context, conn *db.Connection, p reportParams) (err error) {
	singer, err := fzfui.SelectLeader(ctx, conn, p.Name)
	if err != nil {
		return err
	}

	w, err := p.getWriter(singer, "popular", "experience")
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(w.Close()) }()

	var ec erc.Collector
	var mb mdwn.Builder

	mb.H2(fmt.Sprintf("Popular in %s's Experience", singer))
	writeSongTable(&mb, erc.HandleAll(conn.PopularSongsInOnesExperience(ctx, singer, cmp.Or(p.Limit, defaultN)), ec.Push))

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func itoa(in int) string { return strconv.Itoa(in) }
func popularInYearsReport(ctx context.Context, conn *db.Connection, p reportParams) error {
	singer, err := fzfui.SelectLeader(ctx, conn, p.Name)
	if err != nil {
		return err
	}
	years, err := fzfui.SelectYears("") // TODO change upstream function to take integers and separate out parings
	if err != nil {
		return err
	}

	w, err := p.getWriter(slices.AppendSeq([]string{"popular", "year"}, irt.Convert(irt.Slice(years), itoa))...)
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(w.Close()) }()

	var ec erc.Collector
	var mb mdwn.Builder

	for part := range strings.SplitSeq(singer, ",") {
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

func locallyPopularReport(ctx context.Context, conn *db.Connection, p reportParams) (err error) {
	var ec erc.Collector
	var mb mdwn.Builder

	var localities []models.SingingLocality
	for part := range strings.SplitSeq(p.Name, ",") {
		localities = append(localities, models.NewSingingLocality(strings.TrimSpace(part)))
	} // TODO have a locality selector, and validate input

	wr, err := p.getWriter("report", "popularity", strings.ReplaceAll(p.Name, ",", "-"))
	if !ec.PushOk(err) {
		return ec.Resolve()
	}
	defer func() { err = erc.Join(wr.Close()) }()

	mb.H2(fmt.Sprintf("Locally Popular: %s", p.Name))
	writeSongTable(&mb, erc.HandleAll(conn.LocallyPopular(ctx, cmp.Or(p.Limit, 32), localities...), ec.Push))

	ec.Push(flush(wr, &mb))
	return ec.Resolve()
}

func neverSungReport(ctx context.Context, conn *db.Connection, p reportParams) error {
	singer, err := fzfui.SelectLeader(ctx, conn, p.Name)
	if err != nil {
		return err
	}

	w, err := p.getWriter(singer)
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(w.Close()) }()

	var ec erc.Collector
	var mb mdwn.Builder

	mb.H2(fmt.Sprintf("Never Sung: %s", singer))
	writeSongTable(&mb, erc.HandleAll(irt.Limit2(conn.NeverSung(ctx, singer), cmp.Or(p.Limit, 20)), ec.Push))

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func neverLedReport(ctx context.Context, conn *db.Connection, p reportParams) error {
	singer, err := fzfui.SelectLeader(ctx, conn, p.Name)
	if err != nil {
		return err
	}

	w, err := p.getWriter(singer)
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(w.Close()) }()

	var ec erc.Collector
	var mb mdwn.Builder

	mb.H2(fmt.Sprintf("Never Led: %s", singer))
	writeSongTable(&mb, erc.HandleAll(irt.Limit2(conn.NeverLed(ctx, singer), cmp.Or(p.Limit, 20)), ec.Push))

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func unfamilarHitsReport(ctx context.Context, conn *db.Connection, p reportParams) error {
	singer, err := fzfui.SelectLeader(ctx, conn, p.Name)
	if err != nil {
		return err
	}

	w, err := p.getWriter(singer)
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(w.Close()) }()

	var ec erc.Collector
	var mb mdwn.Builder

	mb.H2(fmt.Sprintf("Unfamiliar Hits: %s", singer))
	writeSongTable(&mb, erc.HandleAll(conn.TheUnfamilarHits(ctx, singer, cmp.Or(p.Limit, 20)), ec.Push))

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func connectednessReport(ctx context.Context, conn *db.Connection, p reportParams) error {
	var ec erc.Collector
	var mb mdwn.Builder

	w, err := p.getWriter("report", "connectedness")
	if err != nil {
		return err
	}

	defer func() { err = erc.Join(w.Close()) }()
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

func topLeadersReport(ctx context.Context, conn *db.Connection, p reportParams) (err error) {
	var ec erc.Collector
	var mb mdwn.Builder
	years, err := fzfui.SelectYears(p.Name) // TODO change upstream function to take integers and separate out parings
	if err != nil {
		return err
	}

	yearsStr := irt.Collect(irt.Convert(irt.Slice(years), itoa))
	w, err := p.getWriter(append([]string{"report", "top", "leaders"}, yearsStr...)...)
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(w.Close()) }()

	mb.H2("Top Leaders")
	if len(years) > 0 {
		mb.KV("Years", strings.Join(yearsStr, ", "))
	}

	var pos int
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

func leaderShareReport(ctx context.Context, conn *db.Connection, p reportParams) error {
	var ec erc.Collector
	var mb mdwn.Builder

	singer, err := fzfui.SelectLeader(ctx, conn, p.Name)
	if err != nil {
		return err
	}
	years, err := fzfui.SelectYears("") // TODO change upstream function to take integers and separate out parings
	if err != nil {
		return err
	}
	yearsStr := irt.Collect(irt.Convert(irt.Slice(years), itoa))
	wr, err := p.getWriter(append(append([]string{}, singer, "leading", "share"), yearsStr...)...)
	if !ec.PushOk(err) {
		return ec.Resolve()
	}
	defer func() { err = erc.Join(wr.Close()) }()

	v, err := conn.LeaderShareOfLeads(ctx, singer, years...)
	ec.Push(err)

	mb.H2(fmt.Sprintf("Leader Share: %s", singer))
	mb.KV("Leader", singer)
	if len(years) > 0 {
		mb.KV("Year(s)", strings.Join(yearsStr, ", "))
	}
	mb.KV("Share of Leads", fmt.Sprintf("%.4f%%", stw.DerefZ(v)*100))
	mb.Line()

	ec.Push(flush(wr, &mb))
	return ec.Resolve()
}

func leaderFootstepsReport(ctx context.Context, conn *db.Connection, p reportParams) error {
	var ec erc.Collector
	var mb mdwn.Builder

	wr, err := p.getWriter("report-leading-in-the-footsteps")
	if !ec.PushOk(err) {
		return ec.Resolve()
	}
	defer func() { err = erc.Join(wr.Close()) }()

	mb.H2(fmt.Sprintf("Leader Footsteps: %s", p.Name))
	writeLeaderFootstepTable(&mb, erc.HandleAll(conn.LeaderFootsteps(ctx, p.Name, cmp.Or(p.Limit, 20)), ec.Push))

	ec.Push(flush(wr, &mb))
	return ec.Resolve()
}

func dispatcher(op fzfui.MinutesAppOperation) reporterFunc {
	switch op {
	case fzfui.MinutesAppOpLeaders:
		return reportAction
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
	case fzfui.MinutesAppOpExit:
		panic(op.String())
	case fzfui.MinutesAppOpUnknown:
		panic(op.String())
	case fzfui.MinutesAppOpInvalid:
		panic(op.String())
	default:
		return nil
	}
}

func atoi(in string) (n int)                              { n, _ = strconv.Atoi(in); return }
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
