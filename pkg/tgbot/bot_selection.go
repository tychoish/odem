package tgbot

import (
	"fmt"

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
	case dispatch.MinutesAppQueryTypeWord:
		return b.selectWord()
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

func (b *bot) selectSinger() stateFn {
	defer b.queryState.has.Add(dispatch.MinutesAppQueryTypeLeader)
	grip.Debug(b.grip("selecting singer"))
	b.sendMarkdown("which singer are you looking for?")
	return b.wrapInputAsHandler(b.captureLeader, b.discoverNext)
}

func (b *bot) selectSong() stateFn {
	defer b.queryState.has.Add(dispatch.MinutesAppQueryTypeSong)
	grip.Debug(b.grip("selecting song"))
	b.sendMarkdown("which song (title or page number) are you looking for?")
	return b.wrapInputAsHandler(b.captureSong, b.discoverNext)
}

func (b *bot) selectSinging() stateFn {
	defer b.queryState.has.Add(dispatch.MinutesAppQueryTypeSinging)
	grip.Debug(b.grip("selecting singing"))
	b.sendMarkdown("which singing are you looking for?")
	return b.wrapInputAsHandler(b.captureSinging, b.discoverNext)
}

func (b *bot) selectYear() stateFn {
	defer b.queryState.has.Add(dispatch.MinutesAppQueryTypeYear)
	grip.Debug(b.grip("selecting year"))
	b.sendMarkdown("which year would you like to filter by?")
	return b.wrapInputAsHandler(b.captureYears, b.discoverNext)
}

func (b *bot) selectLocality() stateFn {
	defer b.queryState.has.Add(dispatch.MinutesAppQueryTypeLocality)
	grip.Debug(b.grip("selecting locality"))
	b.sendMarkdown("what locality would you like to filter by (state codes)?")
	return b.wrapInputAsHandler(b.captureInputAsName, b.discoverNext)
}

func (b *bot) selectKey() stateFn {
	defer b.queryState.has.Add(dispatch.MinutesAppQueryTypeKey)
	grip.Debug(b.grip("selecting key"))
	b.sendMarkdown("what key would you like to filter by?")
	return b.wrapInputAsHandler(b.captureKey, b.discoverNext)
}

func (b *bot) selectWord() stateFn {
	defer b.queryState.has.Add(dispatch.MinutesAppQueryTypeWord)
	grip.Debug("selecting word")
	b.sendMarkdown("what word would you like to find?")
	return b.wrapInputAsHandler(b.captureWord, b.discoverNext)
}
