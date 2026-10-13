// Command claude-statusline prints the model name for a Claude Code status line.
package main

import (
	"encoding/json"
	"io"
	"os"
	"strings"
)

type payload struct {
	Model struct {
		DisplayName string `json:"display_name"`
	} `json:"model"`
}

func main() {
	run(os.Stdin, os.Stdout)
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
