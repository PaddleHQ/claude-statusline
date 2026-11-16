// Package cli picks the claude-statusline command from its arguments and runs it.
package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/ratebars"
	"github.com/PaddleHQ/claude-statusline/internal/setup"
	"github.com/PaddleHQ/claude-statusline/internal/statusline"
	"github.com/PaddleHQ/claude-statusline/internal/subagent"
)

// usageStatus is the exit status for a command given arguments it cannot use.
const usageStatus = 2

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
	build      Build
	executable func() (string, error)
	stderr     io.Writer
}

// New returns the command for a binary with the given build stamps.
func New(build Build) *App {
	return &App{build: build, executable: os.Executable, stderr: os.Stderr}
}

// Execute runs the command the first argument selects, or the status line itself, and returns the exit status. It
// never blanks the status line over a failure. Only config-check (the config cannot be used) and rate-bars (its flags
// are bad) exit non-zero.
func (a *App) Execute(args []string, stdin io.Reader, stdout io.Writer, getenv func(string) string) int {
	command := ""
	if len(args) > 0 {
		command = args[0]
	}
	if command == "--version" {
		_, _ = fmt.Fprintln(stdout, a.build.String())
		return 0
	}
	exe := a.resolvedExecutable()
	getenv = withStateDir(getenv, exe)
	switch command {
	case "setup":
		a.runSetup(args[1:], stdout, locations(exe, getenv))
	case "uninstall":
		setup.Uninstall(stdout, locations(exe, getenv))
	case "subagent":
		runSubagent(stdin, stdout, getenv)
	case "rate-bars":
		return a.runRateBars(args[1:], stdout, getenv)
	case "help", "config":
		_, _ = io.WriteString(stdout, infoText(command, getenv))
	case "config-check":
		return runConfigCheck(stdout, getenv)
	default:
		run(stdin, stdout, getenv)
	}
	return 0
}

// runSetup wires the status line into settings.json. Without an --event flag it was run by hand, which re-adds the
// lines even after an uninstall.
func (a *App) runSetup(args []string, stdout io.Writer, loc setup.Locations) {
	event := eventFlag(args)
	if !slices.ContainsFunc(args, func(a string) bool { return strings.HasPrefix(a, "--event=") }) {
		event = setup.EventManual
	}
	setup.Run(stdout, event, a.build.Version, loc)
}

func (a *App) resolvedExecutable() string {
	exe, err := a.executable()
	if err != nil {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		return resolved
	}
	return exe
}

// withStateDir makes the binary's own location, not the environment, decide where its private state lives: inside
// the plugin data directory it was installed into, and next to the user's config file otherwise.
func withStateDir(getenv func(string) string, exe string) func(string) string {
	state := ""
	if exe != "" {
		state = config.PluginDataDir(exe)
	}
	return func(name string) string {
		if name == config.StateDirEnv {
			return state
		}
		return getenv(name)
	}
}

func locations(exe string, getenv func(string) string) setup.Locations {
	return setup.Locations{ConfigDir: config.Dir(getenv), StateDir: config.StateDir(getenv), Bin: exe, Home: getenv("HOME")}
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

func infoText(command string, getenv func(string) string) string {
	if command == "help" {
		return statusline.Help(getenv)
	}
	return statusline.DefaultConfig()
}

func (a *App) runRateBars(args []string, stdout io.Writer, getenv func(string) string) int {
	out, err := ratebars.Build(args, getenv)
	if err != nil {
		_, _ = fmt.Fprintln(a.stderr, err)
		return usageStatus
	}
	_, _ = io.WriteString(stdout, out)
	return 0
}

func runConfigCheck(stdout io.Writer, getenv func(string) string) int {
	report, ok := statusline.ConfigCheck(getenv)
	_, _ = io.WriteString(stdout, report)
	if !ok {
		return 1
	}
	return 0
}
