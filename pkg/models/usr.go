package models

import (
	"fmt"
	"time"
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

type LeaderBackground struct {
	Name      string `db:"name"`
	NumLeads  int    `db:"num_leads"`
	FirstYear int    `db:"first_year"`
	LastYear  int    `db:"last_year"`
}

func (lsr *LeaderSongRank) StringFields() []string {
	return []string{lsr.NumLeads, lsr.PageNum, lsr.SongTitle, lsr.Key}
}

type LeaderOfSongInfo struct {
	Name          string `db:"name"`
	Count         int    `db:"count"`
	NumYears      int    `db:"num_years"`        // span: last year led minus first year led
	LedInLastYear bool   `db:"led_in_last_year"` // led within one year of the most recent minutes
}

type DateTime time.Time

func (d *DateTime) Scan(src any) error {
	s, ok := src.(string)
	if !ok {
		return fmt.Errorf("DateTime: expected string, got %T", src)
	}
	if normalized, ok := dateExceptions[s]; ok {
		s = normalized
	}
	t, err := time.Parse("January 2, 2006", s)
	if err != nil {
		return fmt.Errorf("DateTime.Scan: %w", err)
	}
	*d = DateTime(t)
	return nil
}

func (d DateTime) Time() time.Time { return time.Time(d) }
func (d DateTime) String() string  { return time.Time(d).Format(time.DateOnly) }

type LessonInfo struct {
	SingerName     string   `db:"singer_name"`
	SongPageNumber string   `db:"song_page_number"`
	SongName       string   `db:"song_name"`
	SongKey        string   `db:"song_key"`
	SingingDate    DateTime `db:"singing_date"`
	SingingName    string   `db:"singing_name"`
	SingingState   string   `db:"singing_state"`
}

type SingingLessionInfo struct {
	SequenceNumber int    `db:"sequence_number"` // nth lesson of the day
	LessonID       int    `db:"lesson_id"`
	SingerName     string `db:"singer_name"`
	SongPageNumber string `db:"song_page_number"`
	SongName       string `db:"song_name"`
	SongKey        string `db:"song_key"`
}

type LeaderLeadCount struct {
	Name         string  `db:"name"`
	Count        int     `db:"count"`
	LastLeadYear int     `db:"last_lead_year"`
	Percentage   float64 `db:"pct"`
	RunningTotal float64 `db:"running_total"`
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
