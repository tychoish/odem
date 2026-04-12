package msgui

import (
	"iter"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/tychoish/fun/mdwn"
	"github.com/tychoish/odem/pkg/models"
)

// fakeRecord is a minimal LineItemer for edge-case tests.
type fakeRecord struct{ text string }

func (f fakeRecord) LineItem() *mdwn.Builder {
	return mdwn.MakeBuilder(128).PushString(f.text)
}

// seqOf wraps variadic items as an iter.Seq.
func seqOf[T any](items ...T) iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, item := range items {
			if !yield(item) {
				return
			}
		}
	}
}

func TestRenderLineItems(t *testing.T) {
	counter := &atomic.Int64{}

	tests := []struct {
		name         string
		seq          iter.Seq2[*mdwn.Builder, error]
		wantError    bool
		wantMultiple bool
	}{
		{
			name:      "empty sequence",
			seq:       renderLineItems(seqOf[fakeRecord]()),
			wantError: true,
		},
		{
			name: "chunking/large data",
			seq: func() iter.Seq2[*mdwn.Builder, error] {
				recs := make([]fakeRecord, 25)
				for i := range recs {
					recs[i] = fakeRecord{text: strings.Repeat("x", 200)}
				}
				return renderLineItems(seqOf(recs...))
			}(),
			wantMultiple: true,
		},
		// One entry per public function, named for the function and record type it uses.
		{
			name: "MostLed/PopularAsObserved/PopularInYears/PopularLocally/NeverSung/NeverLed/UnfamilarHits/PopularSongsByKey (LeaderSongRank)",
			seq: renderLineItems(seqOf(
				models.LeaderSongRank{Leader: "Alice", NumLeads: "10", PageNum: "123", SongTitle: "Sacred Harp", Key: "G"},
				models.LeaderSongRank{Leader: "Alice", NumLeads: "8", PageNum: "456", SongTitle: "New Britain", Key: "A"},
			)),
		},
		{
			name: "Songs (LeaderOfSongInfo)",
			seq: renderLineItems(seqOf(
				models.LeaderOfSongInfo{Name: "Alice", Count: 10, NumYears: 5, LedInLastYear: true},
				models.LeaderOfSongInfo{Name: "Bob", Count: 7, NumYears: 3, LedInLastYear: false},
			)),
		},
		{
			name: "Singings/LeaderLeadHistory (LessonInfo)",
			seq: renderLineItems(seqOf(
				models.LessonInfo{SingerName: "Alice", SongPageNumber: "123", SongName: "Sacred Harp", SongKey: "G", SingingName: "Shape Note Singing"},
				models.LessonInfo{SingerName: "Bob", SongPageNumber: "456", SongName: "New Britain", SongKey: "A", SingingName: "Convention"},
			)),
		},
		{
			name: "Buddies (SingingBuddy)",
			seq: renderLineItems(seqOf(
				models.SingingBuddy{Name: "Bob", SharedSingings: 5},
				models.SingingBuddy{Name: "Carol", SharedSingings: 3},
			)),
		},
		{
			name: "Strangers (SingingStranger)",
			seq: renderLineItems(seqOf(
				models.SingingStranger{Name: "Dave", MutualConnections: 4},
			)),
		},
		{
			name: "Connectedness (LeaderConnectedness)",
			seq: renderLineItems(seqOf(
				models.LeaderConnectedness{Name: "Alice", Connectedness: 0.042},
				models.LeaderConnectedness{Name: "Bob", Connectedness: 0.031},
			)),
		},
		{
			name: "LeaderRoleModels (LeaderFootstep)",
			seq: renderLineItems(seqOf(
				models.LeaderFootstep{LeaderName: "Bob", SongTitle: "Sacred Harp", SongPage: "123", SongKeys: "G", SelfLeadCount: 5, TheirLeadCount: 20, TheirLastLeadYear: 2023},
			)),
		},
		{
			name: "TopLeaders (models.TopLeaders)",
			seq: renderLineItems(seqOf(
				models.TopLeadersWrapper(counter)(models.LeaderLeadCount{Name: "Alice", Count: 42, LastLeadYear: 2023, Percentage: 0.025, RunningTotal: 0.025}),
				models.TopLeadersWrapper(counter)(models.LeaderLeadCount{Name: "Bob", Count: 38, LastLeadYear: 2022, Percentage: 0.020, RunningTotal: 0.045}),
			)),
		},
		{
			name: "LeaderSingings (LeaderSingingAttendance)",
			seq: renderLineItems(seqOf(
				models.LeaderSingingAttendance{SingingName: "Big Singing", SingingState: "AL", SingingCity: "Birmingham", LeaderLeadCount: 5, NumberOfLeaders: 20},
			)),
		},
		{
			name: "LeaderFavoriteKey (LeaderKeyCount)",
			seq: renderLineItems(seqOf(
				models.LeaderKeyCount{Key: "G", Leads: 15},
				models.LeaderKeyCount{Key: "A", Leads: 10},
			)),
		},
		{
			name: "LeaderDebutsByYear/Top20Leaders/LeadersByKey (LeaderRankingFor)",
			seq: renderLineItems(seqOf(
				models.WrapLeaderSongRank("Leads")(models.LeaderSongRank{Leader: "Alice", NumLeads: "10", PageNum: "123", SongTitle: "Sacred Harp", Key: "G"}),
				models.WrapLeaderSongRank("Leads")(models.LeaderSongRank{Leader: "Bob", NumLeads: "7", PageNum: "456", SongTitle: "New Britain", Key: "A"}),
			)),
		},
		{
			name: "SongsByKey (SongByKey)",
			seq: renderLineItems(seqOf(
				models.WrapSongByKey(models.LeaderSongRank{Key: "G", NumLeads: "42", Ratio: 0.15}),
				models.WrapSongByKey(models.LeaderSongRank{Key: "A", NumLeads: "31", Ratio: 0.11}),
			)),
		},
		{
			name: "LeaderSingingsPerYear (LeaderSingingsInYear)",
			seq: renderLineItems(seqOf(
				models.LeaderSingingsInYear{Year: "2023", Singings: 3},
				models.LeaderSingingsInYear{Year: "2022", Singings: 5},
			)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var builders []*mdwn.Builder
			var errs []error
			for b, err := range tt.seq {
				builders = append(builders, b)
				errs = append(errs, err)
			}

			if len(builders) == 0 {
				t.Fatal("no yields from renderLineItems")
			}

			if tt.wantError {
				for _, err := range errs {
					if err != nil {
						return
					}
				}
				t.Error("expected at least one error but got none")
				return
			}

			for i, err := range errs {
				if err != nil {
					t.Errorf("unexpected error at yield %d: %v", i, err)
				}
			}

			hasContent := false
			for _, b := range builders {
				if b != nil && b.Len() > 0 {
					hasContent = true
					break
				}
			}
			if !hasContent {
				t.Error("expected at least one non-empty builder")
			}

			if tt.wantMultiple && len(builders) <= 1 {
				t.Errorf("expected multiple builders for chunking case, got %d", len(builders))
			}
		})
	}
}
