// Package cli reads the session from standard input and prints the status line.
package cli

import (
	"io"

	"github.com/PaddleHQ/claude-statusline/internal/statusline"
)

// Run prints the status line for the session JSON on stdin. It never reports failures: a blank status line is
// better than a broken one.
func Run(stdin io.Reader, stdout io.Writer) {
	input, err := io.ReadAll(stdin)
	if err != nil {
		return
	}
	_, _ = io.WriteString(stdout, statusline.Build(input))
}
