package logger

import (
	"context"
	"os"
	"time"

	"github.com/tychoish/fun/strut"
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

const plain = "plain"

func SetupDefault()                               { grip.Sender().SetFormatter(Formatter()) }
func Plain(c context.Context) grip.Logger         { return grip.ContextLogger(c, plain) }
func mkplain() grip.Logger                        { return grip.NewLogger(send.MakeWriter(os.Stdout)) }
func WithPlain(c context.Context) context.Context { return grip.WithContextLogger(c, plain, mkplain()) }

func Formatter() send.MessageFormatter {
	return func(m message.Composer) (string, error) {
		mut := strut.MakeMutable(1024)
		defer mut.Release()

		mut.Concat("[p=", m.Priority().String(), " t=", time.Now().Format(time.DateTime), "]: ", m.String())
		return mut.String(), nil
	}
}
