package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/PaddleHQ/claude-statusline/internal/featuretest"
	"github.com/PaddleHQ/claude-statusline/internal/statusline"
	"github.com/PaddleHQ/claude-statusline/internal/subagent"
)

const (
	screenshotTag = "@screenshot"
	subagentName  = "subagent"
)

// ErrDuplicateScreenshot is returned when two scenarios save a screenshot under the same name.
var ErrDuplicateScreenshot = errors.New("duplicate screenshot name")

// Image is one generated file: its name within the images directory and its content.
type Image struct {
	Name string
	SVG  []byte
}

// Generate runs the @screenshot scenarios under features and draws what each saves as a dark and a light
// image, ordered by file name.
func Generate(features string) ([]Image, error) {
	screens := map[string]string{}
	var duplicate error
	suite := featuretest.Suite{
		Execute: execute,
		Paths:   []string{features},
		Tags:    screenshotTag,
		Screenshot: func(name, screen string) {
			if _, ok := screens[name]; ok {
				duplicate = fmt.Errorf("%w: %s", ErrDuplicateScreenshot, name)
			}
			screens[name] = screen
		},
	}
	if err := suite.Run(nil); err != nil {
		return nil, fmt.Errorf("run %s scenarios: %w", screenshotTag, err)
	}
	if duplicate != nil {
		return nil, duplicate
	}

	var images []Image
	for name, text := range screens {
		screen := Parse(text)
		for _, th := range []*Theme{Dark, Light} {
			images = append(images, Image{Name: name + "-" + th.Name + ".svg", SVG: screen.SVG(th)})
		}
	}
	slices.SortFunc(images, func(a, b Image) int { return strings.Compare(a.Name, b.Name) })
	return images, nil
}

// execute is the part of the claude-statusline command the screenshot scenarios use: the status line, and with
// the subagent argument the agent panel rows.
func execute(args []string, stdin io.Reader, stdout io.Writer, getenv func(string) string) int {
	input, err := io.ReadAll(stdin)
	if err != nil {
		return 0
	}
	if len(args) > 0 && args[0] == subagentName {
		_, _ = io.WriteString(stdout, subagent.Build(input, getenv))
		return 0
	}
	_, _ = io.WriteString(stdout, statusline.Build(input, getenv))
	return 0
}

// Write saves the images into dir, creating it if needed.
func Write(dir string, images []Image) error {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	for _, img := range images {
		if err := os.WriteFile(filepath.Join(dir, img.Name), img.SVG, 0o600); err != nil {
			return fmt.Errorf("write %s: %w", img.Name, err)
		}
	}
	return nil
}
