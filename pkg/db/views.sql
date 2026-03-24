CREATE VIEW song_details AS
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

CREATE VIEW "minutes_expanded" AS
SELECT
	COALESCE(leaders.name, '') AS leader,
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
LEFT JOIN songs	ON slj.song_id = songs.id
LEFT JOIN minutes_location_joins AS mlj ON slj.minutes_id = mlj.minutes_id
LEFT JOIN locations ON mlj.location_id = locations.id
LEFT JOIN book_song_joins AS bsj ON slj.song_id = bsj.song_id;

CREATE VIEW "lesson_details" AS
SELECT
	leaders.id,
	COALESCE(leaders.name, '') AS name,
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
WHERE bsj.book_id = 2;

CREATE VIEW "singing_details" AS
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

CREATE VIEW singing_lessons AS
SELECT
	CAST(ROW_NUMBER() OVER (PARTITION BY slj.minutes_id ORDER BY slj.id) AS INTEGER) AS sequence_number,
	COALESCE(slj.lesson_id, 0) AS lesson_id,
	COALESCE(m."Name", '') AS singing_name,
	COALESCE(l.name, '') AS singer_name,
	COALESCE(bsj.page_num, '') AS song_page_number,
	COALESCE(s.title, '') AS song_name,
	COALESCE(bsj.keys, '') AS song_key
FROM song_leader_joins AS slj
JOIN minutes AS m ON slj.minutes_id = m.id
JOIN leaders AS l ON slj.leader_id = l.id
JOIN songs AS s ON slj.song_id = s.id
JOIN book_song_joins AS bsj ON bsj.song_id = s.id AND bsj.book_id = 2;

CREATE VIEW singing_info AS
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

CREATE VIEW song_leader_stats AS
SELECT
	leaders.name,
	bsj.page_num,
	lss.lesson_count AS count,
	MAX(m.Year) - MIN(m.Year) AS num_years,
	CASE WHEN MAX(m.Year) >= (SELECT MAX(Year) FROM minutes) - 1 THEN 1 ELSE 0 END AS led_in_last_year
FROM leader_song_stats AS lss
JOIN leaders ON lss.leader_id = leaders.id
JOIN songs ON lss.song_id = songs.id
JOIN book_song_joins AS bsj ON songs.id = bsj.song_id AND bsj.book_id = 2
JOIN song_leader_joins AS slj ON slj.leader_id = lss.leader_id AND slj.song_id = lss.song_id
JOIN minutes AS m ON slj.minutes_id = m.id
GROUP BY lss.leader_id, bsj.page_num;

CREATE VIEW leader_minutes AS
SELECT
	COALESCE(l.id, 0) AS leader_id,
	COALESCE(l.name, '') AS leader_name,
	COALESCE(slj.minutes_id, 0) AS minutes_id
FROM song_leader_joins AS slj
JOIN leaders AS l ON slj.leader_id = l.id;

-- Indexes for query performance (not in embedded db file)
CREATE INDEX leaders_name ON leaders(name);
CREATE INDEX slj_leader_minutes ON song_leader_joins(leader_id, minutes_id);
CREATE INDEX slj_minutes_leader ON song_leader_joins(minutes_id, leader_id);

-- Deduplicated (leader_id, minutes_id) pairs used for co-attendance computation.
-- song_leader_joins has ~2 rows per (leader, singing); this table deduplicates them.
CREATE TABLE leader_singings AS SELECT DISTINCT leader_id, minutes_id FROM song_leader_joins;
CREATE INDEX ls_minutes_leader ON leader_singings(minutes_id, leader_id);
CREATE INDEX ls_leader_minutes ON leader_singings(leader_id, minutes_id);

-- Precomputed count of shared singings for every (leader_a, leader_b) pair.
-- Building this takes ~7s but makes SuprisingSingingStrangers run in <1s for typical leaders.
CREATE TABLE leader_coattendance AS
SELECT
	a.leader_id AS leader_a_id,
	b.leader_id AS leader_b_id,
	COUNT(*) AS shared_singings
FROM leader_singings a
JOIN leader_singings b ON a.minutes_id = b.minutes_id AND a.leader_id != b.leader_id
GROUP BY a.leader_id, b.leader_id;

CREATE INDEX lca_a ON leader_coattendance(leader_a_id, leader_b_id);
CREATE INDEX lca_b ON leader_coattendance(leader_b_id, leader_a_id);

CREATE VIEW leader_details AS
SELECT
	COALESCE(leaders.name, '') AS leader_name,
	COALESCE(leaders.lesson_count, '') AS leader_total_num_leads,
	COALESCE(songs.title, '') AS song_title,
	COALESCE(bsj.page_num, '') AS page_number,
	COALESCE(lss.lesson_count, '') AS song_num_leads
FROM leaders
JOIN leader_song_stats AS lss ON leaders.id = lss.leader_id
JOIN songs ON songs.id = lss.song_id
LEFT JOIN book_song_joins AS bsj ON songs.id = bsj.song_id;
