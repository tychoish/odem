-- setup.sql: one-time setup operations that cannot be made idempotent.
-- Run this after views.sql. Re-running will fail; drop the tables first if rebuilding.

-- Deduplicated (leader_id, minutes_id) pairs used for co-attendance computation.
-- song_leader_joins has ~2 rows per (leader, singing); this table deduplicates them.
-- Note: CREATE TABLE ... AS SELECT does not support IF NOT EXISTS in SQLite.
CREATE TABLE leader_singings AS SELECT DISTINCT leader_id, minutes_id FROM song_leader_joins;
CREATE INDEX IF NOT EXISTS ls_minutes_leader ON leader_singings(minutes_id, leader_id);
CREATE INDEX IF NOT EXISTS ls_leader_minutes ON leader_singings(leader_id, minutes_id);

-- Precomputed count of shared singings for every (leader_a, leader_b) pair.
-- Building this takes ~7s but makes SuprisingSingingStrangers run in <1s for typical leaders.
-- Note: same CREATE TABLE ... AS SELECT limitation as above.
CREATE TABLE leader_coattendance AS
SELECT
	a.leader_id AS leader_a_id,
	b.leader_id AS leader_b_id,
	COUNT(*) AS shared_singings
FROM leader_singings a
JOIN leader_singings b ON a.minutes_id = b.minutes_id AND a.leader_id != b.leader_id
GROUP BY a.leader_id, b.leader_id;
CREATE INDEX IF NOT EXISTS lca_a ON leader_coattendance(leader_a_id, leader_b_id);
CREATE INDEX IF NOT EXISTS lca_b ON leader_coattendance(leader_b_id, leader_a_id);

-- Precomputed global song totals used by NeverLed, NeverSung, TheUnfamilarHits.
-- Each of those queries inlines "SELECT song_id, SUM(lesson_count) AS total FROM song_stats GROUP BY song_id";
-- materializing it once here avoids three independent full-table aggregations.
-- Note: same CREATE TABLE ... AS SELECT limitation as above.
CREATE TABLE song_stats_totals AS
    SELECT song_id,
    SUM(lesson_count) AS total
FROM song_stats
GROUP BY song_id;
CREATE INDEX IF NOT EXISTS sst_song_id ON song_stats_totals(song_id);

-- Precomputed per-leader song attendance counts: how many times each song was called
-- at a singing that leader attended. Built from leader_singings (distinct per leader+singing)
-- joined to song_leader_joins (all songs at each singing).
-- Used by PopularSongsInOnesExperience, TheUnfamilarHits, and NeverSung — each of which
-- previously re-derived this by joining leader_singings (or leader_minutes) to song_leader_joins
-- and grouping, costing O(leaders × singings × songs/singing) per query.
-- ~4.1M rows; build takes ~15s but queries using it complete in <1ms.
-- Note: same CREATE TABLE ... AS SELECT limitation as above.
CREATE TABLE leader_song_attendance AS
SELECT
    ls.leader_id,
    slj.song_id,
    COUNT(*) AS attendance_count
FROM leader_singings AS ls
JOIN song_leader_joins AS slj ON slj.minutes_id = ls.minutes_id
GROUP BY ls.leader_id, slj.song_id;
CREATE INDEX IF NOT EXISTS lsa_leader_song ON leader_song_attendance(leader_id, song_id, attendance_count);

-- Seed data: known-invalid leader name strings to filter from leader lookups.
-- Note: leader_name_invalid has no UNIQUE constraint on name, so this INSERT is not
-- idempotent. Adding UNIQUE(name) + INSERT OR IGNORE would fix that if needed.
INSERT INTO leader_name_invalid (name) VALUES
	('A Day That Will Be'),
	('A Founders Lesson'),
	('A Founder’s Lesson'),
	('A Shenandoah Harmony'),
	('Alban’s Chapel'),
	('At John Fawcett’s'),
	('At Old Temple Kirk'),
	('At Rosslyn Chapel'),
	('But The Blood'),
	('Captian Kidd'),
	('Convention-New Mexico'),
	('Dynamics Chapter V. The'),
	('E.L. King'),
	('Holly Mana'),
	('In The'),
	('Keys Chapter IV'),
	('King Henry VIII'),
	('L.M.'),
	('Let Us Sing'),
	('McCool'),
	('Melodics Chapter III'),
	('N-Z'),
	('Old Days'),
	('PVADS'),
	('S.M.'),
	 -- likely, just a guess
	('South Hampton'),
	('Tell Me Of The Angels'),
	('Thankful Heart'),
	('The Alexander'),
	('The Avery Family'),
	('The Beasley Family'),
	('The Beasley'),
	('The Brady'),
	('The Bristol'),
	('The Butterfly'),
	('The Chairladies'),
	('The Chicago'),
	('The Cork Singers'),
	('The Creel'),
	('The Dallas'),
	('The Danby'),
	('The Dayton Harmonist'),
	('The Eddins'),
	('The Founders'),
	('The Founders’'),
	('The Founder’s Lesson'),
	('The Friday'),
	('The Geneva'),
	('The Gilmore Family'),
	('The Great Roll Call'),
	('The Green Family'),
	('The Harper'),
	('The Harris Family'),
	('The IV'),
	('The Introductory Lesson'),
	('The Ivey Family'),
	('The Jamulus Singers'),
	('The Kerr Family'),
	('The Kerr'),
	('The Lacy Family-Reba Windom'),
	('The Ladykillers'),
	('The Landing'),
	('The Locating'),
	('The London'),
	('The Lonnie Rogers'),
	('The Man With'),
	('The Monday'),
	('The Montréal'),
	('The Munich'),
	('The New Year’s Day'),
	('The Oliver'),
	('The Olney Hymns'),
	('The Registration'),
	('The Rogers Children'),
	('The Rogers'),
	('The Roots'),
	('The Sheppards'),
	('The Sherbrooke'),
	('The Silverton Grange'),
	('The Supertonic'),
	('The Texas'),
	('The Trumpet'),
	('The Tuesday Night Singers'),
	('The Union'),
	('The Wakefield Singers'),
	('The Wallowa Valley'),
	('The Youth Boys'),
	('Three of these leads should be attributed to Tommy Schultz: 2012 Columbia singing, 2013 Missouri State Convention, 2016 Columbia Singing. Thank you!'),
	('U. S.'),
	('UKSH'),
	('USA. Peter'),
	('USA. The Winchester Singers'),
	('Us Sing'),
	('Zion Hill'),
	('[No such leader, and attributed song was discussed in a Camp FaSoLa class but not minuted as having been sung]'),
	('[No such leader, and attributed song was not sung]'),
	('[No such leader, and attributed song was not sung—parsing error of highway name]'),
	('[No such leader, and attributed songs were not sung—code is parsing building address as song number]'),
	('[No such leader, and song in question was discussed but not specified to have been song]'),
	('[No such leader, and song was NOT sung—error parsing Bible chapter #]'),
	('[No such leader, and song was not sung—incorrect parsing of chapter/verse from Book of Isaiah]'),
	('[No such leader; sung WAS sung]'),
	('[No such leader]'),
	('[No such leader—but song WAS led by other attributed leader]'),
	('[No such leader—parsing error]'),
	('[No such singer, and attributed song was not sung]'),
	('[Parsing error—no such leader. Attributed song was led by Alice Ann Vaughan Borge and David Ivey only]'),
	('[Various—can these be corrected and parsed individually?]'),
	('[Venue name incorrectly parsed as leader; song WAS sung by other attributed leader]');
