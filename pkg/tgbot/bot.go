package tgbot

import (
	"cmp"
	"context"
	"sync/atomic"

	etron "github.com/NicoNex/echotron/v3"
	"github.com/tychoish/fun/dt"
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/odem"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/dispatch"
	"github.com/tychoish/odem/pkg/models"
)

type stateFn func(*etron.Update) stateFn

type bot struct {
	chatID       int64
	threadID     int
	stateMachine stateFn // current state; replaced after every update
	etron.API
	ctx   context.Context
	db    *db.Connection
	conf  *odem.Configuration
	off   *atomic.Bool
	state struct {
		has        *dt.Set[dispatch.MinutesAppQueryType]
		entry      *dispatch.MinutesAppRegistration
		op         *dispatch.MinutesAppOperation
		inProgress bool
		params     models.Params
	}
}

func (b *bot) resetState() stateFn {
	b.state.entry = nil
	b.state.op = nil
	b.state.inProgress = false
	b.state.has = &dt.Set[dispatch.MinutesAppQueryType]{}
	b.state.params = models.Params{
		Limit: 10,
		Years: []int{2025, 2026},
	}
	return b.handleMessage
}

func (b *bot) Update(update *etron.Update) {
	// Execute the current state and store whatever it returns as the next one.
	// A single assignment is all the state-machine machinery needed.
	if b.stateMachine != nil {
		b.stateMachine = b.stateMachine(update)
	} else {
		b.stateMachine = b.handleMessage(update)
	}
}

func (b *bot) handleMessage(u *etron.Update) stateFn {
	switch {
	case u.Message != nil:
		b.threadID = cmp.Or(b.threadID, u.Message.ThreadID)
		return b.dispatchMessage(u.Message)
	case u.CallbackQuery != nil:
		b.threadID = cmp.Or(b.threadID, u.CallbackQuery.Message.ThreadID)
		return b.handleKeyboardResponse(u.CallbackQuery.Data)
	default:
		mut := toJson(u)
		defer mut.Release()
		grip.Debug(mut.String())
	}
	return b.handleMessage
}

func (b *bot) sendMarkdown(msg string) {
	b.handleSendMessage(b.SendMessage(msg, b.chatID, &etron.MessageOptions{ParseMode: etron.Markdown, MessageThreadID: int64(b.threadID)}))
}

func (b *bot) sendPlain(msg string) {
	b.handleSendMessage(b.SendMessage(msg, b.chatID, &etron.MessageOptions{MessageThreadID: int64(b.threadID)}))
}

func (b *bot) handleSendMessage(resp etron.APIResponseMessage, err error) {
	grip.Error(err)
	grip.Debug(message.When(b.conf.Telegram.Quiet, resp))
	grip.Info(message.When(!b.conf.Telegram.Quiet, resp))
}
