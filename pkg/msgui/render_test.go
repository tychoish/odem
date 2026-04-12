package msgui

import (
	"strings"
	"testing"

	"github.com/tychoish/fun/mdwn"
	"github.com/tychoish/odem/pkg/models"
)

type fakeRecord struct{ text string }

func (f fakeRecord) LineItem() *mdwn.Builder {
	return mdwn.MakeBuilder(128).PushString(f.text)
}

func TestRenderLineItemsPopulated(t *testing.T) {
	records := []fakeRecord{
		{text: "first record line item"},
		{text: "second record line item"},
		{text: "third record line item"},
		{text: "fourth record line item"},
		{text: "fifth record line item"},
	}

	seq := func(yield func(fakeRecord) bool) {
		for _, r := range records {
			if !yield(r) {
				return
			}
		}
	}

	var builders []*mdwn.Builder
	var errs []error
	for b, err := range renderLineItems(seq) {
		builders = append(builders, b)
		errs = append(errs, err)
	}

	if len(builders) == 0 {
		t.Fatal("expected at least one builder to be yielded, got none")
	}

	for i, err := range errs {
		if err != nil {
			t.Errorf("yield %d: unexpected error: %v", i, err)
		}
	}

	for i, b := range builders {
		if b == nil {
			t.Errorf("builder %d is nil", i)
			continue
		}
		if b.Len() == 0 {
			t.Errorf("builder %d has zero length", i)
		}
	}
}

func TestRenderLineItemsEmpty(t *testing.T) {
	seq := func(yield func(fakeRecord) bool) {
		// yields nothing
	}

	var builders []*mdwn.Builder
	var errs []error
	for b, err := range renderLineItems(seq) {
		builders = append(builders, b)
		errs = append(errs, err)
	}

	if len(builders) == 0 {
		t.Fatal("expected at least one yield (error yield from flush), got none")
	}

	hasNonNilError := false
	for _, err := range errs {
		if err != nil {
			hasNonNilError = true
			break
		}
	}
	if !hasNonNilError {
		t.Error("expected a non-nil error from flush for empty results, but all errors were nil")
	}
}

func TestRenderLineItemsChunking(t *testing.T) {
	// Each record has ~200 characters; 25 records = ~5000 bytes total, exceeding 4000 byte chunk size.
	longText := strings.Repeat("x", 200)
	const numRecords = 25
	records := make([]fakeRecord, numRecords)
	for i := range records {
		records[i] = fakeRecord{text: longText}
	}

	seq := func(yield func(fakeRecord) bool) {
		for _, r := range records {
			if !yield(r) {
				return
			}
		}
	}

	var builders []*mdwn.Builder
	for b := range renderLineItems(seq) {
		if b != nil {
			builders = append(builders, b)
		}
	}

	if len(builders) <= 1 {
		t.Errorf("expected more than one builder (chunking), got %d", len(builders))
	}

	for i, b := range builders {
		if b == nil {
			t.Errorf("builder %d is nil", i)
		}
	}
}

func TestMostLedRenderLineItemsBehavior(t *testing.T) {
	records := []models.LeaderSongRank{
		{Rank: 1, Leader: "Alice", NumLeads: "10", PageNum: "123", SongTitle: "Sacred Harp", Key: "G"},
		{Rank: 2, Leader: "Bob", NumLeads: "8", PageNum: "456", SongTitle: "New Britain", Key: "A"},
		{Rank: 3, Leader: "Carol", NumLeads: "6", PageNum: "789", SongTitle: "Idumea", Key: "D"},
	}

	seq := func(yield func(models.LeaderSongRank) bool) {
		for _, r := range records {
			if !yield(r) {
				return
			}
		}
	}

	var builders []*mdwn.Builder
	var errs []error
	for b, err := range renderLineItems(seq) {
		builders = append(builders, b)
		errs = append(errs, err)
	}

	if len(builders) == 0 {
		t.Fatal("expected at least one builder to be yielded, got none")
	}

	hasNonNilBuilder := false
	for _, b := range builders {
		if b != nil && b.Len() > 0 {
			hasNonNilBuilder = true
			break
		}
	}
	if !hasNonNilBuilder {
		t.Error("expected at least one non-nil builder with non-zero length")
	}

	for i, err := range errs {
		if err != nil {
			t.Logf("yield %d returned error: %v (may be expected from flush)", i, err)
		}
	}

	// With valid data (3 records, well under 4000 bytes), flush should not return an error.
	for i, err := range errs {
		if err != nil {
			t.Errorf("unexpected error at yield %d: %v", i, err)
		}
	}
}
