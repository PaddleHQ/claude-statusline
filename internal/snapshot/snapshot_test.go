package snapshot_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/snapshot"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrite(t *testing.T) {
	t.Parallel()

	t.Run("replaces an existing file atomically with the new contents", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join(t.TempDir(), "latest.json")
		require.NoError(t, os.WriteFile(path, []byte(`{"old":true}`), 0o644))

		require.NoError(t, snapshot.New(path).Write([]byte(`{"new":true}`)))

		data, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, `{"new":true}`, string(data))
		fi, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), fi.Mode().Perm())
		entries, err := os.ReadDir(filepath.Dir(path))
		require.NoError(t, err)
		assert.Len(t, entries, 1)
	})

	t.Run("creates the directory", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join(t.TempDir(), "a", "b", "latest.json")

		require.NoError(t, snapshot.New(path).Write([]byte(`{}`)))

		assert.FileExists(t, path)
	})

	t.Run("skips invalid JSON and an empty path", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, "latest.json")

		require.NoError(t, snapshot.New(path).Write([]byte(`{"a":`)))
		require.NoError(t, snapshot.New("").Write([]byte(`{}`)))

		assert.NoFileExists(t, path)
	})

	t.Run("reports an unwritable location", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		blocker := filepath.Join(dir, "file")
		require.NoError(t, os.WriteFile(blocker, nil, 0o600))

		assert.Error(t, snapshot.New(filepath.Join(blocker, "latest.json")).Write([]byte(`{}`)))
	})
}
