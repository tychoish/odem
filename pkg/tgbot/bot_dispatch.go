package tgbot

import (
	"fmt"
	"strconv"
	"strings"

	etron "github.com/NicoNex/echotron/v3"
	"github.com/tychoish/fun/mdwn"
	"github.com/tychoish/grip"
)

func (b *bot) dispatchMessage(msg *etron.Message) stateFn {
	defer func() {
		if p := recover(); p != nil {
			resp, err := b.Close()
			grip.Log(b.level(err), b.gmr("recover close", resp.Base()).WithError(err))

			switch p {
			case "exit", "abort", "quit":
				b.off.Store(true)
				return
			default:
				panic(p)
			}
		}
	}()

	grip.Info(b.grip("dispatch message").KV("msg.text", msg.Text))
	switch {
	case msg.Text == "":
		grip.Notice(b.grip("got message with empty text"))
		return b.handleMessage
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
		if !b.queryState.inProgress {
			b.sendPlain("Nothing in progress at the moment...")
			return b.handleMessage
		}

		mb := mdwn.MakeBuilder(1024).
			KV("operation", b.queryState.entry.Command).
			KV("discription", b.queryState.entry.Description).
			KV("selection", b.queryState.params.Name).
			KV("limit", strconv.Itoa(b.queryState.params.Limit)).
			KV("years", fmt.Sprintf("(if relevant) %v", b.queryState.params.Years))
		defer mb.Release()

		b.sendMarkdown(mb.String())

		return b.discoverNext()
	case isOrContainsCmd(msg, "limit reset"):
		if !b.queryState.inProgress {
			b.sendPlain("no operation in progress, can't set a limit right now, but select an option from the menu...")
			return b.handleMessage
		}
		b.queryState.params.Limit = 20
		b.sendPlain(fmt.Sprintln("(re)setting the limit to 20..."))
		return b.discoverNext()
	case isOrContainsCmd(msg, "limit"):
		if !b.queryState.inProgress {
			b.sendPlain("no operation in progress, can't set a limit right now, but select an option from the menu...")
			return b.handleMessage
		} else if n, ok := extractNumber(msg.Text); ok && n > 40 {
			b.sendPlain("going to cap things at 40 right now...")
			b.queryState.params.Limit = 40
		} else if !ok {
			b.sendPlain(fmt.Sprintln("couldn't find a number in the message, going to leave the limit at, and continue here...", b.queryState.params.Limit))
		} else {
			b.queryState.params.Limit = n
			b.sendPlain(fmt.Sprintln("set the limt to", b.queryState.params.Limit))
		}
		return b.discoverNext()
	case isOrContainsCmd(msg, "reset year"):
		if !b.queryState.inProgress {
			b.sendPlain("no operation in progress, so can't reset a limit right now, but select an option from the menu...")
		}
		b.queryState.params.Years = []int{0}
		return b.discoverNext()
	case isOrContainsCmd(msg, "help"):
		b.sendPlain("Hi! I'm the minutes app, in chat form... Select an option from the menu!")
		return b.keyboardMinutesAppQueries()
	case isOrContainsCmd(msg, "keyboard", "menu"):
		return b.keyboardMinutesAppQueries()
	case b.queryState.inProgress:
		b.sendPlain(fmt.Sprintf("Hmm... I'm working on %s, can you try that again?", b.queryState.op.String()))
		return b.discoverNext()
	case b.setupQuery(strings.ToLower(msg.Text)):
		return b.discoverNext()
	default:
		return b.keyboardMinutesAppQueries()
	}
}
