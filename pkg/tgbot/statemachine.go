package tgbot

import (
	"strconv"
	"strings"

	etron "github.com/NicoNex/echotron/v3"
	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/strut"
	"github.com/tychoish/fun/stw"
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/odem/pkg/dispatch"
	"github.com/tychoish/odem/pkg/reportui"
)

func (b *bot) discoverNext() stateFn {
	if b.state.entry.Requires == nil {
		grip.Info(message.NewKV().KV("state", "discoverNext").KV("status", "requirements nil; rendering off the bat").KV("op", b.state.entry.Command))
		return b.renderResults()
	}
	if b.state.entry == nil {
		grip.Info(message.NewKV().KV("state", "discoverNext").KV("status", "entry nil; retry keyboard").KV("op", b.state.entry.Command))
		return b.selectOperationKeyboard()
	}
	if b.state.inProgress && b.state.has == nil {
		grip.Info(message.NewKV().KV("state", "discoverNext").KV("status", "requirements Set undefined; rendering").KV("op", b.state.entry.Command))
		return b.renderResults()
	}
	for requirement := range b.state.entry.Requires.Iterator() {
		if !b.state.has.Check(requirement) {
			grip.Debug(message.NewKV().
				KV("state", "discoverNext").
				KV("status", "discovering next value").
				KV("requirement", requirement).
				KV("op", b.state.entry.Command),
			)
			return b.selectFor(requirement)
		}
	}
	grip.Debug(message.NewKV().
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

func (b *bot) handleKeyboardResponse(kbdValue string) stateFn {
	grip.Debug(message.NewKV().KV("type", "callback").KV("body", kbdValue))
	b.state.op = stw.Ptr(dispatch.NewMinutesAppOperation(kbdValue))
	if !b.state.op.Ok() {
		return b.selectOperationKeyboard()
	}
	b.state.entry = stw.Ptr(b.state.op.Registry())
	b.state.has = &dt.Set[dispatch.MinutesAppQueryType]{}
	b.state.inProgress = true
	b.sendMarkdown(joinstr("ok, lets find **", b.state.entry.Command, "** ... 🎶"))
	return b.discoverNext()
}

func (b *bot) renderResults() stateFn {
	grip.Info(message.NewKV().KV("status", "rendering now...").KV("state", b.state.params).KV("command", b.state.op.String()))

	buf := strut.MakeMutable(1024)
	defer buf.Release()
	buf.PushString("```")
	grip.Error(b.state.entry.Reporter.Report(b.ctx, b.db, reportui.Params{
		Params:                b.state.params,
		ToWriter:              buf,
		SuppressInteractivity: true,
	}))
	buf.PushString("```")

	b.handleSendMessage(b.SendMessage(buf.String(), b.chatID, &etron.MessageOptions{ParseMode: etron.MarkdownV2}))
	return b.resetState()
}
