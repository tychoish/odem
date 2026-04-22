package models

import (
	"cmp"
	"fmt"
	"sync"
	"time"

	"github.com/tychoish/fun/irt"
)

// SongInput returns the song page number to use for song-scoped queries,
// preferring Song over Name so that song-based operations don't consume Name.
func (p Params) SongInput() string { return cmp.Or(p.Song, p.Name) }

type Params struct {
	// Provide input for the name of the singer, the song or the
	// singing, as relevant to the query.
	Name string `json:"name" jsonschema:"the name of the leader (singer), singing, or song, depending on query."`

	// Song is the page number of a song (e.g. "82t", "475") used
	// to filter results to a specific song. When set, queries that
	// support it will scope their results to that song.
	Song string `json:"song,omitempty" jsonschema:"optional; the page number of a song (e.g. '82t', '475') to filter results to a specific song"`

	// Years makes it possible to limit the scope of a query to
	// specific years. Negative numbers exclude years from
	// queries, positive numbers include them. When empty query
	// all years.
	Years []int `json:"years,omitempty" jsonschema:"optional; explicitly constratian or exclude years for some results; optional"`

	// Limit the number of items returned to this number:
	// typically the handlers will restrict this to somewhere
	// between 16 and 64 depending, but it can be overridden.
	Limit int `json:"limit,omitempty" jsonschema:"optional;limit the number of results returned in some cases"`
}

func (p Params) String() string {
	return fmt.Sprintf("name<%q> song<%q> years %v limit<%d>", p.Name, p.Song, p.Years, p.Limit)
}

func FirstValidYear(yrs []int) int {
	input, _ := irt.Initial(irt.Keep(irt.Slice(yrs), isCoveredYear))
	return cmp.Or(input, thisYear()-1)
}

var thisYear = sync.OnceValue(func() int { return time.Now().Year() })

func isCoveredYear(y int) bool { return y >= 1995 && y <= thisYear() }
