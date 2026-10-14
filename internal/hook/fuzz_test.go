package hook_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/hook"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var sgr = regexp.MustCompile("\x1b\\[[0-9;]*m")

func hasControl(s string) bool {
	for _, r := range s {
		if r < 0x20 || (r >= 0x7f && r <= 0x9f) {
			return true
		}
	}
	return false
}

func FuzzSanitize(f *testing.F) {
	for _, s := range []string{
		"plain text",
		"\x1b[32mgreen\x1b[0m",
		"\x1b[2Jcleared",
		"\x1b[1;38;5;33mbold\x1b[m",
		"a\x00b\x07c\rd\te",
		"\x9b31mC1 csi",
		"\x1b\x01[31mnested",
		"\x1b[31",
		"   ",
		"\x1b]0;title\x07x",
	} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		out := hook.Sanitize(s)
		assert.False(t, hasControl(sgr.ReplaceAllString(out, "")), "control character survived in %q", out)
		assert.Equal(t, out, hook.Sanitize(out), "sanitizing is not idempotent")
	})
}

func FuzzHeader(f *testing.F) {
	for _, s := range []string{
		"#!/bin/sh\n# statusline-slot: usage\n# statusline-ttl: 300\n# statusline-help: Shows the branch\n",
		"#!/bin/sh\n#statusline-slot:end\n",
		"#!/bin/sh\n# statusline-ttl: 30 seconds\n",
		"#!/bin/sh\n# statusline-ttl: 99999999999999999999\n",
		"#!/bin/sh\r\n# statusline-help: caf\xc3\xa9 \x1b[2J \x9b done\r\n",
		"#!/bin/sh\n# statusline-slot: bogus\n",
		"",
	} {
		f.Add(s)
	}

	dir := f.TempDir()
	path := filepath.Join(dir, "10-hook.sh")

	f.Fuzz(func(t *testing.T, script string) {
		require.NoError(t, os.WriteFile(path, []byte(script), 0o700))
		hooks := hook.Discover(dir)
		require.Len(t, hooks, 1)
		h := hooks[0]
		assert.Contains(t, []hook.Slot{hook.SlotCtx, hook.SlotUsage, hook.SlotEnd}, h.Slot)
		assert.GreaterOrEqual(t, int64(h.TTL), int64(-1), "ttl is only ever negative as the unset marker")
		assert.False(t, hasControl(h.Help), "help holds a control character: %q", h.Help)
		assert.False(t, strings.ContainsAny(h.Help, "\n"))
	})
}
