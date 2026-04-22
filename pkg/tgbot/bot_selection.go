package tgbot

import (
	"fmt"
	"strings"

	"github.com/tychoish/fun/ers"
	"github.com/tychoish/grip"
	"github.com/tychoish/odem/pkg/dispatch"
	"github.com/tychoish/odem/pkg/infra"
	"github.com/tychoish/odem/pkg/models"
	"github.com/tychoish/odem/pkg/selector"
)

func (b *bot) selectFor(requirement dispatch.MinutesAppQueryType) stateFn {
	switch requirement {
	case dispatch.MinutesAppQueryTypeOperation:
		return b.keyboardMinutesAppQueries()
	case dispatch.MinutesAppQueryTypeLeader:
		return b.promptFor(dispatch.MinutesAppQueryTypeLeader, "which singer are you looking for?", b.captureLeader)
	case dispatch.MinutesAppQueryTypeSong:
		return b.promptFor(dispatch.MinutesAppQueryTypeSong, "which song (title or page number) are you looking for?", b.captureSong)
	case dispatch.MinutesAppQueryTypeSinging:
		return b.promptFor(dispatch.MinutesAppQueryTypeSinging, "which singing are you looking for?", b.captureSinging)
	case dispatch.MinutesAppQueryTypeYear:
		return b.promptFor(dispatch.MinutesAppQueryTypeYear, "which year would you like to filter by?", b.captureYears)
	case dispatch.MinutesAppQueryTypeKey:
		return b.promptFor(dispatch.MinutesAppQueryTypeKey, "what key would you like to filter by?", b.captureKey)
	case dispatch.MinutesAppQueryTypeLocality:
		return b.promptFor(dispatch.MinutesAppQueryTypeLocality, "what locality would you like to filter by (state codes)?", b.captureLocality)
	case dispatch.MinutesAppQueryTypeWord:
		return b.promptFor(dispatch.MinutesAppQueryTypeWord, "what word would you like to find?", b.captureWord)
	case dispatch.MinutesAppQueryTypeInvalid:
		b.sendMarkdown(fmt.Sprintf("❗invalid option: `%s`: %s. Let's start over! ⏪", requirement, requirement.Validate()))
		return b.resetState()
	case dispatch.MinutesAppQueryTypeUnknown:
		b.sendPlain("❗Sorry, something went wrong: we need to start over... 😞")
		return b.resetState()
	default:
		b.sendMarkdown(fmt.Sprintf("Sorry, got an invalid option(`%s`: %s) and need to start over 😥", requirement, requirement.Validate()))
		return b.resetState()
	}
}

func (b *bot) searchParams(input string) *infra.SearchParams {
	return (&infra.SearchParams{}).With(input).WithoutInteractive().UseFirstResult()
}

func (b *bot) promptFor(queryType dispatch.MinutesAppQueryType, prompt string, handler func(string) stateFn) stateFn {
	defer b.queryState.has.Add(queryType)
	grip.Debug(b.grip("selecting").KV("type", queryType))
	b.sendMarkdown(prompt)
	return b.wrapInputAsHandler(handler, b.discoverNext)
}

// captureNameResult is the shared success/failure path for capture functions
// that resolve a lookup to a string stored in queryState.params.Name.
// name is a lazy accessor called only when err == nil, so nil-pointer
// results from pointer-returning selectors are safe.
func (b *bot) captureNameResult(input, noun string, err error, retry func(string) stateFn, result func() string) stateFn {
	if err != nil {
		return b.captureRetry(fmt.Sprintf("coulding find %s matching `%s`", noun, input), retry)
	}
	b.queryState.selectionAttempts = 0
	b.queryState.params.Name = result()
	return b.discoverNext()
}

func (b *bot) captureYears(value string) stateFn {
	years, err := selector.Years(b.searchParams(value))
	if err != nil {
		return b.captureRetry(fmt.Sprintf("couldn't parse years from `%s`", value), b.captureYears)
	}
	b.queryState.selectionAttempts = 0
	b.queryState.params.Years = years
	return b.discoverNext()
}

func (b *bot) captureLocality(value string) stateFn {
	if models.NewSingingLocality(value).Valid() {
		b.queryState.params.Name = value
		return b.discoverNext()
	}
	return b.captureNameResult(value, "locality", ers.New("no matching locality"), b.captureLocality, func() string { return value })
}

func (b *bot) captureLeader(value string) stateFn {
	l, err := selector.Leader(b.ctx, b.db, b.searchParams(value))
	return b.captureNameResult(value, "leader", err, b.captureLeader, func() string { return l.Name })
}

func (b *bot) captureSong(value string) stateFn {
	s, err := selector.Song(b.ctx, b.db, b.searchParams(value))
	return b.captureNameResult(value, "song", err, b.captureSong, func() string { return s.PageNum })
}

func (b *bot) captureSinging(value string) stateFn {
	s, err := selector.Singing(b.ctx, b.db, b.searchParams(value))
	return b.captureNameResult(value, "singing", err, b.captureSinging, func() string { return s.SingingName })
}

func (b *bot) captureKey(value string) stateFn {
	key, err := selector.Key(b.ctx, b.db, b.searchParams(value))
	return b.captureNameResult(value, "key", err, b.captureKey, func() string { return key })
}

func (b *bot) captureWord(value string) stateFn {
	value = strings.TrimSpace(value)
	if value == "" {
		return b.captureRetry("please enter a word or phrase to search for", b.captureWord)
	}
	b.queryState.selectionAttempts = 0
	b.queryState.params.Name = value
	return b.discoverNext()
}
