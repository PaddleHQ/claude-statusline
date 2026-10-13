// Package setup wires the status line into Claude Code's settings.json once.
package setup

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// Events describe what the bootstrap hook just did to the binary.
const (
	EventInstalled = "installed"
	EventUpdated   = "updated"
	EventCurrent   = "current"
)

type config struct {
	dir     string
	bin     string
	event   string
	version string
}

// Run never fails the session: it writes at most one systemMessage line and no errors.
func Run(w io.Writer, event, version string) {
	dir := os.Getenv("CLAUDE_CONFIG_DIR")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return
		}
		dir = filepath.Join(home, ".claude")
	}
	bin, err := os.Executable()
	if err != nil {
		return
	}
	if resolved, err := filepath.EvalSymlinks(bin); err == nil {
		bin = resolved
	}
	emit(w, apply(config{dir: dir, bin: bin, event: event, version: version}))
}

func emit(w io.Writer, msg string) {
	if msg == "" {
		return
	}
	b, err := json.Marshal(struct {
		SystemMessage string `json:"systemMessage"`
	}{msg})
	if err != nil {
		return
	}
	_, _ = w.Write(append(b, '\n'))
}

func apply(c config) string {
	installed := c.event == EventInstalled
	sentinel := filepath.Join(c.dir, "claude-statusline", ".bootstrapped")
	if _, err := os.Stat(sentinel); err == nil {
		if c.event == EventUpdated {
			return "claude-statusline updated to v" + c.version + "."
		}
		return ""
	}

	settings := filepath.Join(c.dir, "settings.json")
	data, err := os.ReadFile(settings)
	mode := os.FileMode(0o600)
	switch {
	case os.IsNotExist(err):
		data = []byte("{}\n")
		if err := writeAtomic(settings, data, mode); err != nil {
			return ""
		}
	case err != nil:
		return ""
	default:
		if fi, err := os.Stat(settings); err == nil {
			mode = fi.Mode().Perm()
		}
	}

	if !gjson.ValidBytes(data) || !gjson.ParseBytes(data).IsObject() {
		if installed {
			return "claude-statusline v" + c.version + " installed, but " + settings + " is not valid JSON so it was left alone. It will retry next session."
		}
		return ""
	}

	if sl := gjson.GetBytes(data, "statusLine"); sl.Exists() {
		if err := touch(sentinel); err != nil {
			return ""
		}
		if sl.Get("command").String() == c.bin {
			if installed {
				return "claude-statusline v" + c.version + " installed."
			}
			return ""
		}
		return "claude-statusline v" + c.version + " installed, but you already have a statusLine so it was left unchanged. To switch, set statusLine.command in " + settings + " to " + c.bin + "."
	}

	command, err := json.Marshal(c.bin)
	if err != nil {
		return ""
	}
	entry := `{"type":"command","command":` + string(command) + `,"refreshInterval":60}`
	out, err := sjson.SetRawBytes(data, "statusLine", []byte(entry))
	if err != nil {
		return ""
	}
	// sjson drops whitespace after the closing brace.
	out = append(out, data[len(bytes.TrimRight(data, " \t\r\n")):]...)
	if err := writeAtomic(settings+".bak", data, mode); err != nil {
		return ""
	}
	if err := writeAtomic(settings, out, mode); err != nil {
		return ""
	}
	if err := touch(sentinel); err != nil {
		return ""
	}
	return "claude-statusline v" + c.version + " installed and enabled. It appears after your first message. Your previous settings are backed up at " + settings + ".bak. To disable it, remove the statusLine block from settings.json."
}

func touch(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, nil, 0o644)
}

func writeAtomic(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".settings-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(f.Name()) }()
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Chmod(mode); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
