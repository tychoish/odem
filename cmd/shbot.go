package main

import (
	_ "github.com/mattn/go-sqlite3"
	"github.com/tychoish/shbot/cmd/ep"
	"github.com/tychoish/shbot/pkg/infra"
)

func main() {
	infra.MainCLI("shbot",
		ep.Version(),
		ep.Hacking(),
		ep.Fuzzy(),
	)
}
