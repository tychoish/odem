package db

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/tychoish/fun/srv"
)

const testLeader = "Bud Oliver"
const testSocialLeader = "Rose Altha Taylor" // fewer singings — faster for social graph queries
const testSong = "32t"
const testSinging = "Liberty Church"

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

func TestSupriringsSingingStrangers(t *testing.T) {
	conn, ctx := testConn(t)
	count := 0
	for _, err := range conn.SupriringsSingingStrangers(ctx, testSocialLeader, 5) {
		if err != nil {
			t.Fatal(err)
		}
		count++
	}
	if count == 0 {
		t.Errorf("SupriringsSingingStrangers(%q): expected at least one result", testSocialLeader)
	}
}
