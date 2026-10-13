// Command claude-statusline prints a configurable status line for Claude Code.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/PaddleHQ/claude-statusline/internal/setup"
	"github.com/PaddleHQ/claude-statusline/internal/statusline"
	"github.com/PaddleHQ/claude-statusline/internal/subagent"
)

// Set at build time via -ldflags -X; every target must exist or the stamp silently no-ops.
var (
	version = "dev"
	commit  = ""
	date    = ""
)

func main() {
	execute(os.Args[1:], os.Stdin, os.Stdout, os.Getenv)
}

func execute(args []string, stdin io.Reader, stdout io.Writer, getenv func(string) string) {
	if len(args) > 0 && args[0] == "--version" {
		_, _ = fmt.Fprintln(stdout, versionString())
		return
	}
	if len(args) > 0 && args[0] == "setup" {
		setup.Run(stdout, eventFlag(args[1:]), version)
		return
	}
	if len(args) > 0 && args[0] == "subagent" {
		runSubagent(stdin, stdout, getenv)
		return
	}
	run(stdin, stdout, getenv)
}

func eventFlag(args []string) string {
	for _, a := range args {
		if v, ok := strings.CutPrefix(a, "--event="); ok {
			switch v {
			case setup.EventInstalled, setup.EventUpdated:
				return v
			}
		}
	}
	return setup.EventCurrent
}

func versionString() string {
	s := "claude-statusline " + version
	if commit != "" {
		s += " (" + commit + ", " + date + ")"
	}
	return s
}

// run never reports failures: a blank status line is better than a broken one.
func run(stdin io.Reader, stdout io.Writer, getenv func(string) string) {
	input, err := io.ReadAll(stdin)
	if err != nil {
		return
	}
	_, _ = io.WriteString(stdout, statusline.Build(input, getenv))
}

// runSubagent prints the agent panel rows; like run, it never reports failures.
func runSubagent(stdin io.Reader, stdout io.Writer, getenv func(string) string) {
	input, err := io.ReadAll(stdin)
	if err != nil {
		return
	}
	_, _ = io.WriteString(stdout, subagent.Build(input, getenv))
}
