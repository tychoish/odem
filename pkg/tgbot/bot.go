package tgbot

import (
	"cmp"
	"context"
	"iter"
	"sync/atomic"
	"time"

	etron "github.com/NicoNex/echotron/v3"
	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/level"
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
	queryState  struct {
		has               *dt.Set[dispatch.MinutesAppQueryType]
		entry             *dispatch.MinutesAppRegistration
		op                *dispatch.MinutesAppOperation
		inProgress        bool
		params            models.Params
		selectionAttempts int
	}
	state struct {
		info             *etron.ChatFullInfo
		trackingKeyboard atomic.Int64
	}
}

func (b *bot) resetState() stateFn {
	b.queryState.entry = nil
	b.queryState.op = nil
	b.queryState.inProgress = false
	b.queryState.has = &dt.Set[dispatch.MinutesAppQueryType]{}
	b.queryState.selectionAttempts = 0
	b.queryState.params = models.Params{
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
		grip.Debug(b.grip("got message").
			KV("from", u.Message.From.Username).
			KV("recv", b.recv.Add(1)).
			KV("since_last", btwn.String()).
			KV("text", u.Message.Text))
	case u.CallbackQuery != nil:
		grip.Debug(b.grip("got query callback").
			KV("from", u.CallbackQuery.From.Username).
			KV("recv", b.recv.Add(1)).
			KV("since_last", btwn.String()).
			KV("data", u.CallbackQuery.Data))
	default:
		mut := toJson(u)
		defer mut.Release()
		grip.Debug(b.grip("recv unknown message").
			KV("recv", b.recv.Add(1)).
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
		if b.state.trackingKeyboard.Load() != 0 {
			return b.handleKeyboardResponse(0)(u.CallbackQuery.Data)
		}
		b.sendPlain("Sorry, that didn't quite work, let's... try the shapes again")
		fallthrough
	default:
		return b.handleMessage
	}
}

func (b *bot) sendMarkdown(msg string) {
	grip.Debug(b.grip("sending markdown message").KV("outID", b.sent.Add(1)))

	b.handleSendMessage(b.SendMessage(msg, b.chatID, &etron.MessageOptions{ParseMode: etron.Markdown, MessageThreadID: int64(b.threadID)}))
}

func (b *bot) sendPlain(msg string) {
	grip.Debug(b.grip("sending plain message").KV("outID", b.sent.Add(1)))

	b.handleSendMessage(b.SendMessage(msg, b.chatID, &etron.MessageOptions{MessageThreadID: int64(b.threadID)}))
}

func (b *bot) handleSendMessage(resp etron.APIResponseMessage, err error) {
	grip.Error(err)
	grip.Send(b.gmr("sent message response", resp.Base()).Level(b.level()).WithError(err).Extend(kvsFromMessage(resp.Result)))
}

func (b *bot) handleAPIResponse(resp etron.APIResponseBase, err error) {
	grip.Error(err)
	grip.Send(b.gmr("sent message response", resp.Base()).Level(b.level()).WithError(err))
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
	list.PushBack(irt.MakeKV("msg.text", any(m.Text)))
	return irt.KVsplit(list.IteratorFront())
}

func (b *bot) grip(op string) *message.KV {
	return grip.KV("op", op).KV("chatID", b.chatID).KV("threadID", b.threadID)
}

func (b *bot) gmr(op string, resp etron.APIResponseBase) *message.KV {
	return b.grip(op).
		WhenKV(!resp.Ok, "code", resp.ErrorCode).
		WhenKV(resp.Description != "", "description", resp.Description)
}

func (b *bot) level() level.Priority {
	if b.conf.Telegram.Quiet {
		return level.Debug
	}
	return level.Info
}
