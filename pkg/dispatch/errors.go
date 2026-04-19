package dispatch

import (
	"github.com/tychoish/fun/ers"
)

// ErrUnavailableOperation is returned when an operation is invoked in a context
// that does not support it (e.g., a bot-only query run from the CLI).
const ErrUnavailableOperation ers.Error = "unavailable operation"

func unavailableOp(name string) error { return ers.Wrap(ErrUnavailableOperation, name) }
