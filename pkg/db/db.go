package db

import (
	"cmp"
	"context"
	"database/sql"
	"iter"

	"github.com/tychoish/dbx"
	"github.com/tychoish/shbot/pkg/models"

	"github.com/tychoish/fun/fnx"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/srv"
)

type Connection struct{ db *sql.DB }

func Connect(ctx context.Context) (*Connection, error) {
	if err := Init(); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", getDBpath())
	if err != nil {
		return nil, err
	}
	srv.AddCleanup(ctx, fnx.MakeWorker(db.Close))

	return &Connection{db: db}, nil
}

func (conn *Connection) AllSongDetails(ctx context.Context) iter.Seq2[models.SongDetail, error] {
	const query = `SELECT * FROM song_details;`

	cur, err := conn.db.QueryContext(ctx, query)
	if err != nil {
		return irt.Two(models.SongDetail{}, err)
	}

	return dbx.Cursor[models.SongDetail](cur)
}

func (conn *Connection) AllSongPageNumbers(ctx context.Context) iter.Seq2[string, error] {
	const query = `SELECT page_num song_details;`

	cur, err := conn.db.QueryContext(ctx, query)
	if err != nil {
		return irt.Two("", err)
	}

	return dbx.Cursor[string](cur)
}

func (conn *Connection) AllLeaderNames(ctx context.Context) iter.Seq2[string, error] {
	const query = `SELECT name FROM leaders;`

	cur, err := conn.db.QueryContext(ctx, query)
	if err != nil {
		return irt.Two("", err)
	}
	return dbx.Cursor[string](cur)
}

func (conn *Connection) MostLeadSongs(ctx context.Context, leader string, limit int) iter.Seq2[models.LeaderSongRank, error] {
	const query = `
SELECT
	name,
	leader_lesson_count AS count,
	song_page,
	song_title,
	song_keys
FROM lesson_details
WHERE name = ?
GROUP BY leader_lesson_rank, song_page
ORDER BY count DESC
LIMIT ?;`

	cur, err := conn.db.QueryContext(ctx, query, leader, cmp.Or(limit, 40))
	if err != nil {
		return irt.Two(models.LeaderSongRank{}, err)
	}
	return dbx.Cursor[models.LeaderSongRank](cur)
}

func (conn *Connection) TopLeadersOfSong(ctx context.Context, song string, limit int) iter.Seq2[models.LeaderOfSongInfo, error] {
	const query = `
SELECT
	leaders.name,
	lss.lesson_count AS count,
	MAX(m.Year) - MIN(m.Year) AS num_years,
	CASE WHEN MAX(m.Year) >= (SELECT MAX(Year) FROM minutes) - 1 THEN 1 ELSE 0 END AS led_in_last_year
FROM leader_song_stats AS lss
JOIN leaders ON lss.leader_id = leaders.id
JOIN songs ON lss.song_id = songs.id
JOIN book_song_joins AS bsj ON songs.id = bsj.song_id AND bsj.book_id = 2
JOIN song_leader_joins AS slj ON slj.leader_id = lss.leader_id AND slj.song_id = lss.song_id
JOIN minutes AS m ON slj.minutes_id = m.id
WHERE bsj.page_num = ?
GROUP BY lss.leader_id
ORDER BY lss.lesson_count DESC
LIMIT ?;`

	cur, err := conn.db.QueryContext(ctx, query, song, cmp.Or(limit, 40))
	if err != nil {
		return irt.Two(models.LeaderOfSongInfo{}, err)
	}
	return dbx.Cursor[models.LeaderOfSongInfo](cur)
}
