package theme

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"go.yaml.in/yaml/v3"
)

//go:embed themes/*.yaml
var builtins embed.FS

const (
	fallbackName = "classic"
	legacyName   = "default"
	maxFileBytes = 1 << 20
	styleSolid   = "solid"
	styleGrad    = "gradient"
	partialCount = 7
	resetSeq     = "\033[0m"
)

type spec struct {
	Extends string      `yaml:"extends"`
	Colors  *colorsSpec `yaml:"colors"`
	Bar     *barSpec    `yaml:"bar"`
}

type colorsSpec struct {
	Text   *string `yaml:"text"`
	Muted  *string `yaml:"muted"`
	Good   *string `yaml:"good"`
	Warn   *string `yaml:"warn"`
	Bad    *string `yaml:"bad"`
	Marker *string `yaml:"marker"`
	Path   *string `yaml:"path"`
}

type barSpec struct {
	Style       *string  `yaml:"style"`
	Full        *string  `yaml:"full"`
	Partials    []string `yaml:"partials"`
	Empty       *string  `yaml:"empty"`
	Marker      *string  `yaml:"marker"`
	Ramp        []string `yaml:"ramp"`
	EmptyColor  *string  `yaml:"empty_color"`
	MarkerColor *string  `yaml:"marker_color"`
}

// Load resolves the theme setting to a theme. A value containing "/" or starting
// with "~" is a file path (relative paths are taken from base, "~" is home);
// anything else is a built-in name. Whatever goes wrong, the result is classic.
func Load(value, home, base string) Theme {
	if t, err := Open(value, home, base); err == nil {
		return t
	}
	t, _ := Open(fallbackName, home, base)
	return t
}

// Open is Load without the fallback: it reports why a theme could not be used.
func Open(value, home, base string) (Theme, error) {
	s, err := resolve(value, home, base, map[string]bool{})
	if err != nil {
		return Theme{}, err
	}
	return s.compile()
}

func isPath(value string) bool {
	return strings.Contains(value, "/") || strings.HasPrefix(value, "~")
}

func resolve(value, home, base string, seen map[string]bool) (spec, error) {
	data, id, dir, err := read(value, home, base)
	if err != nil {
		return spec{}, err
	}
	if seen[id] {
		return spec{}, fmt.Errorf("theme %q extends itself", id)
	}
	seen[id] = true

	var s spec
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&s); err != nil && !errors.Is(err, io.EOF) {
		return spec{}, fmt.Errorf("theme %q: %w", id, err)
	}
	if s.Extends == "" {
		return s, nil
	}
	parent, err := resolve(s.Extends, home, dir, seen)
	if err != nil {
		return spec{}, err
	}
	return parent.overlay(s), nil
}

func read(value, home, base string) (data []byte, id, dir string, err error) {
	if !isPath(value) {
		name := value
		if name == legacyName {
			name = fallbackName
		}
		data, err = builtins.ReadFile("themes/" + name + ".yaml")
		return data, "builtin:" + name, "", err
	}
	path := value
	switch {
	case value == "~" || strings.HasPrefix(value, "~/"):
		if home == "" {
			return nil, "", "", errors.New("no home directory")
		}
		path = filepath.Join(home, value[1:])
	case !filepath.IsAbs(value):
		if base == "" {
			return nil, "", "", errors.New("relative theme path without a base directory")
		}
		path = filepath.Join(base, value)
	}
	path = filepath.Clean(path)
	f, err := os.Open(path)
	if err != nil {
		return nil, "", "", err
	}
	defer func() { _ = f.Close() }()
	data, err = io.ReadAll(io.LimitReader(f, maxFileBytes+1))
	if err != nil {
		return nil, "", "", err
	}
	if len(data) > maxFileBytes {
		return nil, "", "", fmt.Errorf("theme file %q is too large", path)
	}
	return data, path, filepath.Dir(path), nil
}

func pick[T any](child, parent *T) *T {
	if child != nil {
		return child
	}
	return parent
}

func pickList(child, parent []string) []string {
	if child != nil {
		return child
	}
	return parent
}

