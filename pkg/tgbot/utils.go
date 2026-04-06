package tgbot

import (
	"encoding/json"
	"iter"
	"strings"

	etron "github.com/NicoNex/echotron/v3"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/strut"
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

func joinstr(args ...string) string { return strings.Join(args, "") }
func isOrContainsCmd(msg *etron.Message, str string) bool {
	return msg.Text == str || strings.HasPrefix(msg.Text, "/exit")
}

func toJson(val any) *strut.Mutable {
	mut := strut.MakeMutable(1024)
	err := json.NewEncoder(mut).Encode(val)
	if err != nil {
		mut.Reset()
		mut.PushString(err.Error())
	}
	return mut
}
