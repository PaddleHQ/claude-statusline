package setup

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tidwall/gjson"
)

const testBin = "/opt/cs/bin/claude-statusline"

var prettySettings = "{\n  \"theme\": \"dark\",\n  \"env\": {\n    \"B\": \"1\",\n    \"A\": \"2\"\n  },\n  \"model\": \"opus\",\n  \"alpha\": [1, 2]\n}\n"

func newDir(t *testing.T, settings *string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if settings != nil {
		if err := os.WriteFile(path, []byte(*settings), 0o640); err != nil {
			t.Fatal(err)
		}
	}
	return dir, path
}

func run(dir, event string) string {
	return apply(config{dir: dir, bin: testBin, event: event, version: "1.2.3"})
}

func sentinelExists(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "claude-statusline", ".bootstrapped"))
	return err == nil
}

func TestApply(t *testing.T) {
	t.Parallel()

	own := `{"statusLine":{"type":"command","command":"` + testBin + `"}}`
	other := `{"statusLine":{"type":"command","command":"/usr/bin/mine"}}`
	invalid := "not json"
	array := "[1]"

	tests := []struct {
		name         string
		settings     *string
		sentinel     bool
		event        string
		wantMsg      string
		wantSettings string
		wantSentinel bool
	}{
		{"leaves settings alone when already wired and current", &prettySettings, true, EventCurrent, "", prettySettings, true},
		{"announces an update when already wired", &prettySettings, true, EventUpdated, "updated to v1.2.3", prettySettings, true},
		{"stays silent when already wired and installed", &prettySettings, true, EventInstalled, "", prettySettings, true},
		{"leaves invalid JSON alone and says so on install", &invalid, false, EventInstalled, "not valid JSON", invalid, false},
		{"leaves invalid JSON alone silently on later events", &invalid, false, EventCurrent, "", invalid, false},
		{"leaves a non-object alone", &array, false, EventInstalled, "not valid JSON", array, false},
		{"keeps our own statusLine and says installed", &own, false, EventInstalled, "v1.2.3 installed.", own, true},
		{"keeps our own statusLine silently on current", &own, false, EventCurrent, "", own, true},
		{"leaves someone else's statusLine and explains the switch", &other, false, EventCurrent, "already have a statusLine", other, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir, path := newDir(t, tt.settings)
			if tt.sentinel {
				if err := touch(filepath.Join(dir, "claude-statusline", ".bootstrapped")); err != nil {
					t.Fatal(err)
				}
			}

			msg := run(dir, tt.event)

			if tt.wantMsg == "" && msg != "" || !strings.Contains(msg, tt.wantMsg) {
				t.Errorf("message = %q, want containing %q", msg, tt.wantMsg)
			}
			got, _ := os.ReadFile(path)
			if string(got) != tt.wantSettings {
				t.Errorf("settings = %q, want %q", got, tt.wantSettings)
			}
			if sentinelExists(dir) != tt.wantSentinel {
				t.Errorf("sentinel exists = %v, want %v", sentinelExists(dir), tt.wantSentinel)
			}
			if _, err := os.Stat(path + ".bak"); err == nil {
				t.Error("unexpected backup")
			}
		})
	}
}

func TestInsertStatusLine(t *testing.T) {
	t.Parallel()

	block := func(indent, nl string) string {
		return `"statusLine": {` + nl + indent + indent + `"type": "command",` + nl +
			indent + indent + `"command": "` + testBin + `",` + nl +
			indent + indent + `"refreshInterval": 60` + nl + indent + "}"
	}
	compact := `"statusLine":{"type":"command","command":"` + testBin + `","refreshInterval":60}`

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			"two-space file",
			"{\n  \"theme\": \"dark\",\n  \"model\": \"opus\"\n}\n",
			"{\n  \"theme\": \"dark\",\n  \"model\": \"opus\",\n  " + block("  ", "\n") + "\n}\n",
		},
		{
			"four-space file",
			"{\n    \"a\": 1\n}\n",
			"{\n    \"a\": 1,\n    " + block("    ", "\n") + "\n}\n",
		},
		{
			"tab-indented file",
			"{\n\t\"a\": 1,\n\t\"b\": 2\n}\n",
			"{\n\t\"a\": 1,\n\t\"b\": 2,\n\t" + block("\t", "\n") + "\n}\n",
		},
		{
			"compact one-line file",
			`{"a":1,"b":"x"}` + "\n",
			`{"a":1,"b":"x",` + compact + "}\n",
		},
		{
			"empty object",
			"{}",
			"{\n  " + block("  ", "\n") + "\n}",
		},
		{
			"empty object with a newline",
			"{\n}\n",
			"{\n  " + block("  ", "\n") + "\n}\n",
		},
		{
			"empty object with a space",
			"{ }\n",
			"{\n  " + block("  ", "\n") + "\n}\n",
		},
		{
			"file without a trailing newline",
			"{\n  \"a\": 1\n}",
			"{\n  \"a\": 1,\n  " + block("  ", "\n") + "\n}",
		},
		{
			"last member is a nested object",
			"{\n  \"env\": {\n    \"A\": \"1\"\n  }\n}\n",
			"{\n  \"env\": {\n    \"A\": \"1\"\n  },\n  " + block("  ", "\n") + "\n}\n",
		},
		{
			"last member is an array",
			"{\n  \"a\": [1, 2]\n}\n",
			"{\n  \"a\": [1, 2],\n  " + block("  ", "\n") + "\n}\n",
		},
		{
			"closing braces inside strings",
			"{\n  \"a\": \"}\",\n  \"b\": \"x}\"\n}\n",
			"{\n  \"a\": \"}\",\n  \"b\": \"x}\",\n  " + block("  ", "\n") + "\n}\n",
		},
		{
			"CRLF line endings",
			"{\r\n  \"a\": 1\r\n}\r\n",
			"{\r\n  \"a\": 1,\r\n  " + block("  ", "\r\n") + "\r\n}\r\n",
		},
	}

	for _, tt := range tests {
		t.Run("inserts into a "+tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := insertStatusLine([]byte(tt.in), []byte(`{"type":"command","command":"`+testBin+`","refreshInterval":60}`))

			if !ok {
				t.Fatal("insertion failed")
			}
			if string(got) != tt.want {
				t.Errorf("got:\n%q\nwant:\n%q", got, tt.want)
			}
			if !gjson.ValidBytes(got) {
				t.Errorf("result is not valid JSON: %s", got)
			}
		})
	}
}

