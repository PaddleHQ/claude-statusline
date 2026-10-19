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
	build      Build
	executable func() (string, error)
}

// New returns the command for a binary with the given build stamps.
func New(build Build) *App {
	return &App{build: build, executable: os.Executable}
}

// Execute runs the command the first argument selects, or the status line itself. It never reports failures: a
// blank status line is better than a broken one.
func (a *App) Execute(args []string, stdin io.Reader, stdout io.Writer, getenv func(string) string) {
	if len(args) > 0 && args[0] == "--version" {
		_, _ = fmt.Fprintln(stdout, a.build.String())
		return
	}
	exe := a.resolvedExecutable()
	getenv = withStateDir(getenv, exe)
	if len(args) > 0 && a.manage(args, stdout, locations(exe, getenv)) {
		return
	}
	if len(args) > 0 && args[0] == "subagent" {
		runSubagent(stdin, stdout, getenv)
		return
	}
	if len(args) > 0 && args[0] == "help" {
		_, _ = io.WriteString(stdout, statusline.Help(getenv))
		return
	}
	run(stdin, stdout, getenv)
}

// manage runs the commands that edit settings.json, and reports whether the first argument named one.
func (a *App) manage(args []string, stdout io.Writer, loc setup.Locations) bool {
	switch args[0] {
	case "setup":
		event := eventFlag(args[1:])
		if !slices.ContainsFunc(args[1:], func(a string) bool { return strings.HasPrefix(a, "--event=") }) {
			event = setup.EventManual
		}
		setup.Run(stdout, event, a.build.Version, loc)
	case "uninstall":
		setup.Uninstall(stdout, loc)
	default:
		return false
	}
	return true
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
