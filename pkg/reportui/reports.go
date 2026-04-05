package reportui

import (
	"cmp"
	"context"
	"fmt"
	"iter"
	"strconv"
	"strings"
	"time"

	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/fnx"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/stw"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/fzfui"
	"github.com/tychoish/odem/pkg/mdwn"
	"github.com/tychoish/odem/pkg/models"
)

const defaultN = 25

func LeaderJobs(conn *db.Connection, basePath string, leaders []string) iter.Seq[fnx.Worker] {
	return irt.Convert(irt.Slice(leaders), func(leader string) fnx.Worker {
		return func(ctx context.Context) error {
			return Leader(ctx, conn, Params{
				SuppressInteractivity: true,
				PathPrefix:            basePath,
				Params:                models.Params{Name: leader},
			})
		}
	})
}

func Leader(ctx context.Context, conn *db.Connection, in Params) (err error) {
	singer, err := in.SelectLeader(ctx, conn)
	if err != nil {
		return err
	}
	w, err := in.getWriter(singer)
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(w.Close()) }()

	// ---------------- THE FOLD ----------------
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
	mb.H2("Favorite Keys")
	mb.KVTable(
		irt.MakeKV("Count", "Key"),
		irt.Convert2(irt.KVsplit(erc.HandleAll(conn.LeaderFavoriteKey(ctx, singer, 100), ec.Push)), intValToStr),
	)
	mb.Line()

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
	mb.KVTable(
		irt.MakeKV("Name", "Mutual Connections"),
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

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func Songs(ctx context.Context, conn *db.Connection, p Params) (err error) {
	sg, err := fzfui.SelectSong(ctx, conn, p.Name)
	if err != nil {
		return err
	}

	wr, err := p.getWriter(stw.DerefZ(sg).PageNum)
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(err, wr.Close()) }()
	// ---------------- THE FOLD ----------------
	var ec erc.Collector
	var mb mdwn.Builder

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
	wr, err := p.getWriter("siging", stw.DerefZ(info).SingingName)
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(err, wr.Close()) }()
	// ---------------- THE FOLD ----------------
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H2("Singing Details")
	mb.KV("Name", strings.ReplaceAll(info.SingingName, "\\n", "; "))
	mb.KV("Date", info.SingingDate.Time().Format(time.DateOnly))
	mb.KV("Location", info.SingingLocation)
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
	singer, err := p.SelectLeader(ctx, conn)
	if err != nil {
		return err
	}

	w, err := p.getWriter(singer)
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(w.Close()) }()
	// ---------------- THE FOLD ----------------
	var ec erc.Collector
	var mb mdwn.Builder

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
	singer, err := p.SelectLeader(ctx, conn)
	if err != nil {
		return err
	}

	w, err := p.getWriter(singer, "strangers")
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(w.Close()) }()
	// ---------------- THE FOLD ----------------
	var ec erc.Collector
	var mb mdwn.Builder

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
	singer, err := p.SelectLeader(ctx, conn)
	if err != nil {
		return err
	}

	w, err := p.getWriter(singer, "popular", "experience")
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(w.Close()) }()
	// ---------------- THE FOLD ----------------
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H2(fmt.Sprintf("Popular in %s's Experience", singer))
	writeSongTable(&mb, erc.HandleAll(conn.PopularSongsInOnesExperience(ctx, singer, cmp.Or(p.Limit, defaultN)), ec.Push))

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func PopularityInYears(ctx context.Context, conn *db.Connection, p Params) error {
	years, err := fzfui.SelectYears(p.Name) // TODO change upstream function to take integers and separate out parings
	if err != nil {
		return err
	}

	yearsStrs := irt.Collect(irt.Convert(irt.Slice(years), itoa))
	w, err := p.getWriter(append([]string{"popular"}, yearsStrs...)...)
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(w.Close()) }()
	// ---------------- THE FOLD ----------------
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H2("Globally Popular")

	mb.KV("Years", cmp.Or(strings.Join(yearsStrs, ", "), "(all)"))

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
	// ---------------- THE FOLD ----------------
	mb.H2(fmt.Sprintf("Locally Popular: %s", p.Name))
	writeSongTable(&mb, erc.HandleAll(conn.LocallyPopular(ctx, cmp.Or(p.Limit, 32), localities...), ec.Push))

	ec.Push(flush(wr, &mb))
	return ec.Resolve()
}

