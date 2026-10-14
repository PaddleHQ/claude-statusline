package cli

import (
	"io"
)

func Run(stdin io.Reader, stdout io.Writer, getenv func(string) string) { run(stdin, stdout, getenv) }

func EventFlag(args []string) string { return eventFlag(args) }
