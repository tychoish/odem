package models

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
