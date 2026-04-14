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
		return b.keyboardMinutesAppQueries()
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
		b.sendMarkdown(fmt.Sprintf("❗invalid option: `%s`: %s. Let's start over! ⏪", requirement, requirement.Validate()))
		return b.resetState()
	case dispatch.MinutesAppQueryTypeUnknown:
		b.sendPlain("❗Sorry, something went wrong: we need to start over... 😞")
		b.setOperationSelectorButtons()
		return b.resetState()
	default:
		b.sendMarkdown(fmt.Sprintf("Sorry, got an invalid option(`%s`: %s) and need to start over 😥", requirement, requirement.Validate()))
		return b.resetState()
	}
}

func (b *bot) keyboardMinutesAppQueries() stateFn {
	btn := irt.Collect(
		irt.Convert(irt.RemoveValue(dispatch.AllMinutesAppOps(), dispatch.MinutesAppOpExit),
			func(mao dispatch.MinutesAppOperation) etron.InlineKeyboardButton {
				reg := mao.Registry().Info()
				return etron.InlineKeyboardButton{Text: reg.Key, CallbackData: reg.Key}
			},
		),
	)

	b.handleSendMessage(b.SendMessage("Choose an option:", b.chatID, &etron.MessageOptions{
		MessageThreadID: int64(b.threadID),
		ReplyMarkup: etron.InlineKeyboardMarkup{
			InlineKeyboard: irt.Collect(slices.Chunk(btn, len(btn)/8)),
		},
	}))

	return b.wrapInputAsHandler(b.handleKeyboardResponse, b.keyboardMinutesAppQueries)
}

func (b *bot) keyboardHelpMenu() stateFn {
	rsp, err := b.SendMessage("Choose an option:", b.chatID, &etron.MessageOptions{
		MessageThreadID: int64(b.threadID),
		ReplyMarkup: etron.InlineKeyboardMarkup{
			InlineKeyboard: [][]etron.InlineKeyboardButton{
				{},
			},
		},
	})
	grip.Error(err)
	b.state.toDelete.PushBack(rsp.Result.ID)
	return b.wrapInputAsHandler(b.handleKeyboardResponse, b.keyboardMinutesAppQueries)
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
	defer b.queryState.has.Add(dispatch.MinutesAppQueryTypeLeader)
	grip.Debug("selecting singer")
	b.sendMarkdown("which singer are you looking for?")
	return b.wrapInputAsHandler(b.captureLeader, b.discoverNext)
}

func (b *bot) selectSong() stateFn {
	defer b.queryState.has.Add(dispatch.MinutesAppQueryTypeSong)
	grip.Debug("selecting song")
	b.sendMarkdown("which song (title or page number) are you looking for?")
	return b.wrapInputAsHandler(b.captureSong, b.discoverNext)
}

func (b *bot) selectSinging() stateFn {
	defer b.queryState.has.Add(dispatch.MinutesAppQueryTypeSinging)
	grip.Debug("selecting singing")
	b.sendMarkdown("which singing are you looking for?")
	return b.wrapInputAsHandler(b.captureSinging, b.discoverNext)
}

func (b *bot) selectYear() stateFn {
	defer b.queryState.has.Add(dispatch.MinutesAppQueryTypeYear)
	grip.Debug("selecting year")
	b.sendMarkdown("which year would you like to filter by?")
	return b.wrapInputAsHandler(b.captureYears, b.discoverNext)
}

func (b *bot) selectLocality() stateFn {
	defer b.queryState.has.Add(dispatch.MinutesAppQueryTypeLocality)
	grip.Debug("selecting locality")
	b.sendMarkdown("what locality would you like to filter by (state codes)?")
	return b.wrapInputAsHandler(b.captureInputAsName, b.discoverNext)
}

func (b *bot) selectKey() stateFn {
	defer b.queryState.has.Add(dispatch.MinutesAppQueryTypeKey)
	grip.Debug("selecting key")
	b.sendMarkdown("what key would you like to filter by?")
	return b.wrapInputAsHandler(b.captureKey, b.discoverNext)
}
