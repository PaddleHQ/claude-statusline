// Package statusline turns the session payload into the printed line.
package statusline

import (
	"github.com/PaddleHQ/claude-statusline/internal/payload"
	"github.com/PaddleHQ/claude-statusline/segments/model"
)

// Build renders the line for the session JSON in input.
func Build(input []byte) string {
	p := payload.Parse(input)
	return model.New(model.Deps{Session: &p}).Render()
}
