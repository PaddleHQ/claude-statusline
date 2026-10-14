// Package tokens provides the tokens segment: session token totals.
package tokens

import (
	"fmt"
	"strings"
)

const (
	name     = "tokens"
	thousand = 1000
	million  = 1000 * thousand
	tenth    = 10
)

//go:generate go run github.com/vektra/mockery/v2 --name=Session

// Session is what the segment needs to know about the session; a false result means the payload did not report the total.
type Session interface {
	TotalInput() (int64, bool)
	TotalOutput() (int64, bool)
}

//go:generate go run github.com/vektra/mockery/v2 --name=Painter

// Painter colors text.
type Painter interface {
	Text(s string) string
}

// Deps is everything the segment is built from. Separator is the painted text drawn between the two totals.
type Deps struct {
	Session   Session
	Paint     Painter
	Separator string
}

// Totals shows session token totals as "↓ <in> · ↑ <out>".
type Totals struct{ d Deps }

// New builds the tokens segment.
func New(d Deps) *Totals { return &Totals{d: d} }

// Name is the segment's name in the segments list.
func (t *Totals) Name() string { return name }

// Help says what the segment shows, for the help command.
func (t *Totals) Help() string {
	return "Session token totals: input after the down arrow, output after the up arrow."
}

// Keys lists the config keys the segment reads; it has none.
func (t *Totals) Keys() []string { return nil }

// Render returns the segment's text, or "" to hide it.
func (t *Totals) Render() string {
	var parts []string
	if in, ok := t.d.Session.TotalInput(); ok {
		parts = append(parts, t.d.Paint.Text("↓ "+FormatTokens(in)))
	}
	if out, ok := t.d.Session.TotalOutput(); ok {
		parts = append(parts, t.d.Paint.Text("↑ "+FormatTokens(out)))
	}
	return strings.Join(parts, t.d.Separator)
}

// FormatTokens abbreviates a count, truncating: 1500 is 1.5k and 1500000 is 1.5M.
func FormatTokens(n int64) string {
	switch {
	case n >= million:
		return fmt.Sprintf("%d.%dM", n/million, n%million/(million/tenth))
	case n >= thousand:
		return fmt.Sprintf("%d.%dk", n/thousand, n%thousand/(thousand/tenth))
	}
	return fmt.Sprintf("%d", n)
}
