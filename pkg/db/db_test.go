package db

import (
	"context"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/tychoish/fun/srv"
	"github.com/tychoish/odem"
	"github.com/tychoish/odem/pkg/models"
	_ "modernc.org/sqlite"
)

const (
	testLeader       = "Bud Oliver"
	testSocialLeader = "Rose Altha Taylor" // fewer singings — faster for social graph queries
	testSong         = "32t"
	testSinging      = "Liberty Church"
)

var (
	dbInitOnce sync.Once
	dbInitErr  error
)

func testConn(t *testing.T) (*Connection, context.Context) {
	t.Helper()
	dbInitOnce.Do(func() {
		// Always rebuild the test DB from the embedded source. The fast path
		// in Init (views.sql only) assumes setup.sql hasn't changed since the
		// DB was last created; that assumption breaks whenever setup.sql gains
		// new tables (e.g. leader_song_attendance). Resetting ensures the test
		// DB is always schema-current. sync.Once bounds the cost to once per
		// `go test` invocation, not once per test function.
		if os.Getenv("ODEM_TEST_RESTART") != "" {
			if err := Reset(); err != nil {
				dbInitErr = err
				return
			}
			dbInitErr = Init(context.Background())
		} else {
			t.Log("skipping test database refresh.")
		}
	})

	if dbInitErr != nil {
		t.Fatalf("Init: %v", dbInitErr)
	}
	ctx, cancel := context.WithCancel(srv.WithCleanup(context.Background()))
	// ManualReloadDB=true tells Connect to skip Init entirely: the dbInitOnce
	// above already performed the one-time setup, exactly as the program calls
	// Init once at startup before opening any connections.
	conf := &odem.Configuration{}
	conf.Settings.ManualReloadDB = true
	ctx = odem.WithConfiguration(ctx, conf)
	t.Cleanup(cancel)
	conn, err := Connect(ctx)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	return conn, ctx
}

