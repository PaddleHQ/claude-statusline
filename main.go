// Command claude-statusline prints a configurable status line for Claude Code.
package main

import (
	"os"

	"github.com/PaddleHQ/claude-statusline/internal/cli"
)

// Set at build time via -ldflags -X; every target must exist or the stamp silently no-ops.
var (
	version = "dev"
	commit  = ""
	date    = ""
)

func main() {
	app := cli.New(cli.Build{Version: version, Commit: commit, Date: date})
	if code := app.Execute(os.Args[1:], os.Stdin, os.Stdout, os.Getenv); code != 0 {
		os.Exit(code)
	}
}
