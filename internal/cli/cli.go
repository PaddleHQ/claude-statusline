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

// Execute runs the command the first argument selects, or the status line itself, and returns the exit status. It
// never blanks the status line over a failure, and only config-check exits non-zero, when the config cannot be used.
func (a *App) Execute(args []string, stdin io.Reader, stdout io.Writer, getenv func(string) string) int {
	command := ""
	if len(args) > 0 {
		command = args[0]
	}
	switch command {
	case "--version":
		_, _ = fmt.Fprintln(stdout, a.build.String())
	case "setup":
		setup.Run(stdout, eventFlag(args[1:]), a.build.Version)
	case "subagent":
		runSubagent(stdin, stdout, getenv)
	case "help":
		_, _ = io.WriteString(stdout, statusline.Help(getenv))
	case "config":
		_, _ = io.WriteString(stdout, statusline.DefaultConfig())
	case "config-check":
		return runConfigCheck(stdout, getenv)
	default:
		run(stdin, stdout, getenv)
	}
	return 0
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

func runConfigCheck(stdout io.Writer, getenv func(string) string) int {
	report, ok := statusline.ConfigCheck(getenv)
	_, _ = io.WriteString(stdout, report)
	if !ok {
		return 1
	}
	return 0
}
