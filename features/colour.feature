@colour
Feature: Colour and NO_COLOR
  A non-empty NO_COLOR removes every colour; an empty one does not.

  Background:
    Given the environment variable "STATUSLINE_NOW" is "1760000000"
    And the environment variable "COLUMNS" is "200"

  Scenario: NO_COLOR with every segment, a git branch and a stripped path prefix
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
          "current_dir": "@TMP@/code/app"
        },
        "context_window": {
          "used_percentage": 95,
          "total_input_tokens": 1500,
          "total_output_tokens": 23400,
          "current_usage": {
            "input_tokens": 2000,
            "cache_creation_input_tokens": 1000,
            "cache_read_input_tokens": 10
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
    And the config:
      """
      path:
        strip: ["~/code"]
      """
    And the environment variable "NO_COLOR" is "1"
    And the file "code/app/.git/HEAD":
      """
      ref: refs/heads/main
      """
    When the status line renders
    Then the text is:
      """
      app (main) · Opus 4.7 [high] · ctx: █████████▌ 95% · ↓ 1.5k · ↑ 23.4k · ⧉ 0% · 5h: █████████░┃░░░░░░░░░ 45% · 7d: ████████░░┃░░░░░░░░░ 40%
      """
