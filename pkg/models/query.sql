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
	CAST(CAST(COUNT(DISTINCT b.leader_id) AS REAL) / CAST((SELECT COUNT(*) FROM leaders) AS REAL) AS REAL) AS connectedness
FROM song_leader_joins a
JOIN song_leader_joins b ON b.minutes_id = a.minutes_id AND b.leader_id != a.leader_id
WHERE a.leader_id = (SELECT id FROM leaders WHERE leaders.name = ?);

-- name: GetLeaderTopMajorKey :one
SELECT CAST(COALESCE(bsj.keys, '') AS TEXT) AS top_key, CAST(COUNT(*) AS INTEGER) AS lead_count
FROM song_leader_joins AS slj
JOIN leader_name_map AS lnm ON lnm.leader_id = slj.leader_id
JOIN book_song_joins AS bsj ON slj.song_id = bsj.song_id AND bsj.book_id = 2
WHERE lnm.name = ?
AND bsj.keys LIKE '%Major%' AND bsj.keys NOT LIKE '%Minor%'
GROUP BY bsj.keys
ORDER BY lead_count DESC
LIMIT 1;

-- name: GetLeaderTopMinorKey :one
SELECT CAST(COALESCE(bsj.keys, '') AS TEXT) AS top_key, CAST(COUNT(*) AS INTEGER) AS lead_count
FROM song_leader_joins AS slj
JOIN leader_name_map AS lnm ON lnm.leader_id = slj.leader_id
JOIN book_song_joins AS bsj ON slj.song_id = bsj.song_id AND bsj.book_id = 2
WHERE lnm.name = ?
AND bsj.keys LIKE '%Minor%' AND bsj.keys NOT LIKE '%Major%'
GROUP BY bsj.keys
ORDER BY lead_count DESC
LIMIT 1;

-- name: GetLeaderMajorMinorCounts :one
SELECT
    CAST(COALESCE(SUM(CASE WHEN bsj.keys LIKE '%Major%' THEN 1 ELSE 0 END), 0) AS INTEGER) AS major_count,
    CAST(COALESCE(SUM(CASE WHEN bsj.keys LIKE '%Minor%' THEN 1 ELSE 0 END), 0) AS INTEGER) AS minor_count
FROM song_leader_joins AS slj
JOIN leader_name_map AS lnm ON lnm.leader_id = slj.leader_id
JOIN book_song_joins AS bsj ON slj.song_id = bsj.song_id AND bsj.book_id = 2
WHERE lnm.name = ?;

-- name: GetLeaderTopSingingBuddy :one
SELECT CAST(COALESCE(lna2.name, l2.name, '') AS TEXT) AS buddy_name, CAST(COUNT(DISTINCT slj2.minutes_id) AS INTEGER) AS singing_count
FROM song_leader_joins AS slj
JOIN leader_name_map AS lnm ON lnm.leader_id = slj.leader_id
JOIN song_leader_joins AS slj2 ON slj2.minutes_id = slj.minutes_id AND slj2.leader_id != slj.leader_id
JOIN leaders AS l2 ON l2.id = slj2.leader_id
LEFT JOIN (SELECT alias, MIN(name) AS name FROM leader_name_aliases WHERE leader_id IS NOT NULL GROUP BY alias) AS lna2 ON lna2.alias = l2.name
LEFT JOIN leader_name_invalid AS inv ON inv.name = l2.name
WHERE lnm.name = ?
AND inv.name IS NULL
GROUP BY slj2.leader_id
ORDER BY singing_count DESC
LIMIT 1;

-- name: GetLeaderActiveYears :one
SELECT
    CAST(COALESCE(MIN(m.year), 0) AS INTEGER) AS first_year,
    CAST(COALESCE(MAX(m.year), 0) AS INTEGER) AS last_year,
    CAST(COALESCE(MAX(m.year) - MIN(m.year) + 1, 0) AS INTEGER) AS years_active,
    CAST(CASE WHEN MAX(m.year) >= (SELECT MAX(year) FROM minutes) - 5 THEN 1 ELSE 0 END AS INTEGER) AS is_active
FROM song_leader_joins AS slj
JOIN leader_name_map AS lnm ON lnm.leader_id = slj.leader_id
JOIN minutes AS m ON m.id = slj.minutes_id
WHERE lnm.name = ?;

-- name: GetLeaderTopState :one
SELECT CAST(COALESCE(loc.state_province, '') AS TEXT) AS state, CAST(COUNT(*) AS INTEGER) AS lead_count
FROM song_leader_joins AS slj
JOIN leader_name_map AS lnm ON lnm.leader_id = slj.leader_id
JOIN minutes AS m ON m.id = slj.minutes_id
JOIN minutes_location_joins AS mlj ON mlj.minutes_id = m.id
JOIN locations AS loc ON loc.id = mlj.location_id
WHERE lnm.name = ?
AND loc.state_province != '' AND loc.state_province IS NOT NULL
GROUP BY loc.state_province
ORDER BY lead_count DESC
LIMIT 1;