// TestSongQueries covers song lookup, lyrics, and text search.
func TestSongQueries(t *testing.T) {
	t.Parallel()
	conn, ctx := testConn(t)

	tests := []struct {
		name string
		fn   func(*testing.T)
	}{
		{"SongsByWord", func(t *testing.T) {
			count := 0
			for result, err := range conn.SongsByWord(ctx, "canaan", 10) {
				if err != nil {
					t.Fatal(err)
				}
				if result.PageNum == "" {
					t.Error("empty page_num")
				}
				if result.MatchLine == "" {
					t.Errorf("no matching line found for %q in page %s", "canaan", result.PageNum)
				}
				count++
			}
			if count == 0 {
				t.Error("expected at least one result")
			}
		}},
		{"SongLyrics", func(t *testing.T) {
			sl, err := conn.SongLyrics(ctx, testSong)
			if err != nil {
				t.Fatal(err)
			}
			if sl.PageNum == "" {
				t.Errorf("empty page_num for %q", testSong)
			}
			if sl.Text == "" {
				t.Errorf("empty text/lyrics for %q", testSong)
			}
		}},
		{"AllSongTexts", func(t *testing.T) {
			count := 0
			for row, err := range conn.AllSongTexts(ctx) {
				if err != nil {
					t.Fatal(err)
				}
				if row.PageNum == "" {
					t.Error("empty page_num")
				}
				count++
				break // unbounded query — only check first result
			}
			if count == 0 {
				t.Error("expected at least one result")
			}
		}},
		{"AllSongDetails", func(t *testing.T) {
			count := 0
			for _, err := range conn.AllSongDetails(ctx) {
				if err != nil {
					t.Fatal(err)
				}
				count++
				break // unbounded query — only check first result
			}
			if count == 0 {
				t.Error("expected at least one result")
			}
		}},
		{"AllKeys", func(t *testing.T) {
			count := 0
			for key, err := range conn.AllKeys(ctx) {
				if err != nil {
					t.Fatal(err)
				}
				if key == "" {
					t.Error("expected non-empty key")
				}
				count++
			}
			if count == 0 {
				t.Error("expected at least one result")
			}
		}},
		{"PopularSongsByKey", func(t *testing.T) {
			const testKey = "A Major"
			count := 0
			for row, err := range conn.PopularSongsByKey(ctx, testKey, 5) {
				if err != nil {
					t.Fatal(err)
				}
				if row.PageNum == "" {
					t.Errorf("empty page num for key %q", testKey)
				}
				if row.SongTitle == "" {
					t.Errorf("empty song title for key %q", testKey)
				}
				c, err2 := strconv.Atoi(row.NumLeads)
				if err2 != nil || c <= 0 {
					t.Errorf("expected positive count, got %q", row.NumLeads)
				}
				count++
			}
			if count == 0 {
				t.Errorf("expected at least one result for key %q", testKey)
			}
		}},
		{"SongsByKey/AllTime", func(t *testing.T) {
			var rows []models.LeaderSongRank
			for row, err := range conn.SongsByKey(ctx) {
				if err != nil {
					t.Fatal(err)
				}
				rows = append(rows, row)
			}
			if len(rows) == 0 {
				t.Fatal("expected at least one result")
			}
			var sum float64
			for _, r := range rows {
				if r.Ratio <= 0 || r.Ratio > 1 {
					t.Errorf("ratio out of (0,1] for key %q: %v", r.Key, r.Ratio)
				}
				sum += r.Ratio
			}
			if sum < 0.999 || sum > 1.001 {
				t.Errorf("ratios sum to %v, expected ~1.0", sum)
			}
		}},
		{"SongsByKey/Year2023", func(t *testing.T) {
			var rows []models.LeaderSongRank
			for row, err := range conn.SongsByKey(ctx, 2023) {
				if err != nil {
					t.Fatal(err)
				}
				rows = append(rows, row)
			}
			if len(rows) == 0 {
				t.Fatal("expected at least one result")
			}
			var sum float64
			for _, r := range rows {
				if r.Ratio <= 0 || r.Ratio > 1 {
					t.Errorf("ratio out of (0,1] for key %q: %v", r.Key, r.Ratio)
				}
				sum += r.Ratio
			}
			if sum < 0.999 || sum > 1.001 {
				t.Errorf("ratios sum to %v, expected ~1.0", sum)
			}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.fn(t)
		})
	}
}

