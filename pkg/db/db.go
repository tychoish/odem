package db

import (
	"cmp"
	"context"
	"database/sql"
	"fmt"
	"iter"
	"time"

	"github.com/tychoish/dbx"
	"github.com/tychoish/odem"
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
	conf := odem.GetConfiguration(ctx)
	if conf == nil || !conf.Settings.ManualReloadDB {
		if err := Init(); err != nil {
			return nil, err
		}
	}

	db, err := sql.Open("sqlite", getDBpath())
	if err != nil {
		return nil, err
	}
	srv.AddCleanup(ctx, fnx.MakeWorker(db.Close))

	return &Connection{db: db, Queries: models.New(db)}, nil
}

func (conn *Connection) AllSongDetails(ctx context.Context) iter.Seq2[models.SongDetail, error] {
	const query = `SELECT * FROM song_details;`
	return dbx.Query[models.SongDetail](ctx, conn.db.QueryContext, query)
}

func (conn *Connection) AllLeaderProfiles(ctx context.Context) iter.Seq2[models.LeaderProfile, error] {
	const query = `SELECT * FROM leader_profiles;`
	return dbx.Query[models.LeaderProfile](ctx, conn.db.QueryContext, query)
}

func (conn *Connection) AllSongPageNumbers(ctx context.Context) iter.Seq2[string, error] {
	const query = `SELECT page_num FROM song_details;`
	return dbx.Query[string](ctx, conn.db.QueryContext, query)
}

func (conn *Connection) AllLeaderNames(ctx context.Context) iter.Seq2[string, error] {
	const query = `
SELECT COALESCE(lna.name, l.name) AS name
FROM leaders AS l
LEFT JOIN (SELECT alias, MIN(name) AS name FROM leader_name_aliases WHERE leader_id IS NOT NULL GROUP BY alias) AS lna ON lna.alias = l.name
LEFT JOIN leader_name_invalid AS inv ON inv.name = l.name
WHERE inv.name IS NULL;`
	return dbx.Query[string](ctx, conn.db.QueryContext, query)
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
	return dbx.Query[models.LeaderSongRank](ctx, conn.db.QueryContext, query, leader, cmp.Or(limit, 32))
}

func (conn *Connection) TopLeadersOfSong(ctx context.Context, song string, limit int) iter.Seq2[models.LeaderOfSongInfo, error] {
	const query = `
SELECT name, count, num_years, led_in_last_year
FROM song_leader_stats
WHERE page_num = ?
ORDER BY count DESC
LIMIT ?;`
	return dbx.Query[models.LeaderOfSongInfo](ctx, conn.db.QueryContext, query, song, cmp.Or(limit, 32))
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
	return dbx.Query[models.LessonInfo](ctx, conn.db.QueryContext, query, leader)
}

func (conn *Connection) LeaderLeadHistory(ctx context.Context, leader string) iter.Seq2[models.LessonInfo, error] {
	const query = `
SELECT
	CAST(COALESCE(lna.name, leaders.name, '') AS TEXT) AS singer_name,
	COALESCE(bsj.page_num, '') AS song_page_number,
	COALESCE(songs.title, '') AS song_name,
	COALESCE(bsj.keys, '') AS song_key,
	COALESCE(minutes."Date", '') AS singing_date,
	COALESCE(minutes."Name", '') AS singing_name,
	COALESCE(loc.state_province, '') AS singing_state
FROM song_leader_joins AS slj
LEFT JOIN minutes ON slj.minutes_id = minutes.id
LEFT JOIN leaders ON slj.leader_id = leaders.id
LEFT JOIN (SELECT alias, MIN(name) AS name FROM leader_name_aliases WHERE leader_id IS NOT NULL GROUP BY alias) AS lna ON lna.alias = leaders.name
LEFT JOIN songs ON slj.song_id = songs.id
LEFT JOIN (
	SELECT mlj.minutes_id, MIN(loc.state_province) AS state_province
	FROM minutes_location_joins AS mlj
	JOIN locations AS loc ON loc.id = mlj.location_id
	GROUP BY mlj.minutes_id
) AS loc ON loc.minutes_id = slj.minutes_id
LEFT JOIN book_song_joins AS bsj ON slj.song_id = bsj.song_id AND bsj.book_id = 2
WHERE CAST(COALESCE(lna.name, leaders.name, '') AS TEXT) = ?
ORDER BY minutes.Year DESC, slj.minutes_id DESC`
	return dbx.Query[models.LessonInfo](ctx, conn.db.QueryContext, query, leader)
}

