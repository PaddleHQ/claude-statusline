package theme_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/PaddleHQ/claude-statusline/internal/theme"
	"github.com/PaddleHQ/claude-statusline/themes"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadClassic(t *testing.T) {
	t.Parallel()

	got := theme.Load("classic", "", "")
	want := theme.Theme{
		Muted:       "\033[2m",
		Good:        "\033[32m",
		WarnColor:   "\033[33m",
		BadColor:    "\033[31m",
		Marker:      "\033[1m",
		PathColor:   "\033[1m\033[38;5;33m",
		Reset:       "\033[0m",
		Full:        "█",
		Partials:    [7]string{"▏", "▎", "▍", "▌", "▋", "▊", "▉"},
		Empty:       "░",
		MarkerGlyph: "┃",
	}
	assert.Equal(t, want, got)
}

func TestLoadGradient(t *testing.T) {
	t.Parallel()

	got := theme.Load("gradient", "", "")
	classic := theme.Load("classic", "", "")
	assert.True(t, got.Gradient)
	assert.Equal(t, [][3]int{{148, 226, 173}, {249, 226, 175}, {243, 139, 168}}, got.Ramp)
	assert.Equal(t, "\033[38;2;74;74;74m", got.EmptyColor)
	assert.Equal(t, "\033[1m", got.MarkerColor, "the terminal's own text color, so the marker shows on light and dark")
	assert.Equal(t, "▰", got.Full)
	assert.Equal(t, "▱", got.Empty)
	assert.Equal(t, "/", got.MarkerGlyph)
	assert.Equal(t, classic.PathColor, got.PathColor)
	assert.Equal(t, [7]string{}, got.Partials, "a gradient file has no partials")
}

func TestLoadDefaultAlias(t *testing.T) {
	t.Parallel()

	assert.Equal(t, theme.Load("classic", "", ""), theme.Load("default", "", ""))
	_, err := theme.Open("default", "", "")
	assert.NoError(t, err)
}

func writeTheme(t *testing.T, dir, name, text string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte(text), 0o600))
	return path
}

func TestOpenFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	full := `
colors:
  text: cyan
  muted: bright-black
  good: "#00ff00"
  warn: 208
  bad: bold red
  marker: underline
  path: italic blue
bar:
  style: solid
  full: "#"
  partials: ["1", "2", "3", "4", "5", "6", "7"]
  empty: "."
  marker: "|"
`
	abs := writeTheme(t, dir, "full.yaml", full)
	writeTheme(t, dir, "sub/rel.yaml", full)
	home := t.TempDir()
	writeTheme(t, home, ".config/mine.yaml", full)

	for name, value := range map[string]string{"absolute": abs, "relative to base": "sub/rel.yaml", "tilde": "~/.config/mine.yaml"} {
		got, err := theme.Open(value, home, dir)
		require.NoError(t, err, name)
		assert.Equal(t, "\033[36m", got.TextColor, name)
		assert.Equal(t, "\033[90m", got.Muted, name)
		assert.Equal(t, "\033[38;2;0;255;0m", got.Good, name)
		assert.Equal(t, "\033[38;5;208m", got.WarnColor, name)
		assert.Equal(t, "\033[1m\033[31m", got.BadColor, name)
		assert.Equal(t, "\033[4m", got.Marker, name)
		assert.Equal(t, "\033[3m\033[34m", got.PathColor, name)
		assert.Equal(t, "#", got.Full, name)
		assert.Equal(t, "|", got.MarkerGlyph, name)
	}
}

const validTheme = `
colors:
  text: ""
  muted: dim
  good: green
  warn: yellow
  bad: red
  marker: bold
  path: bold 33
bar:
  style: solid
  full: "#"
  partials: ["1", "2", "3", "4", "5", "6", "7"]
  empty: "."
  marker: "|"
`

