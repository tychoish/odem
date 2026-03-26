package ep

import (
	"github.com/tychoish/cmdr"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/odem/pkg/fzfui"
	"github.com/tychoish/odem/pkg/infra"
)

func Fuzzy() *cmdr.Commander {
	return cmdr.MakeCommander().
		SetName("fuzzy").
		Aliases("fzf").
		SetUsage("fuzzy cli UI to minutes data").
		With(infra.DBOperationSpec(fzfui.MinutesAppOpRetry.Dispatch().Op).Add).
		Subcommanders(irt.Collect(fzfui.AllMinutesAppCommanders())...)
}
