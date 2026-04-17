package tgbot

import (
	"fmt"

	"github.com/tychoish/odem/pkg/selector"
)

func (b *bot) captureYears(value string) stateFn {
	years, err := selector.Years(b.searchParams(value))
	if err != nil {
		return b.captureRetry(fmt.Sprintf("couldn't parse years from `%s`", value), b.captureYears)
	}
	b.queryState.selectionAttempts = 0
	b.queryState.params.Years = years
	return b.discoverNext()
}

// captureNameResult is the shared success/failure path for capture functions
// that resolve a lookup to a string stored in queryState.params.Name.
// name is a lazy accessor called only when err == nil, so nil-pointer
// results from pointer-returning selectors are safe.
func (b *bot) captureNameResult(value, errPrefix string, err error, retry func(string) stateFn, name func() string) stateFn {
	if err != nil {
		return b.captureRetry(fmt.Sprintf("%s `%s`", errPrefix, value), retry)
	}
	b.queryState.selectionAttempts = 0
	b.queryState.params.Name = name()
	return b.discoverNext()
}

func (b *bot) captureLeader(value string) stateFn {
	l, err := selector.Leader(b.ctx, b.db, b.searchParams(value))
	return b.captureNameResult(value, "couldn't find a leader matching", err, b.captureLeader, func() string { return l.Name })
}

func (b *bot) captureSong(value string) stateFn {
	s, err := selector.Song(b.ctx, b.db, b.searchParams(value))
	return b.captureNameResult(value, "couldn't find a song matching", err, b.captureSong, func() string { return s.PageNum })
}

func (b *bot) captureSinging(value string) stateFn {
	s, err := selector.Singing(b.ctx, b.db, b.searchParams(value))
	return b.captureNameResult(value, "couldn't find a singing matching", err, b.captureSinging, func() string { return s.SingingName })
}

func (b *bot) captureKey(value string) stateFn {
	key, err := selector.Key(b.ctx, b.db, b.searchParams(value))
	return b.captureNameResult(value, "couldn't find a key matching", err, b.captureKey, func() string { return key })
}

func (b *bot) captureWord(value string) stateFn {
	word, err := selector.Concordance(b.ctx, b.db, b.searchParams(value))
	return b.captureNameResult(value, "couldn't find a word matching", err, b.captureWord, func() string { return word })
}
