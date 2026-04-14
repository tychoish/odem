// Package fzfui is a fuzzy-finder UI interface
package fzfui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/mdwn"
	"github.com/tychoish/grip"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/infra"
	"github.com/tychoish/odem/pkg/models"
	"github.com/tychoish/odem/pkg/selector"
)

func LeaderAction(ctx context.Context, conn *db.Connection, arg string) error {
	singer, err := selector.Leader(ctx, conn, new(infra.SearchParams).With(arg).WithPrompt("leader"))
	if err != nil {
		return err
	}
	grip.Info(grip.MPrintf("songs led by: %s", singer))

	return renderTable(models.WriteTable, conn.MostLedSongs(ctx, singer.Name, 32))
}

func SongsByWordAction(ctx context.Context, conn *db.Connection, word string) error {
	if word == "" {
		var err error
		word, err = selector.Concordance(ctx, conn, new(infra.SearchParams).WithPrompt("concordance"))
		if err != nil {
			return err
		}
	}
	grip.Info(grip.MPrintf("songs containing %q", word))
	return renderTable(models.WriteTable, conn.SongsByWord(ctx, word, 50))
}

func SongLyricsAction(ctx context.Context, conn *db.Connection, song string) error {
	var s *models.SongDetail
	var ec erc.Collector

	if song != "" {
		sg, err := conn.GetSong(ctx, song)
		ec.Push(err)
		s = &sg
	}

	if s == nil {
		sg, err := infra.NewFuzzySearch[models.SongDetail](
			irt.Collect(erc.HandleAll(conn.AllSongDetails(ctx), ec.Push)),
		).WithToString(func(in models.SongDetail) string {
			return fmt.Sprintf("pg %s -- %s", in.PageNum, in.SongTitle)
		}).Prompt("songs").FindOne()

		ec.Push(err)
		s = &sg
	}

	ec.When(s == nil, "no matching song found")
	if !ec.Ok() {
		return ec.Resolve()
	}

	sl, err := conn.SongLyrics(ctx, s.PageNum)
	if ec.PushOk(err) {
		var mb mdwn.Builder
		mb.H1(sl.PageNum, " — ", sl.SongTitle)
		mb.KV("Page", sl.PageNum)
		mb.KV("Music", sl.MusicAttribution)
		mb.KV("Words", sl.WordsAttribution)
		mb.KV("Meter", sl.SongMeter)
		mb.KV("Key", sl.Keys)
		mb.Line()
		mb.Paragraph(sl.Text)
		ec.Push(flush(os.Stdout, &mb))
	}
	return ec.Resolve()
}

func SongAction(ctx context.Context, conn *db.Connection, song string) error {
	var s *models.SongDetail
	var ec erc.Collector

	if song != "" {
		sg, err := conn.GetSong(ctx, song)
		ec.Push(err)
		s = &sg
	}

	if s == nil {
		sg, err := infra.NewFuzzySearch[models.SongDetail](
			irt.Collect(erc.HandleAll(conn.AllSongDetails(ctx), ec.Push)),
		).WithToString(func(in models.SongDetail) string {
			return fmt.Sprintf("pg %s -- %s", in.PageNum, in.SongTitle)
		}).Prompt("songs").FindOne()

		ec.Push(err)
		s = &sg
	}

	ec.When(s == nil, "no matching song found")
	if !ec.Ok() {
		return ec.Resolve()
	}

	grip.Info(grip.MPrintln("song info for", s.PageNum))

	var mb mdwn.Builder
	for k, v := range infra.IterStruct(s) {
		mb.KV(k, fmt.Sprint(v))
	}
	ec.Push(flush(os.Stdout, &mb))
	grip.Info(grip.MPrintln("top leaders of", s.PageNum))
	ec.Push(renderTable(models.WriteTable, conn.TopLeadersOfSong(ctx, s.PageNum, 20)))

	return ec.Resolve()
}

func SingingAction(ctx context.Context, dbconn *db.Connection) error {
	singing, err := selector.Singing(ctx, dbconn, new(infra.SearchParams).With("").WithPrompt("leader"))
	if err != nil {
		return err
	}
	grip.Info("Singing:")
	if err := infra.WriteTabbedKVs(os.Stdout, infra.IterStruct(singing)); err != nil {
		return err
	}
	grip.Info("Lessons:")
	return renderTable(models.WriteTable, dbconn.SingingLessons(ctx, singing.SingingName))
}

func LeaderLeadHistoryAction(ctx context.Context, dbconn *db.Connection, input string) error {
	singer, err := selector.Leader(ctx, dbconn, new(infra.SearchParams).With(input).WithPrompt("leader"))
	if err != nil {
		return err
	}
	grip.Info(grip.MPrintf("lead history for: %s", singer.Name))

	return renderTable(models.WriteTable, dbconn.LeaderLeadHistory(ctx, singer.Name, 50000))
}