func TestApplyWiring(t *testing.T) {
	t.Parallel()

	t.Run("writes the inserted entry, a backup of the original and the sentinel", func(t *testing.T) {
		t.Parallel()
		dir, path := newDir(t, &prettySettings)

		msg := run(dir, EventInstalled)

		if !strings.Contains(msg, "installed and enabled") {
			t.Errorf("message = %q", msg)
		}
		got, _ := os.ReadFile(path)
		if gjson.GetBytes(got, "statusLine.command").String() != testBin || gjson.GetBytes(got, "statusLine.refreshInterval").Int() != 60 {
			t.Errorf("statusLine not set: %s", got)
		}
		bak, _ := os.ReadFile(path + ".bak")
		if string(bak) != prettySettings {
			t.Errorf("backup = %q, want original", bak)
		}
		if !sentinelExists(dir) {
			t.Error("no sentinel")
		}
	})
}

func TestApplyFiles(t *testing.T) {
	t.Parallel()

	t.Run("preserves the file mode", func(t *testing.T) {
		t.Parallel()
		dir, path := newDir(t, &prettySettings)

		run(dir, EventInstalled)

		fi, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if fi.Mode().Perm() != 0o640 {
			t.Errorf("mode = %v, want 0640", fi.Mode().Perm())
		}
	})

	t.Run("creates a missing settings file with mode 0600", func(t *testing.T) {
		t.Parallel()
		dir, path := newDir(t, nil)

		msg := run(dir, EventInstalled)

		if !strings.Contains(msg, "installed and enabled") {
			t.Errorf("message = %q", msg)
		}
		fi, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if fi.Mode().Perm() != 0o600 {
			t.Errorf("mode = %v, want 0600", fi.Mode().Perm())
		}
		got, _ := os.ReadFile(path)
		if !strings.Contains(string(got), testBin) {
			t.Errorf("settings = %s", got)
		}
	})

	t.Run("does nothing on a second run", func(t *testing.T) {
		t.Parallel()
		dir, path := newDir(t, &prettySettings)
		run(dir, EventInstalled)
		first, _ := os.ReadFile(path)

		msg := run(dir, EventCurrent)

		second, _ := os.ReadFile(path)
		if msg != "" || !bytes.Equal(first, second) {
			t.Errorf("second run changed things: msg %q", msg)
		}
	})

	t.Run("leaves no temp files behind", func(t *testing.T) {
		t.Parallel()
		dir, _ := newDir(t, &prettySettings)
		run(dir, EventInstalled)

		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), ".settings-") {
				t.Errorf("left %s behind", e.Name())
			}
		}
	})
}

func TestRun(t *testing.T) {
	t.Run("prints a single JSON line using the config dir from the environment", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("CLAUDE_CONFIG_DIR", dir)

		var out bytes.Buffer
		Run(&out, EventInstalled, "1.2.3")

		var msg struct {
			SystemMessage string `json:"systemMessage"`
		}
		if err := json.Unmarshal(out.Bytes(), &msg); err != nil {
			t.Fatalf("output %q: %v", out.String(), err)
		}
		if !strings.Contains(msg.SystemMessage, "installed and enabled") || strings.Count(out.String(), "\n") != 1 {
			t.Errorf("output = %q", out.String())
		}
	})

	t.Run("prints nothing when there is nothing to say", func(t *testing.T) {
		t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
		var out bytes.Buffer
		Run(&out, EventInstalled, "1.2.3")
		out.Reset()
		Run(&out, EventCurrent, "1.2.3")
		if out.Len() != 0 {
			t.Errorf("output = %q", out.String())
		}
	})
}
