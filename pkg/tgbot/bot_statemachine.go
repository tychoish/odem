package tgbot

import (
	"cmp"
	"fmt"
	"strconv"
	"strings"

	etron "github.com/NicoNex/echotron/v3"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/grip"
	"github.com/tychoish/odem/pkg/infra"
	"github.com/tychoish/odem/pkg/selector"
)

func (b *bot) discoverNext() stateFn {
	if b.queryState.entry == nil {
		grip.Info(grip.KV("state", "discoverNext").
			KV("status", "entry nil; retry keyboard"))
		return b.keyboardMinutesAppQueries()
	}
	if b.queryState.entry.Requires == nil {
		grip.Info(grip.KV("state", "discoverNext").
			KV("status", "requirements nil; rendering off the bat").
			KV("op", b.queryState.entry.Command))
		return b.renderResults()
	}
	if b.queryState.inProgress && b.queryState.has == nil {
		grip.Info(grip.KV("state", "discoverNext").
			KV("status", "requirements Set undefined; rendering").
			KV("op", b.queryState.entry.Command))
		return b.renderResults()
	}
	for requirement := range b.queryState.entry.Requires.Iterator() {
		if !b.queryState.has.Check(requirement) {
			grip.Debug(grip.
				KV("state", "discoverNext").
				KV("status", "discovering next value").
				KV("requirement", requirement).
				KV("op", b.queryState.entry.Command),
			)
			return b.selectFor(requirement)
		}
	}
	grip.Debug(grip.
		KV("state", "discoverNext").
		KV("status", "rendering").
		KV("op", b.queryState.entry.Command),
	)
	return b.renderResults()
}

func (b *bot) wrapInputAsHandler(in func(string) stateFn, fallback func() stateFn) stateFn {
	return func(u *etron.Update) stateFn {
		var text string
		switch {
		case u.Message != nil:
			text = u.Message.Text
		case u.CallbackQuery != nil:
			text = u.CallbackQuery.Data
		default:
			return fallback()
		}
		if isEscapeInput(text) {
			b.sendPlain("ok, starting over...")
			b.queryState.selectionAttempts = 0
			return b.resetState()
		}
		return in(text)
	}
}

func (b *bot) maxSelectionAttempts() int { return cmp.Or(b.conf.Telegram.MaxSelectionAttempts, 3) }

// captureRetry sends errMsg and returns to the selection loop, but aborts
// back to the top level after maxSelectionAttempts consecutive failures.
func (b *bot) captureRetry(errMsg string, retry func(string) stateFn) stateFn {
	b.queryState.selectionAttempts++
	max := b.maxSelectionAttempts()
	if b.queryState.selectionAttempts >= max {
		b.queryState.selectionAttempts = 0
		b.sendMarkdown(fmt.Sprintf("%s after %d tries — starting over", errMsg, max))
		return b.resetState()
	}
	b.sendMarkdown(fmt.Sprintf("%s (attempt %d/%d — or say `cancel` to start over)", errMsg, b.queryState.selectionAttempts, max))
	return b.wrapInputAsHandler(retry, b.discoverNext)
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

func (b *bot) captureInputAsName(value string) stateFn {
	b.queryState.params.Name = value
	return b.discoverNext()
}

func (b *bot) captureInputAsYears(value string) stateFn {
	var err error
	b.queryState.params.Years, err = erc.FromIteratorAll(irt.With2(irt.Modify(strings.SplitSeq(value, ","), strings.TrimSpace), strconv.Atoi))
	grip.Error(err)
	return b.discoverNext()
}

func (b *bot) searchParams(input string) *infra.SearchParams {
	return (&infra.SearchParams{}).With(input).WithoutInteractive().UseFirstResult()
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

func (b *bot) captureYears(value string) stateFn {
	years, err := selector.Years(b.searchParams(value))
	if err != nil {
		return b.captureRetry(fmt.Sprintf("couldn't parse years from `%s`", value), b.captureYears)
	}
	b.queryState.selectionAttempts = 0
	b.queryState.params.Years = years
	return b.discoverNext()
}

func (b *bot) renderResults() stateFn {
	grip.Info(grip.KV("status", "rendering now...").KV("state", b.queryState.params).KV("command", b.queryState.op.String()))

	for msg, err := range b.queryState.entry.Messenger(b.ctx, b.db, b.queryState.params) {
		if err != nil {
			grip.Alert(grip.KV("op", b.queryState.entry.Command).KV("outcome", "overflow").KV("query", b.queryState.params))
			b.sendPlain(fmt.Sprintf("❗got error producing results: %v", err))
			break
		} else {
			b.sendMarkdown(msg.String())
			msg.Release()
		}
	}

	return b.resetState()
}