func LeaderSingingsAttendedAction(ctx context.Context, dbconn *db.Connection, input string) error {
	singer, err := selector.Leader(ctx, dbconn, new(infra.SearchParams).With(input).WithPrompt("leader"))
	if err != nil {
		return err
	}
	grip.Info(grip.MPrintf("singings attended by: %s", singer))

	return renderTable(models.WriteTable, dbconn.LeaderSingingsAttended(ctx, singer.Name, 0))
}

func SingingBuddiesAction(ctx context.Context, dbconn *db.Connection, input string) error {
	singer, err := selector.Leader(ctx, dbconn, new(infra.SearchParams).With(input).WithPrompt("leader"))
	if err != nil {
		return err
	}

	grip.Info(grip.MPrintf("singing buddies for %q", singer.Name))
	return renderTable(models.WriteTable, dbconn.SingingBuddies(ctx, singer.Name, 20))
}

func SingingStrangersAction(ctx context.Context, dbconn *db.Connection, input string) error {
	singer, err := selector.Leader(ctx, dbconn, new(infra.SearchParams).With(input).WithPrompt("leader"))
	if err != nil {
		return err
	}

	grip.Info(grip.MPrintf("singing strangers for %q", singer))
	return renderTable(models.WriteTable, dbconn.SingingStrangers(ctx, singer.Name, 20))
}

func PopularInOnesExperienceAction(ctx context.Context, dbconn *db.Connection, input string) error {
	singer, err := selector.Leader(ctx, dbconn, new(infra.SearchParams).With(input).WithPrompt("leader"))
	if err != nil {
		return err
	}

	grip.Info(grip.MPrintf("most common songs at singings attended by %s", singer.Name))
	return renderTable(models.WriteTable, dbconn.PopularAsObserved(ctx, input, 20))
}

func NeverSungAction(ctx context.Context, dbconn *db.Connection, input string) error {
	singer, err := selector.Leader(ctx, dbconn, new(infra.SearchParams).With(input).WithPrompt("leader"))
	if err != nil {
		return err
	}

	grip.Info(grip.MPrintf("songs never sung at singing %s was present at", singer.Name))
	return renderTable(models.WriteTable, dbconn.NeverSung(ctx, singer.Name))
}

func NeverLedAction(ctx context.Context, dbconn *db.Connection, input string) error {
	singer, err := selector.Leader(ctx, dbconn, new(infra.SearchParams).With(input).WithPrompt("leader"))
	if err != nil {
		return err
	}

	grip.Info(grip.MPrintf("songs never led by %s", singer.Name))
	return renderTable(models.WriteTable, dbconn.NeverLed(ctx, singer.Name, 40))
}

func LocallyPopularAction(ctx context.Context, dbconn *db.Connection, arg string) error {
	localities := irt.Collect(irt.Convert(strings.SplitSeq(arg, " "), models.NewSingingLocality))

	if len(localities) == 0 {
		var err error
		localities, err = erc.FromIteratorAll(infra.NewFuzzySearch[models.SingingLocality](models.AllLocalities()).Prompt("location").Find())
		if err != nil {
			return err
		}
	}

	grip.Info(grip.MPrintf("popular songs in a specific location %v", localities))
	return renderTable(models.WriteTable, dbconn.LocallyPopular(ctx, 20, localities...))
}

func UnfamilarHitsAction(ctx context.Context, dbconn *db.Connection, input string) error {
	singer, err := selector.Leader(ctx, dbconn, new(infra.SearchParams).With(input).WithPrompt("leader"))
	if err != nil {
		return err
	}

	grip.Info(grip.MPrintf("otherwise popular songs less-or-unfamilar to %s", singer.Name))
	return renderTable(models.WriteTable, dbconn.TheUnfamilarHits(ctx, singer.Name, 20))
}

func LeaderFavoriteKeyAction(ctx context.Context, dbconn *db.Connection, input string) error {
	singer, err := selector.Leader(ctx, dbconn, new(infra.SearchParams).With(input).WithPrompt("leader"))
	if err != nil {
		return err
	}

	grip.Info(grip.MPrintf("leads per key for %q", singer.Name))
	return renderTable(models.WriteTable, dbconn.LeaderFavoriteKey(ctx, singer.Name, 20))
}

func SingersByConnectednessAction(ctx context.Context, dbconn *db.Connection) error {
	grip.Info("singers ranked by connectedness ratio")
	return renderTable(models.WriteTable, dbconn.AllLeaderConnectedness(ctx, 32))
}

func LeaderFootstepsAction(ctx context.Context, dbconn *db.Connection, input string) error {
	singer, err := selector.Leader(ctx, dbconn, new(infra.SearchParams).With(input).WithPrompt("leader"))
	if err != nil {
		return err
	}

	grip.Info(grip.MPrintf("songs led by %s, ranked by the most frequent other leader of each song", singer.Name))

	return renderTable(models.WriteTable, dbconn.LeaderFootsteps(ctx, singer.Name, 32))
}

