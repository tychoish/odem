-- lesson_exclusions: idempotent creation for the fast-path (views-only rebuild).
-- On a full rebuild setup.sql creates this first; here we ensure it exists for
-- existing databases being upgraded so active_song_leader_joins can reference it.
CREATE TABLE IF NOT EXISTS lesson_exclusions (
    id     INTEGER NOT NULL PRIMARY KEY,
    reason TEXT    DEFAULT NULL
);

-- active_song_leader_joins: filtered view of song_leader_joins that excludes any
-- lead whose id appears in lesson_exclusions. Use this everywhere instead of the
-- raw table so that exclusions are automatically honoured by all queries and views.
CREATE VIEW IF NOT EXISTS active_song_leader_joins AS
SELECT slj.*
FROM song_leader_joins AS slj
LEFT JOIN lesson_exclusions AS le ON le.id = slj.id
WHERE le.id IS NULL;

CREATE VIEW IF NOT EXISTS song_details AS
SELECT
	COALESCE(song_id, 0) AS song_id,
	COALESCE(page_num, '') AS page_num,
	COALESCE(keys, '') AS keys,
	COALESCE(times, '') AS times,
	COALESCE(songs.title, '') AS song_title,
	COALESCE(songs.meter, '') AS song_meter,
	COALESCE(music_attribution, '') AS music_attribution,
	COALESCE(words_attribution, '') AS words_attribution
FROM book_song_joins
LEFT JOIN songs ON songs.id = book_song_joins.song_id
WHERE book_id = 2;

CREATE VIEW IF NOT EXISTS "minutes_expanded" AS
SELECT
	CAST(COALESCE(lna.name, leaders.name, '') AS TEXT) AS leader,
	COALESCE(bsj.page_num, '') AS song_page_number,
	COALESCE(songs.title, '') AS song_title,
	COALESCE(minutes."Name", '') AS minutes_name,
	COALESCE(minutes."Year", '') AS minutes_year,
	COALESCE(minutes."Date", '') AS minutes_date,
	COALESCE(minutes."Location", '') AS minues_location,
	COALESCE(bsj.keys, '') AS song_keys,
	COALESCE(songs.music_attribution, '') AS song_tune_by,
	COALESCE(bsj.words_attribution, '') AS song_words_by,
	COALESCE(leaders.lesson_count, 0) AS leader_total_num_lessons,
	COALESCE(leaders.top20_count, 0) AS leader_num_in_the_top_20,
	COALESCE(locations.name, '') AS location_name,
	COALESCE(locations.state_province, '') AS location_state,
	COALESCE(locations.city, '') AS location_city,
	COALESCE(locations.country, '') AS location_country,
	COALESCE(locations.postal_code, 0) AS location_zip_code
FROM song_leader_joins AS slj
LEFT JOIN minutes ON slj.minutes_id = minutes.id
LEFT JOIN leaders ON slj.leader_id = leaders.id
LEFT JOIN (SELECT alias, MIN(name) AS name FROM leader_name_aliases WHERE leader_id IS NOT NULL GROUP BY alias) AS lna ON lna.alias = leaders.name
LEFT JOIN songs	ON slj.song_id = songs.id
LEFT JOIN minutes_location_joins AS mlj ON slj.minutes_id = mlj.minutes_id
LEFT JOIN locations ON mlj.location_id = locations.id
LEFT JOIN book_song_joins AS bsj ON slj.song_id = bsj.song_id;

CREATE VIEW IF NOT EXISTS "lesson_details" AS
SELECT
	leaders.id,
	CAST(COALESCE(lna.name, leaders.name, '') AS TEXT) AS name,
	COALESCE(lss.lesson_count, 0) AS leader_lesson_count,
	COALESCE(lss.lesson_rank, 0) AS leader_lesson_rank,
	COALESCE(bsj.page_num, '') AS song_page,
	COALESCE(songs.title, '') AS song_title,
	COALESCE(leaders.lesson_count, 0) AS leader_total_lesson_count,
	COALESCE(songs.meter, '') AS song_meter,
	COALESCE(bsj.keys, '') AS song_keys,
	COALESCE(songs.music_attribution, '') AS song_music_attribution,
	COALESCE(bsj.words_attribution, '') AS song_words_attribution
