package tgbot

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	etron "github.com/NicoNex/echotron/v3"
	"github.com/tychoish/grip"
)

// startWebhook registers the Telegram webhook and runs an HTTP server that
// receives updates. It blocks until ctx is cancelled, then gracefully shuts
// down the server and deletes the webhook registration before returning.
func (srv *Service) startWebhook(ctx context.Context, dsp *etron.Dispatcher) error {
	webhookURL := srv.conf.Telegram.Webhook.URL
	listenAddr, err := resolveListenAddr(srv.conf.Telegram.Webhook.Listen, webhookURL)
	if err != nil {
		return fmt.Errorf("resolving webhook listen address: %w", err)
	}

	grip.Info(grip.KV("op", "webhook").KV("status", "starting").KV("listen", listenAddr).KV("url", webhookURL))

	httpSrv := &http.Server{Addr: listenAddr}
	dsp.SetHTTPServer(httpSrv)

	errCh := make(chan error, 1)
	go func() {
		err := dsp.ListenWebhookOptions(webhookURL, true, nil)
		// http.ErrServerClosed is expected on graceful shutdown; suppress it.
		if err != nil && err != http.ErrServerClosed {
			errCh <- err
		} else {
			errCh <- nil
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		grip.Info(grip.KV("op", "webhook").KV("status", "shutting down"))
		return srv.stopWebhook(httpSrv)
	}
}

// stopWebhook gracefully shuts down the HTTP server. The webhook registration
// on Telegram is removed automatically when polling resumes or the bot is
// restarted in polling mode; we only clean it up here as a courtesy so that
// Telegram stops sending requests immediately.
func (srv *Service) stopWebhook(httpSrv *http.Server) error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Best-effort webhook deletion; don't fail shutdown on error.
	api := etron.NewAPI(srv.conf.Telegram.BotToken)
	if _, err := api.DeleteWebhook(false); err != nil {
		grip.Warning(grip.KV("op", "webhook").KV("status", "delete webhook failed").KV("err", err))
	}

	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("webhook server shutdown: %w", err)
	}

	grip.Info(grip.KV("op", "webhook").KV("status", "stopped"))
	return nil
}

// resolveListenAddr returns the address the HTTP server should bind to.
// If listen is explicitly configured it is used as-is. Otherwise the port
// is extracted from webhookURL so that a minimal config only requires the URL.
func resolveListenAddr(listen, webhookURL string) (string, error) {
	if listen != "" {
		return listen, nil
	}
	u, err := url.Parse(webhookURL)
	if err != nil {
		return "", fmt.Errorf("parsing webhook URL %q: %w", webhookURL, err)
	}
	port := u.Port()
	if port == "" {
		switch u.Scheme {
		case "https":
			port = "443"
		case "http":
			port = "80"
		default:
			return "", fmt.Errorf("cannot determine listen port from webhook URL %q", webhookURL)
		}
	}
	return net.JoinHostPort("", port), nil
}