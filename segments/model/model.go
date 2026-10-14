// Package model provides the model segment: the model's name, and its effort level.
package model

import (
	_ "embed"
	"strings"

	"github.com/PaddleHQ/claude-statusline/internal/layout"
	"github.com/PaddleHQ/claude-statusline/internal/plugin"
)

const name = "model"

//go:embed defaults.conf
var defaults string

// families are the model families recognized by a substring of the model id, checked in order.
var families = []struct{ key, label string }{
	{"haiku", "Haiku"},
	{"sonnet", "Sonnet"},
	{"opus", "Opus"},
	{"fable", "Fable"},
	{"mythos", "Mythos"},
}

// Plugin is the model plugin.
func Plugin() *plugin.Plugin {
	return &plugin.Plugin{Defaults: defaults, Segments: []plugin.Entry{{Name: name, New: New}}}
}

// New builds the model segment.
func New(p *plugin.Providers) plugin.Segment { return &model{p: p} }

type model struct{ p *plugin.Providers }

func (m *model) Name() string { return name }

func (m *model) Group() layout.Group { return layout.Identity }

func (m *model) Help() string {
	help := "The model in use"
	if m.p.Config.Bool("effort") {
		help += " and its effort level, as in Opus 4.7 [high]"
	}
	return help + ". It reads Haiku (skill), for example, while a skill or subagent runs a different model than the session's."
}

func (m *model) Keys() []string { return []string{"effort"} }

func (m *model) Render(s *plugin.Session) string {
	th := m.p.Theme
	display := s.Payload.Model.DisplayName
	level := s.Payload.Effort.Level
	// A skill or subagent can run a different model than the session's, which the payload does not reflect.
	if s.Payload.TranscriptPath != "" {
		if actual, ok := family(m.p.Transcript().ActualModel); ok {
			if session, _ := family(s.Payload.Model.ID); actual.key != session.key {
				display, level = actual.label+" (skill)", ""
			}
		}
	}
	if display == "" {
		return ""
	}
	if level != "" && m.p.Config.Bool("effort") {
		display += " [" + level + "]"
	}
	return th.Paint(th.Text, display)
}

func family(id string) (f struct{ key, label string }, ok bool) {
	for _, f := range families {
		if strings.Contains(id, f.key) {
			return f, true
		}
	}
	return f, false
}
