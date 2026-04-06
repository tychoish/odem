package tgbot

import (
	"context"
	"time"

	etron "github.com/NicoNex/echotron/v3"
	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/ers"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/odem"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/dispatch"
	"github.com/tychoish/odem/pkg/models"
)

type Service struct {
	signal <-chan struct{}
	conf   *odem.Configuration
	db     *db.Connection
	ctx    context.Context
}

func NewService(ctx context.Context, conf *odem.Configuration, conn *db.Connection) *Service {
	return &Service{signal: ctx.Done(), conf: conf, db: conn, ctx: ctx}
}

func (srv *Service) Start(ctx context.Context) error {
	dsp := etron.NewDispatcher(srv.conf.Telegram.BotToken, srv.MakeBot)
	timer := time.NewTimer(0)
	for err, count := dsp.Poll(), int64(0); true; err, count = dsp.Poll(), count+1 {
		grip.Notice(ers.Wrapf(err, "dispatcher loop num %d", count))
		timer.Reset(5 * time.Second)
		select {
		case <-timer.C:
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
	}
	b.resetState()

	resp, err := b.SetMyCommands(&etron.CommandOptions{
		LanguageCode: "en",
		Scope: etron.BotCommandScope{
			Type:   etron.BCSTDefault,
			ChatID: b.chatID,
			// UserID: 0,
		},
	}, irt.Collect(getBotCommands())...)

	grip.Error(err)
	grip.Info(resp)

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
	b.state.params = models.Params{
		Limit: 10,
		Years: []int{2025, 2026},
	}
	return b.sendKeyboard()
}

func (b *bot) Update(update *etron.Update) {
	// Execute the current state and store whatever it returns as the next one.
	// A single assignment is all the state-machine machinery needed.
	if b.stateMachine != nil {
		b.stateMachine = b.stateMachine(update)
	}
	b.stateMachine = b.handleMessage(update)
}

func (b *bot) handleMessage(u *etron.Update) stateFn {
	switch {
	case u.Message != nil:
		return b.handleArbitraryMessage(u.Message, b.sendKeyboard)
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
	grip.Infoln("message", msg.Text)
	switch {
	case isOrContainsCmd(msg, "exit"):
		panic("exit")
	case isOrContainsCmd(msg, "abort"):
		panic("abort")
	case isOrContainsCmd(msg, "quit"):
		panic("quit")
	case isOrContainsCmd(msg, "help"):
		b.selectOperation()
		// TODO print some kind of help text
		return b.sendKeyboard()
	case isOrContainsCmd(msg, "reset"):
		b.sendMarkdown("resetting query...")
		return b.resetState()
	case isOrContainsCmd(msg, "retry"):
		b.sendMarkdown("retrying...")
		return b.resetState()
	case isOrContainsCmd(msg, "restart"):
		b.sendMarkdown("restarting...")
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

func (b *bot) handleSendMessage(resp etron.APIResponseMessage, err error) {
	grip.Error(err)
	grip.Debug(message.When(b.conf.Telegram.Quiet, resp))
	grip.Info(message.When(!b.conf.Telegram.Quiet, resp))
}
