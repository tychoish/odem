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
	"github.com/tychoish/fun/ers"
	"github.com/tychoish/fun/fnx"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/stw"
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/mdwn"
	"github.com/tychoish/odem/pkg/models"
	"github.com/tychoish/odem/pkg/selector"
)

const defaultN = 25

func LeaderJobs(conn *db.Connection, basePath string, leaders []string) iter.Seq[fnx.Worker] {
	return irt.Convert(irt.Slice(leaders), func(leader string) fnx.Worker {
		return func(ctx context.Context) error {
			grip.Info(message.NewKV().KV("leader", leader).KV("op", "batch-report"))
			return Leader(ctx, conn, Params{
				SuppressInteractivity: true,
				PathPrefix:            basePath,
				Params:                models.Params{Name: leader},
			})
		}
	})
}

func MostLed(ctx context.Context, conn *db.Connection, in Params) (err error) {
	singer, err := selector.Leader(ctx, conn, in.Search())
	if err != nil {
		return err
	}

	w, err := in.getWriter(singer.Name, "most-led")
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(w.Close()) }()

	// ---------------- THE FOLD ----------------
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H1(singer.Name, "--", "Most Led")
	models.WriteSongTable(&mb, erc.HandleAll(conn.MostLedSongs(ctx, singer.Name, 40), ec.Push))
	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func Songs(ctx context.Context, conn *db.Connection, p Params) (err error) {
	sg, err := selector.Song(ctx, conn, p.Search())
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
	models.WriteSongLeadersTable(&mb, erc.HandleAll(conn.TopLeadersOfSong(ctx, sg.PageNum, cmp.Or(p.Limit, 20)), ec.Push))

	ec.Push(flush(wr, &mb))
	return ec.Resolve()
}

func Singings(ctx context.Context, conn *db.Connection, p Params) (err error) {
	info, err := selector.Singing(ctx, conn, p.Search())
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
	models.WriteSingingLessonsTable(&mb, erc.HandleAll(conn.SingingLessons(ctx, p.Name), ec.Push))

	ec.Push(flush(wr, &mb))
	return ec.Resolve()
}

func Buddies(ctx context.Context, conn *db.Connection, p Params) (err error) {
	singer, err := selector.Leader(ctx, conn, p.Search())
	if err != nil {
		return err
	}

	w, err := p.getWriter(singer.Name, "buddies")
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(w.Close()) }()
	// ---------------- THE FOLD ----------------
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H2(fmt.Sprintf("Singing Buddies: %s", singer.Name))
	mb.KVTable(
		irt.MakeKV("Name", "Shared Singings"),
		irt.Convert2(irt.KVsplit(erc.HandleAll(conn.SingingBuddies(ctx, singer.Name, cmp.Or(p.Limit, 24)), ec.Push)), intValToStr),
	)
	mb.Line()

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func Strangers(ctx context.Context, conn *db.Connection, p Params) error {
	singer, err := selector.Leader(ctx, conn, p.Search())
	if err != nil {
		return err
	}

	w, err := p.getWriter(singer.Name, "strangers")
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(w.Close()) }()
	// ---------------- THE FOLD ----------------
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H2(fmt.Sprintf("Singing Strangers: %s", singer.Name))
	mb.KVTable(
		irt.MakeKV("Name", "Mutual Connections"),
		irt.Convert2(irt.KVsplit(erc.HandleAll(conn.SingingStrangers(ctx, singer.Name, cmp.Or(p.Limit, 24)), ec.Push)), intValToStr),
	)
	mb.Line()

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func PopularityAsExperienced(ctx context.Context, conn *db.Connection, p Params) (err error) {
	singer, err := selector.Leader(ctx, conn, p.Search())
	if err != nil {
		return err
	}

	w, err := p.getWriter(singer.Name, "popular", "experience")
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(w.Close()) }()
	// ---------------- THE FOLD ----------------
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H2(fmt.Sprintf("Popular in %s's Experience", singer.Name))
	models.WriteSongTable(&mb, erc.HandleAll(conn.PopularSongsInOnesExperience(ctx, singer.Name, cmp.Or(p.Limit, defaultN)), ec.Push))

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func PopularityInYears(ctx context.Context, conn *db.Connection, p Params) error {
	var err error
	years, err := p.selectYears()
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

	models.WriteSongTable(&mb, erc.HandleAll(conn.GloballyPopularForYears(ctx, cmp.Or(p.Limit, 40), years...), ec.Push))

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
	models.WriteSongTable(&mb, erc.HandleAll(conn.LocallyPopular(ctx, cmp.Or(p.Limit, 32), localities...), ec.Push))

	ec.Push(flush(wr, &mb))
	return ec.Resolve()
}

