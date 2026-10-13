package theme

import "testing"

func TestLookupDefault(t *testing.T) {
	t.Parallel()

	got := Lookup("default")
	want := Theme{
		Muted:       "\033[2m",
		Good:        "\033[32m",
		Warn:        "\033[33m",
		Bad:         "\033[31m",
		Marker:      "\033[1m",
		Reset:       "\033[0m",
		Full:        "█",
		Partials:    [7]string{"▏", "▎", "▍", "▌", "▋", "▊", "▉"},
		Empty:       "░",
		MarkerGlyph: "┃",
	}
	if got != want {
		t.Errorf("Lookup(default) = %+v, want %+v", got, want)
	}
}

func TestLookupUnknownFallsBack(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"", "nonexistent", "Default", "../x"} {
		if Lookup(name) != Lookup("default") {
			t.Errorf("Lookup(%q) is not the default theme", name)
		}
	}
}

func TestWithoutColor(t *testing.T) {
	t.Parallel()

	d := Lookup("default")
	got := d.WithoutColor()
	if got.Text != "" || got.Muted != "" || got.Good != "" || got.Warn != "" || got.Bad != "" || got.Marker != "" || got.Reset != "" {
		t.Errorf("WithoutColor() kept a colour: %+v", got)
	}
	if got.Full != d.Full || got.Partials != d.Partials || got.Empty != d.Empty || got.MarkerGlyph != d.MarkerGlyph {
		t.Errorf("WithoutColor() changed the glyphs: %+v", got)
	}
	if d.Muted == "" {
		t.Error("WithoutColor() modified the receiver")
	}
}

func TestPaint(t *testing.T) {
	t.Parallel()

	d := Lookup("default")
	tests := []struct {
		name   string
		colour string
		want   string
	}{
		{"wraps with colour and reset", d.Muted, "\033[2mx\033[0m"},
		{"leaves text bare without colour", d.Text, "x"},
	}
	for _, tt := range tests {
		if got := d.Paint(tt.colour, "x"); got != tt.want {
			t.Errorf("%s: Paint() = %q, want %q", tt.name, got, tt.want)
		}
	}
}
