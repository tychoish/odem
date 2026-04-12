package db

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/tychoish/fun/srv"
	"github.com/tychoish/odem/pkg/models"
	_ "modernc.org/sqlite"
)

const (
	testLeader       = "Bud Oliver"
	testSocialLeader = "Rose Altha Taylor" // fewer singings — faster for social graph queries
	testSong         = "32t"
	testSinging      = "Liberty Church"
)

func testConn(t *testing.T) (*Connection, context.Context) {
	t.Helper()
	ctx, cancel := context.WithCancel(srv.WithCleanup(context.Background()))
	t.Cleanup(cancel)
	conn, err := Connect(ctx)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	return conn, ctx
}

func TestAllSongDetails(t *testing.T) {
	conn, ctx := testConn(t)
	count := 0
	for _, err := range conn.AllSongDetails(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		count++
		break // unbounded query — only check first result
	}
	if count == 0 {
		t.Error("AllSongDetails: expected at least one result")
	}
}

func TestAllLeaderNames(t *testing.T) {
	conn, ctx := testConn(t)
	count := 0
	for _, err := range conn.AllLeaderNames(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		count++
		break // unbounded query — only check first result
	}
	if count == 0 {
		t.Error("AllLeaderNames: expected at least one result")
	}
}

func TestMostLeadSongs(t *testing.T) {
	conn, ctx := testConn(t)
	count := 0
	for _, err := range conn.MostLedSongs(ctx, testLeader, 5) {
		if err != nil {
			t.Fatal(err)
		}
		count++
	}
	if count == 0 {
		t.Errorf("MostLeadSongs(%q): expected at least one result", testLeader)
	}
}

func TestTopLeadersOfSong(t *testing.T) {
	conn, ctx := testConn(t)
	count := 0
	for _, err := range conn.TopLeadersOfSong(ctx, testSong, 5) {
		if err != nil {
			t.Fatal(err)
		}
		count++
	}
	if count == 0 {
		t.Errorf("TopLeadersOfSong(%q): expected at least one result", testSong)
	}
}

func TestAllLessons(t *testing.T) {
	conn, ctx := testConn(t)
	count := 0
	for _, err := range conn.AllLessons(ctx, testLeader) {
		if err != nil {
			t.Fatal(err)
		}
		count++
		break // unbounded query — only check first result
	}
	if count == 0 {
		t.Errorf("AllLessons(%q): expected at least one result", testLeader)
	}
}

func TestLeaderLeadHistory(t *testing.T) {
	conn, ctx := testConn(t)
	count := 0
	for _, err := range conn.LeaderLeadHistory(ctx, testLeader, 20) {
		if err != nil {
			t.Fatal(err)
		}
		count++
		break // unbounded query — only check first result
	}
	if count == 0 {
		t.Errorf("LeaderLeadHistory(%q): expected at least one result", testLeader)
	}
}

func TestLeaderSingingsAttended(t *testing.T) {
	conn, ctx := testConn(t)
	count := 0
	for row, err := range conn.LeaderSingingsAttended(ctx, testLeader, 5) {
		if err != nil {
			t.Fatal(err)
		}
		if row.NumberOfLeaders <= 0 {
			t.Errorf("LeaderSingingsAttended(%q): expected number_of_leaders > 0, got %d", testLeader, row.NumberOfLeaders)
		}
		if row.LeaderLeadCount <= 0 {
			t.Errorf("LeaderSingingsAttended(%q): expected leader_lead_count > 0, got %d", testLeader, row.LeaderLeadCount)
		}
		count++
	}
	if count == 0 {
		t.Errorf("LeaderSingingsAttended(%q): expected at least one result", testLeader)
	}
}

func TestSingingLessons(t *testing.T) {
	conn, ctx := testConn(t)
	count := 0
	for _, err := range conn.SingingLessons(ctx, testSinging) {
		if err != nil {
			t.Fatal(err)
		}
		count++
		break // unbounded query — only check first result
	}
	if count == 0 {
		t.Errorf("SingingLessons(%q): expected at least one result", testSinging)
	}
}

