package main

import (
	"github.com/tychoish/odem/cmd/ep"
	"github.com/tychoish/odem/pkg/odemcli"
	"github.com/tychoish/odem/pkg/release"
)

func main() {
	odemcli.Main(release.Name,
		ep.Setup(),
		ep.Navigate(),
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
