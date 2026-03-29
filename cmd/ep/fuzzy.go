package ep

import (
	"github.com/tychoish/cmdr"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/odem/pkg/dispatch"
	"github.com/tychoish/odem/pkg/infra"
)

func Fuzzy() *cmdr.Commander {
	return cmdr.MakeCommander().
		SetName("fuzzy").
		Aliases("fzf").
		SetUsage("fuzzy cli UI to minutes data").
		With(infra.DBOperationSpec(dispatch.MinutesAppOpRetry.FuzzyDispatcher().Op).Add).
		Subcommanders(irt.Collect(dispatch.AllFuzzyMinutesAppCmdrs())...)
}
