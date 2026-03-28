package reportui

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/stw"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/fzfui"
	"github.com/tychoish/odem/pkg/mdwn"
	"github.com/tychoish/odem/pkg/models"
)

const defaultN = 25

func Leader(ctx context.Context, conn *db.Connection, in Params) (err error) {
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

	share, err := conn.LeaderShareOfLeads(ctx, singer)
	ec.Push(err)
	v, err := conn.SingersConnectedness(ctx, singer)
	ec.Push(err)

	mb.KV("Generated", time.Now().Format(time.DateOnly))
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

func Songs(ctx context.Context, conn *db.Connection, p Params) (err error) {
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

func Singings(ctx context.Context, conn *db.Connection, p Params) (err error) {
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

func Buddies(ctx context.Context, conn *db.Connection, p Params) (err error) {
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

func Strangers(ctx context.Context, conn *db.Connection, p Params) (err error) {
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

func PopularityAsExperienced(ctx context.Context, conn *db.Connection, p Params) (err error) {
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
func PopularityInYears(ctx context.Context, conn *db.Connection, p Params) error {
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

func LocallyPopular(ctx context.Context, conn *db.Connection, p Params) (err error) {
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

func NeverSung(ctx context.Context, conn *db.Connection, p Params) error {
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

func NeverLed(ctx context.Context, conn *db.Connection, p Params) error {
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

func UnfamilarHits(ctx context.Context, conn *db.Connection, p Params) error {
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

func Connectedness(ctx context.Context, conn *db.Connection, p Params) error {
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

func TopLeader(ctx context.Context, conn *db.Connection, p Params) (err error) {
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

func LeadershipShare(ctx context.Context, conn *db.Connection, p Params) error {
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

func LeaderFootsteps(ctx context.Context, conn *db.Connection, p Params) error {
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
