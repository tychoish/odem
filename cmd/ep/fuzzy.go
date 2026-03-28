package ep

import (
	"github.com/tychoish/cmdr"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/odem/pkg/clidispatch"
	"github.com/tychoish/odem/pkg/infra"
)

func Fuzzy() *cmdr.Commander {
	return cmdr.MakeCommander().
		SetName("fuzzy").
		Aliases("fzf").
		SetUsage("fuzzy cli UI to minutes data").
		With(infra.DBOperationSpec(clidispatch.MinutesAppOpRetry.FuzzyDispatcher().Op).Add).
		Subcommanders(irt.Collect(clidispatch.AllFuzzyMinutesAppCmdrs())...)
}
