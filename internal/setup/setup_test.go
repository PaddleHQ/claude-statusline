package setup_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"

	"github.com/PaddleHQ/claude-statusline/internal/setup"
)

const testBin = "/opt/cs/bin/claude-statusline"

var prettySettings = "{\n  \"theme\": \"dark\",\n  \"env\": {\n    \"B\": \"1\",\n    \"A\": \"2\"\n  },\n  \"model\": \"opus\",\n  \"alpha\": [1, 2]\n}\n"

func newDir(t *testing.T, settings *string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if settings != nil {
		require.NoError(t, os.WriteFile(path, []byte(*settings), 0o640))
	}
	return dir, path
}

func run(dir, event string) string {
	return setup.Apply(dir, testBin, event, "1.2.3")
}

func sentinelExists(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "claude-statusline", ".bootstrapped"))
	return err == nil
}

func TestApply(t *testing.T) {
	t.Parallel()

	own := `{"statusLine":{"type":"command","command":"` + testBin + `"},"subagentStatusLine":{"type":"command","command":"` + testBin + ` subagent"}}`
	other := `{"statusLine":{"type":"command","command":"/usr/bin/mine"},"subagentStatusLine":{"type":"command","command":"/usr/bin/mine"}}`
	otherSub := `{"statusLine":{"type":"command","command":"` + testBin + `"},"subagentStatusLine":{"type":"command","command":"/usr/bin/mine"}}`
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
		{"leaves settings alone when already wired and current", &prettySettings, true, setup.EventCurrent, "", prettySettings, true},
		{"announces an update when already wired", &prettySettings, true, setup.EventUpdated, "updated to v1.2.3", prettySettings, true},
		{"stays silent when already wired and installed", &prettySettings, true, setup.EventInstalled, "", prettySettings, true},
		{"leaves invalid JSON alone and says so on install", &invalid, false, setup.EventInstalled, "not valid JSON", invalid, false},
		{"leaves invalid JSON alone silently on later events", &invalid, false, setup.EventCurrent, "", invalid, false},
		{"leaves a non-object alone", &array, false, setup.EventInstalled, "not valid JSON", array, false},
		{"keeps our own statusLine and says installed", &own, false, setup.EventInstalled, "v1.2.3 installed.", own, true},
		{"keeps our own statusLine silently on current", &own, false, setup.EventCurrent, "", own, true},
		{"leaves someone else's statusLine and explains the switch", &other, false, setup.EventCurrent, "already have a statusLine", other, true},
		{"leaves someone else's subagentStatusLine and explains the switch", &otherSub, false, setup.EventInstalled, "already have a subagentStatusLine", otherSub, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir, path := newDir(t, tt.settings)
			if tt.sentinel {
				require.NoError(t, setup.Touch(filepath.Join(dir, "claude-statusline", ".bootstrapped")))
				require.NoError(t, setup.Touch(filepath.Join(dir, "claude-statusline", ".bootstrapped-subagent")))
			}

			msg := run(dir, tt.event)

			if tt.wantMsg == "" {
				assert.Empty(t, msg)
			} else {
				assert.Contains(t, msg, tt.wantMsg)
			}
			got, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Equal(t, tt.wantSettings, string(got))
			assert.Equal(t, tt.wantSentinel, sentinelExists(dir))
			assert.NoFileExists(t, path+".bak")
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

			got, ok := setup.InsertMember([]byte(tt.in), "statusLine", []byte(`{"type":"command","command":"`+testBin+`","refreshInterval":60}`))

			require.True(t, ok, "insertion failed")
			assert.Equal(t, tt.want, string(got))
			assert.True(t, gjson.ValidBytes(got), "result is not valid JSON: %s", got)
		})
	}
}

func TestApplyWiring(t *testing.T) {
	t.Parallel()

	t.Run("writes the inserted entry, a backup of the original and the sentinel", func(t *testing.T) {
		t.Parallel()
		dir, path := newDir(t, &prettySettings)

		msg := run(dir, setup.EventInstalled)

		assert.Contains(t, msg, "installed and enabled")
		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, testBin, gjson.GetBytes(got, "statusLine.command").String())
		assert.EqualValues(t, 60, gjson.GetBytes(got, "statusLine.refreshInterval").Int())
		bak, err := os.ReadFile(path + ".bak")
		require.NoError(t, err)
		assert.Equal(t, prettySettings, string(bak))
		assert.True(t, sentinelExists(dir), "no sentinel")
	})
}

