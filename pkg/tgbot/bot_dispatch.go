package tgbot

import (
	etron "github.com/NicoNex/echotron/v3"
	"github.com/tychoish/grip"
	"github.com/tychoish/odem/pkg/dispatch"
)

func (b *bot) dispatchMessage(msg *etron.Message, fallback func() stateFn) stateFn {
	defer func() {
		if p := recover(); p != nil {
			resp, err := b.Close()
			grip.Error(err)
			msg := grip.KV("code", p).KV("close", resp)
			grip.Debug(grip.When(resp.Ok, msg))
			grip.Warning(grip.When(!resp.Ok, msg))

			switch p {
			case "exit", "abort", "quit":
				b.off.Store(true)
				return
			default:
				panic(p)
			}
		}
	}()

	grip.Info(grip.MPrintln("message", msg.Text))
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
