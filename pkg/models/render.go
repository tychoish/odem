package models

import (
	"fmt"
	"iter"
	"strconv"
	"strings"

	"github.com/tychoish/fun/irt"
	"github.com/tychoish/odem/pkg/mdwn"
)

func WriteSongTable(mb *mdwn.Builder, seq iter.Seq[LeaderSongRank]) {
	mb.NewTable(
		mdwn.Column{Name: "Count", RightAlign: true},
		mdwn.Column{Name: "Page"},
		mdwn.Column{Name: "Title"},
		mdwn.Column{Name: "Key"},
	).Extend(irt.Convert(seq, func(row LeaderSongRank) []string { return (&row).StringFields() })).Build()

	mb.Line()
}

func WriteLeaderCountTableForCount(mb *mdwn.Builder, seq iter.Seq[LeaderSongRank]) {
	WriteLeaderCountTable(mb, "Count", seq)
}

func WriteLeaderCountTable(mb *mdwn.Builder, countColName string, seq iter.Seq[LeaderSongRank]) {
	mb.NewTable(
		mdwn.Column{Name: "Name"},
		mdwn.Column{Name: countColName, RightAlign: true},
	).Extend(irt.Convert(seq, func(row LeaderSongRank) []string {
		return []string{row.Leader, row.NumLeads}
	})).Build()

	mb.Line()
}

func WriteLeaderFootstepTable(mb *mdwn.Builder, seq iter.Seq[LeaderFootstep]) {
	mb.NewTable(
		mdwn.Column{Name: "Song"},
		mdwn.Column{Name: "Page"},
		mdwn.Column{Name: "Key"},
		mdwn.Column{Name: "Top Leader"},
		mdwn.Column{Name: "Their Leads", RightAlign: true},
		mdwn.Column{Name: "Last Year", RightAlign: true},
		mdwn.Column{Name: "Self Leads", RightAlign: true},
	).Extend(irt.Convert(seq, func(row LeaderFootstep) []string {
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

func WriteSongLeadersTable(mb *mdwn.Builder, seq iter.Seq[LeaderOfSongInfo]) {
	mb.NewTable(
		mdwn.Column{Name: "Name"},
		mdwn.Column{Name: "Count", RightAlign: true},
		mdwn.Column{Name: "Led Last Year"},
		mdwn.Column{Name: "Years Active", RightAlign: true},
	).Extend(irt.Convert(seq, func(l LeaderOfSongInfo) []string {
		return []string{l.Name, strconv.Itoa(l.Count), strconv.FormatBool(l.LedInLastYear), strconv.Itoa(l.NumYears)}
	})).Build()

	mb.Line()
}

func WriteSingingLessonsTable(mb *mdwn.Builder, seq iter.Seq[SingingLessionInfo]) {
	mb.NewTable(
		mdwn.Column{Name: "Lesson", RightAlign: true},
		mdwn.Column{Name: "Leader"},
		mdwn.Column{Name: "Song"},
		mdwn.Column{Name: "Key"},
		mdwn.Column{Name: "Title"},
	).Extend(irt.Convert(seq, func(s SingingLessionInfo) []string {
		return []string{strconv.Itoa(s.LessonID), s.SingerName, s.SongPageNumber, s.SongKey, s.SongName}
	})).Build()

	mb.Line()
}

func WriteLessonTable(mb *mdwn.Builder, seq iter.Seq[LessonInfo]) {
	mb.NewTable(
		mdwn.Column{Name: "Date"},
		mdwn.Column{Name: "Singing"},
		mdwn.Column{Name: "Song"},
		mdwn.Column{Name: "Page"},
		mdwn.Column{Name: "Key"},
	).Extend(irt.Convert(seq, func(row LessonInfo) []string {
		return []string{row.SingingDate.String(), strings.ReplaceAll(row.SingingName, "\\n", "; "), row.SongName, row.SongPageNumber, row.SongKey}
	})).Build()

	mb.Line()
}

func WriteLeaderSingingsTable(mb *mdwn.Builder, seq iter.Seq[LeaderSingingAttendance]) {
	mb.NewTable(
		mdwn.Column{Name: "Date"},
		mdwn.Column{Name: "Singing"},
		mdwn.Column{Name: "State"},
		mdwn.Column{Name: "City"},
		mdwn.Column{Name: "Led", RightAlign: true},
		mdwn.Column{Name: "Leaders", RightAlign: true},
	).Extend(irt.Convert(seq, func(row LeaderSingingAttendance) []string {
		return []string{row.SingingDate.String(), strings.ReplaceAll(row.SingingName, "\\n", "; "), row.SingingState, row.SingingCity, strconv.Itoa(row.LeaderLeadCount), strconv.Itoa(row.NumberOfLeaders)}
	})).Build()

	mb.Line()
}

func WriteTopLeadersTable(mb *mdwn.Builder, seq iter.Seq[LeaderLeadCount]) {
	var pos int
	mb.NewTable(
		mdwn.Column{Name: "#", RightAlign: true},
		mdwn.Column{Name: "Name"},
		mdwn.Column{Name: "Leads", RightAlign: true},
		mdwn.Column{Name: "Last Year", RightAlign: true},
		mdwn.Column{Name: "%", RightAlign: true},
		mdwn.Column{Name: "Running Total %", RightAlign: true},
	).Extend(irt.Convert(seq, func(row LeaderLeadCount) []string {
		pos++
		return []string{strconv.Itoa(pos), row.Name, strconv.Itoa(row.Count), strconv.Itoa(row.LastLeadYear), fmt.Sprintf("%.2f%%", row.Percentage*100), fmt.Sprintf("%.2f%%", row.RunningTotal*100)}
	})).Build()

	mb.Line()
}

func WriteSongsByKeyTable(mb *mdwn.Builder, seq iter.Seq[LeaderSongRank]) {
	mb.NewTable(
		mdwn.Column{Name: "Key"},
		mdwn.Column{Name: "Count", RightAlign: true},
		mdwn.Column{Name: "Percentage", RightAlign: true},
	).Extend(irt.Convert(seq, func(row LeaderSongRank) []string {
		return []string{row.Key, row.NumLeads, fmt.Sprintf("%.1f%%", row.Ratio*100)}
	})).Build()

	mb.Line()
}
