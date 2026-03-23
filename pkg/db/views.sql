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
