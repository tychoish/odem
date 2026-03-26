package fzfui

import (
	"context"
	"iter"

	"github.com/cheynewallace/tabby"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/grip"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/models"
)

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
	return ec.Resolve()
}