// TestLeaderQueries covers per-leader statistics and history.
func TestLeaderQueries(t *testing.T) {
	t.Parallel()
	conn, ctx := testConn(t)

	tests := []struct {
		name string
		fn   func(*testing.T)
	}{
		{"MostLedSongs", func(t *testing.T) {
			count := 0
			for _, err := range conn.MostLedSongs(ctx, testLeader, 5) {
				if err != nil {
					t.Fatal(err)
				}
				count++
			}
			if count == 0 {
				t.Errorf("expected results for %q", testLeader)
			}
		}},
		{"AllLessons", func(t *testing.T) {
			count := 0
			for _, err := range conn.AllLessons(ctx, testLeader) {
				if err != nil {
					t.Fatal(err)
				}
				count++
				break // unbounded query — only check first result
			}
			if count == 0 {
				t.Errorf("expected results for %q", testLeader)
			}
		}},
		{"LeaderLeadHistory", func(t *testing.T) {
			count := 0
			for _, err := range conn.LeaderLeadHistory(ctx, testLeader, 20) {
				if err != nil {
					t.Fatal(err)
				}
				count++
				break // unbounded query — only check first result
			}
			if count == 0 {
				t.Errorf("expected results for %q", testLeader)
			}
		}},
		{"LeaderSingingsAttended", func(t *testing.T) {
			count := 0
			for row, err := range conn.LeaderSingingsAttended(ctx, testLeader, 5) {
				if err != nil {
					t.Fatal(err)
				}
				if row.NumberOfLeaders <= 0 {
					t.Errorf("expected number_of_leaders > 0, got %d", row.NumberOfLeaders)
				}
				if row.LeaderLeadCount <= 0 {
					t.Errorf("expected leader_lead_count > 0, got %d", row.LeaderLeadCount)
				}
				count++
			}
			if count == 0 {
				t.Errorf("expected results for %q", testLeader)
			}
		}},
		{"LeaderFavoriteKey", func(t *testing.T) {
			count := 0
			for kv, err := range conn.LeaderFavoriteKey(ctx, testLeader, 5) {
				if err != nil {
					t.Fatal(err)
				}
				if kv.Key == "" {
					t.Error("expected non-empty key")
				}
				if kv.Leads <= 0 {
					t.Errorf("expected positive count for key %q, got %d", kv.Key, kv.Leads)
				}
				count++
			}
			if count == 0 {
				t.Errorf("expected results for %q", testLeader)
			}
		}},
		{"LeaderSingingsPerYear", func(t *testing.T) {
			count := 0
			for kv, err := range conn.LeaderSingingsPerYear(ctx, testLeader) {
				if err != nil {
					t.Fatal(err)
				}
				if kv.Year == "" {
					t.Error("expected non-empty year key")
				}
				if kv.Singings <= 0 {
					t.Errorf("expected positive count for year %q, got %d", kv.Year, kv.Singings)
				}
				count++
			}
			if count == 0 {
				t.Errorf("expected results for %q", testLeader)
			}
		}},
		{"SingersConnectedness", func(t *testing.T) {
			v, err := conn.SingersConnectedness(ctx, testLeader)
			if err != nil {
				t.Fatal(err)
			}
			if v == nil || *v <= 0 || *v > 1 {
				t.Errorf("expected ratio in (0,1], got %v", v)
			}
		}},
		{"LeaderShareOfLeads", func(t *testing.T) {
			v, err := conn.LeaderShareOfLeads(ctx, testLeader, 8)
			if err != nil {
				t.Fatal(err)
			}
			if v == nil || *v <= 0 || *v > 1 {
				t.Errorf("expected ratio in (0,1], got %v", v)
			}
		}},
		{"LeaderShareOfLeadsWithYear", func(t *testing.T) {
			v, err := conn.LeaderShareOfLeads(ctx, "Sam Kleinman", 8, 2023, 2024)
			if err != nil {
				t.Fatal(err)
			}
			if v == nil || *v < 0 || *v > 1 {
				t.Errorf("expected ratio in [0,1], got %v", v)
			}
		}},
		{"LeaderFootsteps", func(t *testing.T) {
			currentYear := time.Now().Year()
			count := 0
			for row, err := range conn.LeaderFootsteps(ctx, testLeader, 5) {
				if err != nil {
					t.Fatal(err)
				}
				if row.TheirLeadCount <= 0 {
					t.Errorf("TheirLeadCount should be > 0, got %d", row.TheirLeadCount)
				}
				if row.SelfLeadCount <= 0 {
					t.Errorf("SelfLeadCount should be > 0, got %d", row.SelfLeadCount)
				}
				if row.TheirLastLeadYear < 1995 || row.TheirLastLeadYear > currentYear {
					t.Errorf("TheirLastLeadYear out of range for %q: %d", row.LeaderName, row.TheirLastLeadYear)
				}
				count++
			}
			if count == 0 {
				t.Errorf("expected results for %q", testLeader)
			}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.fn(t)
		})
	}
}

// TestSocialGraph covers co-attendance queries (buddies, strangers, exposure).
func TestSocialGraph(t *testing.T) {
	t.Parallel()
	conn, ctx := testConn(t)

	tests := []struct {
		name string
		fn   func(*testing.T)
	}{
		{"SingingBuddies", func(t *testing.T) {
			count := 0
			for _, err := range conn.SingingBuddies(ctx, testSocialLeader, 5) {
				if err != nil {
					t.Fatal(err)
				}
				count++
			}
			if count == 0 {
				t.Errorf("expected results for %q", testSocialLeader)
			}
		}},
		{"PopularAsObserved", func(t *testing.T) {
			count := 0
			for _, err := range conn.PopularAsObserved(ctx, testSocialLeader, 5) {
				if err != nil {
					t.Fatal(err)
				}
				count++
			}
			if count == 0 {
				t.Errorf("expected results for %q", testSocialLeader)
			}
		}},
		{"SingingStrangers", func(t *testing.T) {
			count := 0
			for _, err := range conn.SingingStrangers(ctx, testSocialLeader, 5) {
				if err != nil {
					t.Fatal(err)
				}
				count++
			}
			if count == 0 {
				t.Errorf("expected results for %q", testSocialLeader)
			}
		}},
		{"UnfamiliarHits", func(t *testing.T) {
			count := 0
			for _, err := range conn.TheUnfamilarHits(ctx, testLeader, 5) {
				if err != nil {
					t.Fatal(err)
				}
				count++
			}
			if count == 0 {
				t.Errorf("expected results for %q", testLeader)
			}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.fn(t)
		})
	}
}

