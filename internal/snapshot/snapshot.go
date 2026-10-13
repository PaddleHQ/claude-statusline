// Package snapshot saves the latest session payload for other tools to read.
package snapshot

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tidwall/gjson"
)

// FileName is the snapshot's name inside the Claude config directory.
const FileName = "usage-latest.json"

// Snapshot is the file the latest payload is saved to.
type Snapshot struct{ path string }

// New returns a Snapshot that writes to path; an empty path writes nothing.
func New(path string) *Snapshot { return &Snapshot{path: path} }

// Write replaces the file with data, atomically and readable only by the user. Input that is not valid JSON is not written.
func (s *Snapshot) Write(data []byte) error {
	path := s.path
	if path == "" || !gjson.ValidBytes(data) {
		return nil
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create snapshot directory: %w", err)
	}
	f, err := os.CreateTemp(dir, ".usage-*")
	if err != nil {
		return fmt.Errorf("create temporary snapshot: %w", err)
	}
	defer func() { _ = os.Remove(f.Name()) }()
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		return fmt.Errorf("set snapshot mode: %w", err)
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return fmt.Errorf("write temporary snapshot: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close temporary snapshot: %w", err)
	}
	if err := os.Rename(f.Name(), path); err != nil {
		return fmt.Errorf("replace snapshot: %w", err)
	}
	return nil
}