func TestInvalidThemesFallBack(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	broken := func(old, replacement string) string {
		require.Contains(t, validTheme, old)
		return strings.Replace(validTheme, old, replacement, 1)
	}
	tests := []struct {
		name  string
		value string
		text  string
	}{
		{"unknown built-in name", "nonexistent", ""},
		{"wrong case", "Classic", ""},
		{"missing file", filepath.Join(dir, "missing.yaml"), ""},
		{"a directory", dir, ""},
		{"relative path with no base", "x/y.yaml", ""},
		{"tilde with no home", "~/x.yaml", ""},
		{"invalid YAML", "", "colors: [unclosed"},
		{"unknown top-level key", "", "unknown: key\n" + validTheme},
		{"unknown nested key", "", broken("good: green", "goood: green")},
		{"bad color spec", "", broken("good: green", "good: chartreuse")},
		{"color out of range", "", broken("good: green", "good: 256")},
		{"short hex", "", broken("good: green", "good: \"#fff\"")},
		{"two colors", "", broken("good: green", "good: red blue")},
		{"missing color", "", broken("  good: green\n", "")},
		{"missing colors section", "", "bar:\n  style: solid\n"},
		{"missing bar section", "", "colors:\n  good: green\n"},
		{"missing style", "", broken("  style: solid\n", "")},
		{"unknown style", "", broken("style: solid", "style: rainbow")},
		{"missing glyph", "", broken("  full: \"#\"\n", "")},
		{"empty glyph", "", broken("full: \"#\"", "full: \"\"")},
		{"control character in a glyph", "", broken("full: \"#\"", "full: \"\\e[0m\"")},
		{"solid without partials", "", broken("  partials: [\"1\", \"2\", \"3\", \"4\", \"5\", \"6\", \"7\"]\n", "")},
		{"wrong number of partials", "", broken("partials: [\"1\", ", "partials: [")},
		{"gradient without a ramp", "", broken("style: solid", "style: gradient")},
		{"bad ramp stop", "", broken("style: solid", "style: gradient\n  ramp: [\"#000000\", \"red\", \"#ffffff\"]")},
		{"short hex ramp stop", "", broken("style: solid", "style: gradient\n  ramp: [\"#fff\", \"#000000\", \"#000000\"]")},
		{"misspelled colors section", "", strings.Replace(validTheme, "colors:", "colrs:", 1)},
		{"empty file", "", ""},
	}
	classic := theme.Load("classic", "", "")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			value := tt.value
			if value == "" {
				value = writeTheme(t, t.TempDir(), "t.yaml", tt.text)
			}
			_, err := theme.Open(value, "", "")
			require.Error(t, err)
			assert.Equal(t, classic, theme.Load(value, "", ""))
		})
	}
}

func TestOpenCompleteGradientFile(t *testing.T) {
	t.Parallel()

	noPartials := "  partials: [\"1\", \"2\", \"3\", \"4\", \"5\", \"6\", \"7\"]\n"
	text := strings.Replace(strings.Replace(validTheme, noPartials, "", 1), "style: solid", "style: gradient\n  ramp: [\"#000000\", \"#808080\", \"#ffffff\"]", 1)
	got, err := theme.Open(writeTheme(t, t.TempDir(), "g.yaml", text), "", "")
	require.NoError(t, err)
	assert.True(t, got.Gradient)
	assert.Equal(t, [][3]int{{0, 0, 0}, {128, 128, 128}, {255, 255, 255}}, got.Ramp)
}

func TestParseColor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		spec string
		want string
	}{
		{"", ""},
		{"   ", ""},
		{"bold", "\033[1m"},
		{"dim", "\033[2m"},
		{"italic", "\033[3m"},
		{"underline", "\033[4m"},
		{"black", "\033[30m"},
		{"red", "\033[31m"},
		{"green", "\033[32m"},
		{"yellow", "\033[33m"},
		{"blue", "\033[34m"},
		{"magenta", "\033[35m"},
		{"cyan", "\033[36m"},
		{"white", "\033[37m"},
		{"bright-black", "\033[90m"},
		{"bright-white", "\033[97m"},
		{"0", "\033[38;5;0m"},
		{"33", "\033[38;5;33m"},
		{"255", "\033[38;5;255m"},
		{"#000000", "\033[38;2;0;0;0m"},
		{"#4a4a4a", "\033[38;2;74;74;74m"},
		{"#F38BA8", "\033[38;2;243;139;168m"},
		{"bold 33", "\033[1m\033[38;5;33m"},
		{"33 bold", "\033[38;5;33m\033[1m"},
		{"dim italic underline cyan", "\033[2m\033[3m\033[4m\033[36m"},
		{"bold #ffffff", "\033[1m\033[38;2;255;255;255m"},
	}
	for _, tt := range tests {
		got, err := theme.ParseColor(tt.spec)
		require.NoError(t, err, "spec %q", tt.spec)
		assert.Equal(t, tt.want, got, "spec %q", tt.spec)
	}

	for _, spec := range []string{"purple", "bright-", "bright-grey", "Bold", "256", "-1", "1e2", "+5", "#12345", "#1234567", "#gggggg", "#", "red green", "33 #fff000", "bold,red", "99999999999999999999"} {
		_, err := theme.ParseColor(spec)
		assert.Error(t, err, "spec %q", spec)
	}
}

