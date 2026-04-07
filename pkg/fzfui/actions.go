package fzfui

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/grip"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/infra"
	"github.com/tychoish/odem/pkg/mdwn"
	"github.com/tychoish/odem/pkg/models"
)

func LeaderAction(ctx context.Context, conn *db.Connection, arg string) error {
	singer, err := interactivelyResolveSingerName(ctx, conn, arg)
	if err != nil {
		return err
	}
	grip.Infof("songs led by: %s", singer)

	return renderTopLedSongs(conn.MostLedSongs(ctx, singer, 32))
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

	grip.Infoln("song info for", s.PageNum)

	var mb mdwn.Builder
	for k, v := range infra.IterStruct(s) {
		mb.KV(k, fmt.Sprint(v))
	}
	ec.Push(flush(os.Stdout, &mb))
	grip.Infoln("top leaders of", s.PageNum)
	ec.Push(renderTopLeaders(conn.TopLeadersOfSong(ctx, s.PageNum, 20)))

	return ec.Resolve()
}

func SingingAction(ctx context.Context, dbconn *db.Connection) error {
	singing, err := SelectSinging(ctx, dbconn)
	if err != nil {
		return err
	}
	grip.Info("Singing:")
	if err := infra.WriteTabbedKVs(os.Stdout, infra.IterStruct(singing)); err != nil {
		return err
	}
	grip.Info("Lessons:")
	var ec erc.Collector
	var mb mdwn.Builder
	mb.NewTable(
		mdwn.Column{Name: "Lesson", RightAlign: true},
		mdwn.Column{Name: "Leader"},
		mdwn.Column{Name: "Song"},
		mdwn.Column{Name: "Key"},
		mdwn.Column{Name: "Title"},
	).Extend(irt.Convert(erc.HandleAll(dbconn.SingingLessons(ctx, singing.SingingName), ec.Push), func(s models.SingingLessionInfo) []string {
		return []string{strconv.Itoa(s.LessonID), s.SingerName, s.SongPageNumber, s.SongKey, s.SongName}
	})).Build()

	if !ec.Ok() || !ec.PushOk(flush(os.Stdout, &mb)) {
		return ec.Resolve()
	}
	return nil
}

func LeaderLeadHistoryAction(ctx context.Context, dbconn *db.Connection, singer string) error {
	singer, err := interactivelyResolveSingerName(ctx, dbconn, singer)
	if err != nil {
		return err
	}
	grip.Infof("lead history for: %s", singer)

	var ec erc.Collector
	var mb mdwn.Builder
	mb.NewTable(
		mdwn.Column{Name: "Date"},
		mdwn.Column{Name: "Singing"},
		mdwn.Column{Name: "Song"},
		mdwn.Column{Name: "Page"},
		mdwn.Column{Name: "Key"},
	).Extend(irt.Convert(erc.HandleAll(dbconn.LeaderLeadHistory(ctx, singer, 50000), ec.Push), func(row models.LessonInfo) []string {
		return []string{row.SingingDate.String(), strings.ReplaceAll(row.SingingName, "\\n", "; "), row.SongName, row.SongPageNumber, row.SongKey}
	})).Build()

	if !ec.Ok() || !ec.PushOk(flush(os.Stdout, &mb)) {
		return ec.Resolve()
	}
	return nil
}

func LeaderSingingsAttendedAction(ctx context.Context, dbconn *db.Connection, singer string) error {
	singer, err := interactivelyResolveSingerName(ctx, dbconn, singer)
	if err != nil {
		return err
	}
	grip.Infof("singings attended by: %s", singer)

	var ec erc.Collector
	var mb mdwn.Builder
	mb.NewTable(
		mdwn.Column{Name: "Date"},
		mdwn.Column{Name: "Singing"},
		mdwn.Column{Name: "State"},
		mdwn.Column{Name: "City"},
		mdwn.Column{Name: "Led", RightAlign: true},
		mdwn.Column{Name: "Leaders", RightAlign: true},
	).Extend(irt.Convert(erc.HandleAll(dbconn.LeaderSingingsAttended(ctx, singer, 0), ec.Push), func(row models.LeaderSingingAttendance) []string {
		return []string{row.SingingDate.String(), strings.ReplaceAll(row.SingingName, "\\n", "; "), row.SingingState, row.SingingCity, fmt.Sprint(row.LeaderLeadCount), fmt.Sprint(row.NumberOfLeaders)}
	})).Build()

	if !ec.Ok() || !ec.PushOk(flush(os.Stdout, &mb)) {
		return ec.Resolve()
	}
	return nil
}

