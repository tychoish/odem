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
		return fmt.Errorf("LessonDate: expected string, got %T", src)
	}
	t, err := time.Parse("January 2, 2006", s)
	if err != nil {
		return err
	}
	*d = DateTime(t)
	return nil
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

type SingingLessionInfo struct {
	SequenceNumber int    `db:"sequence_number"` // nth lesson of the day
	SingerName     string `db:"singer_name"`
	SongPageNumber string `db:"song_page_number"`
	SongName       string `db:"song_name"`
	SongKey        string `db:"song_key"`
}

type SingingInfo struct {
	SingingDate     DateTime `db:"singing_date"`
	SingingName     string   `db:"singing_name"`
	SingingState    string   `db:"singing_state"`
	SingingLocation string   `db:"singing_location"`
	NumberOfLessons int      `db:"number_of_lessons"`
	NumberOfLeaders int      `db:"number_of_leaders"`
}
