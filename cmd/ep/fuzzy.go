package ep

import (
	"context"
	"fmt"

	"github.com/cheynewallace/tabby"
	fzf "github.com/koki-develop/go-fzf"
	"github.com/tychoish/cmdr"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/ers"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/stw"
	"github.com/tychoish/grip"
	"github.com/tychoish/shbot/pkg/db"
	"github.com/tychoish/shbot/pkg/infra"
	"github.com/tychoish/shbot/pkg/models"
)

func Fuzzy() *cmdr.Commander {
	return cmdr.MakeCommander().
		SetName("fuzzy").
		Aliases("fzf").
		SetUsage("fuzzy commandline search").
		With(infra.DBOperationSpec(func(ctx context.Context, conn *db.Connection, operation string) error {
			switch operation {
			case "leaders", "leader", "singer":
				return leaderAction(ctx, conn, "")
			case "songs", "song":
				return songAction(ctx, conn, "")
			default:
				options := stw.Slice[string]{"leaders", "songs"}

				idx, err := erc.Must(fzf.New(
					fzf.WithPrompt("search => "),
					fzf.WithCaseSensitive(false),
					fzf.WithLimit(32), //
				)).Find(options, options.Index)
				if err != nil {
					return err
				}
				if len(idx) == 0 {
					return ers.New("no selection")
				}

				switch options.Index(idx[0]) {
				case "leaders":
					return leaderAction(ctx, conn, "")
				case "songs":
					return songAction(ctx, conn, "")
				default:
					return ers.New("selection not found")
				}
			}
		}).Add).
		Subcommanders(
			cmdr.MakeCommander().
				SetName("leaders").
				Aliases("leader").
				With(infra.DBOperationSpec(leaderAction).Add),
			cmdr.MakeCommander().
				SetName("songs").
				SetName("song").
				With(infra.DBOperationSpec(songAction).Add),
		)
}

func leaderAction(ctx context.Context, conn *db.Connection, leader string) error {
	if leader == "" {
		var err error
		leader, err = SelectLeader(ctx, conn)
		if err != nil {
			return err
		}
	}

	grip.Infof("selection: %s", leader)
	table := tabby.New()
	table.AddHeader("Count", "Song", "Title", "Key")
	var ct int
	for song, err := range conn.MostLeadSongs(ctx, leader, 20) {
		ct++
		if err != nil {
			return err
		}
		table.AddLine(song.NumLeads, song.PageNum, song.SongTitle, song.Key)
	}
	table.Print()
	grip.Infof("saw %d songs", ct)
	return nil
}

func songAction(ctx context.Context, conn *db.Connection, song string) error {
	var ec erc.Collector

	if song != "" {
		for s := range erc.HandleAll(conn.AllSongDetails(ctx), ec.Push) {
			if song == s.PageNum {
				grip.Infof("selection:\n%s---", infra.NewYAML(s).String())
				ec.Push(renderTopLeaders(ctx, conn, s.PageNum))
				return ec.Resolve()

			}
		}
	}
	if !ec.Ok() {
		return ec.Resolve()
	}

	s, err := SelectSong(ctx, conn)
	if err != nil {
		return err
	}

	grip.Infoln("selection:\n", infra.NewYAML(s).String(), "\n---")

	return renderTopLeaders(ctx, conn, s.PageNum)
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

func SelectLeader(ctx context.Context, dbconn *db.Connection) (string, error) {
	var ec erc.Collector

	names := irt.Collect(erc.HandleAll(dbconn.AllLeaderNames(ctx), ec.Push))
	if !ec.Ok() {
		return "", ec.Resolve()
	}

	idx, err := erc.Must(fzf.New(
		fzf.WithPrompt("leaders => "),
		fzf.WithCaseSensitive(false),
		fzf.WithLimit(32), //
	)).Find(names, func(in int) string { return names[in] })
	ec.Push(err)
	if !ec.Ok() {
		return "", ec.Resolve()
	}
	return names[idx[0]], nil
}

func SelectSong(ctx context.Context, dbconn *db.Connection) (*models.SongDetail, error) {
	var ec erc.Collector

	songs := irt.Collect(erc.HandleAll(dbconn.AllSongDetails(ctx), ec.Push))
	if !ec.Ok() {
		return nil, ec.Resolve()
	}
	idx, err := erc.Must(fzf.New(
		fzf.WithPrompt("songs => "),
		fzf.WithCaseSensitive(false),
		fzf.WithLimit(32), //
	)).Find(songs, func(in int) string { return fmt.Sprintf("pg %s -- %s", songs[in].PageNum, songs[in].SongTitle) })
	ec.Push(err)
	if !ec.Ok() {
		return nil, ec.Resolve()
	}
	return &songs[idx[0]], nil
}