func (conn *Connection) LeaderSingingsAttended(ctx context.Context, leader string, limit int) iter.Seq2[models.LeaderSingingAttendance, error] {
	const query = `
SELECT
	COALESCE(m."Name", '') AS singing_name,
	COALESCE(m."Date", '') AS singing_date,
	COALESCE(loc.state_province, '') AS singing_state,
	COALESCE(loc.city, '') AS singing_city,
	COUNT(slj.id) AS leader_lead_count,
	COALESCE(total.number_of_leaders, 0) AS number_of_leaders
FROM song_leader_joins AS slj
JOIN minutes AS m ON slj.minutes_id = m.id
JOIN leaders AS l ON slj.leader_id = l.id
LEFT JOIN (SELECT alias, MIN(name) AS name FROM leader_name_aliases WHERE leader_id IS NOT NULL GROUP BY alias) AS lna ON lna.alias = l.name
LEFT JOIN minutes_location_joins AS mlj ON m.id = mlj.minutes_id
LEFT JOIN locations AS loc ON mlj.location_id = loc.id
LEFT JOIN (
	SELECT minutes_id, COUNT(DISTINCT leader_id) AS number_of_leaders
	FROM song_leader_joins
	GROUP BY minutes_id
) AS total ON total.minutes_id = m.id
WHERE CAST(COALESCE(lna.name, l.name, '') AS TEXT) = ?
GROUP BY m.id
ORDER BY m.Year DESC, m.id DESC
LIMIT ?`
	return dbx.Query[models.LeaderSingingAttendance](ctx, conn.db.QueryContext, query, leader, cmp.Or(limit, 100))
}

func (conn *Connection) SingingLessons(ctx context.Context, singing string) iter.Seq2[models.SingingLessionInfo, error] {
	const query = `
SELECT lesson_id, sequence_number, singer_name, song_page_number, song_name, song_key
FROM singing_lessons
WHERE singing_name = ?;`
	return dbx.Query[models.SingingLessionInfo](ctx, conn.db.QueryContext, query, singing)
}

func (conn *Connection) AllSingings(ctx context.Context) iter.Seq2[models.SingingInfo, error] {
	const query = `
SELECT singing_date, singing_name, singing_state, singing_location, number_of_lessons, number_of_leaders
FROM singing_info
ORDER BY minutes_id DESC;`
	return dbx.Query[models.SingingInfo](ctx, conn.db.QueryContext, query)
}

func (conn *Connection) SingingBuddies(ctx context.Context, name string, limit int) iter.Seq2[irt.KV[string, int], error] {
	const query = `
SELECT COALESCE(lna2.name, l2.name) AS key, COUNT(*) AS value
FROM leader_singings AS ls_me
JOIN leaders AS l_me ON l_me.id = ls_me.leader_id
JOIN leader_singings AS ls_other ON ls_other.minutes_id = ls_me.minutes_id AND ls_other.leader_id != ls_me.leader_id
JOIN leaders AS l2 ON l2.id = ls_other.leader_id
LEFT JOIN (SELECT alias, MIN(name) AS name FROM leader_name_aliases WHERE leader_id IS NOT NULL GROUP BY alias) AS lna2 ON lna2.alias = l2.name
LEFT JOIN leader_name_invalid AS inv ON inv.name = l2.name
WHERE l_me.name = ?
AND l2.name != ?
AND inv.name IS NULL
GROUP BY ls_other.leader_id
ORDER BY value DESC
LIMIT ?;`
	return dbx.Query[irt.KV[string, int]](ctx, conn.db.QueryContext, query, name, name, cmp.Or(limit, 32))
}