func TestAllSingings(t *testing.T) {
	conn, ctx := testConn(t)
	count := 0
	for _, err := range conn.AllSingings(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		count++
		break // unbounded query — only check first result
	}
	if count == 0 {
		t.Error("AllSingings: expected at least one result")
	}
}

func TestSingingBuddies(t *testing.T) {
	conn, ctx := testConn(t)
	count := 0
	for _, err := range conn.SingingBuddies(ctx, testSocialLeader, 5) {
		if err != nil {
			t.Fatal(err)
		}
		count++
	}
	if count == 0 {
		t.Errorf("SingingBuddies(%q): expected at least one result", testSocialLeader)
	}
}

func TestPopularSongsInOnesExperience(t *testing.T) {
	conn, ctx := testConn(t)
	count := 0
	for _, err := range conn.PopularAsObserved(ctx, testSocialLeader, 5) {
		if err != nil {
			t.Fatal(err)
		}
		count++
	}
	if count == 0 {
		t.Errorf("PopularSongsInOnesExperience(%q): expected at least one result", testSocialLeader)
	}
}

func TestSingingStrangers(t *testing.T) {
	conn, ctx := testConn(t)
	count := 0
	for _, err := range conn.SingingStrangers(ctx, testSocialLeader, 5) {
		if err != nil {
			t.Fatal(err)
		}
		count++
	}
	if count == 0 {
		t.Errorf("SingingStrangers(%q): expected at least one result", testSocialLeader)
	}
}

func TestSingingStrangersVariesByInput(t *testing.T) {
	// Regression: the old query used a slow NOT IN subquery on leader_minutes that
	// caused SQLite to short-circuit and return the same alphabetical results for
	// every input. Results are now ordered by mutual-connection count descending.
	//
	// "Allison Whitener" is a frequent co-attendee of testLeader (so not a stranger
	// to them) but has never co-attended with testSocialLeader. With mutual-connection
	// ordering she appears in the top ~30 results for testSocialLeader.
	const sentinel = "Allison Whitener"

	conn, ctx := testConn(t)

	contains := func(name string, limit int, target string) bool {
		t.Helper()
		for s, err := range conn.SingingStrangers(ctx, name, limit) {
			if err != nil {
				t.Fatalf("SingingStrangers(%q): %v", name, err)
			}
			if s.Name == target {
				return true
			}
		}
		return false
	}

	if contains(testLeader, 40, sentinel) {
		t.Errorf("SingingStrangers(%q): %q should not be a stranger (they are frequent co-attendees)", testLeader, sentinel)
	}
	if !contains(testSocialLeader, 40, sentinel) {
		t.Errorf("SingingStrangers(%q): %q should be a stranger (they have never co-attended)", testSocialLeader, sentinel)
	}
}

func TestUnfamiliarHits(t *testing.T) {
	conn, ctx := testConn(t)
	count := 0
	for _, err := range conn.TheUnfamilarHits(ctx, testLeader, 5) {
		if err != nil {
			t.Fatal(err)
		}
		count++
	}
	if count == 0 {
		t.Errorf("TheUnfamilarHits(%q): expected at least one result", testLeader)
	}
}

func TestUnfamiliarHitsExcludesFamiliarSongs(t *testing.T) {
	// Regression: the old query used leader_song_stats (lead count) to measure
	// exposure. Because leaders lead only a small fraction of the book, almost
	// every song had count=0 and results were identical to the global most-popular
	// list for any input — familiar songs appeared as "unfamiliar".
	//
	// The fix uses attendance exposure (times the song was called at a singing
	// the leader attended). Top-attended songs must not appear in the top
	// unfamiliar hits.
	const limit = 15

	conn, ctx := testConn(t)

	experienced := make(map[string]bool)
	for song, err := range conn.PopularAsObserved(ctx, testLeader, limit) {
		if err != nil {
			t.Fatalf("PopularSongsInOnesExperience(%q): %v", testLeader, err)
		}
		experienced[song.PageNum] = true
	}
	if len(experienced) == 0 {
		t.Fatalf("PopularSongsInOnesExperience(%q): no results; cannot run regression check", testLeader)
	}

	for song, err := range conn.TheUnfamilarHits(ctx, testLeader, limit) {
		if err != nil {
			t.Fatalf("TheUnfamilarHits(%q): %v", testLeader, err)
		}
		if experienced[song.PageNum] {
			t.Errorf("TheUnfamilarHits(%q): %s (%q) is highly familiar (top attended) but appears in unfamiliar hits",
				testLeader, song.PageNum, song.SongTitle)
		}
	}
}

