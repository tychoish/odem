package ep

import (
	"context"
	"fmt"
	"iter"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cheynewallace/tabby"
	"github.com/tychoish/cmdr"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/ers"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/stw"
	"github.com/tychoish/grip"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/infra"
	"github.com/tychoish/odem/pkg/models"
)

func Fuzzy() *cmdr.Commander {
	return cmdr.MakeCommander().
		SetName("fuzzy").
		Aliases("fzf").
		SetUsage("fuzzy commandline search").
		With(infra.DBOperationSpec(func(ctx context.Context, conn *db.Connection, operation string) error {
			return NewMinutesAppOperation("retry").Dispatch(selectMinutesAppAction).Handle(ctx, conn, "")
		}).Add).
		Subcommanders(
			cmdr.MakeCommander().
				SetName("leaders").
				Aliases("leader", "singer", "singers").
				SetUsage("search for a leader").
				With(infra.MakeDBOperationSpec("name", leaderAction).Add),
			cmdr.MakeCommander().
				SetName("singing").
				Aliases("singings", "allday").
				SetUsage("search for a specific singing").
				With(infra.SimpleDBOperationSpec(singingAction).Add),
			cmdr.MakeCommander().
				SetName("connections").
				Aliases("neighbors", "friends", "buddy", "buddies").
				SetUsage("find the people that you've sung with the most").
				With(infra.MakeDBOperationSpec("name", singerBuddiesAction).Add),
			cmdr.MakeCommander().
				SetName("strangers").
				SetUsage("find the people that you've never sung with, surprisingly").
				With(infra.MakeDBOperationSpec("name", singerStrangersAction).Add),
			cmdr.MakeCommander().
				SetName("songs").
				Aliases("song").
				SetUsage("find out more about a song and it's top leaders.").
				With(infra.DBOperationSpec(songAction).Add),
			cmdr.MakeCommander().
				SetName("prevalent").
				SetUsage("find out what the top songs are at singing's you've been to.").
				With(infra.DBOperationSpec(popularInOnesExperienceAction).Add),
			cmdr.MakeCommander().
				SetName("never-sung").
				SetUsage("find songs that have never been performed at a singing you attended.").
				With(infra.DBOperationSpec(neverSungAction).Add),
			cmdr.MakeCommander().
				SetName("never-led").
				SetUsage("find songs from the book you have never led.").
				With(infra.MakeDBOperationSpec("name", neverLedAction).Add),
			cmdr.MakeCommander().
				SetName("locally-popular").
				SetUsage("find out what the top songs that are popular in a specific locality.").
				With(infra.SimpleDBOperationSpec(locallyPopularAction).Add),
			cmdr.MakeCommander().
				SetName("popular-for-years").
				SetUsage("find out what the top songs that are popular at a specific year.").
				With(infra.DBOperationSpec(popularInYearsAction).Add),
		)
}

type MinutesAppOperationHandler func(context.Context, *db.Connection, ...string) error

func (maoh MinutesAppOperationHandler) Handle(ctx context.Context, conn *db.Connection, args ...string) error {
	return maoh(ctx, conn, args...)
}

type MinutesAppOperation int

func (mao MinutesAppOperation) Validate() error {
	return ers.Whenf(mao >= MinutesAppOpInvalid || mao <= MinutesAppOpUnknown, "invalid OperationID %s %d", mao, mao)
}

const (
	MinutesAppOpUnknown MinutesAppOperation = iota
	MinutesAppOpLeaders
	MinutesAppOpSongs
	MinutesAppOpSingings
	MinutesAppOpBuddies
	MinutesAppOpStrangers
	MinutesAppOpPopularInOnesExperience
	MinutesAppOpPopularInYears
	MinutesAppOpLocallyPopular
	MinutesAppOpRetry
	MinutesAppOpNeverSung
	MinutesAppOpNeverLed
	MinutesAppOpUnfamilarHits
	MinutesAppOpInvalid
	MinutesAppOpExit = 181
)

