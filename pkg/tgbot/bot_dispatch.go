package tgbot

import (
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/tychoish/fun/mdwn"
	"github.com/tychoish/fun/strut"
	"github.com/tychoish/grip"
	"github.com/tychoish/odem/pkg/dispatch"
	"github.com/tychoish/odem/pkg/release"
)

func (b *bot) dispatchMessage(msg string) stateFn {
	defer func() {
		if p := recover(); p != nil {
			grip.Alert(b.grip("recover").KV("panic", p).KV("stack", string(debug.Stack())))
		}
	}()
	msg = strings.ToLower(msg)

	grip.Info(b.grip("dispatch message").KV("msg.text", msg))
	switch {
	case msg == "":
		grip.Notice(b.grip("got message with empty text"))
		return b.handleMessage
	case isOrContainsCmd(msg, "reset"):
		b.sendPlain("resetting query...")
		return b.resetState()
	case isOrContainsCmd(msg, "retry"):
		b.sendPlain("retrying...")
		return b.resetState()
	case isOrContainsCmd(msg, "restart"):
		b.sendPlain("restarting...")
		return b.resetState()
	case isOrContainsCmd(msg, "sysinfo", "odmeinfo", "appinfo"):
		mb := mdwn.MakeBuilder(1024).
			KV("in_progress", fmt.Sprint(b.queryState.inProgress)).
			KV("version", release.Version.Resolve().String()).
			KV("sent", strconv.Itoa(int(b.metrics.sent.Load()))).
			KV("recv", strconv.Itoa(int(b.metrics.recv.Load()))).
			KV("uptime", time.Since(b.metrics.createdAt).String()).
			KV("interval", time.Since(b.lastUpdated).String())
		mb.WhenWriteMutable(b.queryState.inProgress, b.stateMessage().Deref())
		b.sendMarkdown(mb.Resolve())
		return b.handleMessage
	case isOrContainsCmd(msg, "state", "status"):
		if !b.queryState.inProgress {
			b.sendPlain("Nothing in progress at the moment...")
			return b.handleMessage
		}

		b.sendMarkdown(b.stateMessage().Resolve())

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
		} else if n, ok := extractNumber(msg); ok && n > 40 {
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
	case isOrContainsCmd(msg, "commands", "cmds", "ops", "cmd", "opts", "opt"):
		b.sendMarkdown(renderCommands().Resolve())
		return b.handleMessage
	case isOrContainsCmd(msg, "help"):
		b.sendPlain("Hi! I'm the minutes app, in chat form... Just send me a message with any of these operations to start! (Or say `menu` to select from a button.)")
		b.sendMarkdown(renderCommands().Resolve())
		return b.handleMessage
	case isOrContainsCmd(msg, "keyboard", "menu"):
		return b.keyboardMinutesAppQueries()
	case b.queryState.inProgress:
		b.sendPlain(strut.Mprintf("Hmm... I'm working on %s, can you try that again?", b.queryState.op.String()).Resolve())
		return b.discoverNext()
	case b.setupQuery(msg):
		return b.discoverNext()
	default:
		return b.keyboardMinutesAppQueries()
	}
}

func renderCommands() *mdwn.Builder {
	mb := mdwn.MakeBuilder(4096)
	for op := range dispatch.AllMinutesAppOps() {
		reg := op.Registry()
		if reg.Messenger == nil {
			continue
		}
		mb.PushString("🎵 ").Concat("*", reg.Command, "*", " ➣ ")
		mb.PushString(reg.Description).Line()
	}
	return mb
}

func (b *bot) stateMessage() *mdwn.Builder {
	mb := mdwn.MakeBuilder(1024)

	if b.queryState.entry != nil {
		mb.KV("operation", b.queryState.entry.Command).
			KV("discription", b.queryState.entry.Description)
	}
	return mb.KV("selection", b.queryState.params.Name).
		KV("song", b.queryState.params.Song).
		KV("limit", strconv.Itoa(b.queryState.params.Limit)).
		KV("years", fmt.Sprintf("(if relevant) %v", b.queryState.params.Years))
}