func TestLeaderFootsteps(t *testing.T) {
	conn, ctx := testConn(t)
	currentYear := time.Now().Year()
	count := 0
	for row, err := range conn.LeaderFootsteps(ctx, testLeader, 5) {
		if err != nil {
			t.Fatal(err)
		}
		if row.TheirLeadCount <= 0 {
			t.Errorf("LeaderFootsteps(%q): TheirLeadCount should be > 0, got %d", testLeader, row.TheirLeadCount)
		}
		if row.SelfLeadCount <= 0 {
			t.Errorf("LeaderFootsteps(%q): MyLeadCount should be > 0, got %d", testLeader, row.SelfLeadCount)
		}
		if row.TheirLastLeadYear < 1995 || row.TheirLastLeadYear > currentYear {
			t.Errorf("LeaderFootsteps(%q): TheirLastLeadYear out of range for %q: %d", testLeader, row.LeaderName, row.TheirLastLeadYear)
		}
		count++
	}
	if count == 0 {
		t.Errorf("LeaderFootsteps(%q): expected at least one result", testLeader)
	}
}

func TestSingersConnectedness(t *testing.T) {
	conn, ctx := testConn(t)
	v, err := conn.SingersConnectedness(ctx, testLeader)
	if err != nil {
		t.Fatal(err)
	}
	if v == nil || *v <= 0 || *v > 1 {
		t.Errorf("SingersConnectedness(%q): expected ratio in (0,1], got %v", testLeader, v)
	}
}

func TestLeaderShareOfLeads(t *testing.T) {
	conn, ctx := testConn(t)
	v, err := conn.LeaderShareOfLeads(ctx, testLeader, 8)
	if err != nil {
		t.Fatal(err)
	}
	if v == nil || *v <= 0 || *v > 1 {
		t.Errorf("LeaderShareOfLeads(%q): expected ratio in (0,1], got %v", testLeader, v)
	}
}

func TestLeaderShareOfLeadsWithYear(t *testing.T) {
	conn, ctx := testConn(t)
	v, err := conn.LeaderShareOfLeads(ctx, "Sam Kleinman", 8, 2023, 2024)
	if err != nil {
		t.Fatal(err)
	}
	if v == nil || *v < 0 || *v > 1 {
		t.Errorf("LeaderShareOfLeads(Sam Kleinman, 2023, 2024): expected ratio in [0,1], got %v", v)
	}
}

func TestTopLeadersByLeads(t *testing.T) {
	conn, ctx := testConn(t)
	currentYear := time.Now().Year()
	var prev models.LeaderLeadCount
	count := 0
	for row, err := range conn.TopLeadersByLeads(ctx, 5) {
		if err != nil {
			t.Fatal(err)
		}
		if row.Count <= 0 {
			t.Errorf("TopLeadersByLeads: expected positive lead count, got %d for %q", row.Count, row.Name)
		}
		if row.Percentage <= 0 || row.Percentage > 1 {
			t.Errorf("TopLeadersByLeads: pct out of range for %q: %v", row.Name, row.Percentage)
		}
		if row.LastLeadYear < 1995 || row.LastLeadYear > currentYear {
			t.Errorf("TopLeadersByLeads: last_lead_year out of range for %q: %d", row.Name, row.LastLeadYear)
		}
		if count > 0 && row.RunningTotal < prev.RunningTotal {
			t.Errorf("TopLeadersByLeads: running_total decreased from %v to %v between %q and %q",
				prev.RunningTotal, row.RunningTotal, prev.Name, row.Name)
		}
		if count > 0 && row.Percentage > prev.Percentage {
			t.Errorf("TopLeadersByLeads: pct increased from %v to %v (not sorted DESC) between %q and %q",
				prev.Percentage, row.Percentage, prev.Name, row.Name)
		}
		prev = row
		count++
	}
	if count == 0 {
		t.Error("TopLeadersByLeads: expected at least one result")
	}
}

