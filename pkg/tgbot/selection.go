package tgbot

import (
	"iter"
	"strings"

	etron "github.com/NicoNex/echotron/v3"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/odem/pkg/dispatch"
)

func getBotCommands() iter.Seq[etron.BotCommand] {
	return irt.Convert(
		irt.RemoveValue(dispatch.AllMinutesAppOps(), dispatch.MinutesAppOpExit),
		func(mao dispatch.MinutesAppOperation) etron.BotCommand {
			reg := mao.Registry().Info()
			return etron.BotCommand{Command: joinstr("/", strings.ReplaceAll(reg.Key, "-", "")), Description: reg.Value}
		},
	)
}

func (b *bot) selectSinger() stateFn   { return nil }
func (b *bot) selectSing() stateFn     { return nil }
func (b *bot) selectSinging() stateFn  { return nil }
func (b *bot) selectYear() stateFn     { return nil }
func (b *bot) selectLocality() stateFn { return nil }
func (b *bot) selectKey() stateFn      { return nil }