FROM leaders
LEFT JOIN song_leader_joins AS slj ON leaders.id = slj.leader_id
LEFT JOIN songs ON slj.song_id = songs.id
LEFT JOIN leader_song_stats AS lss ON (slj.leader_id = lss.leader_id AND songs.id = lss.song_id)
LEFT JOIN book_song_joins AS bsj ON songs.id = bsj.song_id
LEFT JOIN (SELECT alias, MIN(name) AS name FROM leader_name_aliases WHERE leader_id IS NOT NULL GROUP BY alias) AS lna ON lna.alias = leaders.name
WHERE bsj.book_id = 2;

CREATE VIEW IF NOT EXISTS "singing_details" AS
SELECT
	minutes.id AS minutes_id,
	COALESCE(minutes."Name", '') AS minutes_name,
	COALESCE(minutes."Location", '') AS minutes_location,
	COALESCE(minutes."Date" , '') AS minutes_date,
	COALESCE(minutes."Year", 0) AS minutes_year,
	COALESCE(minutes."Minutes", '') AS minutes_body,
	COALESCE(singings.name, '') AS singing,
	COALESCE(locations.name, '') AS location_name,
	COALESCE(locations.country, '') AS location_country,
	COALESCE(locations.state_province, '') AS location_state_province,
	COALESCE(locations.city, '') AS location_city
FROM minutes
LEFT JOIN minutes_singing_joins AS msj ON minutes.id = msj.minutes_id
LEFT JOIN minutes_location_joins AS mlg ON minutes.id = mgl.minutes_id
LEFT JOIN singings ON mlg.singing_id = singings.id
LEFT JOIN locations ON mlg.location_id = locations.id;

DROP VIEW IF EXISTS singing_lessons;
CREATE VIEW IF NOT EXISTS singing_lessons AS
SELECT
	CAST(ROW_NUMBER() OVER (PARTITION BY slj.minutes_id ORDER BY slj.id) AS INTEGER) AS sequence_number,
	COALESCE(slj.lesson_id, 0) AS lesson_id,
	COALESCE(m."Name", '') AS singing_name,
	m.Year AS singing_year,
	CAST(COALESCE(lna.name, l.name, '') AS TEXT) AS singer_name,
	COALESCE(bsj.page_num, '') AS song_page_number,
	COALESCE(s.title, '') AS song_name,
	COALESCE(bsj.keys, '') AS song_key
FROM song_leader_joins AS slj
JOIN minutes AS m ON slj.minutes_id = m.id
JOIN leaders AS l ON slj.leader_id = l.id
LEFT JOIN (SELECT alias, MIN(name) AS name FROM leader_name_aliases WHERE leader_id IS NOT NULL GROUP BY alias) AS lna ON lna.alias = l.name
JOIN songs AS s ON slj.song_id = s.id
JOIN (SELECT song_id, MIN(page_num) AS page_num, MIN(keys) AS keys FROM book_song_joins WHERE book_id = 2 GROUP BY song_id) AS bsj ON bsj.song_id = s.id;

CREATE VIEW IF NOT EXISTS singing_info AS
SELECT
	m.id AS minutes_id,
	COALESCE(slj.lesson_id, 0) AS lesson_id,
	CAST(COALESCE(MIN(m."Date"), '') AS TEXT) AS singing_date,
	COALESCE(m."Name", '') AS singing_name,
	COALESCE(loc.state_province, '') AS singing_state,
	COALESCE(m."Location", '') AS singing_location,
	COUNT(slj.id) AS number_of_lessons,
	COUNT(DISTINCT slj.leader_id) AS number_of_leaders
FROM minutes AS m
LEFT JOIN song_leader_joins AS slj ON m.id = slj.minutes_id
LEFT JOIN minutes_location_joins AS mlj ON m.id = mlj.minutes_id
LEFT JOIN locations AS loc ON mlj.location_id = loc.id
GROUP BY m.id;

CREATE VIEW IF NOT EXISTS song_leader_stats AS
SELECT
	CAST(COALESCE(lna.name, leaders.name, '') AS TEXT) AS name,
	bsj.page_num,
	lss.lesson_count AS count,
	MAX(m.Year) - MIN(m.Year) AS num_years,
	CASE WHEN MAX(m.Year) >= (SELECT MAX(Year) FROM minutes) - 1 THEN 1 ELSE 0 END AS led_in_last_year
