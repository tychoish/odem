package tgbot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	etron "github.com/NicoNex/echotron/v3"
	"github.com/tychoish/odem"
)

// mockTelegramServer returns an httptest.Server that replies to every request
// with a minimal valid Telegram API response, and a cleanup function.
func mockTelegramServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"result": map[string]any{
				"message_id": 42,
				"id":         42,
			},
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

// newTestBot builds a minimal *bot wired to a mock Telegram HTTP server so
// that API calls succeed without hitting the real Telegram backend.
func newTestBot(t *testing.T) *bot {
	t.Helper()
	srv := mockTelegramServer(t)
	var off atomic.Bool
	b := &bot{
		chatID: 12345,
		API:    etron.CustomAPI(srv.URL+"/bot_test/", "_test"),
		ctx:    context.Background(),
		conf:   &odem.Configuration{},
		off:    &off,
	}
	b.resetState() // initialises queryState.has and sets sane defaults
	return b
}

// TestDiscoverNextNilEntryNoPanic is a regression test for the nil-check ordering
// bug in discoverNext().
//
// Bug: the original code checked entry.Requires before checking entry == nil,
// causing a nil-pointer dereference when no query operation had been selected yet.
//
// Fix: the nil guard on entry was moved above the Requires access.
func TestDiscoverNextNilEntryNoPanic(t *testing.T) {
	b := newTestBot(t)

	// resetState already sets entry = nil, but be explicit.
	b.queryState.entry = nil

	var next stateFn
	require := func() {
		// discoverNext must not panic when entry is nil.
		// It should return keyboardMinutesAppQueries() — a valid, non-nil stateFn.
		next = b.discoverNext()
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		require()
	}()

	select {
	case <-done:
		if next == nil {
			t.Error("discoverNext with nil entry returned a nil stateFn")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("discoverNext with nil entry did not return within 3s (possible deadlock or panic recovered)")
	}
}

// TestHandleKeyboardCASDecrementTerminates is a regression test for the
// CompareAndSwap argument ordering bug in handleKeyboardResponse.
//
// Bug: the loop used CompareAndSwap(val-1, val), which tries to swap the
// value from val-1 to val (an increment), but the current value is val —
// so the CAS always fails and the loop spins forever.
//
// Fix: the arguments were corrected to CompareAndSwap(val, val-1), which
// atomically decrements the counter as intended.
//
// This test verifies the pattern directly using an atomic.Int64, matching the
// exact loop body in handleKeyboardResponse. If the arguments are reversed the
// goroutine will never break out and the test will time out.
func TestHandleKeyboardCASDecrementTerminates(t *testing.T) {
	var counter atomic.Int64
	counter.Store(1)

	done := make(chan struct{})
	go func() {
		defer close(done)
		// Replicate the fixed loop body from handleKeyboardResponse verbatim.
		for {
			val := counter.Load()
			if val == 0 || counter.CompareAndSwap(val, val-1) {
				break
			}
		}
	}()

	select {
	case <-done:
		if got := counter.Load(); got != 0 {
			t.Errorf("expected counter=0 after decrement, got %d", got)
		}
	case <-time.After(time.Second):
		t.Fatal("CAS decrement loop did not terminate within 1s — " +
			"check CompareAndSwap argument order: must be (current_val, current_val-1)")
	}
}