func NeverSung(ctx context.Context, conn *db.Connection, p Params) error {
	record, err := selector.Leader(ctx, conn, p.Search())
	if err != nil {
		return err
	}
	w, err := p.getWriter(record.Name, "never-sung")
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(w.Close()) }()
	// ---------------- THE FOLD ----------------
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H2(fmt.Sprintf("Never Sung: %s", record.Name))
	models.WriteSongTable(&mb, erc.HandleAll(irt.Limit2(conn.NeverSung(ctx, record.Name), cmp.Or(p.Limit, 20)), ec.Push))

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func NeverLed(ctx context.Context, conn *db.Connection, p Params) error {
	singer, err := selector.Leader(ctx, conn, p.Search())
	if err != nil {
		return err
	}

	w, err := p.getWriter(singer.Name, "never-led")
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(w.Close()) }()
	// ---------------- THE FOLD ----------------
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H2(fmt.Sprintf("Never Led: %s", singer.Name))
	models.WriteSongTable(&mb, erc.HandleAll(irt.Limit2(conn.NeverLed(ctx, singer.Name, cmp.Or(p.Limit, 20)), cmp.Or(p.Limit, 20)), ec.Push))

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func UnfamilarHits(ctx context.Context, conn *db.Connection, p Params) error {
	singer, err := selector.Leader(ctx, conn, p.Search())
	if err != nil {
		return err
	}

	w, err := p.getWriter(singer.Name, "unfamiliar-hits")
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(w.Close()) }()

	var ec erc.Collector
	var mb mdwn.Builder
	// ---------------- THE FOLD ----------------
	mb.H2(fmt.Sprintf("Unfamiliar Hits: %s", singer.Name))
	models.WriteSongTable(&mb, erc.HandleAll(conn.TheUnfamilarHits(ctx, singer.Name, cmp.Or(p.Limit, 20)), ec.Push))

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func LeaderFavoriteKey(ctx context.Context, conn *db.Connection, p Params) (err error) {
	record, err := selector.Leader(ctx, conn, p.Search())
	if err != nil {
		return err
	}
	w, err := p.getWriter(record.Name, "favorite-key")
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(w.Close()) }()
	// ---------------- THE FOLD ----------------
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H2(fmt.Sprintf("Leads by Key: %s", record.Name))
	mb.KVTable(
		irt.MakeKV("Key", "Leads"),
		irt.Convert2(irt.KVsplit(erc.HandleAll(conn.LeaderFavoriteKey(ctx, record.Name, cmp.Or(p.Limit, 20)), ec.Push)), intValToStr),
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
	years, err := p.selectYears()
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

	models.WriteTopLeadersTable(&mb, erc.HandleAll(conn.TopLeadersByLeads(ctx, cmp.Or(p.Limit, 40), years...), ec.Push))

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func LeadershipShare(ctx context.Context, conn *db.Connection, p Params) error {
	record, err := selector.Leader(ctx, conn, p.Search())
	if err != nil {
		return err
	}
	years, err := p.selectYears()
	if err != nil {
		return err
	}
	yearsStr := irt.Collect(irt.Convert(irt.Slice(years), itoa))
	wr, err := p.getWriter(append(append([]string{}, record.Name, "leading", "share"), yearsStr...)...)
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(wr.Close()) }()
	// ---------------- THE FOLD ----------------
	var ec erc.Collector
	var mb mdwn.Builder

	v, err := conn.LeaderShareOfLeads(ctx, record.Name, cmp.Or(p.Limit, 20), years...)
	ec.Push(err)

	mb.H2(fmt.Sprintf("Leader Share: %s", record.Name))
	mb.KV("Leader", record.Name)
	if len(years) > 0 {
		mb.KV("Year(s)", strings.Join(yearsStr, ", "))
	}
	mb.KV("Share of Leads", fmt.Sprintf("%.4f%%", stw.DerefZ(v)*100))
	mb.Line()

	ec.Push(flush(wr, &mb))
	return ec.Resolve()
}

func LeaderLeadHistory(ctx context.Context, conn *db.Connection, p Params) (err error) {
	record, err := selector.Leader(ctx, conn, p.Search())
	if err != nil {
		return err
	}

	wr, err := p.getWriter(record.Name, "lead-history")
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(wr.Close()) }()
	// ---------------- THE FOLD ----------------
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H2(fmt.Sprintf("Lead History: %s", record.Name))
	models.WriteLessonTable(&mb, erc.HandleAll(conn.LeaderLeadHistory(ctx, record.Name, cmp.Or(p.Limit, 40)), ec.Push))

	ec.Push(flush(wr, &mb))
	return ec.Resolve()
}

