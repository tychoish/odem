package ep

import (
	"context"
	"fmt"
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
			switch operation {
			case "leaders", "leader", "singer", "singers":
				return leaderAction(ctx, conn, nil)
			case "songs", "song":
				return songAction(ctx, conn, "")
			case "singings", "conventions", "alldays", "all-day", "singing":
				return singingAction(ctx, conn)
			case "neighbors", "friends", "connections", "buddies", "buddy":
				return singerBuddiesAction(ctx, conn, "")
			case "strangers", "never-neighbors", "unknowns":
				return singerStrangersAction(ctx, conn, "")
			default:
				options := stw.Slice[string]{"leaders", "singing", "songs", "exit", "connections", "strangers"}

				op, err := infra.NewFuzzySearch[string](options).FindOne("search")
				if err != nil {
					return err
				}

				switch op {
				case "leaders":
					return leaderAction(ctx, conn, nil)
				case "songs":
					return songAction(ctx, conn, "")
				case "singing":
					return singingAction(ctx, conn)
				case "connections":
					return singerBuddiesAction(ctx, conn, "")
				case "strangers":
					return singerStrangersAction(ctx, conn, "")
				case "exit":
					grip.Info("goodbye!")
					return nil
				default:
					return ers.New("selection not found")
				}
			}
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
		)
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
	table := tabby.New()
	table.AddHeader("Count", "Song", "Title", "Key")
	var ct int
	var ec erc.Collector
	for song := range erc.HandleUntil(conn.MostLeadSongs(ctx, leader, 20), ec.Push) {
		ct++
		table.AddLine(song.NumLeads, song.PageNum, song.SongTitle, song.Key)
	}
	if ec.Ok() {
		table.Print()
	}
	grip.Infof("saw %d songs", ct)
	return ec.Resolve()
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
		ec.Push(infra.WriteTabbedKVs(os.Stdout, infra.IterStruct(s)))
		ec.Push(infra.Write(os.Stdout, []byte{'\n'}))
		ec.Push(renderTopLeaders(ctx, conn, s.PageNum))
	}

	return ec.Resolve()
}

func renderTopLeaders(ctx context.Context, conn *db.Connection, pageNum string) error {
	table := tabby.New()
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