func SingingBuddiesAction(ctx context.Context, dbconn *db.Connection, singer string) error {
	singer, err := interactivelyResolveSingerName(ctx, dbconn, singer)
	if err != nil {
		return err
	}

	var ec erc.Collector
	grip.Infof("singing buddies for %q", singer)
	var mb mdwn.Builder
	mb.KVTable(
		irt.MakeKV("Name", "Shared Singings"),
		irt.Convert2(irt.KVsplit(erc.HandleAll(dbconn.SingingBuddies(ctx, singer, 20), ec.Push)), func(k string, v int) (string, string) {
			return k, strconv.Itoa(v)
		}),
	)
	if !ec.Ok() || !ec.PushOk(flush(os.Stdout, &mb)) {
		return ec.Resolve()
	}
	return nil
}

func SingingStrangersAction(ctx context.Context, dbconn *db.Connection, singer string) error {
	singer, err := interactivelyResolveSingerName(ctx, dbconn, singer)
	if err != nil {
		return err
	}

	var ec erc.Collector
	grip.Infof("singing strangers for %q", singer)
	var mb mdwn.Builder
	mb.KVTable(
		irt.MakeKV("Name", "Count"),
		irt.Convert2(irt.KVsplit(erc.HandleAll(dbconn.SingingStrangers(ctx, singer, 20), ec.Push)), func(k string, v int) (string, string) {
			return k, strconv.Itoa(v)
		}),
	)
	if !ec.Ok() || !ec.PushOk(flush(os.Stdout, &mb)) {
		return ec.Resolve()
	}
	return nil
}

func PopularInOnesExperienceAction(ctx context.Context, dbconn *db.Connection, singer string) error {
	singer, err := interactivelyResolveSingerName(ctx, dbconn, singer)
	if err != nil {
		return err
	}

	grip.Infof("most common songs at singings attended by %s", singer)
	return renderTopLedSongs(dbconn.PopularSongsInOnesExperience(ctx, singer, 20))
}

func NeverSungAction(ctx context.Context, dbconn *db.Connection, singer string) error {
	singer, err := interactivelyResolveSingerName(ctx, dbconn, singer)
	if err != nil {
		return err
	}

	grip.Infof("songs never sung at singing %s was present at", singer)
	return renderTopLedSongs(dbconn.NeverSung(ctx, singer))
}

func NeverLedAction(ctx context.Context, dbconn *db.Connection, singer string) error {
	singer, err := interactivelyResolveSingerName(ctx, dbconn, singer)
	if err != nil {
		return err
	}

	grip.Infof("songs never led by %s", singer)
	return renderTopLedSongs(dbconn.NeverLed(ctx, singer, 40))
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

	grip.Infof("popular songs in a specific location %v", localities)
	return renderTopLedSongs(dbconn.LocallyPopular(ctx, 20, localities...))
}

func UnfamilarHitsAction(ctx context.Context, dbconn *db.Connection, singer string) error {
	singer, err := interactivelyResolveSingerName(ctx, dbconn, singer)
	if err != nil {
		return err
	}

	grip.Infof("otherwise popular songs less-or-unfamilar to %s", singer)
	return renderTopLedSongs(dbconn.TheUnfamilarHits(ctx, singer, 20))
}

func LeaderFavoriteKeyAction(ctx context.Context, dbconn *db.Connection, singer string) error {
	singer, err := interactivelyResolveSingerName(ctx, dbconn, singer)
	if err != nil {
		return err
	}

	var ec erc.Collector
	grip.Infof("leads per key for %q", singer)
	var mb mdwn.Builder
	mb.KVTable(
		irt.MakeKV("Key", "Leads"),
		irt.Convert2(irt.KVsplit(erc.HandleAll(dbconn.LeaderFavoriteKey(ctx, singer, 20), ec.Push)), func(k string, v int) (string, string) {
			return k, strconv.Itoa(v)
		}),
	)
	if !ec.Ok() || !ec.PushOk(flush(os.Stdout, &mb)) {
		return ec.Resolve()
	}
	return nil
}

