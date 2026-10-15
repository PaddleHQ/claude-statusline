// Package model provides the model segment: the model's name, and its effort level.
package model

import (
	_ "embed"
)

// Defaults is the text of the segment's defaults.conf, for layering under the user's config.
//
//go:embed defaults.conf
var Defaults string

const name = "model"

//go:generate go run github.com/vektra/mockery/v2 --name=Session

// Session is what the segment needs to know about the session.
type Session interface {
	DisplayName() string
	EffortLevel() string
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

// Deps is everything the segment is built from.
type Deps struct {
	Session Session
	Config  Config
	Paint   Painter
}

// Model shows the model in use.
type Model struct{ d Deps }

// New builds the model segment.
func New(d Deps) *Model { return &Model{d: d} }

// Name is the segment's name in the segments list.
func (m *Model) Name() string { return name }

// Keys lists the config keys the segment reads, relative to its section.
func (m *Model) Keys() []string { return []string{"effort"} }

// Render returns the segment's text, or "" to hide it.
func (m *Model) Render() string {
	display := m.d.Session.DisplayName()
	if display == "" {
		return ""
	}
	if level := m.d.Session.EffortLevel(); level != "" && m.d.Config.Bool("effort") {
		display += " [" + level + "]"
	}
	return m.d.Paint.Text(display)
}