func TestApplySubagent(t *testing.T) {
	t.Parallel()

	subLine := `"subagentStatusLine": {` + "\n" + `    "type": "command",` + "\n" + `    "command": "` + testBin + ` subagent"` + "\n" + `  }`
	ownStatus := "{\n  \"statusLine\": {\"type\": \"command\", \"command\": \"" + testBin + "\"}\n}\n"
	mineStatus := "{\n  \"statusLine\": {\"type\": \"command\", \"command\": \"/usr/bin/mine\"}\n}\n"
	mineSub := "{\n  \"subagentStatusLine\": {\"type\": \"command\", \"command\": \"/usr/bin/mine\"}\n}\n"

	tests := []struct {
		name         string
		settings     string
		sentinel     bool
		event        string
		wantMsg      string
		wantStatus   string
		wantSub      string
		wantBackup   bool
		wantSentinel bool
	}{
		{"both absent wires both", prettySettings, false, setup.EventInstalled, "also styles the subagent rows", testBin, testBin + " subagent", true, true},
		{"our statusLine from an older install gains the subagent line once", ownStatus, true, setup.EventUpdated, "claude-statusline now also styles the subagent rows", testBin, testBin + " subagent", true, true},
		{"a user's own subagentStatusLine is kept", mineSub, false, setup.EventInstalled, "already have a subagentStatusLine", testBin, "/usr/bin/mine", true, true},
		{"a user's statusLine does not stop the subagent line", mineStatus, true, setup.EventCurrent, "claude-statusline now also styles the subagent rows", "/usr/bin/mine", testBin + " subagent", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir, path := newDir(t, &tt.settings)
			if tt.sentinel {
				require.NoError(t, setup.Touch(filepath.Join(dir, "claude-statusline", ".bootstrapped")))
			}

			msg := run(dir, tt.event)

			assert.Contains(t, msg, tt.wantMsg)
			assert.Equal(t, 0, strings.Count(msg, "\n"), "message must be one line")
			got, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, gjson.GetBytes(got, "statusLine.command").String())
			assert.Equal(t, tt.wantSub, gjson.GetBytes(got, "subagentStatusLine.command").String())
			assert.True(t, gjson.ValidBytes(got))
			if tt.wantBackup {
				bak, err := os.ReadFile(path + ".bak")
				require.NoError(t, err)
				assert.Equal(t, tt.settings, string(bak))
			}
			assert.True(t, sentinelExists(dir))
			assert.FileExists(t, filepath.Join(dir, "claude-statusline", ".bootstrapped-subagent"))
		})
	}

	t.Run("adds the subagent line with the file's indentation", func(t *testing.T) {
		t.Parallel()
		dir, path := newDir(t, &ownStatus)
		require.NoError(t, setup.Touch(filepath.Join(dir, "claude-statusline", ".bootstrapped")))

		run(dir, setup.EventCurrent)

		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSuffix(ownStatus, "\n}\n")+",\n  "+subLine+"\n}\n", string(got))
	})

	t.Run("is idempotent and writes a single backup of the original", func(t *testing.T) {
		t.Parallel()
		dir, path := newDir(t, &prettySettings)

		first := run(dir, setup.EventInstalled)
		after, err := os.ReadFile(path)
		require.NoError(t, err)
		again := run(dir, setup.EventCurrent)
		afterAgain, err := os.ReadFile(path)
		require.NoError(t, err)

		assert.Contains(t, first, "installed and enabled")
		assert.Empty(t, again)
		assert.Equal(t, string(after), string(afterAgain))
		bak, err := os.ReadFile(path + ".bak")
		require.NoError(t, err)
		assert.Equal(t, prettySettings, string(bak))
		assert.Equal(t, testBin, gjson.GetBytes(after, "statusLine.command").String())
		assert.Equal(t, testBin+" subagent", gjson.GetBytes(after, "subagentStatusLine.command").String())
	})

	t.Run("quotes a binary path that contains a space", func(t *testing.T) {
		t.Parallel()
		dir, path := newDir(t, &prettySettings)

		setup.Apply(dir, "/opt/my tools/claude-statusline", setup.EventInstalled, "1.2.3")

		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, "'/opt/my tools/claude-statusline' subagent", gjson.GetBytes(got, "subagentStatusLine.command").String())
	})
}

func TestApplyFiles(t *testing.T) {
	t.Parallel()

	t.Run("preserves the file mode", func(t *testing.T) {
		t.Parallel()
		dir, path := newDir(t, &prettySettings)

		run(dir, setup.EventInstalled)

		fi, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o640), fi.Mode().Perm())
	})

	t.Run("creates a missing settings file with mode 0600", func(t *testing.T) {
		t.Parallel()
		dir, path := newDir(t, nil)

		msg := run(dir, setup.EventInstalled)

		assert.Contains(t, msg, "installed and enabled")
		fi, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), fi.Mode().Perm())
		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Contains(t, string(got), testBin)
	})

	t.Run("does nothing on a second run", func(t *testing.T) {
		t.Parallel()
		dir, path := newDir(t, &prettySettings)
		run(dir, setup.EventInstalled)
		first, err := os.ReadFile(path)
		require.NoError(t, err)

		msg := run(dir, setup.EventCurrent)

		second, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.True(t, bytes.Equal(first, second), "second run changed the settings")
	})

	t.Run("leaves no temp files behind", func(t *testing.T) {
		t.Parallel()
		dir, _ := newDir(t, &prettySettings)
		run(dir, setup.EventInstalled)

		entries, err := os.ReadDir(dir)
		require.NoError(t, err)
		for _, e := range entries {
			assert.False(t, strings.HasPrefix(e.Name(), ".settings-"), "left %s behind", e.Name())
		}
	})
}

func TestRun(t *testing.T) {
	t.Run("prints a single JSON line using the config dir from the environment", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("CLAUDE_CONFIG_DIR", dir)

		var out bytes.Buffer
		setup.Run(&out, setup.EventInstalled, "1.2.3")

		var msg struct {
			SystemMessage string `json:"systemMessage"`
		}
		require.NoError(t, json.Unmarshal(out.Bytes(), &msg), "output %q", out.String())
		assert.Contains(t, msg.SystemMessage, "installed and enabled")
		assert.Equal(t, 1, strings.Count(out.String(), "\n"), "output = %q", out.String())
	})

	t.Run("prints nothing when there is nothing to say", func(t *testing.T) {
		t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
		var out bytes.Buffer
		setup.Run(&out, setup.EventInstalled, "1.2.3")
		out.Reset()
		setup.Run(&out, setup.EventCurrent, "1.2.3")
		assert.Zero(t, out.Len(), "output = %q", out.String())
	})
}