func NeverSung(ctx context.Context, conn *db.Connection, p Params) error {
	singer, err := p.SelectLeader(ctx, conn)
	if err != nil {
		return err
	}

	w, err := p.getWriter(singer)
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(w.Close()) }()
	// ---------------- THE FOLD ----------------
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H2(fmt.Sprintf("Never Sung: %s", singer))
	writeSongTable(&mb, erc.HandleAll(irt.Limit2(conn.NeverSung(ctx, singer), cmp.Or(p.Limit, 20)), ec.Push))

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func NeverLed(ctx context.Context, conn *db.Connection, p Params) error {
	singer, err := p.SelectLeader(ctx, conn)
	if err != nil {
		return err
	}

	w, err := p.getWriter(singer)
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(w.Close()) }()
	// ---------------- THE FOLD ----------------
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H2(fmt.Sprintf("Never Led: %s", singer))
	writeSongTable(&mb, erc.HandleAll(irt.Limit2(conn.NeverLed(ctx, singer), cmp.Or(p.Limit, 20)), ec.Push))

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func UnfamilarHits(ctx context.Context, conn *db.Connection, p Params) error {
	singer, err := p.SelectLeader(ctx, conn)
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
	// ---------------- THE FOLD ----------------
	mb.H2(fmt.Sprintf("Unfamiliar Hits: %s", singer))
	writeSongTable(&mb, erc.HandleAll(conn.TheUnfamilarHits(ctx, singer, cmp.Or(p.Limit, 20)), ec.Push))

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func LeaderFavoriteKey(ctx context.Context, conn *db.Connection, p Params) (err error) {
	singer, err := p.SelectLeader(ctx, conn)
	if err != nil {
		return err
	}

	w, err := p.getWriter(singer, "favorite-key")
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(w.Close()) }()
	// ---------------- THE FOLD ----------------
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H2(fmt.Sprintf("Leads by Key: %s", singer))
	mb.KVTable(
		irt.MakeKV("Key", "Leads"),
		irt.Convert2(irt.KVsplit(erc.HandleAll(conn.LeaderFavoriteKey(ctx, singer, cmp.Or(p.Limit, 20)), ec.Push)), intValToStr),
	)
	mb.Line()

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func Connectedness(ctx context.Context, conn *db.Connection, p Params) error {
	w, err := p.getWriter("report", "connectedness")
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(w.Close()) }()
	// ---------------- THE FOLD ----------------
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H2("Leaders by Connectedness")
	mb.KVTable(
		irt.MakeKV("Name", "Connectedness"),
		irt.Convert2(irt.KVsplit(erc.HandleAll(conn.AllLeaderConnectedness(ctx, cmp.Or(p.Limit, 40)), ec.Push)), fmtPercentKVs),
	)
	mb.Line()

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func TopLeader(ctx context.Context, conn *db.Connection, p Params) (err error) {
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
	// ---------------- THE FOLD ----------------
	var ec erc.Collector
	var mb mdwn.Builder

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
	singer, err := p.SelectLeader(ctx, conn)
	if err != nil {
		return err
	}
	years, err := fzfui.SelectYears("") // TODO change upstream function to take integers and separate out parings
	if err != nil {
		return err
	}
	yearsStr := irt.Collect(irt.Convert(irt.Slice(years), itoa))
	wr, err := p.getWriter(append(append([]string{}, singer, "leading", "share"), yearsStr...)...)
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(wr.Close()) }()
	// ---------------- THE FOLD ----------------
	var ec erc.Collector
	var mb mdwn.Builder

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

func LeaderLeadHistory(ctx context.Context, conn *db.Connection, p Params) (err error) {
	singer, err := p.SelectLeader(ctx, conn)
	if err != nil {
		return err
	}

	wr, err := p.getWriter(singer, "lead-history")
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(wr.Close()) }()
	// ---------------- THE FOLD ----------------
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H2(fmt.Sprintf("Lead History: %s", singer))
	mb.NewTable(
		mdwn.Column{Name: "Date"},
		mdwn.Column{Name: "Singing"},
		mdwn.Column{Name: "Song"},
		mdwn.Column{Name: "Page"},
		mdwn.Column{Name: "Key"},
	).Extend(irt.Convert(erc.HandleAll(conn.LeaderLeadHistory(ctx, singer), ec.Push), func(row models.LessonInfo) []string {
		return []string{row.SingingDate.String(), strings.ReplaceAll(row.SingingName, "\\n", "; "), row.SongName, row.SongPageNumber, row.SongKey}
	})).Build()
	mb.Line()

	ec.Push(flush(wr, &mb))
	return ec.Resolve()
}

