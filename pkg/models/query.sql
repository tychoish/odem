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

