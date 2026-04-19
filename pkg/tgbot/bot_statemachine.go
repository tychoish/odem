package tgbot

import (
	"bytes"
	"fmt"

	etron "github.com/NicoNex/echotron/v3"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/strut"
	"github.com/tychoish/grip"
	"github.com/tychoish/odem/pkg/dispatch"
)

func (b *bot) discoverNext() stateFn {
	if b.queryState.entry == nil {
		grip.Info(grip.KV("state", "discoverNext").
			KV("status", "entry nil; retry keyboard"))
		return b.keyboardMinutesAppQueries()
	}
	if b.queryState.entry.Requires == nil {
		grip.Info(grip.KV("state", "discoverNext").
			KV("status", "requirements nil; rendering off the bat").
			KV("op", b.queryState.entry.Command))
		return b.renderResults()
	}
	if b.queryState.inProgress && b.queryState.has == nil {
		grip.Info(grip.KV("state", "discoverNext").
			KV("status", "requirements Set undefined; rendering").
			KV("op", b.queryState.entry.Command))
		return b.renderResults()
	}
	for requirement := range irt.Remove(irt.RemoveValue(b.queryState.entry.Requires.Iterator(),
		dispatch.MinutesAppQueryTypeDocumentOutput), b.queryState.has.Check,
	) {
		grip.Debug(grip.
			KV("state", "discoverNext").
			KV("status", "discovering next value").
			KV("requirement", requirement).
			KV("op", b.queryState.entry.Command),
		)
		return b.selectFor(requirement)
	}
	grip.Debug(grip.
		KV("state", "discoverNext").
		KV("status", "rendering").
		KV("op", b.queryState.entry.Command),
	)
	return b.renderResults()
}

func (b *bot) wrapInputAsHandler(in func(string) stateFn, fallback func() stateFn) stateFn {
	return func(u *etron.Update) stateFn {
		var text string
		switch {
		case u.Message != nil:
			text = u.Message.Text
		case u.CallbackQuery != nil:
			text = u.CallbackQuery.Data
		default:
			return fallback()
		}
		if isEscapeInput(text) {
			b.sendPlain("ok, starting over...")
			b.queryState.selectionAttempts = 0
			return b.resetState()
		}
		return in(text)
	}
}

// captureRetry sends errMsg and returns to the selection loop, but aborts
// back to the top level after maxSelectionAttempts consecutive failures.
func (b *bot) captureRetry(errMsg string, retry func(string) stateFn) stateFn {
	b.queryState.selectionAttempts++
	max := b.maxSelectionAttempts()
	if b.queryState.selectionAttempts >= max {
		b.queryState.selectionAttempts = 0
		b.sendMarkdown(fmt.Sprintf("%s after %d tries — starting over", errMsg, max))
		return b.resetState()
	}
	b.sendMarkdown(fmt.Sprintf("%s (attempt %d/%d — or say `cancel` to start over)", errMsg, b.queryState.selectionAttempts, max))
	return b.wrapInputAsHandler(retry, b.discoverNext)
}

func (b *bot) renderResults() stateFn {
	grip.Info(grip.KV("status", "rendering now...").KV("state", b.queryState.params).KV("command", b.queryState.op.String()))

	working := strut.MakeMutable(64)
	defer working.Release()

	working.Concat("⏳ working on ", b.queryState.op.String())
	working.WhenConcat(b.queryState.params.Name != "", " for ", b.queryState.params.Name)
	if len(b.queryState.params.Years) > 0 {
		working.PushString(" (")
		for i, y := range b.queryState.params.Years {
			working.WhenConcat(i > 0, ", ")
			working.PushInt(y)
		}
		working.PushString(") ")
	}
	working.PushString("...")

	if resp, err := b.SendMessage(working.String(), b.chatID, &etron.MessageOptions{MessageThreadID: int64(b.threadID)}); err == nil && resp.Result != nil {
		defer func() { b.handleAPIResponse(b.DeleteMessage(b.chatID, resp.Result.ID)) }()
	}

	entry := b.queryState.entry
	if entry.IsDocumentOp() {
		var buf bytes.Buffer
		if err := entry.CallReporterToWriter(b.ctx, b.db, b.queryState.params, &buf); err != nil {
			grip.Alert(grip.KV("op", entry.Command).KV("outcome", "error").KV("query", b.queryState.params))
			b.sendPlain(fmt.Sprintf("❗got error producing results: %v", err))
		} else {
			b.sendDocument(entry.DocumentFilename(b.queryState.params), buf.Bytes())
		}
		return b.resetState()
	}

	for msg, err := range entry.GetMessenger()(b.ctx, b.db, b.queryState.params) {
		if err != nil {
			grip.Alert(grip.KV("op", entry.Command).KV("outcome", "overflow").KV("query", b.queryState.params))
			b.sendPlain(fmt.Sprintf("❗got error producing results: %v", err))
			break
		}
		b.sendMarkdown(msg.Resolve())
	}

	return b.resetState()
}
