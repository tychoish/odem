package tgbot

import (
	"slices"
	"strconv"
	"strings"

	etron "github.com/NicoNex/echotron/v3"
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
		return b.renderResults()
	}

	for requirement := range b.state.entry.Requires.Iterator() {
		if !b.state.has.Check(requirement) {
			return b.selectFor(requirement)
		}
	}
	return b.renderResults()
}

func (b *bot) wrapInputAsHandler(in func(string) stateFn, fallback func() stateFn) stateFn {
	return func(u *etron.Update) stateFn {
		switch {
		case u.Message != nil:
			return b.handleArbitraryMessage(u.Message, fallback)
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
		return b.selectOperation()
	}
	b.state.entry = stw.Ptr(b.state.op.Registry())
	b.state.inProgress = true
	return b.discoverNext()
}

func (b *bot) renderResults() stateFn {
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
	return b.handleMessage
}

func (b *bot) sendKeyboard() stateFn {
	btn := irt.Collect(
		irt.Convert(irt.RemoveValue(dispatch.AllMinutesAppOps(), dispatch.MinutesAppOpExit),
			func(mao dispatch.MinutesAppOperation) etron.InlineKeyboardButton {
				reg := mao.Registry().Info()
				return etron.InlineKeyboardButton{Text: reg.Key, CallbackData: reg.Key}
			},
		),
	)

	b.handleSendMessage(b.SendMessage("Choose an option:", b.chatID, &etron.MessageOptions{
		ReplyMarkup: etron.InlineKeyboardMarkup{
			InlineKeyboard: irt.Collect(slices.Chunk(btn, len(btn)/8)),
		},
	}))
	return nil
}
