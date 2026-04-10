package models

import (
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/tychoish/odem/pkg/mdwn"
)

type LeaderSongRank struct {
	Rank      int     `db:"rank"`
	Leader    string  `db:"name"`
	NumLeads  string  `db:"count"`
	PageNum   string  `db:"song_page"`
	SongTitle string  `db:"song_title"`
	Key       string  `db:"song_keys"`
	Ratio     float64 `db:"ratio"` // ratio of this song to total number of leads.
}

func (lsr LeaderSongRank) ColumnNames() []mdwn.Column {
	return []mdwn.Column{
		{Name: "Count", RightAlign: true},
		{Name: "Page"},
		{Name: "Title"},
		{Name: "Key"},
	}
}

func (lsr LeaderSongRank) RowValues() []string {
	return []string{
		lsr.NumLeads,
		lsr.PageNum,
		lsr.SongTitle,
		lsr.Key,
	}
}

type LeaderRankingFor struct {
	Name string
	LeaderSongRank
}

func WrapLeaderSongRank(name string) func(LeaderSongRank) LeaderRankingFor {
	return func(lsr LeaderSongRank) LeaderRankingFor {
		return LeaderRankingFor{Name: name, LeaderSongRank: lsr}
	}
}

func (lrf LeaderRankingFor) ColumnNames() []mdwn.Column {
	return []mdwn.Column{
		{Name: "Name"},
		{Name: lrf.Name, RightAlign: true},
	}
}

func (lrf LeaderRankingFor) RowValues() []string { return []string{lrf.Leader, lrf.NumLeads} }

type SongByKey struct{ LeaderSongRank }

func WrapSongByKey(lsr LeaderSongRank) SongByKey { return SongByKey{LeaderSongRank: lsr} }

func (SongByKey) ColumnNames() []mdwn.Column {
	return []mdwn.Column{
		{Name: "Key"},
		{Name: "Count", RightAlign: true},
		{Name: "Percentage", RightAlign: true},
	}
}

func (s SongByKey) RowValues() []string {
	return []string{
		s.Key,
		s.NumLeads,
		fmt.Sprintf("%.1f%%", s.Ratio*100),
	}
}

type LeaderBackground struct {
	Name      string `db:"name"`
	NumLeads  int    `db:"num_leads"`
	FirstYear int    `db:"first_year"`
	LastYear  int    `db:"last_year"`
}

type LeaderOfSongInfo struct {
	Name          string `db:"name"`
	Count         int    `db:"count"`
	NumYears      int    `db:"num_years"`        // span: last year led minus first year led
	LedInLastYear bool   `db:"led_in_last_year"` // led within one year of the most recent minutes
}

func (LeaderOfSongInfo) ColumnNames() []mdwn.Column {
	return []mdwn.Column{
		{Name: "Name"},
		{Name: "Count", RightAlign: true},
		{Name: "Led Last Year"},
		{Name: "Years Active", RightAlign: true},
	}
}

func (r LeaderOfSongInfo) RowValues() []string {
	return []string{
		r.Name,
		strconv.Itoa(r.Count),
		strconv.FormatBool(r.LedInLastYear),
		strconv.Itoa(r.NumYears),
	}
}

type LessonInfo struct {
	SingerName     string   `db:"singer_name"`
	SongPageNumber string   `db:"song_page_number"`
	SongName       string   `db:"song_name"`
	SongKey        string   `db:"song_key"`
	SingingDate    DateTime `db:"singing_date"`
	SingingName    string   `db:"singing_name"`
	SingingState   string   `db:"singing_state"`
}

func (LessonInfo) ColumnNames() []mdwn.Column {
	return []mdwn.Column{
		{Name: "Date"},
		{Name: "Singing"},
		{Name: "Song"},
		{Name: "Page"},
		{Name: "Key"},
	}
}

func (r LessonInfo) RowValues() []string {
	return []string{
		r.SingingDate.String(),
		strings.ReplaceAll(r.SingingName, "\\n", "; "),
		r.SongName,
		r.SongPageNumber,
		r.SongKey,
	}
}

type SingingLessionInfo struct {
	SequenceNumber int    `db:"sequence_number"` // nth lesson of the day
	LessonID       int    `db:"lesson_id"`
	SingerName     string `db:"singer_name"`
	SongPageNumber string `db:"song_page_number"`
	SongName       string `db:"song_name"`
	SongKey        string `db:"song_key"`
}

func (SingingLessionInfo) ColumnNames() []mdwn.Column {
	return []mdwn.Column{
		{Name: "Lesson", RightAlign: true},
		{Name: "Leader"},
		{Name: "Song"},
		{Name: "Key"},
		{Name: "Title"},
	}
}

