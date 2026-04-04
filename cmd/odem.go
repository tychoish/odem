package main

import (
	"github.com/tychoish/odem/cmd/ep"
	"github.com/tychoish/odem/pkg/infra"
	"github.com/tychoish/odem/pkg/release"
	_ "modernc.org/sqlite"
)

/*
TODO implement larger plans and
- [ ] strict mode where missing or invalid input for fzfui and reports result in an error. should be envvar setable
- [x] attempt to use the fzf search api wiht input from the user to avoid needing to ask someone if there is only one match. if there are multiple matches and not in strict mode, the user can start narrowed. this could be applied to input for some other queries, so providing an isolated implementation.
- [x] should move some of the core dispatching code from fzfui to a new package (cmdln) [this would include the enum, and core methods, but nothing else Actions would still be in fzfui, and reports would be in a reportui package]
- [x] complete a report UI for  (as a prototype in this directory).
- [x] add a query/fzfui/report for leader ordered by number of leads, potentially allow filtering by year (in the way of the song popularity)
- [x] add a query (etc.) for "number of leaders who led N% of songs," also filtered by year.
*/

func main() {
	infra.MainCLI(release.Name,
		ep.Setup(),
		ep.Fuzzy(),
		ep.Report(),
		ep.MCP(),
		ep.Version(),
		ep.Docs(),
		ep.Hacking(),
		ep.Release(),
	)
}