FROM leader_song_stats AS lss
JOIN leaders ON lss.leader_id = leaders.id
LEFT JOIN (SELECT alias, MIN(name) AS name FROM leader_name_aliases WHERE leader_id IS NOT NULL GROUP BY alias) AS lna ON lna.alias = leaders.name
JOIN songs ON lss.song_id = songs.id
JOIN book_song_joins AS bsj ON songs.id = bsj.song_id AND bsj.book_id = 2
JOIN song_leader_joins AS slj ON slj.leader_id = lss.leader_id AND slj.song_id = lss.song_id
JOIN minutes AS m ON slj.minutes_id = m.id
GROUP BY lss.leader_id, bsj.page_num;

CREATE VIEW IF NOT EXISTS leader_minutes AS
SELECT
	COALESCE(l.id, 0) AS leader_id,
	CAST(COALESCE(lna.name, l.name, '') AS TEXT) AS leader_name,
	COALESCE(slj.minutes_id, 0) AS minutes_id
FROM song_leader_joins AS slj
JOIN leaders AS l ON slj.leader_id = l.id
LEFT JOIN (SELECT alias, MIN(name) AS name FROM leader_name_aliases WHERE leader_id IS NOT NULL GROUP BY alias) AS lna ON lna.alias = l.name;

CREATE VIEW IF NOT EXISTS leader_details AS
SELECT
	CAST(COALESCE(lna.name, leaders.name, '') AS TEXT) AS leader_name,
	COALESCE(leaders.lesson_count, '') AS leader_total_num_leads,
	COALESCE(songs.title, '') AS song_title,
	COALESCE(bsj.page_num, '') AS page_number,
	COALESCE(lss.lesson_count, '') AS song_num_leads
FROM leaders
JOIN leader_song_stats AS lss ON leaders.id = lss.leader_id
JOIN songs ON songs.id = lss.song_id
LEFT JOIN book_song_joins AS bsj ON songs.id = bsj.song_id
LEFT JOIN (SELECT alias, MIN(name) AS name FROM leader_name_aliases WHERE leader_id IS NOT NULL GROUP BY alias) AS lna ON lna.alias = leaders.name;

CREATE VIEW IF NOT EXISTS leader_profiles AS
SELECT
	leaders.id AS leader_id,
	CAST(COALESCE(lna.name, leaders.name, '') AS TEXT) AS name,
	COALESCE(leaders.lesson_count, 0) AS lesson_count,
	CAST(COUNT(DISTINCT slj.lesson_id) AS INTEGER) AS unique_lesson_count,
	CAST(COUNT(DISTINCT slj.minutes_id) AS INTEGER) AS singing_count,
	CAST(COALESCE(MIN(m.Year), 0) AS INTEGER) AS first_year,
	CAST(COALESCE(MAX(m.Year), 0) AS INTEGER) AS last_year
FROM leaders
JOIN song_leader_joins AS slj ON leaders.id = slj.leader_id
JOIN minutes AS m ON slj.minutes_id = m.id
LEFT JOIN (SELECT alias, MIN(name) AS name FROM leader_name_aliases WHERE leader_id IS NOT NULL GROUP BY alias) AS lna ON lna.alias = leaders.name
LEFT JOIN leader_name_invalid AS inv ON inv.name = leaders.name
WHERE inv.name IS NULL
GROUP BY leaders.id;

-- leader_name_map: canonical-name → leader_id mapping for sqlc queries.
-- Resolves aliases and filters invalid names.
CREATE VIEW IF NOT EXISTS leader_name_map AS
SELECT
    l.id AS leader_id,
    CAST(COALESCE(lna.name, l.name, '') AS TEXT) AS name
FROM leaders AS l
LEFT JOIN (SELECT alias, MIN(name) AS name FROM leader_name_aliases WHERE leader_id IS NOT NULL GROUP BY alias) AS lna ON lna.alias = l.name
LEFT JOIN leader_name_invalid AS inv ON inv.name = l.name
WHERE inv.name IS NULL;