func TestTopLeadersByLeadsWithYear(t *testing.T) {
	conn, ctx := testConn(t)
	count := 0
	for row, err := range conn.TopLeadersByLeads(ctx, 5, 2023) {
		if err != nil {
			t.Fatal(err)
		}
		if row.LastLeadYear != 2023 {
			t.Errorf("TopLeadersByLeads(2023): last_lead_year should be 2023, got %d for %q", row.LastLeadYear, row.Name)
		}
		count++
	}
	if count == 0 {
		t.Error("TopLeadersByLeads(2023): expected at least one result")
	}
}

func TestLeaderFavoriteKey(t *testing.T) {
	conn, ctx := testConn(t)
	count := 0
	for kv, err := range conn.LeaderFavoriteKey(ctx, testLeader, 5) {
		if err != nil {
			t.Fatal(err)
		}
		if kv.Key == "" {
			t.Errorf("LeaderFavoriteKey(%q): expected non-empty key", testLeader)
		}
		if kv.Leads <= 0 {
			t.Errorf("LeaderFavoriteKey(%q): expected positive count for key %q, got %d", testLeader, kv.Key, kv.Leads)
		}
		count++
	}
	if count == 0 {
		t.Errorf("LeaderFavoriteKey(%q): expected at least one result", testLeader)
	}
}

func TestNewLeadersByYear(t *testing.T) {
	conn, ctx := testConn(t)
	for _, year := range []int{2010, 2023} {
		count := 0
		for row, err := range conn.NewLeadersByYear(ctx, year, 10) {
			if err != nil {
				t.Fatalf("NewLeadersByYear(%d): %v", year, err)
			}
			if row.Leader == "" {
				t.Errorf("NewLeadersByYear(%d): expected non-empty leader name", year)
			}
			count++
		}
		if count == 0 {
			t.Errorf("NewLeadersByYear(%d): expected at least one result", year)
		}
	}
}

func TestSongsByKey(t *testing.T) {
	conn, ctx := testConn(t)

	t.Run("AllTime", func(t *testing.T) {
		var rows []models.LeaderSongRank
		for row, err := range conn.SongsByKey(ctx) {
			if err != nil {
				t.Fatal(err)
			}
			rows = append(rows, row)
		}
		if len(rows) == 0 {
			t.Fatal("SongsByKey: expected at least one result")
		}
		var sum float64
		for _, r := range rows {
			if r.Ratio <= 0 || r.Ratio > 1 {
				t.Errorf("SongsByKey: ratio out of (0,1] for key %q: %v", r.Key, r.Ratio)
			}
			sum += r.Ratio
		}
		if sum < 0.999 || sum > 1.001 {
			t.Errorf("SongsByKey: ratios sum to %v, expected ~1.0", sum)
		}
	})

	t.Run("Year2023", func(t *testing.T) {
		var rows []models.LeaderSongRank
		for row, err := range conn.SongsByKey(ctx, 2023) {
			if err != nil {
				t.Fatal(err)
			}
			rows = append(rows, row)
		}
		if len(rows) == 0 {
			t.Fatal("SongsByKey(2023): expected at least one result")
		}
		var sum float64
		for _, r := range rows {
			if r.Ratio <= 0 || r.Ratio > 1 {
				t.Errorf("SongsByKey(2023): ratio out of (0,1] for key %q: %v", r.Key, r.Ratio)
			}
			sum += r.Ratio
		}
		if sum < 0.999 || sum > 1.001 {
			t.Errorf("SongsByKey(2023): ratios sum to %v, expected ~1.0", sum)
		}
	})
}

