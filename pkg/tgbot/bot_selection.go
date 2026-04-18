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
		return b.promptFor(dispatch.MinutesAppQueryTypeLeader, "which singer are you looking for?", b.captureLeader)
	case dispatch.MinutesAppQueryTypeSong:
		return b.promptFor(dispatch.MinutesAppQueryTypeSong, "which song (title or page number) are you looking for?", b.captureSong)
	case dispatch.MinutesAppQueryTypeSinging:
		return b.promptFor(dispatch.MinutesAppQueryTypeSinging, "which singing are you looking for?", b.captureSinging)
	case dispatch.MinutesAppQueryTypeYear:
		return b.promptFor(dispatch.MinutesAppQueryTypeYear, "which year would you like to filter by?", b.captureYears)
	case dispatch.MinutesAppQueryTypeKey:
		return b.promptFor(dispatch.MinutesAppQueryTypeKey, "what key would you like to filter by?", b.captureKey)
	case dispatch.MinutesAppQueryTypeLocality:
		return b.promptFor(dispatch.MinutesAppQueryTypeLocality, "what locality would you like to filter by (state codes)?", b.captureInputAsName)
	case dispatch.MinutesAppQueryTypeWord:
		return b.promptFor(dispatch.MinutesAppQueryTypeWord, "what word would you like to find?", b.captureWord)
	case dispatch.MinutesAppQueryTypeInvalid:
		b.sendMarkdown(fmt.Sprintf("❗invalid option: `%s`: %s. Let's start over! ⏪", requirement, requirement.Validate()))
		return b.resetState()
	case dispatch.MinutesAppQueryTypeUnknown:
		b.sendPlain("❗Sorry, something went wrong: we need to start over... 😞")
		return b.resetState()
	default:
		b.sendMarkdown(fmt.Sprintf("Sorry, got an invalid option(`%s`: %s) and need to start over 😥", requirement, requirement.Validate()))
		return b.resetState()
	}
}

func (b *bot) promptFor(queryType dispatch.MinutesAppQueryType, prompt string, handler func(string) stateFn) stateFn {
	defer b.queryState.has.Add(queryType)
	grip.Debug(b.grip("selecting").KV("type", queryType))
	b.sendMarkdown(prompt)
	return b.wrapInputAsHandler(handler, b.discoverNext)
}