// TestGlobalRankings covers leaderboard, aggregate, and singing-level queries.
func TestGlobalRankings(t *testing.T) {
	t.Parallel()
	conn, ctx := testConn(t)
	currentYear := time.Now().Year()

	tests := []struct {
		name string
		fn   func(*testing.T)
	}{
		{"TopLeadersOfSong", func(t *testing.T) {
			count := 0
			for _, err := range conn.TopLeadersOfSong(ctx, testSong, 5) {
				if err != nil {
					t.Fatal(err)
				}
				count++
			}
			if count == 0 {
				t.Errorf("expected results for song %q", testSong)
			}
		}},
		{"AllSingings", func(t *testing.T) {
			count := 0
			for _, err := range conn.AllSingings(ctx) {
				if err != nil {
					t.Fatal(err)
				}
				count++
				break // unbounded query — only check first result
			}
			if count == 0 {
				t.Error("expected at least one result")
			}
		}},
		{"SingingLessons", func(t *testing.T) {
			count := 0
			for _, err := range conn.SingingLessons(ctx, testSinging, 2009) {
				if err != nil {
					t.Fatal(err)
				}
				count++
				break // unbounded query — only check first result
			}
			if count == 0 {
				t.Errorf("expected results for singing %q", testSinging)
			}
		}},
		{"AllLeaderNames", func(t *testing.T) {
			count := 0
			for _, err := range conn.AllLeaderNames(ctx) {
				if err != nil {
					t.Fatal(err)
				}
				count++
				break // unbounded query — only check first result
			}
			if count == 0 {
				t.Error("expected at least one result")
			}
		}},
		{"NewLeadersByYear/2010", func(t *testing.T) {
			count := 0
			for row, err := range conn.NewLeadersByYear(ctx, 2010, 10) {
				if err != nil {
					t.Fatalf("NewLeadersByYear(2010): %v", err)
				}
				if row.Leader == "" {
					t.Error("expected non-empty leader name")
				}
				count++
			}
			if count == 0 {
				t.Error("NewLeadersByYear(2010): expected at least one result")
			}
		}},
		{"NewLeadersByYear/2023", func(t *testing.T) {
			count := 0
			for row, err := range conn.NewLeadersByYear(ctx, 2023, 10) {
				if err != nil {
					t.Fatalf("NewLeadersByYear(2023): %v", err)
				}
				if row.Leader == "" {
					t.Error("expected non-empty leader name")
				}
				count++
			}
			if count == 0 {
				t.Error("NewLeadersByYear(2023): expected at least one result")
			}
		}},
		{"LeadersByKey", func(t *testing.T) {
			const testKey = "A Major"
			count := 0
			for row, err := range conn.LeadersByKey(ctx, testKey, 5) {
				if err != nil {
					t.Fatal(err)
				}
				if row.Leader == "" {
					t.Errorf("expected non-empty leader name for key %q", testKey)
				}
				c, err2 := strconv.Atoi(row.NumLeads)
				if err2 != nil || c <= 0 {
					t.Errorf("expected positive count, got %q", row.NumLeads)
				}
				count++
			}
			if count == 0 {
				t.Errorf("expected results for key %q", testKey)
			}
		}},
		{"LeadersByTop20Leads", func(t *testing.T) {
			count := 0
			for row, err := range conn.LeadersByTop20Leads(ctx, 20) {
				if err != nil {
					t.Fatal(err)
				}
				if row.NumLeads == "" || row.NumLeads == "0" {
					t.Errorf("expected count > 0, got %q for %q", row.NumLeads, row.Leader)
				}
				count++
			}
			if count == 0 {
				t.Error("expected at least one result")
			}
		}},
		{"Top20LeadersActiveInLastYear", func(t *testing.T) {
			count := 0
			for row, err := range conn.Top20LeadersActiveInLastYear(ctx, 20) {
				if err != nil {
					t.Fatal(err)
				}
				if row.NumLeads == "" || row.NumLeads == "0" {
					t.Errorf("expected count > 0, got %q for %q", row.NumLeads, row.Leader)
				}
				count++
			}
			if count == 0 {
				t.Error("expected at least one result")
			}
		}},
		{"AllLeaderConnectedness", func(t *testing.T) {
			count := 0
			for kv, err := range conn.AllLeaderConnectedness(ctx, 20) {
				if err != nil {
					t.Fatal(err)
				}
				if kv.Connectedness < 0 || kv.Connectedness > 1 {
					t.Errorf("ratio out of range for %q: %v", kv.Name, kv.Connectedness)
				}
				count++
				break // unbounded — only check first result
			}
			if count == 0 {
				t.Error("expected at least one result")
			}
		}},
		{"TopLeadersByLeads", func(t *testing.T) {
			var prev models.LeaderLeadCount
			count := 0
			for row, err := range conn.TopLeadersByLeads(ctx, 5) {
				if err != nil {
					t.Fatal(err)
				}
				if row.Count <= 0 {
					t.Errorf("expected positive lead count, got %d for %q", row.Count, row.Name)
				}
				if row.Percentage <= 0 || row.Percentage > 1 {
					t.Errorf("pct out of range for %q: %v", row.Name, row.Percentage)
				}
				if row.LastLeadYear < 1995 || row.LastLeadYear > currentYear {
					t.Errorf("last_lead_year out of range for %q: %d", row.Name, row.LastLeadYear)
				}
				if count > 0 && row.RunningTotal < prev.RunningTotal {
					t.Errorf("running_total decreased from %v to %v between %q and %q",
						prev.RunningTotal, row.RunningTotal, prev.Name, row.Name)
				}
				if count > 0 && row.Percentage > prev.Percentage {
					t.Errorf("pct increased from %v to %v (not sorted DESC) between %q and %q",
						prev.Percentage, row.Percentage, prev.Name, row.Name)
				}
				prev = row
				count++
			}
			if count == 0 {
				t.Error("expected at least one result")
			}
		}},
		{"TopLeadersByLeadsWithYear", func(t *testing.T) {
			count := 0
			for row, err := range conn.TopLeadersByLeads(ctx, 5, 2023) {
				if err != nil {
					t.Fatal(err)
				}
				if row.LastLeadYear != 2023 {
					t.Errorf("last_lead_year should be 2023, got %d for %q", row.LastLeadYear, row.Name)
				}
				count++
			}
			if count == 0 {
				t.Error("TopLeadersByLeads(2023): expected at least one result")
			}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.fn(t)
		})
	}
}

// TestSingingStrangersVariesByInput is a regression test verifying that stranger
// results vary by input leader rather than returning identical alphabetical results.
func TestSingingStrangersVariesByInput(t *testing.T) {
	t.Parallel()
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

// TestUnfamiliarHitsExcludesFamiliarSongs is a regression test verifying that
// familiar (high-attendance) songs do not appear in the unfamiliar-hits list.
func TestUnfamiliarHitsExcludesFamiliarSongs(t *testing.T) {
	t.Parallel()
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
			t.Fatalf("PopularAsObserved(%q): %v", testLeader, err)
		}
		experienced[song.PageNum] = true
	}
	if len(experienced) == 0 {
		t.Fatalf("PopularAsObserved(%q): no results; cannot run regression check", testLeader)
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
