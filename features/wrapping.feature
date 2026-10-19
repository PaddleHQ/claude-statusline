@wrapping
Feature: Wrapping to the terminal width
  When the segments do not fit the terminal, the line wraps into groups, with continuation rows
  marked by an arrow.

  Background:
    Given the environment variable "STATUSLINE_NOW" is "1760000000"
    And the environment variable "COLUMNS" is "200"

  Scenario: Every segment fits on one row at 200 columns
    Given the session:
      """json
      {
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
        }
      }
      """
    When the status line renders at 200 columns
    Then the output is:
      """
      /srv/app · Opus 4.7 [high] · ctx: ████▎░░░░░ 42% · ↓ 1.5k · ↑ 23.4k · ⧉ 85% · 5h: █████████░┃░░░░░░░░░ 45% · 7d: ████████░░┃░░░░░░░░░ 40%
      11111111 d                 d      gggggddddd     d        d         d       d     gggggggggdBddddddddd     d     ggggggggddBddddddddd

      1 = bold 256:33
      """

  Scenario: Segments wrap into two rows at 80 columns
    Given the session:
      """json
      {
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
        }
      }
      """
    When the status line renders at 80 columns
    Then the output is:
      """
      /srv/app · Opus 4.7 [high] · ctx: ██▌░░░ 42%
      11111111 d                 d      gggddd
      ↓ 1.5k · ↑ 23.4k · ⧉ 85% 5h: ████▌┃░░░░ 45% · 7d: ████░┃░░░░ 40%
             d         d           gggggBdddd     d     ggggdBdddd

      1 = bold 256:33
      """

  Scenario: Segments wrap into several rows at 40 columns, with continuation rows marked by an arrow
    Given the session:
      """json
      {
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
        }
      }
      """
    When the status line renders at 40 columns
    Then the output is:
      """
      /srv/app · Opus 4.7 [high]
      11111111 d
      ↳          ctx: ██▌░░░ 42%
      d               gggddd
      ↓ 1.5k · ↑ 23.4k · ⧉ 85%
             d         d
      ↳          5h: ████▌┃░░░░ 45% · 7d: ████░┃░░░░ 40%
      d              gggggBdddd     d     ggggdBdddd

      1 = bold 256:33
      """

  Scenario: At 10 columns every group of segments gets its own row
    Given the session:
      """json
      {
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
        }
      }
      """
    When the status line renders at 10 columns
    Then the output is:
      """
      /srv/app
      11111111
      Opus 4.7 [high]

      ctx: ██▌░░░ 42%
           gggddd
      ↓ 1.5k · ↑ 23.4k · ⧉ 85%
             d         d
      5h: ████▌┃░░░░ 45% · 7d: ████░┃░░░░ 40%
          gggggBdddd     d     ggggdBdddd

      1 = bold 256:33
      """
