-- name: GetLeader :one
SELECT
	COALESCE(name, ''),
	COALESCE(lesson_count, 0),
	COALESCE(top20_count, 0),
	COALESCE(location_count, 0)
FROM leaders
WHERE name = ?;

-- name: GetSong :one
SELECT
	*
FROM song_details
WHERE page_num = ?;

-- name: GetSingerConnectedness :one
SELECT
	CAST(COUNT(DISTINCT b.leader_id) AS REAL) / (SELECT COUNT(*) FROM leaders) AS connectedness
FROM song_leader_joins a
JOIN song_leader_joins b ON b.minutes_id = a.minutes_id AND b.leader_id != a.leader_id
WHERE a.leader_id = (SELECT id FROM leaders WHERE leaders.name = ?);
