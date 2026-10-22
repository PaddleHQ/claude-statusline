// Package cost provides the cost segment: what the session has cost so far, in US dollars.
package cost

import (
	_ "embed"
	"fmt"
	"math"
)

// Defaults is the text of the segment's defaults.yaml, for layering under the user's config.
//
//go:embed defaults.yaml
var Defaults string

const (
	name           = "cost"
	centsPerDollar = 100
)

//go:generate go run github.com/vektra/mockery/v2 --name=Session

// Session is what the segment needs to know about the session.
type Session interface {
	// TotalCost is the session's cost in US dollars; false means it was not reported.
	TotalCost() (float64, bool)
}

//go:generate go run github.com/vektra/mockery/v2 --name=Config

// Config reads the segment's settings.
type Config interface {
	Int(key string) int
}

//go:generate go run github.com/vektra/mockery/v2 --name=Painter

// Painter colors text.
type Painter interface {
	Text(s string) string
	Warn(s string) string
	Bad(s string) string
}

// Deps is everything the segment is built from.
type Deps struct {
	Session Session
	Config  Config
	Paint   Painter
}

// Cost shows the session's cost.
type Cost struct{ d Deps }

// New builds the cost segment.
func New(d Deps) *Cost { return &Cost{d: d} }

// Name is the segment's name in the segments list.
func (c *Cost) Name() string { return name }

// Help says what the segment shows, for the help command.
func (c *Cost) Help() string {
	return fmt.Sprintf("What the session would cost at API prices, in US dollars. "+
		"On a Pro or Max plan you are not charged this. Amber from $%d, red from $%d.",
		c.d.Config.Int("warn"), c.d.Config.Int("bad"))
}

// Keys lists the config keys the segment reads, relative to its section.
func (c *Cost) Keys() []string { return []string{"warn", "bad"} }

// Render returns the segment's text, or "" to hide it.
func (c *Cost) Render() string {
	usd, ok := c.d.Session.TotalCost()
	if !ok {
		return ""
	}
	// Compare in whole cents, so the color agrees with the figure shown: 4.999 prints as $5.00 and is amber.
	usd = math.Round(usd*centsPerDollar) / centsPerDollar
	text := fmt.Sprintf("$%.2f", usd)
	switch {
	case usd >= float64(c.d.Config.Int("bad")):
		return c.d.Paint.Bad(text)
	case usd >= float64(c.d.Config.Int("warn")):
		return c.d.Paint.Warn(text)
	default:
		return c.d.Paint.Text(text)
	}
}
