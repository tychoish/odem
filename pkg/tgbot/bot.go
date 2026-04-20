package tgbot

import (
	"cmp"
	"context"
	"strings"
	"sync/atomic"
	"time"

	etron "github.com/NicoNex/echotron/v3"
	"github.com/tychoish/fun/dt"
	"github.com/tychoish/grip"
	"github.com/tychoish/odem"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/dispatch"
	"github.com/tychoish/odem/pkg/models"
)

type stateFn func(*etron.Update) stateFn

// bot is a pure state machine for a single chat thread. All routing is handled
// by metabot; bot instances are created and dispatched to exclusively by their
// parent metabot and never route updates themselves.
type bot struct {
	chatID       int64
	threadID     int
	botID        int64   // Telegram user ID of the bot itself (0 if unknown)
	botName      string  // @username without leading @ (empty if unknown)
	stateMachine stateFn // current state; replaced after every update
	etron.API
	ctx     context.Context
	db      *db.Connection
	conf    *odem.Configuration
	off     *atomic.Bool
	metrics struct {
		recv          atomic.Int64
		sent          atomic.Int64
		filesSent     atomic.Int64
		filesSentSize atomic.Int64
	}
	lastUpdated time.Time
	queryState  struct {
		has               *dt.Set[dispatch.MinutesAppQueryType]
		entry             *dispatch.MinutesOpRegistration
		op                *dispatch.MinutesOperation
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
	b.queryState.params = models.Params{Limit: 10}
	return b.handleMessage
}

// Update executes one state-machine step. It is called by metabot on every
// routed update.
func (b *bot) Update(update *etron.Update) {
	b.updateMetrics(update)
	if b.stateMachine != nil {
		b.stateMachine = b.stateMachine(update)
	} else {
		b.stateMachine = b.handleMessage(update)
	}
}

func (b *bot) maxSelectionAttempts() int  { return cmp.Or(b.conf.Telegram.MaxSelectionAttempts, 3) }
func (b *bot) setLastUpdated(t time.Time) { b.lastUpdated = t }
func (b *bot) updateMetrics(u *etron.Update) {
	btwn := time.Since(b.lastUpdated)
	defer b.setLastUpdated(b.lastUpdated.Add(btwn))
	switch {
	case u.Message != nil:
		grip.Debug(b.grip("got message").
			KV("from", u.Message.From.Username).
			KV("recv", b.metrics.recv.Add(1)).
			KV("since_last", btwn.String()).
			KV("text", u.Message.Text))
	case u.CallbackQuery != nil:
		grip.Debug(b.grip("got query callback").
			KV("from", u.CallbackQuery.From.Username).
			KV("recv", b.metrics.recv.Add(1)).
			KV("since_last", btwn.String()).
			KV("data", u.CallbackQuery.Data))
	default:
		mut := toJson(u)
		defer mut.Release()
		grip.Debug(b.grip("recv unknown message").
			KV("recv", b.metrics.recv.Add(1)).
			KV("since_last", btwn.String()).
			KV("body", mut.String()))
	}
}

func (b *bot) handleMessage(u *etron.Update) stateFn {
	switch {
	case u.Message != nil:
		return b.dispatchMessage(u.Message)
	case u.EditedMessage != nil:
		return b.dispatchMessage(u.EditedMessage)
	case u.CallbackQuery != nil:
		switch {
		case b.state.trackingKeyboard.Load() != 0:
			return b.handleKeyboardResponse(u.CallbackQuery.Data)
		case b.setupQuery(strings.ToLower(u.CallbackQuery.Data)):
			return b.discoverNext()
		case u.CallbackQuery.Message != nil:
			b.sendPlain("Sorry, that didn't quite work, let's... try the shapes again")
			return b.dispatchMessage(u.CallbackQuery.Message)
		}

		fallthrough
	default:
		return b.handleMessage
	}
}

func (b *bot) sendDocument(filename string, content []byte) {
	grip.Debug(b.grip("sending document").
		KV("outID", b.metrics.sent.Add(1)).
		KV("fileID", b.metrics.filesSent.Add(1)).
		KV("filename", filename).
		KV("size", b.metrics.filesSentSize.Add(int64(len(content)))))
	b.handleSendMessage(b.SendDocument(etron.NewInputFileBytes(filename, content), b.chatID, &etron.DocumentOptions{MessageThreadID: int(b.threadID)}))
}

func (b *bot) sendMarkdown(msg string) {
	grip.Debug(b.grip("sending markdown message").KV("outID", b.metrics.sent.Add(1)))
	b.handleSendMessage(b.SendMessage(msg, b.chatID, &etron.MessageOptions{ParseMode: etron.Markdown, MessageThreadID: int64(b.threadID)}))
}

func (b *bot) sendPlain(msg string) {
	grip.Debug(b.grip("sending plain message").KV("outID", b.metrics.sent.Add(1)))
	b.handleSendMessage(b.SendMessage(msg, b.chatID, &etron.MessageOptions{MessageThreadID: int64(b.threadID)}))
}
