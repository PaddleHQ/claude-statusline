package theme

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/PaddleHQ/claude-statusline/themes"

	"go.yaml.in/yaml/v3"
)

var (
	errMissing           = errors.New("is required")
	errNoHome            = errors.New("no home directory")
	errNoBase            = errors.New("relative theme path without a base directory")
	errTooLarge          = errors.New("too large")
	errNotRegular        = errors.New("not a regular file")
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
	Colors *colorsSpec `yaml:"colors"`
	Bar    *barSpec    `yaml:"bar"`
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
	data, id, err := read(value, home, base)
	if err != nil {
		return Theme{}, err
	}
	var s spec
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&s); err != nil && !errors.Is(err, io.EOF) {
		return Theme{}, fmt.Errorf("theme %q: %w", id, err)
	}
	return s.compile()
}

func isPath(value string) bool {
	return strings.Contains(value, "/") || strings.HasPrefix(value, "~")
}

func read(value, home, base string) (data []byte, id string, err error) {
	if !isPath(value) {
		name := value
		if name == legacyName {
			name = fallbackName
		}
		data, err = themes.FS.ReadFile(name + ".yaml")
		if err != nil {
			return nil, "", fmt.Errorf("read built-in theme %q: %w", name, err)
		}
		return data, name, nil
	}
	path, err := themePath(value, home, base)
	if err != nil {
		return nil, "", err
	}
	data, err = readLimited(path)
	if err != nil {
		return nil, "", err
	}
	return data, path, nil
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
	// A FIFO or device would block the open, and a status line must never hang.
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("open theme file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("theme file %q is %w", path, errNotRegular)
	}
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

func (s *spec) compile() (Theme, error) {
	if s.Colors == nil {
		return Theme{}, fmt.Errorf("colors %w", errMissing)
	}
	if s.Bar == nil {
		return Theme{}, fmt.Errorf("bar %w", errMissing)
	}
	t := Theme{Reset: resetSeq}
	if err := t.compileColors(*s.Colors, *s.Bar); err != nil {
		return Theme{}, err
	}
	if err := t.compileGlyphs(*s.Bar); err != nil {
		return Theme{}, err
	}
	if err := t.compilePartials(s.Bar.Partials); err != nil {
		return Theme{}, err
	}
	if err := t.compileStyle(*s.Bar); err != nil {
		return Theme{}, err
	}
	return t, nil
}

func (t *Theme) compileStyle(b barSpec) error {
	if b.Style == nil {
		return fmt.Errorf("bar.style %w", errMissing)
	}
	switch *b.Style {
	case styleSolid:
		if len(b.Partials) == 0 {
			return fmt.Errorf("%w %d glyphs", errPartialsCount, partialCount)
		}
		return nil
	case styleGrad:
		return t.compileRamp(b.Ramp)
	default:
		return fmt.Errorf("bar.style %q is %w", *b.Style, errStyle)
	}
}

func (t *Theme) compileColors(c colorsSpec, b barSpec) error {
	for _, f := range []struct {
		dst      *string
		src      *string
		name     string
		optional bool
	}{
		{&t.TextColor, c.Text, "colors.text", false},
		{&t.Muted, c.Muted, "colors.muted", false},
		{&t.Good, c.Good, "colors.good", false},
		{&t.WarnColor, c.Warn, "colors.warn", false},
		{&t.BadColor, c.Bad, "colors.bad", false},
		{&t.Marker, c.Marker, "colors.marker", false},
		{&t.PathColor, c.Path, "colors.path", false},
		{&t.EmptyColor, b.EmptyColor, "bar.empty_color", true},
		{&t.MarkerColor, b.MarkerColor, "bar.marker_color", true},
	} {
		if f.src == nil {
			if f.optional {
				continue
			}
			return fmt.Errorf("%s %w", f.name, errMissing)
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