func TestRampColor(t *testing.T) {
	t.Parallel()

	th := theme.Load("gradient", "", "")
	tests := []struct {
		name        string
		cell, width int
		want        string
	}{
		{"first cell is the good stop", 0, 20, "\033[38;2;148;226;173m"},
		{"last cell is the bad stop", 19, 20, "\033[38;2;243;139;168m"},
		{"middle cell of an odd width is the warn stop", 1, 3, "\033[38;2;249;226;175m"},
		{"first leg step", 1, 5, "\033[38;2;198;226;174m"},
		{"second leg step truncates towards zero", 3, 5, "\033[38;2;246;183;172m"},
		{"a single cell takes the good stop", 0, 1, "\033[38;2;148;226;173m"},
		{"width ten, cell four", 4, 10, "\033[38;2;237;226;174m"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, th.RampColor(tt.cell, tt.width), tt.name)
	}
	classic := theme.Load("classic", "", "")
	assert.Empty(t, classic.RampColor(0, 10), "solid themes have no ramp")
	plain := th.WithoutColor()
	assert.Empty(t, plain.RampColor(0, 10), "NO_COLOR removes the ramp")
}

func TestWithoutColor(t *testing.T) {
	t.Parallel()

	d := theme.Load("gradient", "", "")
	got := d.WithoutColor()
	for _, color := range []string{got.TextColor, got.Muted, got.Good, got.WarnColor, got.BadColor, got.Marker, got.PathColor, got.Reset, got.EmptyColor, got.MarkerColor} {
		assert.Empty(t, color, "WithoutColor() kept a color: %+v", got)
	}
	assert.Empty(t, got.Ramp)
	assert.Equal(t, d.Full, got.Full)
	assert.Equal(t, d.Partials, got.Partials)
	assert.Equal(t, d.Empty, got.Empty)
	assert.Equal(t, d.MarkerGlyph, got.MarkerGlyph)
	assert.True(t, got.Gradient, "WithoutColor() keeps the style")
	assert.NotEmpty(t, d.Muted, "WithoutColor() modified the receiver")
	assert.Len(t, d.Ramp, 3, "WithoutColor() modified the receiver")
}

func TestPaint(t *testing.T) {
	t.Parallel()

	d := theme.Load("classic", "", "")
	tests := []struct {
		name  string
		color string
		want  string
	}{
		{"wraps with color and reset", d.Muted, "\033[2mx\033[0m"},
		{"leaves text bare without color", d.TextColor, "x"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, d.Paint(tt.color, "x"), tt.name)
	}
}

func TestEveryBuiltinLoads(t *testing.T) {
	t.Parallel()

	files, err := fs.Glob(themes.FS, "*.yaml")
	require.NoError(t, err)
	require.NotEmpty(t, files)
	for _, file := range files {
		name := strings.TrimSuffix(filepath.Base(file), ".yaml")
		_, err := theme.Open(name, "", "")
		assert.NoError(t, err, "built-in %q", name)
	}
}

func TestRolePaintersWrapInTheRoleColor(t *testing.T) {
	t.Parallel()

	d := theme.Load("classic", "", "")
	assert.Equal(t, d.Paint(d.TextColor, "x"), d.Text("x"))
	assert.Equal(t, d.Paint(d.WarnColor, "x"), d.Warn("x"))
	assert.Equal(t, d.Paint(d.BadColor, "x"), d.Bad("x"))
	assert.Equal(t, d.Paint(d.PathColor, "x"), d.Path("x"))
	plain := d.WithoutColor()
	assert.Equal(t, "x", plain.Bad("x"))
}

func TestOpenRejectsFIFO(t *testing.T) {
	t.Parallel()

	fifo := filepath.Join(t.TempDir(), "theme.yaml")
	require.NoError(t, syscall.Mkfifo(fifo, 0o600))

	got := make(chan error, 1)
	go func() {
		_, err := theme.Open(fifo, "", "")
		got <- err
	}()

	select {
	case err := <-got:
		assert.Error(t, err)
	case <-time.After(2 * time.Second):
		require.FailNow(t, "Open blocked on a FIFO")
	}
}