func LeaderSingings(ctx context.Context, conn *db.Connection, p Params) (err error) {
	record, err := selector.Leader(ctx, conn, p.Search())
	if err != nil {
		return err
	}

	wr, err := p.getWriter(record.Name, "singings")
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(wr.Close()) }()
	// ---------------- THE FOLD ----------------
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H2(fmt.Sprintf("Singings Attended: %s", record.Name))
	models.WriteLeaderSingingsTable(&mb, erc.HandleAll(conn.LeaderSingingsAttended(ctx, record.Name, cmp.Or(p.Limit, 0)), ec.Push))

	ec.Push(flush(wr, &mb))
	return ec.Resolve()
}

func NewLeadersByYear(ctx context.Context, conn *db.Connection, p Params) (err error) {
	years, err := p.selectYears()
	switch {
	case err != nil:
		return err
	case len(years) <= 0:
		return ers.New("not found")
	case len(years) > 1:
		grip.Warning(message.NewKV().KV("op", "got more than ").KV("size", len(years)))
	}

	year := years[0]

	w, err := p.getWriter("report", "new-leaders", strconv.Itoa(year))
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(w.Close()) }()
	// ---------------- THE FOLD ----------------
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H2(fmt.Sprintf("Debut Leaders: %d", year))
	models.WriteLeaderCountTable(&mb, "Leads", erc.HandleAll(conn.NewLeadersByYear(ctx, year, cmp.Or(p.Limit, 40)), ec.Push))

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func SongsByKey(ctx context.Context, conn *db.Connection, p Params) (err error) {
	years, err := p.selectYears()
	switch {
	case err != nil:
		return err
	case len(years) <= 0:
		return ers.New("not found")
	}

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
	if len(p.Years) > 0 {
		heading = fmt.Sprintf("Songs by Key (%s)", strings.Join(yearsStrs, ", "))
	}
	mb.H2(heading)

	models.WriteSongsByKeyTable(&mb, erc.HandleAll(conn.SongsByKey(ctx, p.Years...), ec.Push))

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
	models.WriteLeaderCountTable(&mb, "Top-20 Leads", erc.HandleAll(conn.LeadersByTop20Leads(ctx, cmp.Or(p.Limit, 40)), ec.Push))

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func LeaderSingingsPerYear(ctx context.Context, conn *db.Connection, p Params) (err error) {
	singer, err := selector.Leader(ctx, conn, p.Search())
	if err != nil {
		return err
	}

	w, err := p.getWriter(singer.Name, "singings-per-year")
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(w.Close()) }()
	// ---------------- THE FOLD ----------------
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H2(fmt.Sprintf("Singings Per Year: %s", singer.Name))
	mb.KVTable(
		irt.MakeKV("Year", "Singings"),
		irt.Convert2(irt.KVsplit(erc.HandleAll(conn.LeaderSingingsPerYear(ctx, singer.Name), ec.Push)), intValToStr),
	)
	mb.Line()

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func LeadersByKey(ctx context.Context, conn *db.Connection, p Params) (err error) {
	key, err := selector.Key(ctx, conn, p.Search())
	if err != nil {
		return err
	}

	w, err := p.getWriter("report", "leaders-in-key", key)
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(w.Close()) }()
	// ---------------- THE FOLD ----------------
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H2(fmt.Sprintf("Leaders in Key: %s", key))
	models.WriteLeaderCountTable(&mb, "Count", erc.HandleAll(conn.LeadersByKey(ctx, key, cmp.Or(p.Limit, 40)), ec.Push))

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func PopularSongsByKey(ctx context.Context, conn *db.Connection, p Params) (err error) {
	key, err := selector.Key(ctx, conn, p.Search())
	if err != nil {
		return err
	}

	w, err := p.getWriter("report", "songs-in-key", key)
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(w.Close()) }()
	// ---------------- THE FOLD ----------------
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H2(fmt.Sprintf("Popular Songs in Key: %s", key))
	models.WriteSongTable(&mb, erc.HandleAll(conn.PopularSongsByKey(ctx, key, cmp.Or(p.Limit, 40)), ec.Push))

	ec.Push(flush(w, &mb))
	return ec.Resolve()
}

func LeaderFootsteps(ctx context.Context, conn *db.Connection, p Params) error {
	record, err := selector.Leader(ctx, conn, p.Search())
	if err != nil {
		return err
	}

	wr, err := p.getWriter(record.Name, "footsteps")
	if err != nil {
		return err
	}
	defer func() { err = erc.Join(wr.Close()) }()
	// ---------------- THE FOLD ----------------
	var ec erc.Collector
	var mb mdwn.Builder
	mb.H2(fmt.Sprintf("Leader Footsteps: %s", record.Name))
	models.WriteLeaderFootstepTable(&mb, erc.HandleAll(conn.LeaderFootsteps(ctx, p.Name, cmp.Or(p.Limit, 20)), ec.Push))

	ec.Push(flush(wr, &mb))
	return ec.Resolve()
}