func SingersByConnectednessAction(ctx context.Context, dbconn *db.Connection) error {
	var ec erc.Collector
	grip.Info("singers ranked by connectedness ratio")
	var mb mdwn.Builder
	mb.KVTable(
		irt.MakeKV("Name", "Connectedness"),
		irt.Convert2(irt.KVsplit(erc.HandleAll(dbconn.AllLeaderConnectedness(ctx, 32), ec.Push)), func(k string, v float64) (string, string) {
			return k, fmt.Sprintf("%.4f", v)
		}),
	)
	if !ec.Ok() || !ec.PushOk(flush(os.Stdout, &mb)) {
		return ec.Resolve()
	}
	return nil
}

func LeaderFootstepsAction(ctx context.Context, dbconn *db.Connection, singer string) error {
	singer, err := interactivelyResolveSingerName(ctx, dbconn, singer)
	if err != nil {
		return err
	}

	grip.Infof("songs led by %s, ranked by the most frequent other leader of each song", singer)

	var ec erc.Collector
	var mb mdwn.Builder
	mb.NewTable(
		mdwn.Column{Name: "Song"},
		mdwn.Column{Name: "Page"},
		mdwn.Column{Name: "Key"},
		mdwn.Column{Name: "Top Leader"},
		mdwn.Column{Name: "Their Leads", RightAlign: true},
		mdwn.Column{Name: "Last Year", RightAlign: true},
		mdwn.Column{Name: "Self Leads", RightAlign: true},
	).Extend(irt.Convert(erc.HandleAll(dbconn.LeaderFootsteps(ctx, singer, 32), ec.Push), func(row models.LeaderFootstep) []string {
		return []string{row.SongTitle, row.SongPage, row.SongKeys, row.LeaderName, fmt.Sprint(row.TheirLeadCount), fmt.Sprint(row.TheirLastLeadYear), fmt.Sprint(row.SelfLeadCount)}
	})).Build()

	if !ec.Ok() || !ec.PushOk(flush(os.Stdout, &mb)) {
		return ec.Resolve()
	}
	return nil
}

func LeadersShareOfLeadsAction(ctx context.Context, dbconn *db.Connection, input string) error {
	// input may be "Singer Name" or "Singer Name,2023,2024"
	parts := strings.SplitN(input, ",", 2)
	singer, err := interactivelyResolveSingerName(ctx, dbconn, strings.TrimSpace(parts[0]))
	if err != nil {
		return err
	}

	years, err := SelectYears(input)
	if err != nil {
		return err
	}

	grip.Infof("lead share for %q in year(s) %v", singer, years)
	v, err := dbconn.LeaderShareOfLeads(ctx, singer, 16, years...)
	if err != nil {
		return err
	}
	label := "Share of All Leads"
	if len(years) > 0 {
		label = fmt.Sprintf("Share of Leads (%v)", years)
	}
	var mb mdwn.Builder
	mb.KV("Leader", singer)
	mb.KV(label, fmt.Sprintf("%.4f%%", *v*100))

	return flush(os.Stdout, &mb)
}

func TopLeadersByLeadsAction(ctx context.Context, dbconn *db.Connection, yrs string) error {
	years, err := SelectYears(yrs)
	if err != nil {
		return err
	}

	grip.Infof("leaders by total leads in year(s) %v", years)

	var ec erc.Collector
	var mb mdwn.Builder
	pos := 0
	mb.NewTable(
		mdwn.Column{Name: "#", RightAlign: true},
		mdwn.Column{Name: "Name"},
		mdwn.Column{Name: "Leads", RightAlign: true},
		mdwn.Column{Name: "Last Year", RightAlign: true},
		mdwn.Column{Name: "%", RightAlign: true},
		mdwn.Column{Name: "Running Total %", RightAlign: true},
	).Extend(irt.Convert(erc.HandleAll(dbconn.TopLeadersByLeads(ctx, 40, years...), ec.Push), func(row models.LeaderLeadCount) []string {
		pos++
		return []string{strconv.Itoa(pos), row.Name, strconv.Itoa(row.Count), strconv.Itoa(row.LastLeadYear), fmt.Sprintf("%.2f%%", row.Percentage*100), fmt.Sprintf("%.2f%%", row.RunningTotal*100)}
	})).Build()
	if !ec.Ok() || !ec.PushOk(flush(os.Stdout, &mb)) {
		return ec.Resolve()
	}
	return nil
}

