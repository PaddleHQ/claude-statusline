// Package segments is the composition root of the status line: the one list of plugins it can show, and the one place
// that connects them to the rest of the program. Plugins declare what they need as interfaces in their own packages,
// and the program's internal types satisfy those interfaces directly.
package segments

import (
	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/layout"
	"github.com/PaddleHQ/claude-statusline/internal/payload"
	"github.com/PaddleHQ/claude-statusline/internal/theme"
	"github.com/PaddleHQ/claude-statusline/segments/model"
)

// Segment is what every plugin's segment provides.
type Segment interface {
	Name() string
	// Keys lists every config key Render reads, relative to the segment's section, so a test can check them against defaults.yaml.
	Keys() []string
	// Render returns the segment's text, or "" to hide it.
	Render() string
}

// Defaults returns the text of every plugin's defaults.yaml, for layering under the user's config.
func Defaults() []string {
	return []string{model.Defaults}
}

// Env is one render's world, from which the segments are built.
type Env struct {
	Payload payload.Payload
	Config  *config.Config
	Theme   theme.Theme
	Columns int
}

// Separator is the painted glyph drawn between segments, with a space either side.
func (e *Env) Separator() string {
	return " " + e.Theme.Paint(e.Theme.Muted, e.Config.String("separator")) + " "
}

// Lookup builds the segment with the given name, with the layout group it belongs to.
func (e *Env) Lookup(name string) (Segment, layout.Group, bool) {
	for _, r := range e.registrations() {
		if r.name == name {
			return r.build(e.Config.Section(name)), r.group, true
		}
	}
	return nil, 0, false
}

// All builds every segment, in the order they are listed here.
func (e *Env) All() []Segment {
	regs := e.registrations()
	all := make([]Segment, 0, len(regs))
	for _, r := range regs {
		all = append(all, r.build(e.Config.Section(r.name)))
	}
	return all
}

type registration struct {
	name  string
	group layout.Group
	build func(cfg *config.Section) Segment
}

// registrations is the one list of segments. To add a plugin, add its entries here.
func (e *Env) registrations() []registration {
	return []registration{
		{"model", layout.Identity, func(cfg *config.Section) Segment {
			return model.New(model.Deps{Session: &e.Payload, Config: cfg, Paint: &e.Theme})
		}},
	}
}
