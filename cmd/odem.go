package main

import (
	_ "modernc.org/sqlite"
	"github.com/tychoish/odem/cmd/ep"
	"github.com/tychoish/odem/pkg/infra"
)

func main() {
	infra.MainCLI("odem",
		ep.Setup(),
		ep.Fuzzy(),
		ep.Report(),
		ep.Version(),
		ep.Hacking(),
	)
}