func (conn *Connection) PopularSongsInOnesExperience(ctx context.Context, name string, limit int) iter.Seq2[models.LeaderSongRank, error] {
	const query = `
SELECT lsa.attendance_count AS count, bsj.page_num AS song_page, s.title AS song_title, bsj.keys AS song_keys
FROM leader_song_attendance AS lsa
JOIN leaders AS l ON l.id = lsa.leader_id
JOIN songs AS s ON s.id = lsa.song_id
JOIN book_song_joins AS bsj ON bsj.song_id = lsa.song_id AND bsj.book_id = 2
WHERE l.name = ?
ORDER BY count DESC
LIMIT ?;`
	return dbx.Query[models.LeaderSongRank](ctx, conn.db.QueryContext, query, name, cmp.Or(limit, 32))
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
        COALESCE(lna.name, l.name) AS key,
        mutual AS value
FROM stranger_scores
JOIN leaders AS l ON l.id = stranger_scores.stranger_id
LEFT JOIN (SELECT alias, MIN(name) AS name FROM leader_name_aliases WHERE leader_id IS NOT NULL GROUP BY alias) AS lna ON lna.alias = l.name
LEFT JOIN leader_name_invalid AS inv ON inv.name = l.name
WHERE inv.name IS NULL
ORDER BY value DESC
LIMIT ?;`
	return dbx.Query[irt.KV[string, int]](ctx, conn.db.QueryContext, query, name, cmp.Or(limit, 32))
}

func (conn *Connection) LeaderFavoriteKey(ctx context.Context, leader string, limit int) iter.Seq2[irt.KV[string, int], error] {
	const query = `
SELECT bsj.keys AS key, COUNT(*) AS value
FROM song_leader_joins AS slj
JOIN leaders AS l ON l.id = slj.leader_id
LEFT JOIN (SELECT alias, MIN(name) AS name FROM leader_name_aliases WHERE leader_id IS NOT NULL GROUP BY alias) AS lna ON lna.alias = l.name
JOIN book_song_joins AS bsj ON slj.song_id = bsj.song_id AND bsj.book_id = 2
WHERE CAST(COALESCE(lna.name, l.name, '') AS TEXT) = ?
GROUP BY bsj.keys
ORDER BY value DESC
LIMIT ?;`
	return dbx.Query[irt.KV[string, int]](ctx, conn.db.QueryContext, query, leader, cmp.Or(limit, 20))
}

func (conn *Connection) AllLeaderConnectedness(ctx context.Context, limit int) iter.Seq2[irt.KV[string, float64], error] {
	const query = `
SELECT
        COALESCE(lna.name, l.name) AS key,
        CAST(COUNT(lca.leader_b_id) AS REAL) / (SELECT COUNT(*) FROM leaders) AS value
FROM leaders l
LEFT JOIN leader_coattendance lca ON lca.leader_a_id = l.id
LEFT JOIN (SELECT alias, MIN(name) AS name FROM leader_name_aliases WHERE leader_id IS NOT NULL GROUP BY alias) AS lna ON lna.alias = l.name
LEFT JOIN leader_name_invalid AS inv ON inv.name = l.name
WHERE inv.name IS NULL
GROUP BY l.id
ORDER BY value DESC
LIMIT ?;`
	return dbx.Query[irt.KV[string, float64]](ctx, conn.db.QueryContext, query, cmp.Or(limit, 32))
}

func (conn *Connection) SingersConnectedness(ctx context.Context, name string) (*float64, error) {
	const query = `SELECT CAST(COUNT(*) AS REAL) / (SELECT COUNT(*) FROM leaders) AS connectedness
FROM leader_coattendance
WHERE leader_a_id = (SELECT id FROM leaders WHERE leaders.name = ?)`

	v, err := dbx.QueryRow[float64](ctx, conn.db.QueryContext, query, name)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (conn *Connection) TheUnfamilarHits(ctx context.Context, name string, limit int) iter.Seq2[models.LeaderSongRank, error] {
	// Exposure is measured by attendance: how many times the song was called at
	// a singing the leader attended. Using leader_song_stats (lead count) instead
	// was the source of a bug — leaders lead a small fraction of the book, so
	// nearly every song had count=0 and results were identical to the global
	// most-popular list regardless of the input leader.
	const query = `
SELECT
    ? AS name,
    COALESCE(lsa.attendance_count, 0) AS count,
    bsj.page_num AS song_page,
    s.title AS song_title,
    bsj.keys AS song_keys,
    CASE WHEN COALESCE(sst.total, 0) > 0
         THEN CAST(COALESCE(lsa.attendance_count, 0) AS REAL) / sst.total
         ELSE 0.0
    END AS ratio
FROM book_song_joins AS bsj
JOIN songs AS s ON s.id = bsj.song_id
LEFT JOIN song_stats_totals AS sst ON sst.song_id = bsj.song_id
LEFT JOIN leader_song_attendance AS lsa ON lsa.song_id = bsj.song_id
    AND lsa.leader_id = (SELECT id FROM leaders WHERE name = ?)
WHERE bsj.book_id = 2
ORDER BY count ASC, sst.total DESC
LIMIT ?`
	return dbx.Query[models.LeaderSongRank](ctx, conn.db.QueryContext, query, name, name, cmp.Or(limit, 32))
}

func (conn *Connection) GloballyPopularForYears(ctx context.Context, years ...int) iter.Seq2[models.LeaderSongRank, error] {
	currentYear := time.Now().Year()

	var includeYears, excludeYears []int
	for _, y := range years {
		abs := y
		if y < 0 {
			abs = -y
		}
		if abs < 1995 || abs > currentYear {
			return irt.Two(models.LeaderSongRank{}, fmt.Errorf("year %d out of valid range [1995, %d]", y, currentYear))
		}
		if y < 0 {
			excludeYears = append(excludeYears, abs)
		} else {
			includeYears = append(includeYears, y)
		}
	}
	if len(includeYears) > 0 && len(excludeYears) > 0 {
		return irt.Two(models.LeaderSongRank{}, fmt.Errorf("cannot mix included and excluded years"))
	}

	var qb dbx.Builder
	qb.WithSQL(`
SELECT
    bsj.page_num AS song_page,
    s.title AS song_title,
    bsj.keys AS song_keys,
    SUM(ss.lesson_count) AS count
FROM song_stats AS ss
JOIN songs AS s ON s.id = ss.song_id
JOIN book_song_joins AS bsj ON bsj.song_id = ss.song_id AND bsj.book_id = 2`)

	switch {
	case len(includeYears) > 0:
		qb.With(" WHERE ss.year IN (%+?)", includeYears)
	case len(excludeYears) > 0:
		qb.With(" WHERE ss.year NOT IN (%+?)", excludeYears)
	}

	qb.WithSQL(`
GROUP BY ss.song_id
ORDER BY count DESC
LIMIT 40`)

	query, args := qb.Build()
	return dbx.Query[models.LeaderSongRank](ctx, conn.db.QueryContext, query, args...)
}

func (conn *Connection) LocallyPopular(ctx context.Context, limit int, states ...models.SingingLocality) iter.Seq2[models.LeaderSongRank, error] {
	stateStrs := make([]string, len(states))
	for i, s := range states {
		stateStrs[i] = string(s)
	}

	var qb dbx.Builder
	qb.WithSQL(`
SELECT
    COUNT(*) AS count,
    bsj.page_num AS song_page,
    s.title AS song_title,
    bsj.keys AS song_keys
FROM song_leader_joins AS slj
JOIN songs AS s ON s.id = slj.song_id
JOIN book_song_joins AS bsj ON bsj.song_id = s.id AND bsj.book_id = 2
JOIN minutes_location_joins AS mlj ON mlj.minutes_id = slj.minutes_id
JOIN locations AS loc ON loc.id = mlj.location_id`)

	if len(stateStrs) > 0 {
		qb.With(" WHERE loc.state_province IN (%+?)", stateStrs)
	}

	qb.WithSQL(`
GROUP BY slj.song_id
ORDER BY count DESC`)

	if limit > 0 {
		qb.With(" LIMIT %?", limit)
	}

	query, args := qb.Build()
	return dbx.Query[models.LeaderSongRank](ctx, conn.db.QueryContext, query, args...)
}

func (conn *Connection) NeverLed(ctx context.Context, name string) iter.Seq2[models.LeaderSongRank, error] {
	const query = `
SELECT
    ? AS name,
    COALESCE(sst.total, 0) AS count,
    bsj.page_num AS song_page,
    s.title AS song_title,
    bsj.keys AS song_keys
FROM book_song_joins AS bsj
JOIN songs AS s ON s.id = bsj.song_id
LEFT JOIN song_stats_totals AS sst ON sst.song_id = bsj.song_id
WHERE bsj.book_id = 2
AND bsj.song_id NOT IN (
    SELECT lss.song_id
    FROM leader_song_stats AS lss
    JOIN leaders AS l ON l.id = lss.leader_id
    WHERE l.name = ?
)
ORDER BY count DESC`
	return dbx.Query[models.LeaderSongRank](ctx, conn.db.QueryContext, query, name, name)
}

func (conn *Connection) LeaderFootsteps(ctx context.Context, name string, limit int) iter.Seq2[models.LeaderFootstep, error] {
	const query = `
WITH my_songs AS (
    SELECT lss.song_id, lss.lesson_count AS self_lead_count
    FROM leader_song_stats AS lss
    JOIN leaders AS l ON l.id = lss.leader_id
    WHERE l.name = ?
),
top_other_leaders AS (
    SELECT
        lss.song_id,
        COALESCE(lna.name, l.name) AS other_leader_name,
        lss.lesson_count AS other_count,
        MAX(m.Year) AS their_last_lead_year,
        ROW_NUMBER() OVER (PARTITION BY lss.song_id ORDER BY lss.lesson_count DESC) AS rn
    FROM leader_song_stats AS lss
    JOIN leaders AS l ON l.id = lss.leader_id
    JOIN song_leader_joins AS slj ON slj.leader_id = lss.leader_id AND slj.song_id = lss.song_id
    JOIN minutes AS m ON m.id = slj.minutes_id
    LEFT JOIN (SELECT alias, MIN(name) AS name FROM leader_name_aliases WHERE leader_id IS NOT NULL GROUP BY alias) AS lna ON lna.alias = l.name
    LEFT JOIN leader_name_invalid AS inv ON inv.name = l.name
    WHERE l.name != ?
    AND inv.name IS NULL
    GROUP BY lss.song_id, lss.leader_id
)
SELECT
    tol.other_leader_name AS leader_name,
    bsj.page_num AS song_page,
    s.title AS song_title,
    bsj.keys AS song_keys,
    ms.self_lead_count AS self_lead_count,
    tol.other_count AS their_lead_count,
    tol.their_last_lead_year
FROM my_songs AS ms
JOIN top_other_leaders AS tol ON tol.song_id = ms.song_id AND tol.rn = 1
JOIN songs AS s ON s.id = ms.song_id
JOIN book_song_joins AS bsj ON bsj.song_id = ms.song_id AND bsj.book_id = 2
ORDER BY ms.self_lead_count DESC
LIMIT ?`
	return dbx.Query[models.LeaderFootstep](ctx, conn.db.QueryContext, query, name, name, cmp.Or(limit, 32))
}

func (conn *Connection) LeaderShareOfLeads(ctx context.Context, name string, years ...int) (*float64, error) {
	currentYear := time.Now().Year()

	var includeYears, excludeYears []int
	for _, y := range years {
		abs := y
		if y < 0 {
			abs = -y
		}
		if abs < 1995 || abs > currentYear {
			return nil, fmt.Errorf("year %d out of valid range [1995, %d]", y, currentYear)
		}
		if y < 0 {
			excludeYears = append(excludeYears, abs)
		} else {
			includeYears = append(includeYears, y)
		}
	}
	if len(includeYears) > 0 && len(excludeYears) > 0 {
		return nil, fmt.Errorf("cannot mix included and excluded years")
	}

	var qb dbx.Builder
	// denominator subquery (total leads for the year filter)
	qb.WithSQL(`
SELECT CAST(COUNT(slj.id) AS REAL) / (
    SELECT COUNT(slj2.id)
    FROM song_leader_joins AS slj2
    JOIN minutes AS m2 ON m2.id = slj2.minutes_id`)
	switch {
	case len(includeYears) > 0:
		qb.With(" WHERE m2.Year IN (%+?)", includeYears)
	case len(excludeYears) > 0:
		qb.With(" WHERE m2.Year NOT IN (%+?)", excludeYears)
	}
	// numerator outer query — name param placed after denominator years
	qb.WithSQL(`) AS value
FROM song_leader_joins AS slj
JOIN leaders AS l ON l.id = slj.leader_id`)
	if len(includeYears)+len(excludeYears) > 0 {
		qb.WithSQL(`
JOIN minutes AS m ON m.id = slj.minutes_id`)
	}
	qb.With("\nWHERE l.name = %?", name)
	switch {
	case len(includeYears) > 0:
		qb.With(" AND m.Year IN (%+?)", includeYears)
	case len(excludeYears) > 0:
		qb.With(" AND m.Year NOT IN (%+?)", excludeYears)
	}

	query, args := qb.Build()
	v, err := dbx.QueryRow[float64](ctx, conn.db.QueryContext, query, args...)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (conn *Connection) TopLeadersByLeads(ctx context.Context, limit int, years ...int) iter.Seq2[models.LeaderLeadCount, error] {
	currentYear := time.Now().Year()

	var includeYears, excludeYears []int
	for _, y := range years {
		abs := y
		if y < 0 {
			abs = -y
		}
		if abs < 1995 || abs > currentYear {
			return irt.Two(models.LeaderLeadCount{}, fmt.Errorf("year %d out of valid range [1995, %d]", y, currentYear))
		}
		if y < 0 {
			excludeYears = append(excludeYears, abs)
		} else {
			includeYears = append(includeYears, y)
		}
	}
	if len(includeYears) > 0 && len(excludeYears) > 0 {
		return irt.Two(models.LeaderLeadCount{}, fmt.Errorf("cannot mix included and excluded years"))
	}

	var qb dbx.Builder
	qb.WithSQL(`
WITH counts AS (
    SELECT COALESCE(lna.name, l.name) AS name, COUNT(slj.id) AS count, MAX(m.Year) AS last_lead_year
    FROM leaders AS l
    JOIN song_leader_joins AS slj ON slj.leader_id = l.id
    JOIN minutes AS m ON m.id = slj.minutes_id
    LEFT JOIN (SELECT alias, MIN(name) AS name FROM leader_name_aliases WHERE leader_id IS NOT NULL GROUP BY alias) AS lna ON lna.alias = l.name
    LEFT JOIN leader_name_invalid AS inv ON inv.name = l.name
    WHERE inv.name IS NULL`)

	switch {
	case len(includeYears) > 0:
		qb.With(" AND m.Year IN (%+?)", includeYears)
	case len(excludeYears) > 0:
		qb.With(" AND m.Year NOT IN (%+?)", excludeYears)
	}

	qb.WithSQL(`
    GROUP BY l.id
    ORDER BY count DESC`)
	qb.With(" LIMIT %?", cmp.Or(limit, 40))

	qb.WithSQL(`
),
total AS (
    SELECT COUNT(slj2.id) AS grand_total
    FROM leaders AS l2
    JOIN song_leader_joins AS slj2 ON slj2.leader_id = l2.id
    JOIN minutes AS m2 ON m2.id = slj2.minutes_id
    LEFT JOIN leader_name_invalid AS inv2 ON inv2.name = l2.name
    WHERE inv2.name IS NULL`)

	switch {
	case len(includeYears) > 0:
		qb.With(" AND m2.Year IN (%+?)", includeYears)
	case len(excludeYears) > 0:
		qb.With(" AND m2.Year NOT IN (%+?)", excludeYears)
	}

	qb.WithSQL(`
),
totaled AS (
    SELECT name, count, last_lead_year, CAST(count AS REAL) / (SELECT grand_total FROM total) AS pct
    FROM counts
)
SELECT
    name,
    count,
    last_lead_year,
    pct,
    SUM(pct) OVER (ORDER BY count DESC ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) AS running_total
FROM totaled
ORDER BY count DESC`)

	query, args := qb.Build()
	return dbx.Query[models.LeaderLeadCount](ctx, conn.db.QueryContext, query, args...)
}

func (conn *Connection) NewLeadersByYear(ctx context.Context, year int, limit int) iter.Seq2[models.LeaderSongRank, error] {
	const query = `
SELECT
    CAST(COALESCE(lna.name, l.name, '') AS TEXT) AS name,
    COUNT(slj.id) AS count
FROM leaders AS l
JOIN song_leader_joins AS slj ON slj.leader_id = l.id
JOIN minutes AS m ON m.id = slj.minutes_id
LEFT JOIN (SELECT alias, MIN(name) AS name FROM leader_name_aliases WHERE leader_id IS NOT NULL GROUP BY alias) AS lna ON lna.alias = l.name
LEFT JOIN leader_name_invalid AS inv ON inv.name = l.name
WHERE inv.name IS NULL
  AND m.Year = ?
  AND l.id NOT IN (
      SELECT slj2.leader_id
      FROM song_leader_joins AS slj2
      JOIN minutes AS m2 ON m2.id = slj2.minutes_id
      WHERE m2.Year < ?
  )
GROUP BY l.id
ORDER BY count DESC
LIMIT ?`
	return dbx.Query[models.LeaderSongRank](ctx, conn.db.QueryContext, query, year, year, cmp.Or(limit, 40))
}

func (conn *Connection) SongsByKey(ctx context.Context, years ...int) iter.Seq2[models.LeaderSongRank, error] {
	var qb dbx.Builder
	qb.WithSQL(`SELECT bsj.keys AS song_keys, COUNT(*) AS count,
    CAST(COUNT(*) AS REAL) / SUM(COUNT(*)) OVER () AS ratio
FROM song_leader_joins AS slj
JOIN minutes AS m ON m.id = slj.minutes_id
JOIN book_song_joins AS bsj ON bsj.song_id = slj.song_id AND bsj.book_id = 2
WHERE bsj.keys != ''`)
	if len(years) > 0 {
		qb.With(` AND m.Year IN (%+?)`, years)
	}
	qb.WithSQL(` GROUP BY bsj.keys ORDER BY count DESC`)
	query, args := qb.Build()
	return dbx.Query[models.LeaderSongRank](ctx, conn.db.QueryContext, query, args...)
}

func (conn *Connection) LeadersByTop20Leads(ctx context.Context, limit int) iter.Seq2[models.LeaderSongRank, error] {
	const query = `
SELECT CAST(COALESCE(lna.name, l.name, '') AS TEXT) AS name,
       l.top20_count AS count
FROM leaders AS l
LEFT JOIN (SELECT alias, MIN(name) AS name FROM leader_name_aliases WHERE leader_id IS NOT NULL GROUP BY alias) AS lna ON lna.alias = l.name
LEFT JOIN leader_name_invalid AS inv ON inv.name = l.name
WHERE inv.name IS NULL AND l.top20_count > 0
ORDER BY count DESC
LIMIT ?`
	return dbx.Query[models.LeaderSongRank](ctx, conn.db.QueryContext, query, cmp.Or(limit, 40))
}

func (conn *Connection) LeaderSingingsPerYear(ctx context.Context, name string) iter.Seq2[irt.KV[string, int], error] {
	const query = `
SELECT CAST(m.Year AS TEXT) AS key, COUNT(DISTINCT slj.minutes_id) AS value
FROM song_leader_joins AS slj
JOIN minutes AS m ON slj.minutes_id = m.id
JOIN leaders AS l ON slj.leader_id = l.id
LEFT JOIN (SELECT alias, MIN(name) AS name FROM leader_name_aliases WHERE leader_id IS NOT NULL GROUP BY alias) AS lna ON lna.alias = l.name
LEFT JOIN leader_name_invalid AS inv ON inv.name = l.name
WHERE inv.name IS NULL
AND CAST(COALESCE(lna.name, l.name, '') AS TEXT) = ?
GROUP BY m.Year
ORDER BY m.Year ASC`
	return dbx.Query[irt.KV[string, int]](ctx, conn.db.QueryContext, query, name)
}

func (conn *Connection) AllKeys(ctx context.Context) iter.Seq2[string, error] {
	const query = `SELECT DISTINCT keys FROM book_song_joins WHERE book_id = 2 AND keys != '' ORDER BY keys;`
	return dbx.Query[string](ctx, conn.db.QueryContext, query)
}

func (conn *Connection) PopularSongsByKey(ctx context.Context, key string, limit int) iter.Seq2[models.LeaderSongRank, error] {
	const query = `
SELECT
    bsj.page_num AS song_page,
    s.title AS song_title,
    COUNT(DISTINCT slj.id) AS count,
    bsj.keys AS song_keys
FROM book_song_joins AS bsj
JOIN songs AS s ON s.id = bsj.song_id
JOIN song_leader_joins AS slj ON slj.song_id = bsj.song_id
WHERE bsj.book_id = 2
  AND bsj.keys = ?
GROUP BY bsj.song_id
ORDER BY count DESC
LIMIT ?;`
	return dbx.Query[models.LeaderSongRank](ctx, conn.db.QueryContext, query, key, cmp.Or(limit, 40))
}

func (conn *Connection) LeadersByKey(ctx context.Context, key string, limit int) iter.Seq2[models.LeaderSongRank, error] {
	const query = `
SELECT
	CAST(COALESCE(lna.name, l.name, '') AS TEXT) AS name,
	COUNT(slj.id) AS count
FROM song_leader_joins AS slj
JOIN leaders AS l ON l.id = slj.leader_id
JOIN book_song_joins AS bsj ON bsj.song_id = slj.song_id AND bsj.book_id = 2
LEFT JOIN (SELECT alias, MIN(name) AS name FROM leader_name_aliases WHERE leader_id IS NOT NULL GROUP BY alias) AS lna ON lna.alias = l.name
LEFT JOIN leader_name_invalid AS inv ON inv.name = l.name
WHERE inv.name IS NULL
  AND bsj.keys = ?
GROUP BY l.id
ORDER BY count DESC
LIMIT ?`
	return dbx.Query[models.LeaderSongRank](ctx, conn.db.QueryContext, query, key, cmp.Or(limit, 40))
}

func (conn *Connection) NeverSung(ctx context.Context, name string) iter.Seq2[models.LeaderSongRank, error] {
	const query = `
SELECT
    ? AS name,
    COALESCE(sst.total, 0) AS count,
    bsj.page_num AS song_page,
    s.title AS song_title,
    bsj.keys AS song_keys
FROM book_song_joins AS bsj
JOIN songs AS s ON s.id = bsj.song_id
LEFT JOIN song_stats_totals AS sst ON sst.song_id = bsj.song_id
WHERE bsj.book_id = 2
AND bsj.song_id NOT IN (
    SELECT lsa.song_id FROM leader_song_attendance AS lsa
    JOIN leaders AS l ON l.id = lsa.leader_id
    WHERE l.name = ?
)
ORDER BY count DESC`
	return dbx.Query[models.LeaderSongRank](ctx, conn.db.QueryContext, query, name, name)
}
