package tgbot

import (
	"encoding/json"
	"strings"

	etron "github.com/NicoNex/echotron/v3"
	"github.com/tychoish/fun/strut"
)

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
