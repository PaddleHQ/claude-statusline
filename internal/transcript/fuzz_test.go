package transcript_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PaddleHQ/claude-statusline/internal/transcript"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	readBound = 2*(4<<20) + (64 << 10)
	readLimit = 5 * time.Second
)

func FuzzRead(f *testing.F) {
	f.Add([]byte(strings.Join([]string{user, assistant, mem, good, bare}, "\n") + "\n"))
	f.Add([]byte(user + "\n" + assistant))
	f.Add([]byte(`{"type":"assistant"`))
	f.Add([]byte(`{"type":"assistant","message":{"usage":{"input_tokens":-7}}}` + "\n"))
	f.Add([]byte("\n\n\n"))
	f.Add([]byte{})
	f.Add([]byte(strings.Repeat("x", 300<<10) + "\n" + assistant + "\n"))

	path := filepath.Join(f.TempDir(), "t.jsonl")

	f.Fuzz(func(t *testing.T, data []byte) {
		require.NoError(t, os.WriteFile(path, data, 0o600))
		start := time.Now()
		info, total := transcript.Counted(path)
		assert.Less(t, time.Since(start), readLimit, "read did not terminate promptly")
		assert.LessOrEqual(t, total, int64(readBound))
		assert.LessOrEqual(t, total, 2*int64(len(data)), "read more than the file twice over")
		assert.GreaterOrEqual(t, info.Boot, int64(0))
		assert.GreaterOrEqual(t, info.SkillsChars, 0)
		assert.GreaterOrEqual(t, info.SkillsBare, 0)
		assert.LessOrEqual(t, info.SkillsBare, info.SkillsTotal)
		assert.GreaterOrEqual(t, info.SubBare, 0)
		assert.LessOrEqual(t, info.SubBare, info.SubTotal)
	})
}