func (mao MinutesAppOperation) String() string {
	switch mao {
	case MinutesAppOpUnknown:
		return "unknown"
	case MinutesAppOpLeaders:
		return "leaders"
	case MinutesAppOpSongs:
		return "songs"
	case MinutesAppOpSingings:
		return "singings"
	case MinutesAppOpBuddies:
		return "buddies"
	case MinutesAppOpStrangers:
		return "strangers"
	case MinutesAppOpPopularInOnesExperience:
		return "popular-in-ones-experience"
	case MinutesAppOpLocallyPopular:
		return "locally-popular"
	case MinutesAppOpPopularInYears:
		return "popular-for-years"
	case MinutesAppOpNeverSung:
		return "never-sung"
	case MinutesAppOpNeverLed:
		return "never-led"
	case MinutesAppOpRetry:
		return "retry"
	case MinutesAppOpUnfamilarHits:
		return "unfamilar-hits"
	case MinutesAppOpExit:
		return "exit<181>"
	case MinutesAppOpInvalid:
		fallthrough
	default:
		return fmt.Sprintf("invalid<%d>", mao)
	}
}

func NewMinutesAppOperation(arg string) MinutesAppOperation {
	switch arg {
	case "leaders", "leader", "singer", "person":
		return MinutesAppOpLeaders
	case "song", "tune", "hymn", "songs":
		return MinutesAppOpSongs
	case "singing", "allday", "convention":
		return MinutesAppOpSingings
	case "buddies", "connections", "neighbors":
		return MinutesAppOpBuddies
	case "strangers", "enemies", "never-neighbors":
		return MinutesAppOpStrangers
	case "exit", "return", "abort":
		return MinutesAppOpExit
	case "retry", "restart":
		return MinutesAppOpRetry
	case "prevalent", "popular-in-ones-experience":
		return MinutesAppOpPopularInOnesExperience
	case "never-sung", "unknown":
		return MinutesAppOpNeverSung
	case "never-led", "neverled":
		return MinutesAppOpNeverLed
	case "locally-popular", "localpop", "locally":
		return MinutesAppOpLocallyPopular
	case "popular-for-years", "popular-in-years":
		return MinutesAppOpPopularInYears
	default:
		return MinutesAppOpInvalid
	}
}

func (mao MinutesAppOperation) Dispatch(restart MinutesAppOperationHandler) MinutesAppOperationHandler {
	return func(ctx context.Context, conn *db.Connection, args ...string) error {
		switch mao {
		case MinutesAppOpLeaders:
			return leaderAction(ctx, conn, args)
		case MinutesAppOpSongs:
			return songAction(ctx, conn, strings.Join(args, " "))
		case MinutesAppOpSingings:
			return singingAction(ctx, conn)
		case MinutesAppOpBuddies:
			return singerBuddiesAction(ctx, conn, strings.Join(args, " "))
		case MinutesAppOpStrangers:
			return singerStrangersAction(ctx, conn, "")
		case MinutesAppOpPopularInOnesExperience:
			return popularInOnesExperienceAction(ctx, conn, strings.Join(args, " "))
		case MinutesAppOpNeverSung:
			return neverSungAction(ctx, conn, strings.Join(args, " "))
		case MinutesAppOpNeverLed:
			return neverLedAction(ctx, conn, strings.Join(args, " "))
		case MinutesAppOpLocallyPopular:
			return locallyPopularAction(ctx, conn)
		case MinutesAppOpPopularInYears:
			return popularInYearsAction(ctx, conn, strings.Join(args, ","))
		case MinutesAppOpExit:
			grip.Info("goodbye!")
			return nil
		case MinutesAppOpRetry:
			return restart(ctx, conn, args...)
		case MinutesAppOpInvalid, MinutesAppOpUnknown:
			return ers.New("invalid/undefined operation")
		default:
			return fmt.Errorf("unknown operation at %d (%s)", mao, mao)
		}
	}
}

