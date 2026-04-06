package tgbot

import (
	"fmt"
	"slices"

	etron "github.com/NicoNex/echotron/v3"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/grip"
	"github.com/tychoish/odem/pkg/dispatch"
)

func (b *bot) selectFor(requirement dispatch.MinutesAppQueryType) stateFn {
	switch requirement {
	case dispatch.MinutesAppQueryTypeOperation:
		return b.selectOperationKeyboard()
	case dispatch.MinutesAppQueryTypeLeader:
		return b.selectSinger()
	case dispatch.MinutesAppQueryTypeSong:
		return b.selectSong()
	case dispatch.MinutesAppQueryTypeSinging:
		return b.selectSinging()
	case dispatch.MinutesAppQueryTypeYear:
		return b.selectYear()
	case dispatch.MinutesAppQueryTypeKey:
		return b.selectKey()
	case dispatch.MinutesAppQueryTypeLocality:
		return b.selectLocality()
	case dispatch.MinutesAppQueryTypeInvalid:
		b.sendMarkdown(fmt.Sprintf("invalid option: `%s`: %s", requirement, requirement.Validate()))
		return b.selectOperationKeyboard()
	case dispatch.MinutesAppQueryTypeUnknown:
		b.sendMarkdown("unknown operation")
		return b.selectOperationKeyboard()
	default:
		b.sendMarkdown(fmt.Sprintf("invalid option: `%s`: %s", requirement, requirement.Validate()))
		return b.selectOperationKeyboard()
	}
}

func (b *bot) selectOperationKeyboard() stateFn {
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

	return b.wrapInputAsHandler(b.handleKeyboardResponse, b.selectOperationKeyboard)
}

func (b *bot) setOperationSelectorButtons() {
	resp, err := b.SetMyCommands(&etron.CommandOptions{
		LanguageCode: "en",
		Scope: etron.BotCommandScope{
			Type:   etron.BCSTDefault,
			ChatID: b.chatID,
			// UserID: 0,
		},
	}, irt.Collect(getBotCommands())...)

	grip.Error(err)
	grip.Info(resp)
}

func (b *bot) selectSinger() stateFn {
	defer b.state.has.Add(dispatch.MinutesAppQueryTypeLeader)
	grip.Debug("selecting singer")
	return b.wrapInputAsHandler(b.captureInputAsName, b.discoverNext)
}

func (b *bot) selectSong() stateFn {
	defer b.state.has.Add(dispatch.MinutesAppQueryTypeSong)
	grip.Debug("selecting song")
	return b.wrapInputAsHandler(b.captureInputAsName, b.discoverNext)
}

func (b *bot) selectSinging() stateFn {
	defer b.state.has.Add(dispatch.MinutesAppQueryTypeSinging)
	grip.Debug("selecting singing")
	return b.wrapInputAsHandler(b.captureInputAsName, b.discoverNext)
}

func (b *bot) selectYear() stateFn {
	defer b.state.has.Add(dispatch.MinutesAppQueryTypeYear)
	grip.Debug("selecting year")
	return b.wrapInputAsHandler(b.captureInputAsYears, b.discoverNext)
}

func (b *bot) selectLocality() stateFn {
	defer b.state.has.Add(dispatch.MinutesAppQueryTypeLocality)
	grip.Debug("selecting locality")
	return b.wrapInputAsHandler(b.captureInputAsName, b.discoverNext)
}

func (b *bot) selectKey() stateFn {
	defer b.state.has.Add(dispatch.MinutesAppQueryTypeKey)
	grip.Debug("selecting key")
	return b.wrapInputAsHandler(b.captureInputAsName, b.discoverNext)
}
