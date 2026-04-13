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
	if b.state.entry.Requires == nil {
		grip.Info(grip.KV("state", "discoverNext").
			KV("status", "requirements nil; rendering off the bat").
			KV("op", b.state.entry.Command))
		return b.renderResults()
	}
	if b.state.entry == nil {
		grip.Info(grip.KV("state", "discoverNext").
			KV("status", "entry nil; retry keyboard").
			KV("op", b.state.entry.Command))
		return b.selectOperationKeyboard()
	}
	if b.state.inProgress && b.state.has == nil {
		grip.Info(grip.KV("state", "discoverNext").
			KV("status", "requirements Set undefined; rendering").
			KV("op", b.state.entry.Command))
		return b.renderResults()
	}
	for requirement := range b.state.entry.Requires.Iterator() {
		if !b.state.has.Check(requirement) {
			grip.Debug(grip.
				KV("state", "discoverNext").
				KV("status", "discovering next value").
				KV("requirement", requirement).
				KV("op", b.state.entry.Command),
			)
			return b.selectFor(requirement)
		}
	}
	grip.Debug(grip.
		KV("state", "discoverNext").
		KV("status", "rendering").
		KV("op", b.state.entry.Command),
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
	b.state.params.Name = value
	return b.discoverNext()
}

func (b *bot) captureInputAsYears(value string) stateFn {
	var err error
	b.state.params.Years, err = erc.FromIteratorAll(irt.With2(irt.Modify(strings.SplitSeq(value, ","), strings.TrimSpace), strconv.Atoi))
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
	b.state.params.Name = leader.Name
	return b.discoverNext()
}

func (b *bot) captureSong(value string) stateFn {
	song, err := selector.Song(b.ctx, b.db, b.searchParams(value))
	if err != nil {
		b.sendMarkdown(fmt.Sprintf("couldn't find song for `%s`: please try again", value))
		return b.wrapInputAsHandler(b.captureSong, b.discoverNext)
	}
	b.state.params.Name = song.PageNum
	return b.discoverNext()
}

func (b *bot) captureSinging(value string) stateFn {
	singing, err := selector.Singing(b.ctx, b.db, b.searchParams(value))
	if err != nil {
		b.sendMarkdown(fmt.Sprintf("couldn't find singing for `%s`: please try again", value))
		return b.wrapInputAsHandler(b.captureSinging, b.discoverNext)
	}
	b.state.params.Name = singing.SingingName
	return b.discoverNext()
}

func (b *bot) captureKey(value string) stateFn {
	key, err := selector.Key(b.ctx, b.db, b.searchParams(value))
	if err != nil {
		b.sendMarkdown(fmt.Sprintf("couldn't find key for `%s`: please try again", value))
		return b.wrapInputAsHandler(b.captureKey, b.discoverNext)
	}
	b.state.params.Name = key
	return b.discoverNext()
}

func (b *bot) captureYears(value string) stateFn {
	years, err := selector.Years(b.searchParams(value))
	if err != nil {
		b.sendMarkdown(fmt.Sprintf("couldn't find years for `%s`: please try again", value))
		return b.wrapInputAsHandler(b.captureYears, b.discoverNext)
	}
	b.state.params.Years = years
	return b.discoverNext()
}

func (b *bot) handleKeyboardResponse(kbdValue string) stateFn {
	grip.Debug(grip.KV("type", "callback").KV("body", kbdValue))
	b.state.op = stw.Ptr(dispatch.NewMinutesAppOperation(kbdValue))
	if !b.state.op.Ok() {
		return b.selectOperationKeyboard()
	}
	b.state.entry = stw.Ptr(b.state.op.Registry())
	b.state.has = &dt.Set[dispatch.MinutesAppQueryType]{}
	b.state.inProgress = true
	b.sendMarkdown(joinstr("🎶 ok, lets find **", b.state.entry.Command, "** ... 🎶"))
	return b.discoverNext()
}

func (b *bot) renderResults() stateFn {
	grip.Info(grip.KV("status", "rendering now...").KV("state", b.state.params).KV("command", b.state.op.String()))

	for msg, err := range b.state.entry.Messenger(b.ctx, b.db, b.state.params) {
		if err != nil {
			grip.Alert(grip.KV("op", b.state.entry.Command).KV("outcome", "overflow").KV("query", b.state.params))
			b.sendPlain(fmt.Sprintf("❗got error producing results: %v", err))
			break
		} else {
			b.sendMarkdown(msg.String())
			msg.Release()
		}
	}

	return b.resetState()
}
