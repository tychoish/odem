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

func (b *bot) setOperationSelectorButtons() {
	resp, err := b.SetMyCommands(&etron.CommandOptions{
		LanguageCode: "en",
		Scope: etron.BotCommandScope{
			Type:   etron.BCSTDefault,
			ChatID: b.chatID,
			// UserID: 0,
		},
	}, irt.Collect(getBotCommands())...)

	grip.Info(b.gmr("set bot selection menu", resp.Base()).
		KV("result", resp.Result).WithError(err))
}

func (b *bot) keyboardMinutesAppQueries() stateFn {
	b.state.trackingKeyboard.Add(1)
	btn := irt.Collect(
		irt.Convert(irt.RemoveValue(dispatch.AllMinutesAppOps(), dispatch.MinutesAppOpExit),
			func(mao dispatch.MinutesAppOperation) etron.InlineKeyboardButton {
				reg := mao.Registry().Info()
				return etron.InlineKeyboardButton{Text: reg.Key, CallbackData: reg.Key}
			},
		),
	)

	arm, err := b.SendMessage("Choose an option:", b.chatID, &etron.MessageOptions{
		MessageThreadID: int64(b.threadID),
		ReplyMarkup: etron.InlineKeyboardMarkup{
			InlineKeyboard: irt.Collect(slices.Chunk(btn, len(btn)/8)),
		},
	})

	b.handleSendMessage(arm, err)
	return b.wrapInputAsHandler(b.handleKeyboardResponse(arm.Result.ID), b.keyboardMinutesAppQueries)
}

func (b *bot) handleKeyboardResponse(kbdID int) func(kbdValue string) stateFn {
	return func(kbdValue string) stateFn {
		if kbdID != 0 {
			for {
				val := b.state.trackingKeyboard.Load()
				if val == 0 || b.state.trackingKeyboard.CompareAndSwap(val, val-1) {
					grip.Info(b.grip("deleting keyboard").KV("kbd", kbdID))
					b.handleAPIResponse(b.DeleteMessage(b.chatID, kbdID))
					break
				}
			}
		}

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
}

func (b *bot) KeyboardHelpMenu() stateFn {
	// TODO (defer) make a shorter 3-4 button help menu
	rsp, err := b.SendMessage("Choose an option:", b.chatID, &etron.MessageOptions{
		MessageThreadID: int64(b.threadID),
		ReplyMarkup: etron.InlineKeyboardMarkup{
			InlineKeyboard: [][]etron.InlineKeyboardButton{
				{},
			},
		},
	})
	grip.Info(b.grip("send keyboard menu").WithError(err).Extend(kvsFromMessage(rsp.Result)))
	return b.wrapInputAsHandler(b.handleKeyboardResponse(rsp.Result.ID), b.keyboardMinutesAppQueries)
}
