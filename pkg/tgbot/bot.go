package tgbot

import (
	"cmp"
	"context"
	"sync/atomic"
	"time"

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
	ctx         context.Context
	db          *db.Connection
	conf        *odem.Configuration
	off         *atomic.Bool
	recv        atomic.Int64
	sent        atomic.Int64
	lastUpdated time.Time
	state       struct {
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
	b.updateMetrics(update)
	if b.stateMachine != nil {
		b.stateMachine = b.stateMachine(update)
	} else {
		b.stateMachine = b.handleMessage(update)
	}
}

func (b *bot) setLastUpdated(t time.Time) { b.lastUpdated = t }
func (b *bot) updateMetrics(u *etron.Update) {
	btwn := time.Since(b.lastUpdated)
	defer b.setLastUpdated(b.lastUpdated.Add(btwn))
	switch {
	case u.Message != nil:
		b.threadID = cmp.Or(b.threadID, u.Message.ThreadID)
		grip.Debug(grip.KV("op", "got message").
			KV("from", u.Message.From.Username).
			KV("recv", b.recv.Add(1)).
			KV("chatID", b.chatID).
			KV("since_last", btwn.String()).
			KV("text", u.Message.Text))
	case u.CallbackQuery != nil:
		b.threadID = cmp.Or(b.threadID, u.CallbackQuery.Message.ThreadID)
		grip.Debug(grip.KV("op", "got query callback").
			KV("from", u.CallbackQuery.From.Username).
			KV("recv", b.recv.Add(1)).
			KV("chatID", b.chatID).
			KV("thread", b.threadID).
			KV("since_last", btwn.String()).
			KV("data", u.CallbackQuery.Data))
	default:
		mut := toJson(u)
		defer mut.Release()
		grip.Debug(grip.KV("op", "recv unknown message").
			KV("recv", b.recv.Add(1)).
			KV("chatID", b.chatID).
			KV("thread", b.threadID).
			KV("since_last", btwn.String()).
			KV("body", mut.String()))
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
		return b.handleMessage
	}
}

func (b *bot) sendMarkdown(msg string) {
	grip.Debug(grip.KV("op", "sending markdown message").
		KV("chatID", b.chatID).
		KV("threadID", b.threadID).
		KV("outID", b.sent.Add(1)))

	b.handleSendMessage(b.SendMessage(msg, b.chatID, &etron.MessageOptions{ParseMode: etron.Markdown, MessageThreadID: int64(b.threadID)}))
}

func (b *bot) sendPlain(msg string) {
	grip.Debug(grip.KV("op", "sending plain message").
		KV("chatID", b.chatID).
		KV("threadID", b.threadID).
		KV("outID", b.sent.Add(1)))

	b.handleSendMessage(b.SendMessage(msg, b.chatID, &etron.MessageOptions{MessageThreadID: int64(b.threadID)}))
}

func (b *bot) handleSendMessage(resp etron.APIResponseMessage, err error) {
	grip.Error(err)
	grip.Debug(message.When(b.conf.Telegram.Quiet, resp))
	grip.Info(message.When(!b.conf.Telegram.Quiet, resp))
}
