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

var (
	errExtendsItself     = errors.New("extends itself")
	errNoHome            = errors.New("no home directory")
	errNoBase            = errors.New("relative theme path without a base directory")
	errTooLarge          = errors.New("too large")
	errGlyph             = errors.New("a printable glyph")
	errPartialsPrintable = errors.New("bar.partials must be printable glyphs")
	errPartialsCount     = errors.New("bar.partials must hold")
	errRampCount         = errors.New("bar.ramp must hold")
	errStyle             = errors.New("not solid or gradient")
	errRampFormat        = errors.New("#rrggbb")
)

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
		return spec{}, fmt.Errorf("theme %q %w", id, errExtendsItself)
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
		if err != nil {
			return nil, "", "", fmt.Errorf("read built-in theme %q: %w", name, err)
		}
		return data, "builtin:" + name, "", nil
	}
	path, err := themePath(value, home, base)
	if err != nil {
		return nil, "", "", err
	}
	data, err = readLimited(path)
	if err != nil {
		return nil, "", "", err
	}
	return data, path, filepath.Dir(path), nil
}

func themePath(value, home, base string) (string, error) {
	path := value
	switch {
	case value == "~" || strings.HasPrefix(value, "~/"):
		if home == "" {
			return "", errNoHome
		}
		path = filepath.Join(home, value[1:])
	case !filepath.IsAbs(value):
		if base == "" {
			return "", errNoBase
		}
		path = filepath.Join(base, value)
	}
	return filepath.Clean(path), nil
}

func readLimited(path string) ([]byte, error) {
	f, err := os.Open(path) //nolint:gosec // G304: opening the theme file the user configured is the feature
	if err != nil {
		return nil, fmt.Errorf("open theme file: %w", err)
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, maxFileBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read theme file %q: %w", path, err)
	}
	if len(data) > maxFileBytes {
		return nil, fmt.Errorf("theme file %q is %w", path, errTooLarge)
	}
	return data, nil
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
func (s *spec) overlay(child spec) spec {
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

func (s *spec) compile() (Theme, error) {
	t := Theme{Reset: resetSeq}
	c := deref(s.Colors)
	b := deref(s.Bar)

	if err := t.compileColors(c, b); err != nil {
		return Theme{}, err
	}
	if err := t.compileGlyphs(b); err != nil {
		return Theme{}, err
	}
	if err := t.compilePartials(b.Partials); err != nil {
		return Theme{}, err
	}
	if err := t.compileStyle(b); err != nil {
		return Theme{}, err
	}
	return t, nil
}

func deref[T any](p *T) T {
	var zero T
	if p == nil {
		return zero
	}
	return *p
}

func (t *Theme) compileStyle(b barSpec) error {
	style := styleSolid
	if b.Style != nil {
		style = *b.Style
	}
	switch style {
	case styleSolid:
		if len(b.Partials) == 0 {
			return fmt.Errorf("%w %d glyphs", errPartialsCount, partialCount)
		}
	case styleGrad:
		return t.compileRamp(b.Ramp)
	default:
		return fmt.Errorf("bar.style %q is %w", style, errStyle)
	}
	return nil
}

func (t *Theme) compileColors(c colorsSpec, b barSpec) error {
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
		seq, err := ParseColor(*f.src)
		if err != nil {
			return err
		}
		*f.dst = seq
	}
	return nil
}

func (t *Theme) compileGlyphs(b barSpec) error {
	for _, g := range []struct {
		dst  *string
		src  *string
		name string
	}{{&t.Full, b.Full, "full"}, {&t.Empty, b.Empty, "empty"}, {&t.MarkerGlyph, b.Marker, "marker"}} {
		if g.src == nil || !validGlyph(*g.src) {
			return fmt.Errorf("bar.%s must be %w", g.name, errGlyph)
		}
		*g.dst = *g.src
	}
	return nil
}

func (t *Theme) compilePartials(partials []string) error {
	if len(partials) != 0 && len(partials) != partialCount {
		return fmt.Errorf("%w %d glyphs", errPartialsCount, partialCount)
	}
	for i, p := range partials {
		if !validGlyph(p) {
			return errPartialsPrintable
		}
		t.Partials[i] = p
	}
	return nil
}

func (t *Theme) compileRamp(ramp []string) error {
	t.Gradient = true
	if len(ramp) != rampStops {
		return fmt.Errorf("%w %d colors", errRampCount, rampStops)
	}
	for _, hex := range ramp {
		r, g, bl, err := parseRampStop(hex)
		if err != nil {
			return err
		}
		t.Ramp = append(t.Ramp, [channelsRGB]int{r, g, bl})
	}
	return nil
}

func parseRampStop(s string) (r, g, b int, err error) {
	rgb, ok := strings.CutPrefix(s, "#")
	if !ok {
		return 0, 0, 0, fmt.Errorf("ramp stop %q must be %w", s, errRampFormat)
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
