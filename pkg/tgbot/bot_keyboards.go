package tgbot

import (
	"slices"

	etron "github.com/NicoNex/echotron/v3"
	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/stw"
	"github.com/tychoish/grip"
	"github.com/tychoish/odem/pkg/dispatch"
)

func (b *bot) keyboardMinutesAppQueries() stateFn {
	b.state.trackingKeyboard.Add(1)
	btn := irt.Collect(
		irt.Convert(dispatch.AllMinutesAppMessengerOps(),
			func(mao dispatch.MinutesOperation) etron.InlineKeyboardButton {
				reg := mao.Registry().Info()
				return etron.InlineKeyboardButton{Text: reg.Key, CallbackData: reg.Key}
			},
		),
	)
	var message string
	switch {
	case b.metrics.sent.Load() == 0:
		message = "Hello! Select a query to get started:"
	case b.state.trackingKeyboard.Load() >= 1:
		message = "Let's select a new query:"
	case b.queryState.selectionAttempts >= b.conf.Telegram.MaxSelectionAttempts:
		message = "Let's start over and select a new query:"
	default:
		message = "Minutes App Queries:"
	}

	arm, err := b.SendMessage(message, b.chatID, &etron.MessageOptions{
		MessageThreadID: int64(b.threadID),
		ReplyMarkup: etron.InlineKeyboardMarkup{
			InlineKeyboard: irt.Collect(slices.Chunk(btn, len(btn)/8)),
		},
	})

	if arm.Result == nil {
		grip.Error("no result from setting the keyboard")
		return b.handleMessage
	} else if err != nil {
		b.handleAPIResponse(arm.Base(), err)
	}

	if prev := b.state.trackingKeyboard.Swap(int64(arm.Result.ID)); prev != 0 {
		b.handleAPIResponse(b.DeleteMessage(b.chatID, int(prev)))
	}

	return b.wrapInputAsHandler(b.handleKeyboardResponse, b.keyboardMinutesAppQueries)
}

func (b *bot) handleKeyboardResponse(kbdValue string) stateFn {
	if kbdID := b.state.trackingKeyboard.Load(); kbdID != 0 {
		for {
			if b.state.trackingKeyboard.CompareAndSwap(kbdID, 0) {
				grip.Info(b.grip("deleting keyboard").KV("kbd", kbdID))
				b.handleAPIResponse(b.DeleteMessage(b.chatID, int(kbdID)))
				break
			}
		}
	}

	grip.Debug(grip.KV("type", "callback").KV("body", kbdValue))
	if b.setupQuery(kbdValue) {
		return b.discoverNext()
	}
	return b.handleMessage
}

func (b *bot) setupQuery(opName string) bool {
	op := dispatch.NewMinutesAppOperation(opName)
	if !op.Ok() {
		return false
	}
	b.queryState.op = stw.Ptr(op)
	b.queryState.entry = stw.Ptr(b.queryState.op.Registry())
	b.queryState.has = &dt.Set[dispatch.MinutesAppQueryType]{}
	b.queryState.inProgress = true
	b.sendMarkdown(joinstr("🎶 ok, lets find **", b.queryState.entry.Command, "** ... 🎶"))
	return true
}
