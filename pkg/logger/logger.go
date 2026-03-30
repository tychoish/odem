package logger

import (
	"context"
	"time"

	"github.com/tychoish/fun/strut"
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

const plain = "plain"

func Plain(c context.Context) grip.Logger { return grip.ContextLogger(c, plain) }

func Setup(ctx context.Context) context.Context {
	senderRoot := send.MakeWriterSender(send.MakeStdError())
	senderPlain := send.MakeWriter(senderRoot)
	senderDefault := send.MakeWriter(senderRoot)
	senderDefault.SetFormatter(Formatter())

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
