// Command screenshots turns the status line's terminal output into SVG images for the documentation.
// It regenerates the images under docs/images from the @screenshot scenarios in features/screenshots.feature.
// Run it from the repository root: go run ./tools/screenshots.
package main

import (
	"fmt"
	"os"
)

const (
	fixtures  = "features"
	outputDir = "docs/images"
)

func main() {
	if err := generateAll(); err != nil {
		fmt.Fprintln(os.Stderr, "screenshots:", err)
		os.Exit(1)
	}
}

func generateAll() error {
	if _, err := os.Stat(fixtures); err != nil {
		return fmt.Errorf("%s not found; run this from the repository root: %w", fixtures, err)
	}
	images, err := Generate(fixtures)
	if err != nil {
		return fmt.Errorf("generate: %w", err)
	}
	if err := Write(outputDir, images); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	_, _ = fmt.Fprintf(os.Stdout, "wrote %d images to %s\n", len(images), outputDir)
	return nil
}
