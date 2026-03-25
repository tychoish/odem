package ep

import (
	"context"
	"fmt"
	"iter"
	"os"
	"strings"

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
				With(infra.DBOperationSpec(locallyPopularAction).Add),
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
	MinutesAppOpLocallyPopular
	MinutesAppOpRetry
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
	case MinutesAppOpLocallyPopular:
		return "locally-popular"
	case MinutesAppOpRetry:
		return "retry"
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
	case "prevalent", "locally-popular", "localpop", "locally":
		return MinutesAppOpLocallyPopular
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
		case MinutesAppOpLocallyPopular:
			return locallyPopularAction(ctx, conn, strings.Join(args, " "))
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
	options := stw.Slice[string]{"leaders", "singing", "songs", "connections", "strangers", "retry", "prevalent", "exit"}

	operation, err := infra.NewFuzzySearch[string](options).FindOne("search")
	if err != nil {
		return err
	}

	grip.Debugln("dispatching", operation)
	return NewMinutesAppOperation(operation).Dispatch(selectMinutesAppAction).Handle(ctx, dbconn, args...)
}

func leaderAction(ctx context.Context, conn *db.Connection, args []string) error {
	leader := strings.Join(args, " ")
	if leader == "" {
		var err error
		leader, err = selectLeader(ctx, conn)
		if err != nil {
			return err
		}
	}

	grip.Infof("selection: %s", leader)

	return renderTopLedSongs(conn.MostLeadSongs(ctx, leader, 20))
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
	if singer == "" {
		var err error
		singer, err = selectLeader(ctx, dbconn)
		if err != nil {
			return err
		}
	}

	var ec erc.Collector
	table := tabby.New()
	grip.Infof("buddies for %q", singer)
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
	if singer == "" {
		var err error
		singer, err = selectLeader(ctx, dbconn)
		if err != nil {
			return err
		}
	}

	var ec erc.Collector
	table := tabby.New()
	grip.Infof("strangers for %q", singer)
	table.AddHeader("Name", "Count")
	for name, count := range irt.KVsplit(erc.HandleAll(dbconn.SingingStrangers(ctx, singer, 40), ec.Push)) {
		table.AddLine(name, count)
	}
	if ec.Ok() {
		table.Print()
	}
	return ec.Resolve()
}

func locallyPopularAction(ctx context.Context, dbconn *db.Connection, singer string) error {
	if singer == "" {
		var err error
		singer, err = selectLeader(ctx, dbconn)
		if err != nil {
			return err
		}
	}

	return renderTopLedSongs(dbconn.PopularSongsInOnesExperience(ctx, singer, 25))
}
