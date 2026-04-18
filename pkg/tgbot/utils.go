package tgbot

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	etron "github.com/NicoNex/echotron/v3"
	"github.com/tychoish/fun/strut"
)

func joinstr(args ...string) string { return strings.Join(args, "") }

// extractThreadID returns the forum thread ID from an update, or 0 if the
// update is not part of a thread.
func extractThreadID(u *etron.Update) int {
	switch {
	case u.Message != nil:
		return u.Message.ThreadID
	case u.CallbackQuery != nil && u.CallbackQuery.Message != nil:
		return u.CallbackQuery.Message.ThreadID
	default:
		return 0
	}
}

// isEscapeInput reports whether text is a user bail-out command that should
// exit any active selection loop and return to the top level.
func isEscapeInput(text string) bool {
	switch strings.ToLower(strings.TrimPrefix(strings.TrimSpace(text), "/")) {
	case "reset", "cancel", "back", "abort", "quit", "exit", "stop":
		return true
	}
	return false
}

func isOrContainsCmd(msg *etron.Message, strs ...string) bool {
	for _, str := range strs {
		switch {
		case msg.Text == str:
			return true
		case strings.HasPrefix(msg.Text, fmt.Sprint("/", str)):
			return true
		case strings.HasPrefix(msg.Text, str):
			return true
		case strings.Contains(msg.Text, str):
			return true
		}
	}
	return false
}

// extractNumber scans the words in text and returns the first one that
// parses as a positive integer, along with true. Returns 0, false if none
// is found.
func extractNumber(text string) (int, bool) {
	for _, word := range strings.Fields(text) {
		if n, err := strconv.Atoi(word); err == nil && n > 0 {
			return n, true
		}
	}
	return 0, false
}

func toJson(val any) *strut.Mutable {
	mut := strut.MakeMutable(1024)
	err := json.NewEncoder(mut).Encode(val)
	if err != nil {
		mut.Reset()
		mut.PushString(err.Error())
	}
	return mut
}