func LeaderSingings(ctx context.Context, conn *db.Connection, p Params) (err error) {
	singer, err := p.SelectLeader(ctx, conn)
	if err != nil {
		return err
	}

	wr, err := p.getWriter(singer, "singings")
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(wr.Close()) }()
	// ---------------- THE FOLD ----------------
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H2(fmt.Sprintf("Singings Attended: %s", singer))
	mb.NewTable(
		mdwn.Column{Name: "Date"},
		mdwn.Column{Name: "Singing"},
		mdwn.Column{Name: "State"},
		mdwn.Column{Name: "City"},
		mdwn.Column{Name: "Led", RightAlign: true},
		mdwn.Column{Name: "Leaders", RightAlign: true},
	).Extend(irt.Convert(erc.HandleAll(conn.LeaderSingingsAttended(ctx, singer, cmp.Or(p.Limit, 0)), ec.Push), func(row models.LeaderSingingAttendance) []string {
		return []string{row.SingingDate.String(), strings.ReplaceAll(row.SingingName, "\\n", "; "), row.SingingState, row.SingingCity, strconv.Itoa(row.LeaderLeadCount), strconv.Itoa(row.NumberOfLeaders)}
	})).Build()
	mb.Line()

	ec.Push(flush(wr, &mb))
	return ec.Resolve()
}

