package db

import (
	"cmp"
	"context"
	"database/sql"
	"iter"

	"github.com/tychoish/dbx"
	"github.com/tychoish/odem/pkg/models"

	"github.com/tychoish/fun/fnx"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/srv"
)

type Connection struct {
	db *sql.DB
	*models.Queries
}

func Connect(ctx context.Context) (*Connection, error) {
	if err := Init(); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", getDBpath())
	if err != nil {
		return nil, err
	}
	srv.AddCleanup(ctx, fnx.MakeWorker(db.Close))

	return &Connection{db: db, Queries: models.New(db)}, nil
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
SELECT name, count, num_years, led_in_last_year
FROM song_leader_stats
WHERE page_num = ?
ORDER BY count DESC
LIMIT ?;`

	cur, err := conn.db.QueryContext(ctx, query, song, cmp.Or(limit, 40))
	if err != nil {
		return irt.Two(models.LeaderOfSongInfo{}, err)
	}
	return dbx.Cursor[models.LeaderOfSongInfo](cur)
}

func (conn *Connection) AllLessons(ctx context.Context, leader string) iter.Seq2[models.LessonInfo, error] {
	const query = `
SELECT
	leader AS singer_name,
	song_page_number,
	song_title AS song_name,
	song_keys AS song_key,
	minutes_date AS singing_date,
	minutes_name AS singing_name,
	location_state AS singing_state
FROM minutes_expanded
WHERE leader = ?;`

	cur, err := conn.db.QueryContext(ctx, query, leader)
	if err != nil {
		return irt.Two(models.LessonInfo{}, err)
	}
	return dbx.Cursor[models.LessonInfo](cur)
}

func (conn *Connection) SingingLessons(ctx context.Context, singing string) iter.Seq2[models.SingingLessionInfo, error] {
	const query = `
SELECT sequence_number, singer_name, song_page_number, song_name, song_key
FROM singing_lessons
WHERE singing_name = ?;`

	cur, err := conn.db.QueryContext(ctx, query, singing)
	if err != nil {
		return irt.Two(models.SingingLessionInfo{}, err)
	}
	return dbx.Cursor[models.SingingLessionInfo](cur)
}

func (conn *Connection) AllSingings(ctx context.Context, singing string) iter.Seq2[models.SingingInfo, error] {
	const query = `
SELECT singing_date, singing_name, singing_state, singing_location, number_of_lessons, number_of_leaders
FROM singing_info
WHERE (? = '' OR singing_name = ?)
ORDER BY singing_date;`

	cur, err := conn.db.QueryContext(ctx, query, singing, singing)
	if err != nil {
		return irt.Two(models.SingingInfo{}, err)
	}
	return dbx.Cursor[models.SingingInfo](cur)
}
