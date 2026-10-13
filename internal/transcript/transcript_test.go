package transcript_test

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/PaddleHQ/claude-statusline/internal/transcript"
)

const (
	user      = `{"type":"user","message":{"role":"user","content":"hi"}}`
	assistant = `{"type":"assistant","message":{"model":"claude-opus-4-7","usage":{"input_tokens":6,"cache_creation_input_tokens":20000,"cache_read_input_tokens":44500}}}`
	mem       = `{"type":"attachment","attachment":{"type":"hook_success","hookEvent":"SessionStart","content":"<persisted-output> too big"}}`
	good      = `{"type":"attachment","attachment":{"type":"skill_listing","isInitial":true,"content":"- a: x\n- b: y\n"}}`
	bare      = `{"type":"attachment","attachment":{"type":"skill_listing","isInitial":true,"content":"- a\n- b: y\n- c\nnot an entry\n"}}`
)

func write(t *testing.T, path string, lines ...string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600))
}

func TestReadUnreadable(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	assert.Equal(t, transcript.Info{}, transcript.Read(""))
	assert.Equal(t, transcript.Info{}, transcript.Read(filepath.Join(dir, "missing.jsonl")))
	assert.Equal(t, transcript.Info{}, transcript.Read(dir), "a directory is not a transcript")
}

func TestReadBoot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		lines []string
		want  int64
	}{
		{"first assistant line", []string{user, assistant}, 64506},
		{"later assistant lines are ignored", []string{user, assistant, strings.Replace(assistant, "20000", "1", 1)}, 64506},
		{"first assistant line without usage hides it", []string{user, `{"type":"assistant","message":{}}`, assistant}, 0},
		{"missing counts are zero", []string{`{"type":"assistant","message":{"usage":{"input_tokens":7}}}`}, 7},
		{"no assistant line", []string{user}, 0},
		{"malformed assistant line", []string{`{"type":"assistant"`, assistant}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "t.jsonl")
			write(t, path, tt.lines...)
			assert.Equal(t, tt.want, transcript.Read(path).Boot)
		})
	}
}

func TestReadSkills(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "t.jsonl")
	write(t, path, user, bare)
	got := transcript.Read(path)
	assert.Equal(t, len("- a\n- b: y\n- c\nnot an entry\n"), got.SkillsChars)
	assert.Equal(t, 2, got.SkillsBare)
	assert.Equal(t, 3, got.SkillsTotal)

	notInitial := strings.Replace(bare, `"isInitial":true`, `"isInitial":false`, 1)
	write(t, path, notInitial)
	got = transcript.Read(path)
	assert.Positive(t, got.SkillsChars, "the length counts any listing")
	assert.Zero(t, got.SkillsBare, "only the initial listing is checked")

	write(t, path, good)
	assert.Zero(t, transcript.Read(path).SkillsBare)
}

func TestReadOnlyScansOpeningLines(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "t.jsonl")
	lines := make([]string, 0, 50)
	for range 40 {
		lines = append(lines, user)
	}
	lines = append(lines, bare, mem)
	write(t, path, lines...)

	got := transcript.Read(path)
	assert.Zero(t, got.SkillsChars, "line 41 is out of range")
	assert.False(t, got.MemDropped)

	lines = append([]string{}, lines[2:]...)
	write(t, path, lines...)
	got = transcript.Read(path)
	assert.Positive(t, got.SkillsChars, "line 39 and 40 are in range")
	assert.True(t, got.MemDropped)
}

func TestReadMem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		line string
		want bool
	}{
		{"persisted output", mem, true},
		{"small output", strings.Replace(mem, "<persisted-output> too big", "ok", 1), false},
		{"another hook event", strings.Replace(mem, "SessionStart", "PreToolUse", 1), false},
		{"another attachment type", strings.Replace(mem, "hook_success", "hook_failure", 1), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "t.jsonl")
			write(t, path, user, tt.line)
			assert.Equal(t, tt.want, transcript.Read(path).MemDropped)
		})
	}
}

func TestReadActualModel(t *testing.T) {
	t.Parallel()

	haiku := strings.Replace(assistant, "opus-4-7", "haiku-4-5", 1)
	tests := []struct {
		name  string
		lines []string
		want  string
	}{
		{"latest wins", []string{assistant, haiku}, "claude-haiku-4-5"},
		{"none", []string{user}, ""},
		{"must be the first key of the message", []string{`{"message":{"role":"assistant","model":"claude-haiku-4-5"}}`}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "t.jsonl")
			write(t, path, tt.lines...)
			assert.Equal(t, tt.want, transcript.Read(path).ActualModel)
		})
	}

	t.Run("only the last 200 lines", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join(t.TempDir(), "t.jsonl")
		lines := []string{haiku}
		for range 200 {
			lines = append(lines, user)
		}
		write(t, path, lines...)
		assert.Empty(t, transcript.Read(path).ActualModel)
		write(t, path, append([]string{haiku}, lines[2:]...)...)
		assert.Equal(t, "claude-haiku-4-5", transcript.Read(path).ActualModel)
	})
}