func NewLeadersByYearAction(ctx context.Context, dbconn *db.Connection, arg string) error {
	years, err := SelectYears(arg)
	if err != nil {
		return err
	}
	year := time.Now().Year()
	if len(years) > 0 && years[0] > 0 {
		year = years[0]
	}
	grip.Infof("debut leaders in %d", year)
	return renderLeaderCounts(dbconn.NewLeadersByYear(ctx, year, 40))
}

func SongsByKeyAction(ctx context.Context, dbconn *db.Connection, yrs string) error {
	years, err := SelectYears(yrs)
	if err != nil {
		return err
	}

	grip.Infof("lessons by key in year(s) %v", years)

	var ec erc.Collector
	var mb mdwn.Builder
	mb.NewTable(
		mdwn.Column{Name: "Key"},
		mdwn.Column{Name: "Count", RightAlign: true},
		mdwn.Column{Name: "Percentage", RightAlign: true},
	).Extend(irt.Convert(erc.HandleAll(dbconn.SongsByKey(ctx, years...), ec.Push), func(row models.LeaderSongRank) []string {
		return []string{row.Key, row.NumLeads, fmt.Sprintf("%.1f%%", row.Ratio*100)}
	})).Build()

	if !ec.Ok() || !ec.PushOk(flush(os.Stdout, &mb)) {
		return ec.Resolve()
	}
	return nil
}

func LeadersByTop20LeadsAction(ctx context.Context, dbconn *db.Connection, _ string) error {
	grip.Info("singers ordered by number of top-20 leads")
	return renderLeaderCounts(dbconn.LeadersByTop20Leads(ctx, 40))
}

func LeaderSingingsPerYearAction(ctx context.Context, dbconn *db.Connection, singer string) error {
	singer, err := interactivelyResolveSingerName(ctx, dbconn, singer)
	if err != nil {
		return err
	}

	var ec erc.Collector
	grip.Infof("singings per year for %q", singer)
	var mb mdwn.Builder
	mb.KVTable(
		irt.MakeKV("Year", "Singings"),
		irt.Convert2(irt.KVsplit(erc.HandleAll(dbconn.LeaderSingingsPerYear(ctx, singer), ec.Push)), func(k string, v int) (string, string) {
			return k, strconv.Itoa(v)
		}),
	)
	if !ec.Ok() || !ec.PushOk(flush(os.Stdout, &mb)) {
		return ec.Resolve()
	}
	return nil
}

func LeadersByKeyAction(ctx context.Context, dbconn *db.Connection, key string) error {
	var err error
	key, err = SelectKey(ctx, dbconn, key)
	if err != nil {
		return err
	}

	grip.Infof("leaders by number of leads in key %q", key)
	return renderLeaderCounts(dbconn.LeadersByKey(ctx, key, 40))
}

func PopularSongsByKeyAction(ctx context.Context, dbconn *db.Connection, key string) error {
	var err error
	key, err = SelectKey(ctx, dbconn, key)
	if err != nil {
		return err
	}

	grip.Infof("popular songs in key %q", key)
	return renderTopLedSongs(dbconn.PopularSongsByKey(ctx, key, 40))
}

func PopularInYearsAction(ctx context.Context, dbconn *db.Connection, yrs string) error {
	years, err := SelectYears(yrs)
	if err != nil {
		return err
	}

	grip.Infof("songs by popularity in year(s) %v", years)
	return renderTopLedSongs(dbconn.GloballyPopularForYears(ctx, 20, years...))
}
