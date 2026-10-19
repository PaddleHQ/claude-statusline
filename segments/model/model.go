// Package model provides the model segment: the model's name, and its effort level.
package model

import (
	_ "embed"
	"strings"
)

// Defaults is the text of the segment's defaults.yaml, for layering under the user's config.
//
//go:embed defaults.yaml
var Defaults string

const name = "model"

// families are the model families recognized by a substring of the model id, checked in order.
var families = []struct{ key, label string }{
	{"haiku", "Haiku"},
	{"sonnet", "Sonnet"},
	{"opus", "Opus"},
	{"fable", "Fable"},
	{"mythos", "Mythos"},
}

//go:generate go run github.com/vektra/mockery/v2 --name=Session

// Session is what the segment needs to know about the session.
type Session interface {
	DisplayName() string
	ModelID() string
	EffortLevel() string
	TranscriptFile() string
}

//go:generate go run github.com/vektra/mockery/v2 --name=Transcript

// Transcript is what the segment needs from the session transcript.
type Transcript interface {
	LatestModel() string
}

//go:generate go run github.com/vektra/mockery/v2 --name=Config

// Config reads the segment's settings.
type Config interface {
	Bool(key string) bool
}

//go:generate go run github.com/vektra/mockery/v2 --name=Painter

// Painter colors text.
type Painter interface {
	Text(s string) string
}

// Deps is everything the segment is built from. Transcript is a function so that the transcript is only read when the segment needs it.
type Deps struct {
	Session    Session
	Transcript func() Transcript
	Config     Config
	Paint      Painter
}

// Model shows the model in use.
type Model struct{ d Deps }

// New builds the model segment.
func New(d Deps) *Model { return &Model{d: d} }

// Name is the segment's name in the segments list.
func (m *Model) Name() string { return name }

// Help says what the segment shows, for the help command.
func (m *Model) Help() string {
	help := "The model in use"
	if m.d.Config.Bool("effort") {
		help += " and its effort level, as in Opus 4.7 [high]"
	}
	return help + ". It reads Haiku (skill), for example, while a skill or subagent runs a different model than the session's."
}

// Keys lists the config keys the segment reads, relative to its section.
func (m *Model) Keys() []string { return []string{"effort"} }

// Render returns the segment's text, or "" to hide it.
func (m *Model) Render() string {
	display := m.d.Session.DisplayName()
	level := m.d.Session.EffortLevel()
	// A skill or subagent can run a different model than the session's, which the payload does not reflect.
	if m.d.Session.TranscriptFile() != "" {
		if actual, ok := family(m.d.Transcript().LatestModel()); ok {
			if session, _ := family(m.d.Session.ModelID()); actual.key != session.key {
				display, level = actual.label+" (skill)", ""
			}
		}
	}
	if display == "" {
		return ""
	}
	if level != "" && m.d.Config.Bool("effort") {
		display += " [" + level + "]"
	}
	return m.d.Paint.Text(display)
}

func family(id string) (f struct{ key, label string }, ok bool) {
	for _, f := range families {
		if strings.Contains(id, f.key) {
			return f, true
		}
	}
	return f, false
}
