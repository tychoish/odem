package infra

import "github.com/tychoish/fun/fnx"

func ErrWorker(err error) fnx.Worker { return fnx.MakeWorker(func() error { return err }) }
