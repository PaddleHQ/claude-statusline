// Package cli picks the claude-statusline command from its arguments and runs it.
package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/PaddleHQ/claude-statusline/internal/setup"
	"github.com/PaddleHQ/claude-statusline/internal/statusline"
	"github.com/PaddleHQ/claude-statusline/internal/subagent"
)

// Build identifies the binary: the values the release stamps in at build time.
type Build struct {
	Version string
	Commit  string
	Date    string
}

// String is the text of the --version command.
func (b *Build) String() string {
	s := "claude-statusline " + b.Version
	if b.Commit != "" {
		s += " (" + b.Commit + ", " + b.Date + ")"
	}
	return s
}

// App is the claude-statusline command.
type App struct {
	build Build
}

// New returns the command for a binary with the given build stamps.
func New(build Build) *App {
	return &App{build: build}
}

// Execute runs the command the first argument selects, or the status line itself. It never reports failures: a
// blank status line is better than a broken one.
func (a *App) Execute(args []string, stdin io.Reader, stdout io.Writer, getenv func(string) string) {
	if len(args) > 0 && args[0] == "--version" {
		_, _ = fmt.Fprintln(stdout, a.build.String())
		return
	}
	if len(args) > 0 && args[0] == "setup" {
		setup.Run(stdout, eventFlag(args[1:]), a.build.Version)
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

func run(stdin io.Reader, stdout io.Writer, getenv func(string) string) {
	input, err := io.ReadAll(stdin)
	if err != nil {
		return
	}
	_, _ = io.WriteString(stdout, statusline.Build(input, getenv))
}

func runSubagent(stdin io.Reader, stdout io.Writer, getenv func(string) string) {
	input, err := io.ReadAll(stdin)
	if err != nil {
		return
	}
	_, _ = io.WriteString(stdout, subagent.Build(input, getenv))
}
