// Package theme defines the colours and glyphs a status line is drawn with.
package theme

// Theme holds role colours as raw escape sequences and the glyphs for bars.
// An empty colour means "leave the terminal's own".
type Theme struct {
	Text   string
	Muted  string
	Good   string
	Warn   string
	Bad    string
	Marker string
	Path   string
	Reset  string

	Full        string
	Partials    [7]string
	Empty       string
	MarkerGlyph string
}

const defaultName = "default"

// Basic ANSI colours and a bold default-foreground marker read well on light and dark terminals.
var themes = map[string]Theme{
	defaultName: {
		Muted:  "\033[2m",
		Good:   "\033[32m",
		Warn:   "\033[33m",
		Bad:    "\033[31m",
		Marker: "\033[1m",
		Path:   "\033[1m\033[38;5;33m",
		Reset:  "\033[0m",

		Full:        "█",
		Partials:    [7]string{"▏", "▎", "▍", "▌", "▋", "▊", "▉"},
		Empty:       "░",
		MarkerGlyph: "┃",
	},
}

// Lookup returns the named theme, or the default for an unknown name.
func Lookup(name string) Theme {
	if t, ok := themes[name]; ok {
		return t
	}
	return themes[defaultName]
}

// WithoutColor clears every colour and style, keeping the glyphs.
func (t Theme) WithoutColor() Theme {
	t.Text, t.Muted, t.Good, t.Warn, t.Bad, t.Marker, t.Path, t.Reset = "", "", "", "", "", "", "", ""
	return t
}

// Paint wraps s in the colour and a reset, or returns it untouched when the colour is empty.
func (t Theme) Paint(colour, s string) string {
	if colour == "" {
		return s
	}
	return colour + s + t.Reset
}