func (r SingingLessionInfo) RowValues() []string {
	return []string{
		strconv.Itoa(r.LessonID),
		r.SingerName,
		r.SongPageNumber,
		r.SongKey,
		r.SongName,
	}
}

type LeaderLeadCount struct {
	Name         string  `db:"name"`
	Count        int     `db:"count"`
	LastLeadYear int     `db:"last_lead_year"`
	Percentage   float64 `db:"pct"`
	RunningTotal float64 `db:"running_total"`
}

func (LeaderLeadCount) ColumnNames() []mdwn.Column {
	return []mdwn.Column{
		{Name: "Name"},
		{Name: "Leads", RightAlign: true},
		{Name: "Last Year", RightAlign: true},
		{Name: "%", RightAlign: true},
		{Name: "Running Total %", RightAlign: true},
	}
}

func (r LeaderLeadCount) RowValues() []string {
	return []string{
		r.Name,
		strconv.Itoa(r.Count),
		strconv.Itoa(r.LastLeadYear),
		fmt.Sprintf("%.2f%%", r.Percentage*100),
		fmt.Sprintf("%.2f%%", r.RunningTotal*100),
	}
}

type TopLeaders struct {
	ct *atomic.Int64
	LeaderLeadCount
}

func TopLeadersWrapper(ct *atomic.Int64) func(LeaderLeadCount) TopLeaders {
	return func(llc LeaderLeadCount) TopLeaders {
		return TopLeaders{ct: ct, LeaderLeadCount: llc}
	}
}

func (TopLeaders) ColumnNames() []mdwn.Column {
	var t TopLeaders
	return append([]mdwn.Column{
		{Name: "#", RightAlign: true},
	}, t.LeaderLeadCount.ColumnNames()...)
}

func (tl TopLeaders) RowValues() []string {
	return append(
		[]string{strconv.FormatInt(tl.ct.Add(1), 10)},
		tl.LeaderLeadCount.RowValues()...,
	)
}

// LeaderSingingAttendance represents a singing a leader attended, with their
// lead count for that singing and the total number of leaders present.
type LeaderSingingAttendance struct {
	SingingName     string   `db:"singing_name"`
	SingingDate     DateTime `db:"singing_date"`
	SingingState    string   `db:"singing_state"`
	SingingCity     string   `db:"singing_city"`
	LeaderLeadCount int      `db:"leader_lead_count"`
	NumberOfLeaders int      `db:"number_of_leaders"`
}

func (LeaderSingingAttendance) ColumnNames() []mdwn.Column {
	return []mdwn.Column{
		{Name: "Date"},
		{Name: "Singing"},
		{Name: "State"},
		{Name: "City"},
		{Name: "Led", RightAlign: true},
		{Name: "Leaders", RightAlign: true},
	}
}

func (r LeaderSingingAttendance) RowValues() []string {
	return []string{
		r.SingingDate.String(),
		strings.ReplaceAll(r.SingingName, "\\n", "; "),
		r.SingingState,
		r.SingingCity,
		strconv.Itoa(r.LeaderLeadCount),
		strconv.Itoa(r.NumberOfLeaders),
	}
}

// LeaderFootstep represents a song the queried singer has led, paired with
// the most frequent other leader of that same song.
type LeaderFootstep struct {
	LeaderName        string `db:"leader_name"` // most frequent other leader of this song
	SongTitle         string `db:"song_title"`
	SongPage          string `db:"song_page"`
	SongKeys          string `db:"song_keys"`
	SelfLeadCount     int    `db:"self_lead_count"`      // times the queried singer has led it
	TheirLeadCount    int    `db:"their_lead_count"`     // times the most frequent other leader has led it
	TheirLastLeadYear int    `db:"their_last_lead_year"` // last year the top other leader led this song
}

func (LeaderFootstep) ColumnNames() []mdwn.Column {
	return []mdwn.Column{
		{Name: "Song"},
		{Name: "Page"},
		{Name: "Key"},
		{Name: "Top Leader"},
		{Name: "Their Leads", RightAlign: true},
		{Name: "Last Year", RightAlign: true},
		{Name: "Self Leads", RightAlign: true},
	}
}

func (lf LeaderFootstep) RowValues() []string {
	return []string{
		lf.SongTitle,
		lf.SongPage,
		lf.SongKeys,
		lf.LeaderName,
		strconv.Itoa(lf.TheirLeadCount),
		strconv.Itoa(lf.TheirLastLeadYear),
		strconv.Itoa(lf.SelfLeadCount),
	}
}
