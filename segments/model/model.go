// Package model provides the model segment: the model's name, and its effort level.
package model

import (
	_ "embed"
	"regexp"
	"strings"
)

// Defaults is the text of the segment's defaults.yaml, for layering under the user's config.
//
//go:embed defaults.yaml
var Defaults string

const name = "model"

var (
	dateSuffix = regexp.MustCompile(`-\d{8}$`)
	// A modern id is claude-<family words>[-<major>[-<minor>...]], so new models need no table entry (the transcript always holds full ids).
	modelID = regexp.MustCompile(`^claude-([a-z]+(?:-[a-z]+)*)(?:-(\d+(?:-\d+)*))?$`)
)

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
	return help + ". It reads Haiku 4.5 (skill), for example, while a skill or subagent runs a different model than the session's."
}

// Keys lists the config keys the segment reads, relative to its section.
func (m *Model) Keys() []string { return []string{"effort"} }

// Label turns claude-opus-4-7-20251001 into Opus 4.7, and shows an id it cannot parse as it is minus the claude- prefix.
func Label(id string) string {
	if label, _, ok := parse(id); ok {
		return label
	}
	return strings.TrimPrefix(id, "claude-")
}

// Render returns the segment's text, or "" to hide it.
func (m *Model) Render() string {
	display := m.d.Session.DisplayName()
	level := m.d.Session.EffortLevel()
	// A skill or subagent can run a different model than the session's, which the payload does not reflect.
	if m.d.Session.TranscriptFile() != "" {
		if actualLabel, actualKey, ok := parse(m.d.Transcript().LatestModel()); ok {
			if _, sessionKey, _ := parse(m.d.Session.ModelID()); actualKey != sessionKey {
				display, level = actualLabel+" (skill)", ""
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

// parse reads a modern model id into its label and lower-case family.
func parse(id string) (label, family string, ok bool) {
	m := modelID.FindStringSubmatch(dateSuffix.ReplaceAllString(id, ""))
	if m == nil {
		return "", "", false
	}
	words := strings.Split(m[1], "-")
	for i, w := range words {
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	label = strings.Join(words, " ")
	if m[2] != "" {
		label += " " + strings.ReplaceAll(m[2], "-", ".")
	}
	return label, m[1], true
}
