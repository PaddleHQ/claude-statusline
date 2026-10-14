@transcript_diagnostics
Feature: Transcript diagnostics
  The boot, skills and mem segments, and the model family, are read from the session transcript.

  Background:
    Given the environment variable "STATUSLINE_NOW" is "1760000000"
    And the environment variable "COLUMNS" is "200"

  Scenario: The transcript diagnostics are hidden when the transcript file named in the payload does not exist
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"transcript_path":"@TMP@/transcript.jsonl"}
      """
    When the status line renders at 200 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7 [high] \e[2m·\e[0m ctx: \e[32m████▎\e[0m\e[2m░░░░░\e[0m 42% \e[2m·\e[0m ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%
      """

  Scenario: Skill listings in sub-agent transcripts are checked too, and the agent with the most stripped skills is flagged
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"transcript_path":"@TMP@/transcript.jsonl"}
      """
    And the file "transcript.jsonl":
      """
      {"type":"user","message":{"role":"user","content":"hi"}}
      {"type":"attachment","attachment":{"type":"skill_listing","isInitial":true,"content":"- alpha: does a thing\u000a- beta: another\u000a- gamma: third thing here\u000a"}}
      """
    And the file "transcript/subagents/agent-a.jsonl":
      """
      {"type":"user","message":{"role":"user","content":"hi"}}
      {"type":"attachment","attachment":{"type":"skill_listing","isInitial":true,"content":"- alpha: does a thing\u000a- beta\u000a- gamma\u000a- delta: has text\u000a- eps\u000a"}}
      """
    And the file "transcript/subagents/agent-b.jsonl":
      """
      {"type":"user","message":{"role":"user","content":"hi"}}
      {"type":"attachment","attachment":{"type":"skill_listing","isInitial":true,"content":"- a\u000a- b\u000a- c\u000a- d: x\u000a- e: y\u000a- f: z\u000a"}}
      """
    And the file "transcript/subagents/agent-c.jsonl":
      """
      {"type":"user","message":{"role":"user","content":"hi"}}
      {"type":"attachment","attachment":{"type":"skill_listing","isInitial":true,"content":"- alpha: does a thing\u000a- beta: another\u000a- gamma: third thing here\u000a"}}
      """
    When the status line renders at 200 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7 [high] \e[2m·\e[0m ctx: \e[32m████▎\e[0m\e[2m░░░░░\e[0m 42% \e[2m·\e[0m ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% \e[2m·\e[0m ⚙ 16 \e[2m·\e[0m \e[33m✗ agent skills 3/5\e[0m \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%
      """

  Scenario: Every transcript diagnostic at once on a 200 column terminal
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"transcript_path":"@TMP@/transcript.jsonl"}
      """
    And the file "transcript.jsonl":
      """
      {"type":"user","message":{"role":"user","content":"hi"}}
      {"type":"attachment","attachment":{"type":"skill_listing","isInitial":true,"content":"- alpha: does a thing\u000a- beta\u000a- gamma\u000a- delta: has text\u000a- eps\u000a"}}
      {"type":"attachment","attachment":{"type":"hook_success","hookEvent":"SessionStart","content":"<persisted-output>\u000aOutput too large"}}
      {"type":"assistant","message":{"model":"claude-opus-4-7","usage":{"input_tokens":6,"cache_creation_input_tokens":20000,"cache_read_input_tokens":44500,"output_tokens":5}}}
      {"type":"assistant","message":{"model":"claude-sonnet-5-5","usage":{"input_tokens":6,"cache_creation_input_tokens":20000,"cache_read_input_tokens":44500,"output_tokens":5}}}
      """
    And the file "transcript/subagents/agent-a.jsonl":
      """
      {"type":"user","message":{"role":"user","content":"hi"}}
      {"type":"attachment","attachment":{"type":"skill_listing","isInitial":true,"content":"- a\u000a- b\u000a- c\u000a- d: x\u000a- e: y\u000a- f: z\u000a"}}
      """
    When the status line renders at 200 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Sonnet (skill) \e[2m·\e[0m ctx: \e[32m████▎\e[0m\e[2m░░░░░\e[0m 42% \e[2m·\e[0m ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% \e[2m·\e[0m ⌂ 64.5k \e[2m·\e[0m ⚙ 15 \e[2m·\e[0m \e[31m✗ skills 3/5\e[0m \e[2m·\e[0m \e[33m✗ agent skills 3/6\e[0m \e[2m·\e[0m \e[31m✗ mem\e[0m \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%
      """

  Scenario: Every transcript diagnostic at 120 columns, where the line wraps into groups
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"transcript_path":"@TMP@/transcript.jsonl"}
      """
    And the file "transcript.jsonl":
      """
      {"type":"user","message":{"role":"user","content":"hi"}}
      {"type":"attachment","attachment":{"type":"skill_listing","isInitial":true,"content":"- alpha: does a thing\u000a- beta\u000a- gamma\u000a- delta: has text\u000a- eps\u000a"}}
      {"type":"attachment","attachment":{"type":"hook_success","hookEvent":"SessionStart","content":"<persisted-output>\u000aOutput too large"}}
      {"type":"assistant","message":{"model":"claude-opus-4-7","usage":{"input_tokens":6,"cache_creation_input_tokens":20000,"cache_read_input_tokens":44500,"output_tokens":5}}}
      {"type":"assistant","message":{"model":"claude-sonnet-5-5","usage":{"input_tokens":6,"cache_creation_input_tokens":20000,"cache_read_input_tokens":44500,"output_tokens":5}}}
      """
    And the file "transcript/subagents/agent-a.jsonl":
      """
      {"type":"user","message":{"role":"user","content":"hi"}}
      {"type":"attachment","attachment":{"type":"skill_listing","isInitial":true,"content":"- a\u000a- b\u000a- c\u000a- d: x\u000a- e: y\u000a- f: z\u000a"}}
      """
    When the status line renders at 120 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Sonnet (skill) \e[2m·\e[0m ctx: \e[32m███▍\e[0m\e[2m░░░░\e[0m 42%
      ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% \e[2m·\e[0m ⌂ 64.5k \e[2m·\e[0m ⚙ 15 \e[2m·\e[0m \e[31m✗ skills 3/5\e[0m \e[2m·\e[0m \e[33m✗ agent skills 3/6\e[0m \e[2m·\e[0m \e[31m✗ mem\e[0m
      \e[2m↳\e[0m          5h: \e[32m██████▎\e[0m\e[1m┃\e[0m\e[2m░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m█████▋\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░\e[0m 40%
      """

  Scenario: Every transcript diagnostic at 80 columns, where the line wraps into more groups
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"transcript_path":"@TMP@/transcript.jsonl"}
      """
    And the file "transcript.jsonl":
      """
      {"type":"user","message":{"role":"user","content":"hi"}}
      {"type":"attachment","attachment":{"type":"skill_listing","isInitial":true,"content":"- alpha: does a thing\u000a- beta\u000a- gamma\u000a- delta: has text\u000a- eps\u000a"}}
      {"type":"attachment","attachment":{"type":"hook_success","hookEvent":"SessionStart","content":"<persisted-output>\u000aOutput too large"}}
      {"type":"assistant","message":{"model":"claude-opus-4-7","usage":{"input_tokens":6,"cache_creation_input_tokens":20000,"cache_read_input_tokens":44500,"output_tokens":5}}}
      {"type":"assistant","message":{"model":"claude-sonnet-5-5","usage":{"input_tokens":6,"cache_creation_input_tokens":20000,"cache_read_input_tokens":44500,"output_tokens":5}}}
      """
    And the file "transcript/subagents/agent-a.jsonl":
      """
      {"type":"user","message":{"role":"user","content":"hi"}}
      {"type":"attachment","attachment":{"type":"skill_listing","isInitial":true,"content":"- a\u000a- b\u000a- c\u000a- d: x\u000a- e: y\u000a- f: z\u000a"}}
      """
    When the status line renders at 80 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Sonnet (skill) \e[2m·\e[0m ctx: \e[32m██▌\e[0m\e[2m░░░\e[0m 42%
      ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% \e[2m·\e[0m ⌂ 64.5k \e[2m·\e[0m ⚙ 15 \e[2m·\e[0m \e[31m✗ skills 3/5\e[0m \e[2m·\e[0m \e[33m✗ agent skills 3/6\e[0m \e[2m·\e[0m \e[31m✗ mem\e[0m
      \e[2m↳\e[0m          5h: \e[32m████▌\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 40%
      """

  Scenario: The boot segment shows the prompt tokens of the first assistant turn, and the turn's model family replaces the model name
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"transcript_path":"@TMP@/transcript.jsonl"}
      """
    And the file "transcript.jsonl":
      """
      {"type":"user","message":{"role":"user","content":"hi"}}
      {"type":"assistant","message":{"model":"claude-opus-4-7","usage":{"input_tokens":6,"cache_creation_input_tokens":20000,"cache_read_input_tokens":44500,"output_tokens":5}}}
      """
    When the status line renders at 200 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus (skill) \e[2m·\e[0m ctx: \e[32m████▎\e[0m\e[2m░░░░░\e[0m 42% \e[2m·\e[0m ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% \e[2m·\e[0m ⌂ 64.5k \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%
      """

  Scenario: The boot segment is hidden when the first assistant turn in the transcript has no token usage
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"transcript_path":"@TMP@/transcript.jsonl"}
      """
    And the file "transcript.jsonl":
      """
      {"type":"user","message":{"role":"user","content":"hi"}}
      {"type":"assistant","message":{"model":"claude-opus-4-7"}}
      {"type":"assistant","message":{"model":"claude-opus-4-7","usage":{"input_tokens":6,"cache_creation_input_tokens":20000,"cache_read_input_tokens":44500,"output_tokens":5}}}
      """
    When the status line renders at 200 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus (skill) \e[2m·\e[0m ctx: \e[32m████▎\e[0m\e[2m░░░░░\e[0m 42% \e[2m·\e[0m ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%
      """

  Scenario: Malformed lines in the transcript are skipped and the valid ones still count
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"transcript_path":"@TMP@/transcript.jsonl"}
      """
    And the file "transcript.jsonl":
      """
      not json
      {"type":"assistant"
      {"type":"user","message":{"role":"user","content":"hi"}}
      {"type":"assistant","message":{"model":"claude-opus-4-7","usage":{"input_tokens":6,"cache_creation_input_tokens":20000,"cache_read_input_tokens":44500,"output_tokens":5}}}
      {broken
      {"type":"attachment","attachment":{"type":"skill_listing","isInitial":true,"content":"- alpha: does a thing\u000a- beta\u000a- gamma\u000a- delta: has text\u000a- eps\u000a"}}
      {"type":"attachment","attachment":{"type":"hook_success","hookEvent":"SessionStart","content":"<persisted-output>\u000aOutput too large"}}
      """
    When the status line renders at 200 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus (skill) \e[2m·\e[0m ctx: \e[32m████▎\e[0m\e[2m░░░░░\e[0m 42% \e[2m·\e[0m ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% \e[2m·\e[0m ⚙ 15 \e[2m·\e[0m \e[31m✗ skills 3/5\e[0m \e[2m·\e[0m \e[31m✗ mem\e[0m \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%
      """

  Scenario: A SessionStart hook whose output was too large to inject and was replaced by a preview is flagged as mem
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"transcript_path":"@TMP@/transcript.jsonl"}
      """
    And the file "transcript.jsonl":
      """
      {"type":"user","message":{"role":"user","content":"hi"}}
      {"type":"attachment","attachment":{"type":"hook_success","hookEvent":"SessionStart","content":"<persisted-output>\u000aOutput too large"}}
      """
    When the status line renders at 200 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7 [high] \e[2m·\e[0m ctx: \e[32m████▎\e[0m\e[2m░░░░░\e[0m 42% \e[2m·\e[0m ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% \e[2m·\e[0m \e[31m✗ mem\e[0m \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%
      """

  Scenario: Small SessionStart output, and a persisted-output marker from another hook event, is not flagged as mem
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"transcript_path":"@TMP@/transcript.jsonl"}
      """
    And the file "transcript.jsonl":
      """
      {"type":"user","message":{"role":"user","content":"hi"}}
      {"type":"attachment","attachment":{"type":"hook_success","hookEvent":"SessionStart","content":"small output"}}
      {"type":"attachment","attachment":{"type":"hook_success","hookEvent":"PreToolUse","content":"<persisted-output> x"}}
      """
    When the status line renders at 200 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7 [high] \e[2m·\e[0m ctx: \e[32m████▎\e[0m\e[2m░░░░░\e[0m 42% \e[2m·\e[0m ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%
      """

  Scenario: A later assistant turn on a different model family replaces the model name, marked (skill)
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"transcript_path":"@TMP@/transcript.jsonl"}
      """
    And the file "transcript.jsonl":
      """
      {"type":"user","message":{"role":"user","content":"hi"}}
      {"type":"assistant","message":{"model":"claude-opus-4-7","usage":{"input_tokens":6,"cache_creation_input_tokens":20000,"cache_read_input_tokens":44500,"output_tokens":5}}}
      {"type":"assistant","message":{"model":"claude-haiku-4-5-20251001","usage":{"input_tokens":6,"cache_creation_input_tokens":20000,"cache_read_input_tokens":44500,"output_tokens":5}}}
      """
    When the status line renders at 200 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Haiku (skill) \e[2m·\e[0m ctx: \e[32m████▎\e[0m\e[2m░░░░░\e[0m 42% \e[2m·\e[0m ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% \e[2m·\e[0m ⌂ 64.5k \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%
      """

  Scenario: The model family shown is the one from the latest assistant turn in the transcript
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"transcript_path":"@TMP@/transcript.jsonl"}
      """
    And the file "transcript.jsonl":
      """
      {"type":"user","message":{"role":"user","content":"hi"}}
      {"type":"assistant","message":{"model":"claude-haiku-4-5","usage":{"input_tokens":6,"cache_creation_input_tokens":20000,"cache_read_input_tokens":44500,"output_tokens":5}}}
      {"type":"assistant","message":{"model":"claude-sonnet-5-5","usage":{"input_tokens":6,"cache_creation_input_tokens":20000,"cache_read_input_tokens":44500,"output_tokens":5}}}
      {"type":"assistant","message":{"model":"claude-opus-4-7","usage":{"input_tokens":6,"cache_creation_input_tokens":20000,"cache_read_input_tokens":44500,"output_tokens":5}}}
      """
    When the status line renders at 200 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus (skill) \e[2m·\e[0m ctx: \e[32m████▎\e[0m\e[2m░░░░░\e[0m 42% \e[2m·\e[0m ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% \e[2m·\e[0m ⌂ 64.5k \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%
      """

  Scenario: Repeated assistant turns on one model family show that family name, marked (skill)
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"transcript_path":"@TMP@/transcript.jsonl"}
      """
    And the file "transcript.jsonl":
      """
      {"type":"user","message":{"role":"user","content":"hi"}}
      {"type":"assistant","message":{"model":"claude-opus-4-7","usage":{"input_tokens":6,"cache_creation_input_tokens":20000,"cache_read_input_tokens":44500,"output_tokens":5}}}
      {"type":"assistant","message":{"model":"claude-opus-4-7","usage":{"input_tokens":6,"cache_creation_input_tokens":20000,"cache_read_input_tokens":44500,"output_tokens":5}}}
      """
    When the status line renders at 200 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus (skill) \e[2m·\e[0m ctx: \e[32m████▎\e[0m\e[2m░░░░░\e[0m 42% \e[2m·\e[0m ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% \e[2m·\e[0m ⌂ 64.5k \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%
      """

  Scenario: A latest assistant turn on a model of no known family is ignored and the session's model name stays
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"transcript_path":"@TMP@/transcript.jsonl"}
      """
    And the file "transcript.jsonl":
      """
      {"type":"user","message":{"role":"user","content":"hi"}}
      {"type":"assistant","message":{"model":"claude-opus-4-7","usage":{"input_tokens":6,"cache_creation_input_tokens":20000,"cache_read_input_tokens":44500,"output_tokens":5}}}
      {"type":"assistant","message":{"model":"some-other-model","usage":{"input_tokens":6,"cache_creation_input_tokens":20000,"cache_read_input_tokens":44500,"output_tokens":5}}}
      """
    When the status line renders at 200 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7 [high] \e[2m·\e[0m ctx: \e[32m████▎\e[0m\e[2m░░░░░\e[0m 42% \e[2m·\e[0m ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% \e[2m·\e[0m ⌂ 64.5k \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%
      """

  Scenario: The diagnostic segments are hidden when the payload has no transcript path
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}}}
      """
    When the status line renders at 200 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7 [high] \e[2m·\e[0m ctx: \e[32m████▎\e[0m\e[2m░░░░░\e[0m 42% \e[2m·\e[0m ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%
      """

  Scenario: A skill listing that is not the initial one does not count towards the skills diagnostics
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"transcript_path":"@TMP@/transcript.jsonl"}
      """
    And the file "transcript.jsonl":
      """
      {"type":"user","message":{"role":"user","content":"hi"}}
      {"type":"attachment","attachment":{"type":"skill_listing","isInitial":false,"content":"- alpha: does a thing\u000a- beta\u000a- gamma\u000a- delta: has text\u000a- eps\u000a"}}
      """
    When the status line renders at 200 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7 [high] \e[2m·\e[0m ctx: \e[32m████▎\e[0m\e[2m░░░░░\e[0m 42% \e[2m·\e[0m ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% \e[2m·\e[0m ⚙ 15 \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%
      """

  Scenario: The skills segment shows the size of the initial skill listing
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"transcript_path":"@TMP@/transcript.jsonl"}
      """
    And the file "transcript.jsonl":
      """
      {"type":"user","message":{"role":"user","content":"hi"}}
      {"type":"attachment","attachment":{"type":"skill_listing","isInitial":true,"content":"- alpha: does a thing\u000a- beta: another\u000a- gamma: third thing here\u000a"}}
      """
    When the status line renders at 200 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7 [high] \e[2m·\e[0m ctx: \e[32m████▎\e[0m\e[2m░░░░░\e[0m 42% \e[2m·\e[0m ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% \e[2m·\e[0m ⚙ 16 \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%
      """

  Scenario: A skill listing in which three of five entries lost their descriptions is flagged as stripped skills
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"transcript_path":"@TMP@/transcript.jsonl"}
      """
    And the file "transcript.jsonl":
      """
      {"type":"user","message":{"role":"user","content":"hi"}}
      {"type":"attachment","attachment":{"type":"skill_listing","isInitial":true,"content":"- alpha: does a thing\u000a- beta\u000a- gamma\u000a- delta: has text\u000a- eps\u000a"}}
      """
    When the status line renders at 200 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7 [high] \e[2m·\e[0m ctx: \e[32m████▎\e[0m\e[2m░░░░░\e[0m 42% \e[2m·\e[0m ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% \e[2m·\e[0m ⚙ 15 \e[2m·\e[0m \e[31m✗ skills 3/5\e[0m \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%
      """
