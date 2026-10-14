package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/PaddleHQ/claude-statusline/internal/rendertest"
)

// Image is one generated file: its name within the images directory and its content.
type Image struct {
	Name string
	SVG  []byte
}

// Generate renders every fixture file under fixtures into a dark and a light image, ordered by file name.
func Generate(fixtures string) ([]Image, error) {
	cases, err := rendertest.Load(fixtures)
	if err != nil {
		return nil, fmt.Errorf("load fixtures: %w", err)
	}
	var images []Image
	for _, c := range cases {
		f := Open(c)
		text, err := f.Render()
		if err != nil {
			return nil, fmt.Errorf("render %s: %w", f.Name(), err)
		}
		screen := Parse(text)
		for _, th := range []*Theme{Dark, Light} {
			images = append(images, Image{Name: f.Name() + "-" + th.Name + ".svg", SVG: screen.SVG(th)})
		}
	}
	slices.SortFunc(images, func(a, b Image) int { return compareNames(a.Name, b.Name) })
	return images, nil
}

func compareNames(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
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