// overlay returns s with every field the child sets taking precedence.
func (s spec) overlay(child spec) spec {
	var c, pc colorsSpec
	if s.Colors != nil {
		pc = *s.Colors
	}
	if child.Colors != nil {
		c = *child.Colors
	}
	var b, pb barSpec
	if s.Bar != nil {
		pb = *s.Bar
	}
	if child.Bar != nil {
		b = *child.Bar
	}
	return spec{
		Colors: &colorsSpec{
			Text:   pick(c.Text, pc.Text),
			Muted:  pick(c.Muted, pc.Muted),
			Good:   pick(c.Good, pc.Good),
			Warn:   pick(c.Warn, pc.Warn),
			Bad:    pick(c.Bad, pc.Bad),
			Marker: pick(c.Marker, pc.Marker),
			Path:   pick(c.Path, pc.Path),
		},
		Bar: &barSpec{
			Style:       pick(b.Style, pb.Style),
			Full:        pick(b.Full, pb.Full),
			Partials:    pickList(b.Partials, pb.Partials),
			Empty:       pick(b.Empty, pb.Empty),
			Marker:      pick(b.Marker, pb.Marker),
			Ramp:        pickList(b.Ramp, pb.Ramp),
			EmptyColor:  pick(b.EmptyColor, pb.EmptyColor),
			MarkerColor: pick(b.MarkerColor, pb.MarkerColor),
		},
	}
}

func (s spec) compile() (Theme, error) {
	t := Theme{Reset: resetSeq}
	c := colorsSpec{}
	if s.Colors != nil {
		c = *s.Colors
	}
	b := barSpec{}
	if s.Bar != nil {
		b = *s.Bar
	}

	var err error
	for _, f := range []struct {
		dst *string
		src *string
	}{
		{&t.Text, c.Text},
		{&t.Muted, c.Muted},
		{&t.Good, c.Good},
		{&t.Warn, c.Warn},
		{&t.Bad, c.Bad},
		{&t.Marker, c.Marker},
		{&t.Path, c.Path},
		{&t.EmptyColor, b.EmptyColor},
		{&t.MarkerColor, b.MarkerColor},
	} {
		if f.src == nil {
			continue
		}
		if *f.dst, err = ParseColor(*f.src); err != nil {
			return Theme{}, err
		}
	}

	style := styleSolid
	if b.Style != nil {
		style = *b.Style
	}
	for _, g := range []struct {
		dst  *string
		src  *string
		name string
	}{{&t.Full, b.Full, "full"}, {&t.Empty, b.Empty, "empty"}, {&t.MarkerGlyph, b.Marker, "marker"}} {
		if g.src == nil || !validGlyph(*g.src) {
			return Theme{}, fmt.Errorf("bar.%s must be a printable glyph", g.name)
		}
		*g.dst = *g.src
	}

	if len(b.Partials) != 0 && len(b.Partials) != partialCount {
		return Theme{}, fmt.Errorf("bar.partials must hold %d glyphs", partialCount)
	}
	for i, p := range b.Partials {
		if !validGlyph(p) {
			return Theme{}, errors.New("bar.partials must be printable glyphs")
		}
		t.Partials[i] = p
	}

	switch style {
	case styleSolid:
		if len(b.Partials) == 0 {
			return Theme{}, fmt.Errorf("bar.partials must hold %d glyphs", partialCount)
		}
	case styleGrad:
		t.Gradient = true
		if len(b.Ramp) != rampStops {
			return Theme{}, fmt.Errorf("bar.ramp must hold %d colors", rampStops)
		}
		for _, hex := range b.Ramp {
			r, g, bl, err := parseRampStop(hex)
			if err != nil {
				return Theme{}, err
			}
			t.Ramp = append(t.Ramp, [channelsRGB]int{r, g, bl})
		}
	default:
		return Theme{}, fmt.Errorf("bar.style %q is not solid or gradient", style)
	}
	return t, nil
}

func parseRampStop(s string) (r, g, b int, err error) {
	rgb, ok := strings.CutPrefix(s, "#")
	if !ok {
		return 0, 0, 0, fmt.Errorf("ramp stop %q must be #rrggbb", s)
	}
	if r, g, b, err = parseHex(rgb); err != nil {
		return 0, 0, 0, fmt.Errorf("ramp stop %q: %w", s, err)
	}
	return r, g, b, nil
}

func validGlyph(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}
