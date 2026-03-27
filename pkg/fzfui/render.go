package fzfui

import (
	"fmt"
	"iter"
	"os"

	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/odem/pkg/mdwn"
	"github.com/tychoish/odem/pkg/models"
)

func renderTopLeaders(seq iter.Seq2[models.LeaderOfSongInfo, error]) error {
	var ec erc.Collector
	var mb mdwn.Builder
	mb.NewTable(
		mdwn.Column{Name: "Name"},
		mdwn.Column{Name: "Count", RightAlign: true},
		mdwn.Column{Name: "Led Last Year"},
		mdwn.Column{Name: "Years Active", RightAlign: true},
	).Extend(irt.Convert(erc.HandleAll(seq, ec.Push), func(l models.LeaderOfSongInfo) []string {
		return []string{l.Name, fmt.Sprint(l.Count), fmt.Sprint(l.LedInLastYear), fmt.Sprint(l.NumYears)}
	})).Build()

	if ec.Ok() {
		_, err := mb.WriteTo(os.Stdout)
		ec.Push(err)
	}
	return ec.Resolve()
}

func renderTopLedSongs(seq iter.Seq2[models.LeaderSongRank, error]) error {
	var ec erc.Collector
	var mb mdwn.Builder
	mb.NewTable(
		mdwn.Column{Name: "Count", RightAlign: true},
		mdwn.Column{Name: "Song"},
		mdwn.Column{Name: "Title"},
		mdwn.Column{Name: "Key"},
	).Extend(irt.Convert(erc.HandleAll(seq, ec.Push), func(s models.LeaderSongRank) []string {
		return []string{s.NumLeads, s.PageNum, s.SongTitle, s.Key}
	})).Build()
	if ec.Ok() {
		_, err := mb.WriteTo(os.Stdout)
		ec.Push(err)
	}
	return ec.Resolve()
}
