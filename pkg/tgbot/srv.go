package tgbot

import (
	"context"
	"sync/atomic"

	etron "github.com/NicoNex/echotron/v3"
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/odem"
	"github.com/tychoish/odem/pkg/db"
)

type Service struct {
	signal func()
	conf   *odem.Configuration
	db     *db.Connection
	ctx    context.Context
	off    atomic.Bool
	count  atomic.Int64
}

func NewService(ctx context.Context, conf *odem.Configuration, conn *db.Connection) *Service {
	grip.Sender().SetPriority(level.Trace)
	ctx, cancel := context.WithCancel(ctx)
	return &Service{signal: cancel, conf: conf, db: conn, ctx: ctx}
}

func (srv *Service) Start(ctx context.Context) error {
	grip.Info(grip.KV("op", "telegram bot starting").
		WhenKV(srv.conf.Telegram.Webhook.Enabled, "mode", "webook").
		WhenKV(!srv.conf.Telegram.Webhook.Enabled, "mode", "longpoll"))

	dsp := etron.NewDispatcher(srv.conf.Telegram.BotToken, srv.MakeBot)

	if srv.conf.Telegram.Webhook.Enabled {
		return srv.startWebhook(ctx, dsp)
	}

	return srv.startPolling(ctx, dsp)
}

func (srv *Service) checkServiceExit(ctx context.Context) (bool, error) {
	switch {
	case srv.off.Load():
		return true, nil
	case ctx.Err() != nil:
		return true, ctx.Err()
	default:
		return false, nil
	}
}

func (srv *Service) MakeBot(chatID int64) etron.Bot {
	api := etron.NewAPI(srv.conf.Telegram.BotToken)
	m := &metabot{
		chatID: chatID,
		api:    api,
		db:     srv.db,
		ctx:    srv.ctx,
		conf:   srv.conf,
		off:    &srv.off,
	}

	me, err := api.GetMe()
	grip.Error(err)
	if me.Result != nil {
		m.botID = me.Result.ID
		m.botName = me.Result.Username
	}

	defaultBot := m.newBot(0)

	ar, err := api.GetChat(chatID)
	grip.Error(err)
	defaultBot.state.info = ar.Result

	m.bots.Store(0, defaultBot)

	grip.Info(grip.KV("op", "starting new bot tracking chat").KV("chatID", chatID).KV("count", srv.count.Add(1)))

	return m
}
