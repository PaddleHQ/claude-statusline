// Package snapshot saves the latest session payload for other tools to read.
package snapshot

import (
	"os"
	"path/filepath"

	"github.com/tidwall/gjson"
)

// FileName is the snapshot's name inside the Claude config directory.
const FileName = "usage-latest.json"

// Write replaces path with data, atomically and readable only by the user. Input that is not valid JSON is not written.
func Write(path string, data []byte) error {
	if path == "" || !gjson.ValidBytes(data) {
		return nil
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".usage-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(f.Name()) }()
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
