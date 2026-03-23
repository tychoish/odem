package logger

import (
	"context"
	"os"

	"github.com/tychoish/grip"
	"github.com/tychoish/grip/send"
)

const plain = "plain"

func SetupDefault()                               { grip.Sender().SetFormatter(send.MakeDefaultFormatter()) }
func Plain(c context.Context) grip.Logger         { return grip.ContextLogger(c, plain) }
func mkplain() grip.Logger                        { return grip.NewLogger(send.MakeWriter(os.Stdout)) }
func WithPlain(c context.Context) context.Context { return grip.WithContextLogger(c, plain, mkplain()) }
