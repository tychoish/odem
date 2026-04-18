package tgbot

import (
	"context"
	"errors"
	"time"

	etron "github.com/NicoNex/echotron/v3"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/ers"
	"github.com/tychoish/fun/stw"
	"github.com/tychoish/grip"
)

func (srv *Service) startPolling(ctx context.Context, dsp *etron.Dispatcher) error {
	timer := time.NewTimer(0)
	defer timer.Stop()

	var count int
	for {
		count++
		var apierr *etron.APIError
		select {
		case err := <-srv.doPoll(ctx, count, dsp):
			switch {
			case err == nil:
				grip.Debug(grip.KV("op", "longpoll").KV("status", "normal").KV("outcome", "continue"))
			case ers.IsExpiredContext(err):
				return err
			case errors.As(err, &apierr):
				grip.Info(grip.KV("op", "longpoll").KV("status", "api-error").KV("code", apierr.ErrorCode()).KV("desc", apierr.Description()).KV("outcome", "continue"))
			case ers.IsTerminating(err):
				grip.Alert(grip.KV("op", "longpoll").KV("status", "terminating").WithError(err))
				return nil
			default:
				grip.Error(grip.KV("op", "longpoll").KV("status", "non-terminating").WithError(err).KV("outcome", "continue"))
			}
			timer.Reset(5 * time.Second)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
			}
		case <-ctx.Done():
			return ctx.Err()
		}

	}
}

func (srv *Service) doPoll(ctx context.Context, count int, dsp *etron.Dispatcher) <-chan error {
	startAt := time.Now()
	grip.Debug(grip.KV("op", "longpoll").KV("count", count).KV("status", "starting"))
	ch := make(chan error)
	go func() {
		var ec erc.Collector
		defer close(ch)
		defer ec.Recover()
		defer func() { stw.BlockingSend(ch).Ignore(ctx, ec.Resolve()) }()
		grip.Debug(grip.KV("op", "longpoll").KV("count", count).KV("status", "launching from goroutine").KV("after", time.Since(startAt)))
		ec.Check(dsp.Poll)
		grip.Debug(grip.KV("op", "longpoll").KV("count", count).KV("status", "returning from goroutine").KV("after", time.Since(startAt)))
	}()

	grip.Debug(grip.KV("op", "longpoll").KV("count", count).KV("status", "returning to main loop").KV("after", time.Since(startAt)))
	return ch
}
