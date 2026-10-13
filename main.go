// Command claude-statusline prints the model name for a Claude Code status line.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/PaddleHQ/claude-statusline/internal/setup"
)

type payload struct {
	Model struct {
		DisplayName string `json:"display_name"`
	} `json:"model"`
}

// Set at build time via -ldflags -X; every target must exist or the stamp silently no-ops.
var (
	version = "dev"
	commit  = ""
	date    = ""
)

func main() {
	execute(os.Args[1:], os.Stdin, os.Stdout)
}

func execute(args []string, stdin io.Reader, stdout io.Writer) {
	if len(args) > 0 && args[0] == "--version" {
		_, _ = fmt.Fprintln(stdout, versionString())
		return
	}
	if len(args) > 0 && args[0] == "setup" {
		setup.Run(stdout, eventFlag(args[1:]), version)
		return
	}
	run(stdin, stdout)
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
func run(stdin io.Reader, stdout io.Writer) {
	var p payload
	if err := json.NewDecoder(stdin).Decode(&p); err != nil {
		return
	}
	_, _ = io.WriteString(stdout, sanitize(p.Model.DisplayName))
}

// sanitize drops control characters so a payload cannot inject terminal escapes.
func sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, s)
}
