package theme_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/theme"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const saberBar = `  tip: "'"
  blade: ["#4aa8ff", "#b26bff", "#ff3b30"]
  glow: 50
  glow_cells: 3
  empty_color: "#3a3a3a"
  marker_color: bold`

func saberTheme(t *testing.T, bar string) string {
	t.Helper()
	noPartials := "  partials: [\"1\", \"2\", \"3\", \"4\", \"5\", \"6\", \"7\"]\n"
	return strings.Replace(strings.Replace(validTheme, noPartials, "", 1), "style: solid", "style: saber\n"+bar, 1)
}

func TestLoadSaber(t *testing.T) {
	t.Parallel()

	got := theme.Load("saber", "", "")
	classic := theme.Load("classic", "", "")
	assert.True(t, got.Saber)
	assert.False(t, got.Gradient)
	assert.Equal(t, [][3]int{{74, 168, 255}, {178, 107, 255}, {255, 59, 48}}, got.Blade)
	assert.Equal(t, 50, got.Glow)
	assert.Equal(t, 3, got.GlowCells)
	assert.Equal(t, "╸", got.Tip)
	assert.Equal(t, "━", got.Full)
	assert.Equal(t, "─", got.Empty)
	assert.Equal(t, "┃", got.MarkerGlyph)
	assert.Equal(t, "\033[38;2;74;168;255m", got.Good)
	assert.Equal(t, "\033[38;2;178;107;255m", got.WarnColor)
	assert.Equal(t, "\033[38;2;255;59;48m", got.BadColor)
	assert.Equal(t, "\033[38;2;58;58;58m", got.EmptyColor)
	assert.Equal(t, "\033[1m", got.MarkerColor)
	assert.Equal(t, classic.PathColor, got.PathColor)
	assert.Equal(t, [7]string{}, got.Partials, "a saber file has no partials")
}

func TestOpenCompleteSaberFile(t *testing.T) {
	t.Parallel()

	got, err := theme.Open(writeTheme(t, t.TempDir(), "s.yaml", saberTheme(t, saberBar)), "", "")
	require.NoError(t, err)
	assert.True(t, got.Saber)
	assert.Equal(t, [][3]int{{74, 168, 255}, {178, 107, 255}, {255, 59, 48}}, got.Blade)
	assert.Equal(t, "'", got.Tip)

	edge := strings.Replace(strings.Replace(saberBar, "glow: 50", "glow: 0", 1), "glow_cells: 3", "glow_cells: 1", 1)
	got, err = theme.Open(writeTheme(t, t.TempDir(), "edge.yaml", saberTheme(t, edge)), "", "")
	require.NoError(t, err)
	assert.Equal(t, 0, got.Glow)
	assert.Equal(t, 1, got.GlowCells)

	full := strings.Replace(saberBar, "glow: 50", "glow: 100", 1)
	_, err = theme.Open(writeTheme(t, t.TempDir(), "full.yaml", saberTheme(t, full)), "", "")
	require.NoError(t, err)
}

func TestInvalidSaberThemesAreRejected(t *testing.T) {
	t.Parallel()

	without := func(line string) string { return strings.Replace(saberBar, line, "", 1) }
	replace := func(old, replacement string) string { return strings.Replace(saberBar, old, replacement, 1) }
	tests := []struct {
		name string
		bar  string
	}{
		{"missing blade", without("  blade: [\"#4aa8ff\", \"#b26bff\", \"#ff3b30\"]\n")},
		{"two blade stops", replace("\"#4aa8ff\", ", "")},
		{"bad blade stop", replace("\"#b26bff\"", "\"purple\"")},
		{"short hex blade stop", replace("\"#ff3b30\"", "\"#f30\"")},
		{"missing glow", without("  glow: 50\n")},
		{"glow below range", replace("glow: 50", "glow: -1")},
		{"glow above range", replace("glow: 50", "glow: 101")},
		{"missing glow cells", without("  glow_cells: 3\n")},
		{"zero glow cells", replace("glow_cells: 3", "glow_cells: 0")},
		{"missing tip", without("  tip: \"'\"\n")},
		{"empty tip", replace("tip: \"'\"", "tip: \"\"")},
	}
	classic := theme.Load("classic", "", "")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := writeTheme(t, t.TempDir(), "t.yaml", saberTheme(t, tt.bar))
			_, err := theme.Open(path, "", "")
			require.Error(t, err)
			assert.Equal(t, classic, theme.Load(path, "", ""))
		})
	}
}

func TestBladeColor(t *testing.T) {
	t.Parallel()

	th := theme.Theme{
		Saber: true, Glow: 50, GlowCells: 3,
		Blade: [][3]int{{0, 0, 0}, {100, 100, 100}, {200, 200, 200}},
	}
	c := func(n int) string { return fmt.Sprintf("\033[38;2;%d;%d;%dm", n, n, n) }
	tests := []struct {
		name             string
		state, cell, lit int
		want             string
	}{
		{"cells before the window are the plain state color", 0, 0, 10, c(0)},
		{"the last plain cell", 0, 6, 10, c(0)},
		{"first cell of the window", 0, 7, 10, c(40)},
		{"second cell of the window", 0, 8, 10, c(84)},
		{"the tip takes the full glow", 0, 9, 10, c(127)},
		{"warn blade tip", 1, 9, 10, c(100 + 155*50/100)},
		{"bad blade tip", 2, 9, 10, c(200 + 55*50/100)},
		{"a blade shorter than the window glows over its own length", 0, 0, 2, c(63)},
		{"the tip of a short blade", 0, 1, 2, c(127)},
		{"a blade exactly as long as the window starts glowing at once", 0, 0, 3, c(40)},
		{"and ends at the full glow", 0, 2, 3, c(127)},
		{"a single cell takes the full glow", 0, 0, 1, c(127)},
		{"an out of range state has no color", 3, 0, 1, ""},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, th.BladeColor(tt.state, tt.cell, tt.lit), tt.name)
	}
	classic := theme.Load("classic", "", "")
	assert.Empty(t, classic.BladeColor(0, 0, 1), "solid themes have no blade")
	plain := th.WithoutColor()
	assert.Empty(t, plain.BladeColor(0, 0, 1), "NO_COLOR removes the blade")
	assert.True(t, plain.Saber, "WithoutColor() keeps the style")
	assert.Len(t, th.Blade, 3, "WithoutColor() modified the receiver")
}
