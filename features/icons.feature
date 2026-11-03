@icons
Feature: Nerd Font icons
  With icons: nerd, each label is drawn as a Nerd Font icon. The icons need a Nerd Font set as the
  terminal font, and the harness counts each as one column, as a Mono variant of the font draws it.

  Background:
    Given the environment variable "STATUSLINE_NOW" is "1760000000"
    And the environment variable "COLUMNS" is "200"

  Scenario: The default line with Nerd Font icons
    Given the session:
      """json
      {
        "session_id": "demo",
        "model": {
          "id": "claude-opus-4-7",
          "display_name": "Opus 4.7"
        },
        "effort": {
          "level": "high"
        },
        "workspace": {
          "current_dir": "@TMP@/code/app"
        },
        "context_window": {
          "used_percentage": 42.4,
          "total_input_tokens": 1500,
          "total_output_tokens": 23400,
          "current_usage": {
            "input_tokens": 2000,
            "cache_creation_input_tokens": 1000,
            "cache_read_input_tokens": 17000
          }
        },
        "rate_limits": {
          "five_hour": {
            "used_percentage": 45.4,
            "resets_at": 1760009000
          },
          "seven_day": {
            "used_percentage": 62,
            "resets_at": 1760302400
          }
        },
        "transcript_path": "@TMP@/transcript.jsonl"
      }
      """
    And the file "code/app/.git/HEAD":
      """
      ref: refs/heads/feature/retry-backoff
      """
    And the file "transcript.jsonl":
      """
      {"type":"user","message":{"role":"user","content":"hi"}}
      {"type":"attachment","attachment":{"type":"skill_listing","isInitial":true,"content":"- review: Review a change for correctness, style and missing tests before it is merged\u000a- debug: Work out why a test fails or a service misbehaves, starting from the evidence\u000a- plan: Write an implementation plan for a multi-step change before touching code\u000a- release: Prepare release notes and check the version label on a pull request\u000a- docs: Write or restructure a repository's README and contributing guide\u000a"}}
      {"type":"assistant","message":{"model":"claude-opus-4-7","usage":{"input_tokens":6,"cache_creation_input_tokens":20000,"cache_read_input_tokens":44500,"output_tokens":5}}}
      """
    And the config:
      """
      icons: nerd
      path:
        emoji: true
      """
    When the status line renders at 200 columns
    Then the output is:
      """
       ~/code/app ( feature/retry-backoff) · 󰚩 Opus 4.7 [high] · 󰍛 ████▎░░░░░ 42% ·  1.5k ·  23.4k ·  85% ·  64.5k ·  102 ·  5h █████████░┃░░░░░░░░░ 45% ·  7d ██████████┃█▍░░░░░░░ 62%
      11111111111111111111111111111111111111 d                   d   gggggddddd     d        d         d       d         d       d      gggggggggdBddddddddd     d      yyyyyyyyyyByyddddddd

      1 = bold 256:33
      """

  Scenario: Nerd Font icons without colour
    Given the session:
      """json
      {
        "model": {
          "display_name": "Opus 4.7"
        },
        "cwd": "@TMP@/code/app",
        "context_window": {
          "used_percentage": 42
        }
      }
      """
    And the environment variable "NO_COLOR" is "1"
    And the config:
      """
      icons: nerd
      segments: [path, model, ctx]
      """
    When the status line renders
    Then the text is:
      """
       ~/code/app · 󰚩 Opus 4.7 · 󰍛 ████▎░░░░░ 42%
      """

  Scenario: Nerd Font icons wrap at 80 columns
    Given the session:
      """json
      {
        "session_id": "demo",
        "model": {
          "id": "claude-opus-4-7",
          "display_name": "Opus 4.7"
        },
        "effort": {
          "level": "high"
        },
        "workspace": {
          "current_dir": "@TMP@/code/app"
        },
        "context_window": {
          "used_percentage": 42.4,
          "total_input_tokens": 1500,
          "total_output_tokens": 23400,
          "current_usage": {
            "input_tokens": 2000,
            "cache_creation_input_tokens": 1000,
            "cache_read_input_tokens": 17000
          }
        },
        "rate_limits": {
          "five_hour": {
            "used_percentage": 45.4,
            "resets_at": 1760009000
          },
          "seven_day": {
            "used_percentage": 62,
            "resets_at": 1760302400
          }
        },
        "transcript_path": "@TMP@/transcript.jsonl"
      }
      """
    And the file "code/app/.git/HEAD":
      """
      ref: refs/heads/feature/retry-backoff
      """
    And the file "transcript.jsonl":
      """
      {"type":"user","message":{"role":"user","content":"hi"}}
      {"type":"attachment","attachment":{"type":"skill_listing","isInitial":true,"content":"- review: Review a change for correctness, style and missing tests before it is merged\u000a- debug: Work out why a test fails or a service misbehaves, starting from the evidence\u000a- plan: Write an implementation plan for a multi-step change before touching code\u000a- release: Prepare release notes and check the version label on a pull request\u000a- docs: Write or restructure a repository's README and contributing guide\u000a"}}
      {"type":"assistant","message":{"model":"claude-opus-4-7","usage":{"input_tokens":6,"cache_creation_input_tokens":20000,"cache_read_input_tokens":44500,"output_tokens":5}}}
      """
    And the environment variable "NO_COLOR" is "1"
    And the config:
      """
      icons: nerd
      path:
        emoji: true
      """
    When the status line renders at 80 columns
    Then the text is:
      """
       ~/code/app ( feature/retry-backoff) · 󰚩 Opus 4.7 [high] · 󰍛 ██▌░░░ 42%
       1.5k ·  23.4k ·  85% ·  64.5k ·  102
      ↳                                         5h ████▌┃░░░░ 45% ·  7d █████┃▎░░░ 62%
      """
