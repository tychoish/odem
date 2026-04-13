package tgbot

import (
	"fmt"
	"strconv"

	etron "github.com/NicoNex/echotron/v3"
	"github.com/tychoish/fun/mdwn"
	"github.com/tychoish/grip"
	"github.com/tychoish/odem/pkg/dispatch"
)

func (b *bot) dispatchMessage(msg *etron.Message) stateFn {
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
	case isOrContainsCmd(msg, "reset"):
		b.sendPlain("resetting query...")
		return b.resetState()
	case isOrContainsCmd(msg, "retry"):
		b.sendPlain("retrying...")
		return b.resetState()
	case isOrContainsCmd(msg, "restart"):
		b.sendPlain("restarting...")
		return b.resetState()
	case isOrContainsCmd(msg, "state", "status"):
		if !b.state.inProgress {
			b.sendPlain("Nothing in progress at the moment...")
			return b.handleMessage
		}

		mb := mdwn.MakeBuilder(1024).
			KV("operation", b.state.entry.Command).
			KV("discription", b.state.entry.Description).
			KV("selection", b.state.params.Name).
			KV("limit", strconv.Itoa(b.state.params.Limit)).
			KV("years", fmt.Sprintf("(if relevant) %s", b.state.params.Years))
		defer mb.Release()

		b.sendMarkdown(mb.String())

		return b.discoverNext()
	case isOrContainsCmd(msg, "limit reset"):
		if !b.state.inProgress {
			b.sendPlain("no operation in progress, can't set a limit right now, but select an option from the menu...")
			return b.handleMessage
		}
		b.state.params.Limit = 20
		b.sendPlain(fmt.Sprintln("(re)setting the limit to 20..."))
		return b.discoverNext()
	case isOrContainsCmd(msg, "limit"):
		if !b.state.inProgress {
			b.sendPlain("no operation in progress, can't set a limit right now, but select an option from the menu...")
			return b.handleMessage
		} else if n, ok := extractNumber(msg.Text); ok && n > 40 {
			b.sendPlain("going to cap things at 40 right now...")
			b.state.params.Limit = 40
		} else if !ok {
			b.sendPlain(fmt.Sprintln("couldn't find a number in the message, going to leave the limit at, and continue here...", b.state.params.Limit))
		} else {
			b.state.params.Limit = n
			b.sendPlain(fmt.Sprintln("set the limt to", b.state.params.Limit))
		}
		return b.discoverNext()
	case isOrContainsCmd(msg, "reset year"):
		if !b.state.inProgress {
			b.sendPlain("no operation in progress, so can't reset a limit right now, but select an option from the menu...")
		}
		b.state.params.Years = []int{0}
		return b.discoverNext()
	case isOrContainsCmd(msg, "help"):
		b.sendPlain("Hi! I'm the minutes app, in chat form... Select an option from the menu!")
		return b.selectOperationKeyboard()
	case isOrContainsCmd(msg, "keyboard", "menu"):
		return b.selectOperationKeyboard()
	case !b.state.inProgress:
		if tryTxt := dispatch.NewMinutesAppOperation(msg.Text); tryTxt.Ok() {
			return b.handleKeyboardResponse(msg.Text)
		}
		b.sendPlain("Let's try again! Select an option from the menu...")
		return b.handleMessage
	default:
		b.sendPlain("Hrm... I didn't quite get that?")
		return b.discoverNext()
	}
}
