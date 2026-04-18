package tgbot

import (
	"iter"

	etron "github.com/NicoNex/echotron/v3"
	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/strut"
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
)

func (b *bot) grip(op string) *message.KV {
	return grip.KV("op", op).KV("chatID", b.chatID).KV("threadID", b.threadID)
}

func (b *bot) gmr(op string, resp etron.APIResponseBase) *message.KV {
	return b.grip(op).
		WhenKV(!resp.Ok, "code", resp.ErrorCode).
		WhenKV(resp.Description != "", "description", resp.Description)
}

func (b *bot) level(err error) level.Priority {
	switch {
	case err != nil:
		return level.Error
	case b.conf.Telegram.Quiet:
		return level.Debug
	default:
		return level.Info
	}
}

func (b *bot) handleSendMessage(resp etron.APIResponseMessage, err error) {
	grip.Send(b.gmr("sent message response", resp.Base()).Level(b.level(err)).WithError(err).Extend(kvsFromMessage(resp.Result)))
}

func (b *bot) handleAPIResponse(resp etron.APIResponseBase, err error) {
	grip.Send(b.gmr("sent message response", resp.Base()).Level(b.level(err)).WithError(err))
}

func kvsFromMessage(m *etron.Message) iter.Seq2[string, any] {
	if m == nil {
		return irt.Zero2[string, any]()
	}
	list := &dt.List[irt.KV[string, any]]{}
	if m.From != nil {
		list.PushBack(irt.MakeKV("from.id", any(m.From.ID)))
		list.PushBack(irt.MakeKV("from.username", any(m.From.Username)))
	}
	list.PushBack(irt.MakeKV("msg.id", any(m.ID)))
	body := strut.MutableFrom(m.Text)
	body.Truncate(32)
	body.ReplaceAllString("\n", "; ")
	body.PushString("...")
	list.PushBack(irt.MakeKV("msg.text", any(body.Resolve())))
	return irt.KVsplit(list.IteratorFront())
}
