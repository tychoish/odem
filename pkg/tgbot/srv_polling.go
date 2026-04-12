package tgbot

import (
	"context"
	"time"

	etron "github.com/NicoNex/echotron/v3"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/stw"
	"github.com/tychoish/grip"
)

func (srv *Service) startPolling(ctx context.Context, dsp *etron.Dispatcher) error {
	timer := time.NewTimer(0)
	var count int
	for {
		count++
		select {
		case err := <-srv.doPoll(ctx, count, dsp):
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}

		if shouldExit, err := srv.checkServiceExit(ctx); shouldExit {
			return err
		}

		timer.Reset(5 * time.Second)
		select {
		case <-timer.C:
			continue
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
