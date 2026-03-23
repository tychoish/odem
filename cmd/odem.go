package main

import (
	_ "github.com/mattn/go-sqlite3"
	"github.com/tychoish/odem/cmd/ep"
	"github.com/tychoish/odem/pkg/infra"
)

func main() {
	infra.MainCLI("odem",
		ep.Version(),
		ep.Hacking(),
		ep.Fuzzy(),
	)
}
