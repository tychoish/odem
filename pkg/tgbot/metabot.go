package tgbot

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	etron "github.com/NicoNex/echotron/v3"
	"github.com/tychoish/fun/adt"
	"github.com/tychoish/grip"
	"github.com/tychoish/odem"
	"github.com/tychoish/odem/pkg/db"
)

// metabot sits one level above the per-thread bot instances. It implements
// etron.Bot and is the only object returned to the echotron Dispatcher.
// All routing lives here; individual bot instances are pure state machines.
type metabot struct {
	chatID  int64
	botID   int64  // Telegram user ID of the bot (0 if unknown)
	botName string // @username without leading @
	bots    adt.SyncMap[int, *bot]
	api     etron.API
	db      *db.Connection
	conf    *odem.Configuration
	ctx     context.Context
	off     *atomic.Bool
}

// Update routes each incoming update to the bot instance responsible for the
// update's thread (threadID 0 is the default for DMs and unthreaded groups).
// For threaded groups, irrelevant plain messages are dropped before a thread
// bot is created so that ambient conversation never allocates a bot.
func (m *metabot) Update(update *etron.Update) {
	threadID := extractThreadID(update)
	if threadID != 0 && update.Message != nil && !m.isRelevantThreadMessage(update.Message) {
		grip.Notice(grip.KV("op", "can't find message for thread").KV("chatID", m.chatID).KV("threadID", threadID))
		return
	}
	if b, ok := m.bots.Load(threadID); ok {
		b.advance(update)
		return
	}
	b := m.newBot(threadID)
	m.bots.Store(threadID, b)
	grip.Info(grip.KV("op", "new thread bot").KV("chatID", m.chatID).KV("threadID", threadID))
	b.advance(update)
}

// newBot constructs a fresh bot instance for the given thread and initialises
// its query state. The bot has no routing map (state.threads == nil) so it
// never recurses back into metabot.
func (m *metabot) newBot(threadID int) *bot {
	b := &bot{
		chatID:   m.chatID,
		threadID: threadID,
		botID:    m.botID,
		botName:  m.botName,
		API:      m.api,
		ctx:      m.ctx,
		db:       m.db,
		conf:     m.conf,
		off:      m.off,
	}
	_ = b.resetState()
	b.setLastUpdated(time.Now())
	return b
}

// isRelevantThreadMessage reports whether a message in a threaded group should
// be processed. Slash commands, replies to the bot's own messages, and direct
// @mentions are admitted; everything else is silently ignored so the bot does
// not respond to ambient conversation.
func (m *metabot) isRelevantThreadMessage(msg *etron.Message) bool {
	if strings.HasPrefix(msg.Text, "/") {
		return true
	}
	if m.botID != 0 && msg.ReplyToMessage != nil && msg.ReplyToMessage.From != nil && msg.ReplyToMessage.From.ID == m.botID {
		return true
	}
	if m.botName != "" && strings.Contains(msg.Text, fmt.Sprintf("@%s", m.botName)) {
		return true
	}
	return false
}