func NewLeadersByYear(ctx context.Context, conn *db.Connection, p Params) (err error) {
	year := time.Now().Year()
	if len(p.Years) > 0 && p.Years[0] != 0 {
		year = p.Years[0]
	}

	w, err := p.getWriter("report", "new-leaders", strconv.Itoa(year))
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(w.Close()) }()
	// ---------------- THE FOLD ----------------
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H2(fmt.Sprintf("Debut Leaders: %d", year))
	mb.KV("Year", strconv.Itoa(year))
	mb.Line()

	mb.NewTable(
		mdwn.Column{Name: "Name"},
		mdwn.Column{Name: "Leads", RightAlign: true},
	).Extend(irt.Convert(erc.HandleAll(conn.NewLeadersByYear(ctx, year, cmp.Or(p.Limit, 40)), ec.Push), func(row models.LeaderSongRank) []string {
		return []string{row.Leader, row.NumLeads}
	})).Build()
	mb.Line()

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func SongsByKey(ctx context.Context, conn *db.Connection, p Params) (err error) {
	years := p.Years
	yearsStrs := irt.Collect(irt.Convert(irt.Slice(years), itoa))

	w, err := p.getWriter(append([]string{"report", "songs-by-key"}, yearsStrs...)...)
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(w.Close()) }()
	// ---------------- THE FOLD ----------------
	var ec erc.Collector
	var mb mdwn.Builder

	heading := "Songs by Key (All Time)"
	if len(years) > 0 {
		heading = fmt.Sprintf("Songs by Key (%s)", strings.Join(yearsStrs, ", "))
	}
	mb.H2(heading)

	mb.NewTable(
		mdwn.Column{Name: "Key"},
		mdwn.Column{Name: "Count", RightAlign: true},
		mdwn.Column{Name: "Percentage", RightAlign: true},
	).Extend(irt.Convert(erc.HandleAll(conn.SongsByKey(ctx, years...), ec.Push), func(row models.LeaderSongRank) []string {
		return []string{row.Key, row.NumLeads, fmt.Sprintf("%.1f%%", row.Ratio*100)}
	})).Build()
	mb.Line()

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func LeadersByTop20Leads(ctx context.Context, conn *db.Connection, p Params) (err error) {
	w, err := p.getWriter("report", "top20-leaders")
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(w.Close()) }()
	// ---------------- THE FOLD ----------------
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H2("Leaders by Top-20 Leads")
	mb.NewTable(
		mdwn.Column{Name: "Name"},
		mdwn.Column{Name: "Top-20 Leads", RightAlign: true},
	).Extend(irt.Convert(erc.HandleAll(conn.LeadersByTop20Leads(ctx, cmp.Or(p.Limit, 40)), ec.Push), func(row models.LeaderSongRank) []string {
		return []string{row.Leader, row.NumLeads}
	})).Build()
	mb.Line()

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func LeaderSingingsPerYear(ctx context.Context, conn *db.Connection, p Params) (err error) {
	singer, err := p.SelectLeader(ctx, conn)
	if err != nil {
		return err
	}

	w, err := p.getWriter(singer, "singings-per-year")
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(w.Close()) }()
	// ---------------- THE FOLD ----------------
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H2(fmt.Sprintf("Singings Per Year: %s", singer))
	mb.KVTable(
		irt.MakeKV("Year", "Singings"),
		irt.Convert2(irt.KVsplit(erc.HandleAll(conn.LeaderSingingsPerYear(ctx, singer), ec.Push)), intValToStr),
	)
	mb.Line()

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func LeadersByKey(ctx context.Context, conn *db.Connection, p Params) (err error) {
	key := p.Name

	w, err := p.getWriter("report", "leaders-in-key", key)
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(w.Close()) }()
	// ---------------- THE FOLD ----------------
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H2(fmt.Sprintf("Leaders in Key: %s", key))
	mb.NewTable(
		mdwn.Column{Name: "Name"},
		mdwn.Column{Name: "Count", RightAlign: true},
	).Extend(irt.Convert(erc.HandleAll(conn.LeadersByKey(ctx, key, cmp.Or(p.Limit, 40)), ec.Push), func(row models.LeaderSongRank) []string {
		return []string{row.Leader, row.NumLeads}
	})).Build()
	mb.Line()

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func PopularSongsByKey(ctx context.Context, conn *db.Connection, p Params) (err error) {
	key := p.Name

	w, err := p.getWriter("report", "songs-in-key", key)
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(w.Close()) }()
	// ---------------- THE FOLD ----------------
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H2(fmt.Sprintf("Popular Songs in Key: %s", key))
	writeSongTable(&mb, erc.HandleAll(conn.PopularSongsByKey(ctx, key, cmp.Or(p.Limit, 40)), ec.Push))

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func LeaderFootsteps(ctx context.Context, conn *db.Connection, p Params) error {
	singer, err := p.SelectLeader(ctx, conn)
	if err != nil {
		return err
	}

	wr, err := p.getWriter(singer, "footsteps")
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(wr.Close()) }()
	// ---------------- THE FOLD ----------------
	var ec erc.Collector
	var mb mdwn.Builder
	mb.H2(fmt.Sprintf("Leader Footsteps: %s", singer))
	writeLeaderFootstepTable(&mb, erc.HandleAll(conn.LeaderFootsteps(ctx, p.Name, cmp.Or(p.Limit, 20)), ec.Push))

	ec.Push(flush(wr, &mb))
	return ec.Resolve()
}
