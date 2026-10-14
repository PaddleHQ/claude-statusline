// Package model provides the model segment: the model's name, and its effort level.
package model

//go:generate go run github.com/vektra/mockery/v2 --name=Session

// Session is what the segment needs to know about the session.
type Session interface {
	DisplayName() string
	EffortLevel() string
}

// Deps is everything the segment is built from.
type Deps struct {
	Session Session
}

// Model shows the model in use.
type Model struct{ d Deps }

// New builds the model segment.
func New(d Deps) *Model { return &Model{d: d} }

// Render returns the segment's text, or "" to hide it.
func (m *Model) Render() string {
	display := m.d.Session.DisplayName()
	if display == "" {
		return ""
	}
	if level := m.d.Session.EffortLevel(); level != "" {
		display += " [" + level + "]"
	}
	return display
}