-- Indexes for query performance (not in embedded db file)
CREATE INDEX IF NOT EXISTS leaders_name ON leaders(name);
CREATE INDEX IF NOT EXISTS slj_leader_minutes ON song_leader_joins(leader_id, minutes_id);
CREATE INDEX IF NOT EXISTS slj_minutes_leader ON song_leader_joins(minutes_id, leader_id);

-- leader_song_stats: join by leader+song (NeverLed, LeaderFootsteps, song_leader_stats view)
CREATE INDEX IF NOT EXISTS lss_leader_song ON leader_song_stats(leader_id, song_id);
-- leader_song_stats: window function PARTITION BY song_id ORDER BY lesson_count DESC (LeaderFootsteps)
CREATE INDEX IF NOT EXISTS lss_song_count_desc ON leader_song_stats(song_id, lesson_count DESC);

-- book_song_joins: covering index for book_id=2 filter used in nearly every query
CREATE INDEX IF NOT EXISTS bsj_book_song_cover ON book_song_joins(book_id, song_id, page_num, keys);

-- song_stats: covering index for SUM(lesson_count) GROUP BY song_id (NeverLed, NeverSung, TheUnfamilarHits)
CREATE INDEX IF NOT EXISTS ss_song_id ON song_stats(song_id, lesson_count);
-- song_stats: year filter for GloballyPopularForYears
CREATE INDEX IF NOT EXISTS ss_year_song ON song_stats(year, song_id, lesson_count);

-- minutes_location_joins: join on minutes_id (LocallyPopular, singing_info, minutes_expanded views)
CREATE INDEX IF NOT EXISTS mlj_minutes ON minutes_location_joins(minutes_id, location_id);

-- locations: state filter for LocallyPopular
CREATE INDEX IF NOT EXISTS loc_state ON locations(state_province);

-- song_leader_joins: song_id lookup (NeverSung, PopularSongsInOnesExperience, LocallyPopular)
CREATE INDEX IF NOT EXISTS slj_song_minutes_lesson ON song_leader_joins(song_id, minutes_id, lesson_id);

-- minutes: Year for MAX(Year) correlated subquery in song_leader_stats view
CREATE INDEX IF NOT EXISTS minutes_year ON minutes(Year);

-- song_leader_joins: (leader_id, song_id, minutes_id) covering index for LeaderFootsteps.
-- The modified query joins leader_song_stats with song_leader_joins on both leader_id AND
-- song_id to compute MAX(m.Year). Without this, SQLite uses slj_leader_minutes and scans
-- all lessons for each leader then filters by song_id — O(lessons/leader × songs).
CREATE INDEX IF NOT EXISTS slj_leader_song_minutes ON song_leader_joins(leader_id, song_id, minutes_id);

-- leader_song_attendance: covering index for PopularSongsInOnesExperience, TheUnfamilarHits, NeverSung.
-- The table is built in setup.sql; this index makes leader_id lookups O(songs attended) vs O(all leaders).
CREATE INDEX IF NOT EXISTS lsa_leader_song ON leader_song_attendance(leader_id, song_id, attendance_count);

-- leader_year_stats: pre-aggregated lesson counts per leader per year, analogous to song_stats.
-- Used by TopLeadersByLeads (both the counts and total CTEs) and LeaderShareOfLeads
-- (numerator and denominator). Without this, both queries do a full song_leader_joins scan
-- joined to minutes — O(SLJ) — for every call. With it, year-filtered aggregation is
-- O(leaders × years), matching the performance of GloballyPopularForYears on song_stats.
CREATE TABLE IF NOT EXISTS leader_year_stats (
    leader_id    INTEGER NOT NULL,
    year         INTEGER NOT NULL,
    lesson_count INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (leader_id, year)
);
CREATE INDEX IF NOT EXISTS lys_year_leader ON leader_year_stats(year, leader_id, lesson_count);

-- leader_profiles view: filter invalid names and resolve aliases
CREATE INDEX IF NOT EXISTS inv_name ON leader_name_invalid(name);
CREATE INDEX IF NOT EXISTS lna_alias_name ON leader_name_aliases(alias, name);
-- leader_profiles view: covering index for GROUP BY leader_id with DISTINCT lesson_id and minutes_id
CREATE INDEX IF NOT EXISTS slj_leader_lesson_minutes ON song_leader_joins(leader_id, lesson_id, minutes_id);