func TestLeadersByTop20Leads(t *testing.T) {
	conn, ctx := testConn(t)
	count := 0
	for row, err := range conn.LeadersByTop20Leads(ctx, 20) {
		if err != nil {
			t.Fatal(err)
		}
		if row.NumLeads == "" || row.NumLeads == "0" {
			t.Errorf("LeadersByTop20Leads: expected count > 0, got %q for %q", row.NumLeads, row.Leader)
		}
		count++
	}
	if count == 0 {
		t.Error("LeadersByTop20Leads: expected at least one result")
	}
}

func TestLeaderSingingsPerYear(t *testing.T) {
	conn, ctx := testConn(t)
	count := 0
	for kv, err := range conn.LeaderSingingsPerYear(ctx, testLeader) {
		if err != nil {
			t.Fatal(err)
		}
		if kv.Year == "" {
			t.Errorf("LeaderSingingsPerYear(%q): expected non-empty year key", testLeader)
		}
		if kv.Singings <= 0 {
			t.Errorf("LeaderSingingsPerYear(%q): expected positive count for year %q, got %d", testLeader, kv.Year, kv.Singings)
		}
		count++
	}
	if count == 0 {
		t.Errorf("LeaderSingingsPerYear(%q): expected at least one result", testLeader)
	}
}

func TestLeadersByKey(t *testing.T) {
	conn, ctx := testConn(t)
	const testKey = "A Major"
	count := 0
	for row, err := range conn.LeadersByKey(ctx, testKey, 5) {
		if err != nil {
			t.Fatal(err)
		}
		if row.Leader == "" {
			t.Errorf("LeadersByKey(%q): expected non-empty leader name", testKey)
		}
		c, err2 := strconv.Atoi(row.NumLeads)
		if err2 != nil || c <= 0 {
			t.Errorf("LeadersByKey(%q): expected positive count, got %q", testKey, row.NumLeads)
		}
		count++
	}
	if count == 0 {
		t.Errorf("LeadersByKey(%q): expected at least one result", testKey)
	}
}

func TestAllKeys(t *testing.T) {
	conn, ctx := testConn(t)
	count := 0
	for key, err := range conn.AllKeys(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		if key == "" {
			t.Error("AllKeys: expected non-empty key")
		}
		count++
	}
	if count == 0 {
		t.Error("AllKeys: expected at least one result")
	}
}

func TestPopularSongsByKey(t *testing.T) {
	conn, ctx := testConn(t)
	const testKey = "A Major"
	count := 0
	for row, err := range conn.PopularSongsByKey(ctx, testKey, 5) {
		if err != nil {
			t.Fatal(err)
		}
		if row.PageNum == "" {
			t.Errorf("PopularSongsByKey(%q): expected non-empty page num", testKey)
		}
		if row.SongTitle == "" {
			t.Errorf("PopularSongsByKey(%q): expected non-empty song title", testKey)
		}
		c, err2 := strconv.Atoi(row.NumLeads)
		if err2 != nil || c <= 0 {
			t.Errorf("PopularSongsByKey(%q): expected positive count, got %q", testKey, row.NumLeads)
		}
		count++
	}
	if count == 0 {
		t.Errorf("PopularSongsByKey(%q): expected at least one result", testKey)
	}
}

func TestAllLeaderConnectedness(t *testing.T) {
	conn, ctx := testConn(t)
	count := 0
	for kv, err := range conn.AllLeaderConnectedness(ctx, 20) {
		if err != nil {
			t.Fatal(err)
		}
		if kv.Connectedness < 0 || kv.Connectedness > 1 {
			t.Errorf("AllLeaderConnectedness: ratio out of range for %q: %v", kv.Name, kv.Connectedness)
		}
		count++
		break // unbounded — only check first result
	}
	if count == 0 {
		t.Error("AllLeaderConnectedness: expected at least one result")
	}
}