func LeadersShareOfLeadsAction(ctx context.Context, dbconn *db.Connection, input string) error {
	// input may be "Singer Name" or "Singer Name,2023,2024"
	name, yrstr, _ := strings.Cut(input, ",")
	singer, err := selector.Leader(ctx, dbconn, new(infra.SearchParams).With(name).WithPrompt("leader"))
	if err != nil {
		return err
	}

	years, err := selector.Years(new(infra.SearchParams).With(yrstr).WithPrompt("years (0 = all)").WithMulti())
	if err != nil {
		return err
	}

	grip.Info(grip.MPrintf("lead share for %q in year(s) %v", singer.Name, years))
	v, err := dbconn.LeaderShareOfLeads(ctx, singer.Name, 16, years...)
	if err != nil {
		return err
	}
	label := "Share of All Leads"
	if len(years) > 0 {
		label = fmt.Sprintf("Share of Leads (%v)", years)
	}
	var mb mdwn.Builder
	mb.KV("Leader", singer.Name)
	mb.KV(label, fmt.Sprintf("%.4f%%", *v*100))

	return flush(os.Stdout, &mb)
}

func TopLeadersByLeadsAction(ctx context.Context, dbconn *db.Connection, yrs string) error {
	years, err := selector.Years(new(infra.SearchParams).With(yrs).WithPrompt("years (0 = all)").WithMulti())
	if err != nil {
		return err
	}

	grip.Info(grip.MPrintf("leaders by total leads in year(s) %v", years))

	return renderTable(
		models.WriteTable,
		irt.Convert2(
			dbconn.TopLeadersByLeads(ctx, 40, years...),
			infra.PassErrorThroughConverter(models.TopLeadersWrapper(&atomic.Int64{})),
		),
	)
}

func NewLeadersByYearAction(ctx context.Context, dbconn *db.Connection, arg string) error {
	years, err := selector.Years(new(infra.SearchParams).With(arg).WithPrompt("years (0 = all)").WithMulti())
	if err != nil {
		return err
	}
	year := time.Now().Year()
	if len(years) > 0 && years[0] > 0 {
		year = years[0]
	}
	grip.Info(grip.MPrintf("debut leaders in %d", year))
	return renderTable(writeLeaderCountTable, dbconn.NewLeadersByYear(ctx, year, 40))
}

func SongsByKeyAction(ctx context.Context, dbconn *db.Connection, yrs string) error {
	years, err := selector.Years(new(infra.SearchParams).With(yrs).WithPrompt("years (0 = all)").WithMulti())
	if err != nil {
		return err
	}

	grip.Info(grip.MPrintf("lessons by key in year(s) %v", years))

	return renderTable(
		models.WriteTable,
		irt.Convert2(
			dbconn.SongsByKey(ctx, years...),
			infra.PassErrorThroughConverter(models.WrapSongByKey),
		),
	)
}

func LeadersByTop20LeadsAction(ctx context.Context, dbconn *db.Connection, _ string) error {
	grip.Info("singers ordered by number of top-20 leads")
	return renderTable(writeLeaderCountTable, dbconn.LeadersByTop20Leads(ctx, 40))
}

func Top20LeadersActiveInLastYearAction(ctx context.Context, dbconn *db.Connection, _ string) error {
	grip.Info("top-20 leaders who have led a song in the last year")
	return renderTable(writeLeaderCountTable, dbconn.Top20LeadersActiveInLastYear(ctx, 40))
}

func LeaderSingingsPerYearAction(ctx context.Context, dbconn *db.Connection, input string) error {
	singer, err := selector.Leader(ctx, dbconn, new(infra.SearchParams).With(input).WithPrompt("leader"))
	if err != nil {
		return err
	}

	grip.Info(grip.MPrintf("singings per year for %q", singer.Name))
	return renderTable(models.WriteTable, dbconn.LeaderSingingsPerYear(ctx, singer.Name))
}

func LeadersByKeyAction(ctx context.Context, dbconn *db.Connection, key string) error {
	var err error
	key, err = selector.Key(ctx, dbconn, new(infra.SearchParams).With(key).WithPrompt("key"))
	if err != nil {
		return err
	}

	grip.Info(grip.MPrintf("leaders by number of leads in key %q", key))
	return renderTable(writeLeaderCountTable, dbconn.LeadersByKey(ctx, key, 40))
}

func PopularSongsByKeyAction(ctx context.Context, dbconn *db.Connection, key string) error {
	key, err := selector.Key(ctx, dbconn, new(infra.SearchParams).With(key).WithPrompt("key"))
	if err != nil {
		return err
	}

	grip.Info(grip.MPrintf("popular songs in key %q", key))
	return renderTable(models.WriteTable, dbconn.PopularSongsByKey(ctx, key, 40))
}

func PopularInYearsAction(ctx context.Context, dbconn *db.Connection, yrs string) error {
	years, err := selector.Years(new(infra.SearchParams).With(yrs).WithPrompt("years (0 = all)").WithMulti())
	if err != nil {
		return err
	}

	grip.Info(grip.MPrintf("songs by popularity in year(s) %v", years))
	return renderTable(models.WriteTable, dbconn.GloballyPopularForYears(ctx, 20, years...))
}
