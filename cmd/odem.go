package main

import (
	"github.com/tychoish/odem/cmd/ep"
	"github.com/tychoish/odem/pkg/infra"
	"github.com/tychoish/odem/pkg/release"
)

func main() {
	infra.MainCLI(release.Name,
		ep.Setup(),
		ep.Fuzzy(),
		ep.Report(),
		ep.MCP(),
		ep.Telegram(),
		ep.Version(),
		ep.Docs(),
		ep.Hacking(),
		ep.Build(),
	)
}
