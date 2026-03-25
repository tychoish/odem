package db

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/tychoish/fun/srv"
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
	for _, err := range conn.MostLeadSongs(ctx, testLeader, 5) {
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
	for _, err := range conn.PopularSongsInOnesExperience(ctx, testSocialLeader, 5) {
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
			if s.Key == target {
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

func TestAllLeaderConnectedness(t *testing.T) {
	conn, ctx := testConn(t)
	count := 0
	for kv, err := range conn.AllLeaderConnectedness(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		if kv.Value < 0 || kv.Value > 1 {
			t.Errorf("AllLeaderConnectedness: ratio out of range for %q: %v", kv.Key, kv.Value)
		}
		count++
		break // unbounded — only check first result
	}
	if count == 0 {
		t.Error("AllLeaderConnectedness: expected at least one result")
	}
}
