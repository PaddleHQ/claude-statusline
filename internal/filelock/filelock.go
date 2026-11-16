// Package filelock serializes the sessions that share a state file, through an advisory lock file beside it.
package filelock

import (
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// wait bounds how long a caller waits for another session's lock before carrying on without it, so a stalled
// holder can never freeze every status line.
const (
	wait = time.Second
	poll = 5 * time.Millisecond
)

// Acquire takes an exclusive lock on the file at path, creating it and its directory when missing, and returns the
// function that releases it. When the lock cannot be taken within a second, it returns a function that does nothing.
func Acquire(path string) (release func()) {
	unlocked := func() {}
	if os.MkdirAll(filepath.Dir(path), 0o700) != nil {
		return unlocked
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600) //nolint:gosec // G304: the user's own config directory
	if err != nil {
		return unlocked
	}
	deadline := time.Now().Add(wait)
	for syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) != nil { //nolint:gosec // G115: a file descriptor fits in an int
		if time.Now().After(deadline) {
			_ = f.Close()
			return unlocked
		}
		time.Sleep(poll)
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN) //nolint:gosec // G115: a file descriptor fits in an int
		_ = f.Close()
	}
}
