package tgbot

import (
	"fmt"
	"strconv"
	"strings"

	etron "github.com/NicoNex/echotron/v3"
	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/stw"
	"github.com/tychoish/grip"
	"github.com/tychoish/odem/pkg/dispatch"
	"github.com/tychoish/odem/pkg/infra"
	"github.com/tychoish/odem/pkg/selector"
)

func (b *bot) discoverNext() stateFn {
	if b.queryState.entry.Requires == nil {
		grip.Info(grip.KV("state", "discoverNext").
			KV("status", "requirements nil; rendering off the bat").
			KV("op", b.queryState.entry.Command))
		return b.renderResults()
	}
	if b.queryState.entry == nil {
		grip.Info(grip.KV("state", "discoverNext").
			KV("status", "entry nil; retry keyboard").
			KV("op", b.queryState.entry.Command))
		return b.keyboardMinutesAppQueries()
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
		switch {
		case u.Message != nil:
			return in(u.Message.Text)
		case u.CallbackQuery != nil:
			return in(u.CallbackQuery.Data)
		default:
			return fallback()
		}
	}
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
	leader, err := selector.Leader(b.ctx, b.db, b.searchParams(value))
	if err != nil {
		b.sendMarkdown(fmt.Sprintf("couldn't find leader for `%s`: please try again", value))
		return b.wrapInputAsHandler(b.captureLeader, b.discoverNext)
	}
	b.queryState.params.Name = leader.Name
	return b.discoverNext()
}

func (b *bot) captureSong(value string) stateFn {
	song, err := selector.Song(b.ctx, b.db, b.searchParams(value))
	if err != nil {
		b.sendMarkdown(fmt.Sprintf("couldn't find song for `%s`: please try again", value))
		return b.wrapInputAsHandler(b.captureSong, b.discoverNext)
	}
	b.queryState.params.Name = song.PageNum
	return b.discoverNext()
}

func (b *bot) captureSinging(value string) stateFn {
	singing, err := selector.Singing(b.ctx, b.db, b.searchParams(value))
	if err != nil {
		b.sendMarkdown(fmt.Sprintf("couldn't find singing for `%s`: please try again", value))
		return b.wrapInputAsHandler(b.captureSinging, b.discoverNext)
	}
	b.queryState.params.Name = singing.SingingName
	return b.discoverNext()
}

func (b *bot) captureKey(value string) stateFn {
	key, err := selector.Key(b.ctx, b.db, b.searchParams(value))
	if err != nil {
		b.sendMarkdown(fmt.Sprintf("couldn't find key for `%s`: please try again", value))
		return b.wrapInputAsHandler(b.captureKey, b.discoverNext)
	}
	b.queryState.params.Name = key
	return b.discoverNext()
}

func (b *bot) captureWord(value string) stateFn {
	word, err := selector.Concordance(b.ctx, b.db, b.searchParams(value))
	if err != nil {
		b.sendMarkdown(fmt.Sprintf("couldn't find a word for `%s`: please try again", value))
		return b.wrapInputAsHandler(b.captureWord, b.discoverNext)
	}
	b.queryState.params.Name = word
	return b.discoverNext()
}

func (b *bot) captureYears(value string) stateFn {
	years, err := selector.Years(b.searchParams(value))
	if err != nil {
		b.sendMarkdown(fmt.Sprintf("couldn't find years for `%s`: please try again", value))
		return b.wrapInputAsHandler(b.captureYears, b.discoverNext)
	}
	b.queryState.params.Years = years
	return b.discoverNext()
}

func (b *bot) handleKeyboardResponse(kbdValue string) stateFn {
	grip.Debug(grip.KV("type", "callback").KV("body", kbdValue))
	b.queryState.op = stw.Ptr(dispatch.NewMinutesAppOperation(kbdValue))
	if !b.queryState.op.Ok() {
		return b.keyboardMinutesAppQueries()
	}
	b.queryState.entry = stw.Ptr(b.queryState.op.Registry())
	b.queryState.has = &dt.Set[dispatch.MinutesAppQueryType]{}
	b.queryState.inProgress = true
	b.sendMarkdown(joinstr("🎶 ok, lets find **", b.queryState.entry.Command, "** ... 🎶"))
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