func selectMinutesAppAction(ctx context.Context, dbconn *db.Connection, args ...string) error {
	grip.Debug("selecting operation to dispatch")
	options := stw.Slice[string]{
		"leaders",
		"singing",
		"songs",
		"connections",
		"strangers",
		"retry",
		"prevalent",
		"locally-popular",
		"popular-for-years",
		"never-sung",
		"never-led",
		"exit",
	}

	operation, err := infra.NewFuzzySearch[string](options).FindOne("search")
	if err != nil {
		return err
	}

	grip.Debugln("dispatching", operation)
	return NewMinutesAppOperation(operation).Dispatch(selectMinutesAppAction).Handle(ctx, dbconn, args...)
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
		ec.Push(renderTopLeaders(ctx, conn, s.PageNum))
	}

	return ec.Resolve()
}

func renderTopLeaders(ctx context.Context, conn *db.Connection, pageNum string) error {
	table := tabby.New()
	grip.Infoln("top leader for page:", pageNum)
	table.AddHeader("Name", "Count", "Led Last Year", "Years Active")
	for leader, err := range conn.TopLeadersOfSong(ctx, pageNum, 20) {
		if err != nil {
			return err
		}
		table.AddLine(leader.Name, leader.Count, leader.LedInLastYear, leader.NumYears)
	}
	table.Print()
	return nil
}

func renderTopLedSongs(seq iter.Seq2[models.LeaderSongRank, error]) error {
	table := tabby.New()
	table.AddHeader("Count", "Song", "Title", "Key")
	var ct int
	var ec erc.Collector
	for song := range erc.Handle(seq, ec.Push) {
		ct++
		table.AddLine(song.NumLeads, song.PageNum, song.SongTitle, song.Key)
	}
	if ec.Ok() {
		table.Print()
	}
	grip.Infof("saw %d songs", ct)
	return ec.Resolve()
}

func selectLeader(ctx context.Context, dbconn *db.Connection) (string, error) {
	var ec erc.Collector

	names := irt.Collect(erc.HandleAll(dbconn.AllLeaderNames(ctx), ec.Push))
	if !ec.Ok() {
		return "", ec.Resolve()
	}

	leader, err := infra.NewFuzzySearch[string](names).FindOne("leaders")
	if !ec.PushOk(err) {
		return "", ec.Resolve()
	}

	grip.Debugln("selected leader", leader)
	return leader, nil
}

func selectSinging(ctx context.Context, dbconn *db.Connection) (*models.SingingInfo, error) {
	var ec erc.Collector

	singings := irt.Collect(erc.HandleAll(dbconn.AllSingings(ctx), ec.Push))
	singing, err := infra.NewFuzzySearch[models.SingingInfo](singings).
		WithToString(func(info models.SingingInfo) string {
			return fmt.Sprintf("%s -- %s (%s)", info.SingingDate.Time().Format("2006-01-02"), strings.Split(info.SingingName, "\\n")[0], info.SingingLocation)
		}).
		FindOne("leaders")

	if !ec.PushOk(err) || !ec.Ok() {
		return nil, ec.Resolve()
	}
	grip.Debugln("selected singing", singing.SingingName)
	return &singing, nil
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

func locallyPopularAction(ctx context.Context, dbconn *db.Connection) error {
	localities, err := erc.FromIteratorAll(infra.NewFuzzySearch[models.SingingLocality](models.AllLocalities()).Find("location"))
	if err != nil {
		return err
	}

	grip.Infof("popular songs in a specific location %v", localities)
	return renderTopLedSongs(dbconn.LocallyPopular(ctx, 32, localities...))
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

func tolocality(in string) models.SingingLocality { return models.SingingLocality(in) }
func interactivelyResolveSingerName(ctx context.Context, conn *db.Connection, singer string) (string, error) {
	if singer != "" {
		return singer, nil
	}

	singer, err := selectLeader(ctx, conn)
	if err != nil {
		return "", err
	}
	if singer == "" {
		return "", ers.New("not found")
	}

	return singer, nil
}
