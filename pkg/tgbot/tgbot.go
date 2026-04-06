package tgbot

import (
	"context"
	"slices"
	"time"

	etron "github.com/NicoNex/echotron/v3"
	"github.com/tychoish/fun/ers"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/strut"
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/odem"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/dispatch"
	"github.com/tychoish/odem/pkg/models"
	"github.com/tychoish/odem/pkg/reportui"
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
		entry  *dispatch.MinutesAppRegistration
		op     *dispatch.MinutesAppOperation
		params models.Params
	}
}

func (b *bot) resetState() stateFn {
	b.state.entry = nil
	b.state.op = nil
	b.state.params = models.Params{}
	return b.handleMessage
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
		grip.Infoln("message", u.Message.Text)
		switch {
		case isOrContainsCmd(u.Message, "exit"):
			panic("exit")
		case isOrContainsCmd(u.Message, "abort"):
			panic("abort")
		case isOrContainsCmd(u.Message, "quit"):
			panic("quit")
		case isOrContainsCmd(u.Message, "help"):
			b.sendKeyboard()
			return b.handleMessage // TODO split into its own handler
		case isOrContainsCmd(u.Message, "reset"):
			b.sendMarkdown("resetting query...")
			return b.resetState()
		case isOrContainsCmd(u.Message, "retry"):
			b.sendMarkdown("retrying...")
			return b.resetState()
		case isOrContainsCmd(u.Message, "restart"):
			b.sendMarkdown("restarting...")
			return b.resetState()
		}
	case u.CallbackQuery != nil:
		grip.Debug(message.NewKV().KV("type", "callback").KV("body", u.CallbackQuery.Message.Text))
		reg := dispatch.NewMinutesAppOperation(u.CallbackQuery.Data).Registry()

		buf := strut.MakeMutable(1024)
		defer buf.Release()

		buf.PushString("```")
		grip.Error(reg.Reporter.Report(b.ctx, b.db, reportui.Params{
			Params:   models.Params{Name: "Henry Johnson", Limit: 10, Years: []int{2025, 2026}},
			ToWriter: buf,
		}))
		buf.PushString("```")

		b.handleSendMessage(b.SendMessage(buf.String(), b.chatID, &etron.MessageOptions{ParseMode: etron.MarkdownV2}))
	default:
		mut := toJson(u)
		defer mut.Release()
		grip.Debug(mut.String())
	}
	return b.handleMessage
}

func (b *bot) sendMarkdown(msg string) {
	b.handleSendMessage(b.SendMessage(msg, b.chatID, &etron.MessageOptions{ParseMode: etron.MarkdownV2}))
}

func (b *bot) sendKeyboard() {
	btn := irt.Collect(
		irt.Convert(irt.RemoveValue(dispatch.AllMinutesAppOps(), dispatch.MinutesAppOpExit),
			func(mao dispatch.MinutesAppOperation) etron.InlineKeyboardButton {
				reg := mao.Registry().Info()
				return etron.InlineKeyboardButton{Text: reg.Key, CallbackData: reg.Key}
			},
		),
	)

	b.handleSendMessage(b.SendMessage("Choose an option:", b.chatID, &etron.MessageOptions{
		ReplyMarkup: etron.InlineKeyboardMarkup{
			InlineKeyboard: irt.Collect(slices.Chunk(btn, len(btn)/8)),
		},
	}))
}

func (b *bot) handleSendMessage(resp etron.APIResponseMessage, err error) {
	grip.Error(err)
	grip.Debug(message.When(b.conf.Telegram.Quiet, resp))
	grip.Info(message.When(!b.conf.Telegram.Quiet, resp))
}