func TestReadSkipsMalformedLines(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "t.jsonl")
	write(t, path, "not json", `{"type":`, user, "{broken", bare, mem, "")
	got := transcript.Read(path)
	assert.Equal(t, 2, got.SkillsBare)
	assert.True(t, got.MemDropped)
}

func TestReadSubagents(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	write(t, path, user)
	sub := filepath.Join(dir, "session", "subagents")
	write(t, filepath.Join(sub, "a.jsonl"), user, bare)
	write(t, filepath.Join(sub, "b.jsonl"), user, strings.Replace(bare, `- b: y`, `- b`, 1))
	write(t, filepath.Join(sub, "c.jsonl"), user, good)
	write(t, filepath.Join(sub, "notes.txt"), "- x")

	got := transcript.Read(path)
	assert.Equal(t, 3, got.SubBare, "the worst subagent")
	assert.Equal(t, 3, got.SubTotal)
	assert.Zero(t, got.SkillsBare, "the main listing is separate")
}

func TestReadHugeTranscriptStaysBounded(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "huge.jsonl")
	f, err := os.Create(path)
	require.NoError(t, err)
	filler := `{"type":"user","message":{"role":"user","content":"` + strings.Repeat("x", 1000) + `"}}` + "\n"
	_, err = f.WriteString(user + "\n" + good + "\n" + assistant + "\n")
	require.NoError(t, err)
	for range 40000 {
		_, err = f.WriteString(filler)
		require.NoError(t, err)
	}
	_, err = f.WriteString(strings.Replace(assistant, "opus-4-7", "haiku-4-5", 1) + "\n")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Greater(t, info.Size(), int64(40<<20), "a transcript far larger than any read bound")

	got := transcript.Read(path)
	assert.Equal(t, int64(64506), got.Boot)
	assert.Equal(t, "claude-haiku-4-5", got.ActualModel)
}

func TestReadDropsALineCutByTheTailBound(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "t.jsonl")
	huge := `{"message":{"model":"claude-haiku-4-5","x":"` + strings.Repeat("y", 5<<20) + `"}}`
	write(t, path, user, huge, user)
	assert.Empty(t, transcript.Read(path).ActualModel, "a line larger than the bound is not read")
}

func filler(n int) string {
	return `{"type":"user","message":{"role":"user","content":"` + strings.Repeat("x", n) + `"}}`
}

func TestReadCostsLittleIO(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	f, err := os.Create(path)
	require.NoError(t, err)
	_, err = f.WriteString(user + "\n" + bare + "\n" + mem + "\n" + assistant + "\n")
	require.NoError(t, err)
	body := filler(1000) + "\n"
	for range 40000 {
		_, err = f.WriteString(body)
		require.NoError(t, err)
	}
	_, err = f.WriteString(strings.Replace(assistant, "opus-4-7", "haiku-4-5", 1) + "\n")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	sub := filepath.Join(dir, "session", "subagents")
	for i := range 30 {
		lines := []string{user, bare}
		for range 50000 {
			lines = append(lines, filler(100))
		}
		write(t, filepath.Join(sub, "agent-"+strconv.Itoa(i)+".jsonl"), lines...)
	}

	got, read := transcript.Counted(path)
	assert.Equal(t, int64(64506), got.Boot)
	assert.Equal(t, "claude-haiku-4-5", got.ActualModel)
	assert.Equal(t, 2, got.SkillsBare)
	assert.True(t, got.MemDropped)
	assert.Equal(t, 2, got.SubBare)
	assert.Less(t, read, int64(512<<10), "a 40 MB transcript with 30 subagents of 7 MB each costs only the lines that are used")
}

func TestReadSkipsAnOverLongOpeningLine(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "t.jsonl")
	write(t, path, user, filler(1<<20), mem, assistant)
	got, read := transcript.Counted(path)
	assert.True(t, got.MemDropped, "reading carries on past the long line")
	assert.Equal(t, int64(64506), got.Boot)
	assert.Less(t, read, int64(2<<20))
}

func TestReadModelAcrossChunks(t *testing.T) {
	t.Parallel()

	haiku := strings.Replace(assistant, "opus-4-7", "haiku-4-5", 1)
	path := filepath.Join(t.TempDir(), "t.jsonl")

	lines := []string{haiku}
	for range 149 {
		lines = append(lines, filler(3000))
	}
	write(t, path, lines...)
	got, read := transcript.Counted(path)
	assert.Equal(t, "claude-haiku-4-5", got.ActualModel, "found 149 lines from the end, several chunks back")
	assert.Less(t, read, int64(1<<20))

	lines = []string{haiku}
	for range 200 {
		lines = append(lines, filler(3000))
	}
	write(t, path, lines...)
	assert.Empty(t, transcript.Read(path).ActualModel, "the 201st line from the end is out of range")

	lines = []string{assistant}
	for range 5 {
		lines = append(lines, filler(100000))
	}
	write(t, path, lines...)
	assert.Equal(t, "claude-opus-4-7", transcript.Read(path).ActualModel, "lines longer than a chunk")
}
