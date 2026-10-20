@transcript_diagnostics
Feature: Transcript diagnostics
  The boot, skills and mem segments, and the model family, are read from the session transcript.

  Background:
    Given the environment variable "STATUSLINE_NOW" is "1760000000"
    And the environment variable "COLUMNS" is "200"

  Scenario: Every transcript diagnostic at once on a 200 column terminal
    Given the session:
      """json
      {
        "session_id": "abc123",
        "model": {
          "display_name": "Opus 4.7"
        },
        "effort": {
          "level": "high"
        },
        "cwd": "/srv/app",
        "context_window": {
          "used_percentage": 42,
          "total_input_tokens": 1500,
          "total_output_tokens": 23400,
          "current_usage": {
            "input_tokens": 1000,
            "cache_creation_input_tokens": 500,
            "cache_read_input_tokens": 8500
          }
        },
        "rate_limits": {
          "five_hour": {
            "used_percentage": 45,
            "resets_at": 1760009000
          },
          "seven_day": {
            "used_percentage": 40,
            "resets_at": 1760302400
          }
        },
        "transcript_path": "@TMP@/transcript.jsonl"
      }
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
    When the status line renders at 210 columns
    Then the output is:
      """
      /srv/app · Sonnet 5.5 (skill) · ctx: ████▎░░░░░ 42% · ↓ 1.5k · ↑ 23.4k · ⧉ 85% · ⌂ 64.5k · ⚙ 15 · ✗ skills 3/5 · ✗ agent skills 3/6 · ✗ mem · 5h: █████████░┃░░░░░░░░░ 45% · 7d: ████████░░┃░░░░░░░░░ 40%
      11111111 d                    d      gggggddddd     d        d         d       d         d      d rrrrrrrrrrrr d yyyyyyyyyyyyyyyyyy d rrrrr d     gggggggggdBddddddddd     d     ggggggggddBddddddddd

      1 = bold 256:33
      """
