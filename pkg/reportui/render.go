package reportui

import (
	"iter"
	"strconv"

	"github.com/tychoish/fun/irt"
	"github.com/tychoish/odem/pkg/mdwn"
	"github.com/tychoish/odem/pkg/models"
)

func writeSongTable(mb *mdwn.Builder, seq iter.Seq[models.LeaderSongRank]) {
	mb.NewTable(
		mdwn.Column{Name: "Count", RightAlign: true},
		mdwn.Column{Name: "Page"},
		mdwn.Column{Name: "Title"},
		mdwn.Column{Name: "Key"},
	).Extend(irt.Convert(seq, asRows)).Build()

	mb.Line()
}

func writeLeaderCountTable(mb *mdwn.Builder, countColName string, seq iter.Seq[models.LeaderSongRank]) {
	mb.NewTable(
		mdwn.Column{Name: "Name"},
		mdwn.Column{Name: countColName, RightAlign: true},
	).Extend(irt.Convert(seq, func(row models.LeaderSongRank) []string {
		return []string{row.Leader, row.NumLeads}
	})).Build()

	mb.Line()
}

func writeLeaderFootstepTable(mb *mdwn.Builder, seq iter.Seq[models.LeaderFootstep]) {
	mb.NewTable(
		mdwn.Column{Name: "Song"},
		mdwn.Column{Name: "Page"},
		mdwn.Column{Name: "Key"},
		mdwn.Column{Name: "Top Leader"},
		mdwn.Column{Name: "Their Leads", RightAlign: true},
		mdwn.Column{Name: "Last Year", RightAlign: true},
		mdwn.Column{Name: "Self Leads", RightAlign: true},
	).Extend(irt.Convert(seq, func(row models.LeaderFootstep) []string {
		return []string{
			row.SongTitle,
			row.SongPage,
			row.SongKeys,
			row.LeaderName,
			strconv.Itoa(row.TheirLeadCount),
			strconv.Itoa(row.TheirLastLeadYear),
			strconv.Itoa(row.SelfLeadCount),
		}
	})).Build()

	mb.Line()
}
