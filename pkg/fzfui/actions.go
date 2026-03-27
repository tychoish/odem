package fzfui

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cheynewallace/tabby"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/grip"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/infra"
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
	if ec.Ok() {
		grip.Infoln("song info for", s.PageNum)
		ec.Push(infra.WriteTabbedKVs(os.Stdout, infra.IterStruct(s)))
		ec.Push(infra.Write(os.Stdout, []byte{'\n'}))
		grip.Infoln("top leaders of", s.PageNum)
		ec.Push(renderTopLeaders(conn.TopLeadersOfSong(ctx, s.PageNum, 20)))
	}

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
	table := tabby.New()
	table.AddHeader("Lesson", "Leader", "Song", "Key", "Title")
	for s := range erc.HandleAll(dbconn.SingingLessons(ctx, singing.SingingName), ec.Push) {
		table.AddLine(s.LessonID, s.SingerName, s.SongPageNumber, s.SongKey, s.SongName)
	}
	table.Print()
	return ec.Resolve()
}

func singerBuddiesAction(ctx context.Context, dbconn *db.Connection, singer string) error {
	singer, err := interactivelyResolveSingerName(ctx, dbconn, singer)
	if err != nil {
		return err
	}

	var ec erc.Collector
	table := tabby.New()
	grip.Infof("singing buddies for %q", singer)
	table.AddHeader("Name", "Shared Singings")
	for kv := range erc.HandleAll(dbconn.SingingBuddies(ctx, singer, 40), ec.Push) {
		table.AddLine(kv.Key, kv.Value)
	}
	if ec.Ok() {
		table.Print()
	}
	return ec.Resolve()
}

func singerStrangersAction(ctx context.Context, dbconn *db.Connection, singer string) error {
	singer, err := interactivelyResolveSingerName(ctx, dbconn, singer)
	if err != nil {
		return err
	}

	var ec erc.Collector
	table := tabby.New()
	grip.Infof("singing strangers for %q", singer)
	table.AddHeader("Name", "Count")
	for name, count := range irt.KVsplit(erc.HandleAll(dbconn.SingingStrangers(ctx, singer, 40), ec.Push)) {
		table.AddLine(name, count)
	}
	if ec.Ok() {
		table.Print()
	}
	return ec.Resolve()
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

func popularInYearsAction(ctx context.Context, dbconn *db.Connection, yrs string) error {
	var years []int
	var err error
	if yrs != "" {
		years, err = erc.FromIteratorAll(
			irt.With2(
				irt.Slice(strings.Split(yrs, ",")),
				strconv.Atoi,
			),
		)
	}
	if len(years) == 0 {
		currentYear := time.Now().Year()

		years, err = erc.FromIteratorAll(infra.NewFuzzySearch[int](
			irt.Chain(irt.Args(
				irt.While(irt.MonotonicFrom(1995), func(v int) bool { return v < currentYear }),
				irt.While(irt.MonotonicFrom(-1*currentYear), func(v int) bool { return v < -1995 }),
			)),
		).Find("years"))
	}
	if err != nil {
		return err
	}

	grip.Infof("songs by popularity in year(s) %v", years)
	return renderTopLedSongs(dbconn.GloballyPopularForYears(ctx, years...))
}
