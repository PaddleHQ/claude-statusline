// Command claude-statusline prints a status line for Claude Code.
package main

import (
	"os"

	"github.com/PaddleHQ/claude-statusline/internal/cli"
)

func main() {
	cli.Run(os.Stdin, os.Stdout)
}
