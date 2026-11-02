package icons_test

import (
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/icons"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		nerd  bool
		ok    bool
	}{
		{"", false, true},
		{"none", false, true},
		{"nerd", true, true},
		{" NERD ", true, true},
		{"emoji", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			set, ok := icons.Parse(tt.input)
			assert.Equal(t, tt.nerd, set.Nerd())
			assert.Equal(t, tt.ok, ok)
		})
	}
}

func TestPick(t *testing.T) {
	t.Parallel()

	none, _ := icons.Parse("none")
	nerd, _ := icons.Parse("nerd")
	assert.Equal(t, "ctx: ", none.Pick("ctx: ", "X "))
	assert.Equal(t, "X ", nerd.Pick("ctx: ", "X "))
}

type fakeConfig struct{ icons string }

func (c *fakeConfig) String(key string) string {
	if key == icons.Key {
		return c.icons
	}
	return ""
}

func TestFromConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		nerd  bool
	}{
		{"nerd", "nerd", true},
		{"none", "none", false},
		{"unset", "", false},
		{"an unknown value is none", "emoji", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			set := icons.FromConfig(&fakeConfig{icons: tt.value})
			assert.Equal(t, tt.nerd, set.Nerd())
		})
	}
}
