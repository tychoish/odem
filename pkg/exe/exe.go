package exe

import (
	"context"

	"github.com/tychoish/fun/exc"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/send"
	"github.com/tychoish/odem/pkg/logger"
)

func Command(ctx context.Context) *exc.Command {
	o := send.MakeWriterSender(logger.Plain(ctx).Sender())
	o.Store(level.Info)
	e := send.MakeWriterSender(logger.Plain(ctx).Sender())
	e.Store(level.Error)

	return new(exc.Command).WithStdError(e).WithStdOutput(o)
}
