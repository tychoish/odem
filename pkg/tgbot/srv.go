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
}

func NewService(ctx context.Context, conf *odem.Configuration, conn *db.Connection) *Service {
	grip.Sender().SetPriority(level.Trace)
	ctx, cancel := context.WithCancel(ctx)
	return &Service{signal: cancel, conf: conf, db: conn, ctx: ctx}
}

func (srv *Service) Start(ctx context.Context) error {
	dsp := etron.NewDispatcher(srv.conf.Telegram.BotToken, srv.MakeBot)
	if srv.conf.Telegram.Webhook.Enabled {
		grip.Info("telegram bot starting in webhook mode")
		return srv.startWebhook(ctx, dsp)
	}
	grip.Info("telegram bot starting in polling mode")
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
