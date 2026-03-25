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
	const query = `
SELECT l.name
FROM leaders AS l
LEFT JOIN leader_name_invalid AS inv ON inv.name = l.name
WHERE inv.name IS NULL;`

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
SELECT lesson_id, sequence_number, singer_name, song_page_number, song_name, song_key
FROM singing_lessons
WHERE singing_name = ?;`

	cur, err := conn.db.QueryContext(ctx, query, singing)
	if err != nil {
		return irt.Two(models.SingingLessionInfo{}, err)
	}
	return dbx.Cursor[models.SingingLessionInfo](cur)
}

func (conn *Connection) AllSingings(ctx context.Context) iter.Seq2[models.SingingInfo, error] {
	const query = `
SELECT singing_date, singing_name, singing_state, singing_location, number_of_lessons, number_of_leaders
FROM singing_info
ORDER BY minutes_id DESC;`

	cur, err := conn.db.QueryContext(ctx, query)
	if err != nil {
		return irt.Two(models.SingingInfo{}, err)
	}
	return dbx.Cursor[models.SingingInfo](cur)
}

func (conn *Connection) SingingBuddies(ctx context.Context, name string, limit int) iter.Seq2[irt.KV[string, int], error] {
	const query = `
SELECT lm_other.leader_name AS key, COUNT(DISTINCT lm_me.minutes_id) AS value
FROM leader_minutes AS lm_me
JOIN leader_minutes AS lm_other ON lm_other.minutes_id = lm_me.minutes_id
LEFT JOIN leader_name_invalid AS inv ON inv.name = lm_other.leader_name
WHERE lm_me.leader_name = ?
AND lm_other.leader_name != ?
AND inv.name IS NULL
GROUP BY lm_other.leader_id
ORDER BY value DESC
LIMIT ?;`

	cur, err := conn.db.QueryContext(ctx, query, name, name, cmp.Or(limit, 40))
	if err != nil {
		return irt.Two(irt.KV[string, int]{}, err)
	}
	return dbx.Cursor[irt.KV[string, int]](cur)
}

func (conn *Connection) PopularSongsInOnesExperience(ctx context.Context, name string, limit int) iter.Seq2[models.LeaderSongRank, error] {
	const query = `
SELECT COUNT(*) AS count, bsj.page_num AS song_page, s.title AS song_title, bsj.keys AS song_keys
FROM leader_minutes AS lm
JOIN song_leader_joins AS slj ON slj.minutes_id = lm.minutes_id
JOIN songs AS s ON slj.song_id = s.id
JOIN book_song_joins AS bsj ON bsj.song_id = s.id AND bsj.book_id = 2
WHERE lm.leader_name = ?
GROUP BY bsj.page_num
ORDER BY count DESC
LIMIT ?;`

	cur, err := conn.db.QueryContext(ctx, query, name, cmp.Or(limit, 40))
	if err != nil {
		return irt.Two(models.LeaderSongRank{}, err)
	}
	return dbx.Cursor[models.LeaderSongRank](cur)
}

func (conn *Connection) SingingStrangers(ctx context.Context, name string, limit int) iter.Seq2[irt.KV[string, int], error] {
	const query = `
WITH target AS (SELECT id FROM leaders WHERE name = ?),
my_network AS (
	SELECT leader_b_id AS peer_id
	FROM leader_coattendance
	WHERE leader_a_id = (SELECT id FROM target)
),
stranger_scores AS (
	SELECT
                lca.leader_b_id AS stranger_id,
                COUNT(*) AS mutual
	FROM leader_coattendance AS lca
	WHERE lca.leader_a_id IN (SELECT peer_id FROM my_network)
	AND lca.leader_b_id NOT IN (SELECT peer_id FROM my_network)
	AND lca.leader_b_id != (SELECT id FROM target)
	GROUP BY lca.leader_b_id
)
SELECT
        l.name AS key,
        mutual AS value
FROM stranger_scores
JOIN leaders AS l ON l.id = stranger_scores.stranger_id
LEFT JOIN leader_name_invalid AS inv ON inv.name = l.name
WHERE inv.name IS NULL
ORDER BY value DESC
LIMIT ?;`

	cur, err := conn.db.QueryContext(ctx, query, name, cmp.Or(limit, 40))
	if err != nil {
		return irt.Two(irt.MakeKV("", 0), err)
	}
	return dbx.Cursor[irt.KV[string, int]](cur)
}

func (conn *Connection) AllLeaderConnectedness(ctx context.Context) iter.Seq2[irt.KV[string, float64], error] {
	const query = `
SELECT
        l.name AS key,
        CAST(COUNT(lca.leader_b_id) AS REAL) / (SELECT COUNT(*) FROM leaders) AS value
FROM leaders l
LEFT JOIN leader_coattendance lca ON lca.leader_a_id = l.id
LEFT JOIN leader_name_invalid AS inv ON inv.name = l.name
WHERE inv.name IS NULL
GROUP BY l.id
ORDER BY value DESC;`

	cur, err := conn.db.QueryContext(ctx, query)
	if err != nil {
		return irt.Two(irt.KV[string, float64]{}, err)
	}
	return dbx.Cursor[irt.KV[string, float64]](cur)
}

func (conn *Connection) SingersConnectedness(ctx context.Context, name string) (*float64, error) {
	const query = `SELECT CAST(COUNT(*) AS REAL) / (SELECT COUNT(*) FROM leaders) AS connectedness
FROM leader_coattendance
WHERE leader_a_id = (SELECT id FROM leaders WHERE leaders.name = ?)`

	var v float64
	if err := conn.db.QueryRowContext(ctx, query, name).Scan(&v); err != nil {
		return nil, err
	}
	return &v, nil
}
