// Package icons holds the icon set the user chose. "none" keeps the plain
// text labels; "nerd" swaps them for Nerd Font glyphs, which only render in a
// terminal that uses a patched font.
package icons

import "strings"

// Set is the chosen icon set. The zero value is "none".
type Set struct {
	nerd bool
}

// Parse reads the `icons` setting. An empty value means none. ok is false for
// an unknown value, and the returned set is then none.
func Parse(value string) (set Set, ok bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "none":
		return Set{}, true
	case "nerd":
		return Set{nerd: true}, true
	}
	return Set{}, false
}

// Pick returns the Nerd Font text when the set is nerd, and the plain text otherwise.
func (s *Set) Pick(plain, nerd string) string {
	if s.nerd {
		return nerd
	}
	return plain
}

// Nerd reports whether Nerd Font glyphs are in use.
func (s *Set) Nerd() bool {
	return s.nerd
}
