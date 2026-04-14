package msgui

import (
	"strings"
	"testing"

	"github.com/tychoish/fun/mdwn"
)

// --- flush() unit tests ---

func TestFlushNormal(t *testing.T) {
	md := mdwn.MakeBuilder(256)
	md.Concat(strings.Repeat("x", 100)) // Len() == 100, normal range

	var builders []*mdwn.Builder
	var errs []error
	flush(md, func(b *mdwn.Builder, err error) bool {
		builders = append(builders, b)
		errs = append(errs, err)
		return true
	})

	if len(builders) != 1 {
		t.Fatalf("expected 1 yield, got %d", len(builders))
	}
	if builders[0] != md {
		t.Error("expected the original builder to be yielded")
	}
	if errs[0] != nil {
		t.Errorf("expected nil error, got %v", errs[0])
	}
}

func TestFlushEmpty(t *testing.T) {
	md := mdwn.MakeBuilder(16) // Len() == 0, which is <= 4

	var builders []*mdwn.Builder
	var errs []error
	flush(md, func(b *mdwn.Builder, err error) bool {
		builders = append(builders, b)
		errs = append(errs, err)
		return true
	})

	if len(builders) != 1 {
		t.Fatalf("expected 1 yield, got %d", len(builders))
	}
	if builders[0] != nil {
		t.Error("expected nil builder for empty case")
	}
	if errs[0] == nil {
		t.Error("expected error for empty case")
	}
}

func TestFlushSmall(t *testing.T) {
	md := mdwn.MakeBuilder(16)
	md.Concat("hi") // Len() == 2, which is <= 4

	var errs []error
	flush(md, func(b *mdwn.Builder, err error) bool {
		errs = append(errs, err)
		return true
	})

	if len(errs) != 1 || errs[0] == nil {
		t.Error("expected error for Len() <= 4")
	}
}

func TestFlushBoundary(t *testing.T) {
	// Len() == 5 should be the smallest "normal" case
	md := mdwn.MakeBuilder(16)
	md.Concat("hello") // exactly 5 bytes

	var builders []*mdwn.Builder
	flush(md, func(b *mdwn.Builder, err error) bool {
		builders = append(builders, b)
		return true
	})

	if len(builders) != 1 || builders[0] == nil {
		t.Error("expected normal yield for Len() == 5")
	}
}

func TestFlushOversize(t *testing.T) {
	md := mdwn.MakeBuilder(8192)
	md.Concat(strings.Repeat("x", 4097)) // Len() == 4097 > 4096

	var builders []*mdwn.Builder
	var errs []error
	flush(md, func(b *mdwn.Builder, err error) bool {
		builders = append(builders, b)
		errs = append(errs, err)
		return true
	})

	// Three yields: error builder, truncated content, oversize error
	if len(builders) != 3 {
		t.Fatalf("expected 3 yields for oversize, got %d", len(builders))
	}
	if builders[0] == nil {
		t.Error("expected error builder as first yield")
	}
	if builders[1] == nil {
		t.Error("expected truncated content as second yield")
	}
	if builders[1] != nil && builders[1].Len() != 4095 {
		t.Errorf("expected truncated builder Len() == 4095, got %d", builders[1].Len())
	}
	if builders[2] != nil {
		t.Error("expected nil builder as third yield")
	}
	if errs[2] == nil {
		t.Error("expected oversize error as third yield")
	}
}

func TestFlushOversizeStopsOnFirstFalse(t *testing.T) {
	md := mdwn.MakeBuilder(8192)
	md.Concat(strings.Repeat("x", 4097))

	count := 0
	flush(md, func(b *mdwn.Builder, err error) bool {
		count++
		return false // stop after first
	})

	if count != 1 {
		t.Errorf("expected yield to stop at first false, got %d calls", count)
	}
}

func TestFlushOversizeStopsOnSecondFalse(t *testing.T) {
	md := mdwn.MakeBuilder(8192)
	md.Concat(strings.Repeat("x", 4097))

	count := 0
	flush(md, func(b *mdwn.Builder, err error) bool {
		count++
		return count < 2 // stop after second
	})

	if count != 2 {
		t.Errorf("expected yield to stop at second false, got %d calls", count)
	}
}

// --- smoke tests: verify all Messenger functions satisfy the type ---

func TestMessengerSignatures(t *testing.T) {
	messengers := map[string]Messenger{
		"MostLed":               MostLed,
		"Songs":                 Songs,
		"Singings":              Singings,
		"Buddies":               Buddies,
		"Strangers":             Strangers,
		"PopularAsObserved":     PopularAsObserved,
		"PopularInYears":        PopularInYears,
		"PopularLocally":        PopularLocally,
		"NeverSung":             NeverSung,
		"NeverLed":              NeverLed,
		"UnfamilarHits":         UnfamilarHits,
		"Connectedness":         Connectedness,
		"LeaderRoleModels":      LeaderRoleModels,
		"TopLeaders":            TopLeaders,
		"LeaderShare":           LeaderShare,
		"LeaderLeadHistory":     LeaderLeadHistory,
		"LeaderSingings":        LeaderSingings,
		"LeaderFavoriteKey":     LeaderFavoriteKey,
		"LeaderDebutsByYear":    LeaderDebutsByYear,
		"SongsByKey":            SongsByKey,
		"Top20Leaders":                   Top20Leaders,
		"Top20LeadersActiveInLastYear":   Top20LeadersActiveInLastYear,
		"LeaderSingingsPerYear": LeaderSingingsPerYear,
		"LeadersByKey":          LeadersByKey,
		"PopularSongsByKey":     PopularSongsByKey,
	}
	for name, m := range messengers {
		if m == nil {
			t.Errorf("%s: nil messenger", name)
		}
	}
}
