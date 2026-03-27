package fzfui

import (
	"iter"

	"github.com/cheynewallace/tabby"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/odem/pkg/models"
)

func renderTopLeaders(seq iter.Seq2[models.LeaderOfSongInfo, error]) error {
	table := tabby.New()
	table.AddHeader("Name", "Count", "Led Last Year", "Years Active")
	var ec erc.Collector
	for leader := range erc.Handle(seq, ec.Push) {
		table.AddLine(leader.Name, leader.Count, leader.LedInLastYear, leader.NumYears)
	}
	if ec.Ok() {
		table.Print()
	}
	return ec.Resolve()
}

func renderTopLedSongs(seq iter.Seq2[models.LeaderSongRank, error]) error {
	table := tabby.New()
	table.AddHeader("Count", "Song", "Title", "Key")
	var ec erc.Collector
	for song := range erc.Handle(seq, ec.Push) {
		table.AddLine(song.NumLeads, song.PageNum, song.SongTitle, song.Key)
	}
	if ec.Ok() {
		table.Print()
	}
	return ec.Resolve()
}
