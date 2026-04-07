package tgbot

import (
	"context"
	"sync/atomic"
	"time"

	etron "github.com/NicoNex/echotron/v3"
	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/ers"
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/odem"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/dispatch"
	"github.com/tychoish/odem/pkg/models"
)

/*
TODO:
- [x] figure how why there's double posting
- [x] suppress interactivity throughout the codebase
- [ ] build specific telegram bot rendering. (new package, probably)
*/

type Service struct {
	signal <-chan struct{}
	conf   *odem.Configuration
	db     *db.Connection
	ctx    context.Context
	off    atomic.Bool
}

func NewService(ctx context.Context, conf *odem.Configuration, conn *db.Connection) *Service {
	grip.Sender().SetPriority(level.Trace)

	return &Service{signal: ctx.Done(), conf: conf, db: conn, ctx: ctx}
}

func (srv *Service) Start(ctx context.Context) error {
	dsp := etron.NewDispatcher(srv.conf.Telegram.BotToken, srv.MakeBot)
	timer := time.NewTimer(0)
	grip.Info("telegram bot starting")
	for err, count := dsp.Poll(), int64(0); true; err, count = dsp.Poll(), count+1 {
		grip.Debugf("telegram longpoll loop number: %d", count)
		grip.Notice(ers.Wrapf(err, "dispatcher loop num %d", count))
		timer.Reset(5 * time.Second)
		select {
		case <-timer.C:
			if srv.off.Load() {
				grip.Alert("shutdown triggered")
				return nil
			}
			continue
		case <-srv.signal:
			return ctx.Err()
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (srv *Service) MakeBot(chatID int64) etron.Bot {
	b := &bot{
		chatID: chatID,
		API:    etron.NewAPI(srv.conf.Telegram.BotToken),
		db:     srv.db,
		ctx:    srv.ctx,
		conf:   srv.conf,
		off:    &srv.off,
	}
	b.setOperationSelectorButtons()

	return b
}

type stateFn func(*etron.Update) stateFn

type bot struct {
	chatID       int64
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
		return b.handleArbitraryMessage(u.Message, b.selectOperationKeyboard)
	case u.CallbackQuery != nil:
		return b.handleKeyboardResponse(u.CallbackQuery.Data)
	default:
		mut := toJson(u)
		defer mut.Release()
		grip.Debug(mut.String())
	}
	return b.handleMessage
}

func (b *bot) handleArbitraryMessage(msg *etron.Message, fallback func() stateFn) stateFn {
	defer func() {
		if p := recover(); p != nil {
			resp, err := b.Close()
			grip.Error(err)

			if resp.Ok {
				grip.Debug(resp)
			} else {
				grip.Warning(resp)
			}

			switch p {
			case "exit", "abort", "quit":
				b.off.Store(true)
				return
			default:
				panic(p)
			}
		}
	}()

	grip.Infoln("message", msg.Text)
	switch {
	case isOrContainsCmd(msg, "exit"):
		b.sendPlain("ok, exiting!")
		panic("exit")
	case isOrContainsCmd(msg, "abort"):
		b.sendPlain("ok, aborting!")
		panic("abort")
	case isOrContainsCmd(msg, "quit"):
		b.sendPlain("ok, quitting!")
		panic("quit")
	case isOrContainsCmd(msg, "help"):
		// TODO print help text
		return b.selectOperationKeyboard()
	case isOrContainsCmd(msg, "reset"):
		b.sendPlain("resetting query...")
		return b.resetState()
	case isOrContainsCmd(msg, "retry"):
		b.sendPlain("retrying...")
		return b.resetState()
	case isOrContainsCmd(msg, "restart"):
		b.sendPlain("restarting...")
		return b.resetState()
	case !b.state.inProgress:
		if tryTxt := dispatch.NewMinutesAppOperation(msg.Text); tryTxt.Ok() {
			// note, we do check the match twice here in
			// the case that three is a match. could be
			// refactored but not critical
			return b.handleKeyboardResponse(msg.Text)
		}
		return fallback()
	default:
		// TODO maybe try and parse "leader/year/singing/song"
		// from the input text here...
		return fallback()
	}
}

func (b *bot) sendMarkdown(msg string) {
	b.handleSendMessage(b.SendMessage(msg, b.chatID, &etron.MessageOptions{ParseMode: etron.MarkdownV2}))
}

func (b *bot) sendPlain(msg string) {
	b.handleSendMessage(b.SendMessage(msg, b.chatID, &etron.MessageOptions{}))
}

func (b *bot) handleSendMessage(resp etron.APIResponseMessage, err error) {
	grip.Error(err)
	grip.Debug(message.When(b.conf.Telegram.Quiet, resp))
	grip.Info(message.When(!b.conf.Telegram.Quiet, resp))
}
