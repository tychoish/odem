package fzfui

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/grip"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/infra"
	"github.com/tychoish/odem/pkg/mdwn"
	"github.com/tychoish/odem/pkg/models"
)

func selectMinutesAppAction(ctx context.Context, dbconn *db.Connection, arg string) error {
	grip.Debug("selecting operation to dispatch")

	operation := NewMinutesAppOperation(arg)
	if !operation.Ok() {
		var err error
		operation, err = infra.NewFuzzySearch[MinutesAppOperation](AllMinutesAppOperations()).FindOne("odem operation")
		if err != nil {
			return err
		}
	}

	grip.Debugln("dispatching", operation)
	return operation.Dispatch().Handle(ctx, dbconn)
}

func leaderAction(ctx context.Context, conn *db.Connection, args []string) error {
	singer, err := interactivelyResolveSingerName(ctx, conn, strings.Join(args, " "))
	if err != nil {
		return err
	}

	grip.Infof("songs led by: %s", singer)

	return renderTopLedSongs(conn.MostLeadSongs(ctx, singer, -20))
}

func songAction(ctx context.Context, conn *db.Connection, song string) error {
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
		}).FindOne("songs")

		ec.Push(err)
		s = &sg
	}

	ec.When(s == nil, "no matching song found")
	if !ec.Ok() {
		return ec.Resolve()
	}

	grip.Infoln("song info for", s.PageNum)

	// TODO convert this to use the standard KV table.
	ec.Push(infra.WriteTabbedKVs(os.Stdout, infra.IterStruct(s)))
	ec.Push(infra.Write(os.Stdout, []byte{'\n'}))
	grip.Infoln("top leaders of", s.PageNum)
	ec.Push(renderTopLeaders(conn.TopLeadersOfSong(ctx, s.PageNum, 20)))

	return ec.Resolve()
}

func singingAction(ctx context.Context, dbconn *db.Connection) error {
	singing, err := selectSinging(ctx, dbconn)
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

func singerBuddiesAction(ctx context.Context, dbconn *db.Connection, singer string) error {
	singer, err := interactivelyResolveSingerName(ctx, dbconn, singer)
	if err != nil {
		return err
	}

	var ec erc.Collector
	grip.Infof("singing buddies for %q", singer)
	var mb mdwn.Builder
	mb.KVTable(
		irt.MakeKV("Name", "Shared Singings"),
		irt.Convert2(irt.KVsplit(erc.HandleAll(dbconn.SingingBuddies(ctx, singer, 32), ec.Push)), func(k string, v int) (string, string) {
			return k, strconv.Itoa(v)
		}),
	)
	if !ec.Ok() || !ec.PushOk(flush(os.Stdout, &mb)) {
		return ec.Resolve()
	}
	return nil
}

func singerStrangersAction(ctx context.Context, dbconn *db.Connection, singer string) error {
	singer, err := interactivelyResolveSingerName(ctx, dbconn, singer)
	if err != nil {
		return err
	}

	var ec erc.Collector
	grip.Infof("singing strangers for %q", singer)
	var mb mdwn.Builder
	mb.KVTable(
		irt.MakeKV("Name", "Count"),
		irt.Convert2(irt.KVsplit(erc.HandleAll(dbconn.SingingStrangers(ctx, singer, 32), ec.Push)), func(k string, v int) (string, string) {
			return k, strconv.Itoa(v)
		}),
	)
	if !ec.Ok() || !ec.PushOk(flush(os.Stdout, &mb)) {
		return ec.Resolve()
	}
	return nil
}

func popularInOnesExperienceAction(ctx context.Context, dbconn *db.Connection, singer string) error {
	singer, err := interactivelyResolveSingerName(ctx, dbconn, singer)
	if err != nil {
		return err
	}

	grip.Infof("most common songs at singings attended by %s", singer)
	return renderTopLedSongs(dbconn.PopularSongsInOnesExperience(ctx, singer, 25))
}

func neverSungAction(ctx context.Context, dbconn *db.Connection, singer string) error {
	singer, err := interactivelyResolveSingerName(ctx, dbconn, singer)
	if err != nil {
		return err
	}

	grip.Infof("songs never sung at singing %s was present at", singer)
	return renderTopLedSongs(dbconn.NeverSung(ctx, singer))
}

func neverLedAction(ctx context.Context, dbconn *db.Connection, singer string) error {
	singer, err := interactivelyResolveSingerName(ctx, dbconn, singer)
	if err != nil {
		return err
	}

	grip.Infof("songs never led by %s", singer)
	return renderTopLedSongs(dbconn.NeverLed(ctx, singer))
}

func locallyPopularAction(ctx context.Context, dbconn *db.Connection, localities ...models.SingingLocality) error {
	if len(localities) == 0 {
		var err error
		localities, err = erc.FromIteratorAll(infra.NewFuzzySearch[models.SingingLocality](models.AllLocalities()).Find("location"))
		if err != nil {
			return err
		}
	}

	grip.Infof("popular songs in a specific location %v", localities)
	return renderTopLedSongs(dbconn.LocallyPopular(ctx, 32, localities...))
}

func unfamilarHitsAction(ctx context.Context, dbconn *db.Connection, singer string) error {
	singer, err := interactivelyResolveSingerName(ctx, dbconn, singer)
	if err != nil {
		return err
	}

	grip.Infof("otherwise popular songs less-or-unfamilar to %s", singer)
	return renderTopLedSongs(dbconn.TheUnfamilarHits(ctx, singer, 32))
}

func singersByConnectednessAction(ctx context.Context, dbconn *db.Connection) error {
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

func leaderFootstepsAction(ctx context.Context, dbconn *db.Connection, singer string) error {
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

func leaderShareOfLeadsAction(ctx context.Context, dbconn *db.Connection, input string) error {
	// input may be "Singer Name" or "Singer Name,2023,2024"
	parts := strings.SplitN(input, ",", 2)
	singer, err := interactivelyResolveSingerName(ctx, dbconn, strings.TrimSpace(parts[0]))
	if err != nil {
		return err
	}

	years, err := selectYears(input)
	if err != nil {
		return err
	}

	grip.Infof("lead share for %q in year(s) %v", singer, years)
	v, err := dbconn.LeaderShareOfLeads(ctx, singer, years...)
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

func topLeadersByLeadsAction(ctx context.Context, dbconn *db.Connection, yrs string) error {
	var years []int
	if yrs != "" {
		var err error
		years, err = erc.FromIteratorAll(irt.With2(irt.Slice(strings.Split(yrs, ",")), strconv.Atoi))
		if err != nil {
			return err
		}
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

func popularInYearsAction(ctx context.Context, dbconn *db.Connection, yrs string) error {
	years, err := selectYears(yrs)
	if err != nil {
		return err
	}

	grip.Infof("songs by popularity in year(s) %v", years)
	return renderTopLedSongs(dbconn.GloballyPopularForYears(ctx, years...))
}

func flush(wr io.Writer, payload io.WriterTo) (err error) { _, err = payload.WriteTo(wr); return }
