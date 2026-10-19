@default_line
Feature: The default status line
  With every default segment and a full payload, the status line shows the path, model, effort,
  context bar, tokens, cache hit rate and both rate-limit bars.

  Background:
    Given the environment variable "STATUSLINE_NOW" is "1760000000"
    And the environment variable "COLUMNS" is "200"

  Scenario: The default segments with a full payload: model, effort, and both rate-limit bars with pace markers
    Given the session:
      """json
      {
        "model": {
          "display_name": "Opus 4.7"
        },
        "effort": {
          "level": "high"
        },
        "rate_limits": {
          "five_hour": {
            "used_percentage": 45.4,
            "resets_at": 1760009000
          },
          "seven_day": {
            "used_percentage": 40,
            "resets_at": 1760302400
          }
        }
      }
      """
    When the status line renders
    Then the output is:
      """
      Opus 4.7 [high] · 5h: █████████░┃░░░░░░░░░ 45% · 7d: ████████░░┃░░░░░░░░░ 40%
                      d     gggggggggdBddddddddd     d     ggggggggddBddddddddd
      """

  Scenario: Every default segment with a full payload, a git repository and a path under the home directory
    Given the session:
      """json
      {
        "model": {
          "display_name": "Opus 4.7"
        },
        "effort": {
          "level": "high"
        },
        "workspace": {
          "current_dir": "@TMP@/code/app/internal"
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
            "used_percentage": 40,
            "resets_at": 1760302400
          }
        }
      }
      """
    And the file "code/app/.git/HEAD":
      """
      ref: refs/heads/feature/core-segments
      """
    When the status line renders
    Then the output is:
      """
      ~/code/app/internal (feature/core-segments) · Opus 4.7 [high] · ctx: ████▎░░░░░ 42% · ↓ 1.5k · ↑ 23.4k · ⧉ 85% · 5h: █████████░┃░░░░░░░░░ 45% · 7d: ████████░░┃░░░░░░░░░ 40%
      1111111111111111111111111111111111111111111 d                 d      gggggddddd     d        d         d       d     gggggggggdBddddddddd     d     ggggggggddBddddddddd

      1 = bold 256:33
      """
