// Command screenshots regenerates the images under docs/images from the fixtures in testdata/screenshots.
// Run it from the repository root: go run ./tools/screenshots.
package main

import (
	"fmt"
	"os"

	"github.com/PaddleHQ/claude-statusline/internal/screenshot"
)

const (
	fixtures  = "testdata/screenshots"
	outputDir = "docs/images"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "screenshots:", err)
		os.Exit(1)
	}
}

func run() error {
	if _, err := os.Stat(fixtures); err != nil {
		return fmt.Errorf("%s not found; run this from the repository root: %w", fixtures, err)
	}
	images, err := screenshot.Generate(fixtures)
	if err != nil {
		return fmt.Errorf("generate: %w", err)
	}
	if err := screenshot.Write(outputDir, images); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	_, _ = fmt.Fprintf(os.Stdout, "wrote %d images to %s\n", len(images), outputDir)
	return nil
}
