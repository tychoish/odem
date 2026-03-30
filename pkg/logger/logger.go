package logger

import (
	"context"
	"time"

	"github.com/tychoish/fun/strut"
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

const plain = "plain"

func Plain(c context.Context) grip.Logger { return grip.ContextLogger(c, plain) }

func Setup(ctx context.Context) context.Context {
	// TODO(tycho): eventually would be nice to use the WriterSender, which has a buffer of80
	// characters, so sometimes messages get swallowed in the current implementation.

	// senderRoot := send.MakeWriterSender(send.MakeStdError())
	// senderRoot.SetPriority(level.Info)
	// senderPlain := send.MakeWriter(senderRoot)
	// senderDefault := send.MakeWriter(senderRoot)

	senderPlain := send.MakeStdError()
	senderPlain.SetPriority(level.Trace)
	senderDefault := send.MakeStdError()
	senderDefault.SetFormatter(Formatter())
	senderDefault.SetPriority(level.Info)

	grip.SetSender(senderDefault)

	ctx = grip.WithContextLogger(ctx, plain, grip.NewLogger(senderPlain))

	return ctx
}

func Formatter() send.MessageFormatter {
	return func(m message.Composer) (string, error) {
		mut := strut.MakeMutable(1024)
		defer mut.Release()

		mut.Concat("[p=", m.Priority().String(), " t=", time.Now().Format(time.DateTime), "]: ", m.String())

		return mut.String(), nil
	}
}
