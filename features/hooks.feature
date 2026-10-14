@hooks
Feature: Custom segment hooks
  Scripts in the hook directory add custom segments in the ctx, usage or end slot, with a per-session
  cache, a time to live and only their first line shown.

  Background:
    Given the environment variable "STATUSLINE_NOW" is "1760000000"
    And the environment variable "COLUMNS" is "200"

  Scenario: Custom hooks in the ctx, usage and end slots at 100 columns
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
        }
      }
      """
    And the executable file "statusline.d/10-ctxhook.sh":
      """
      #!/bin/sh
      printf 'ctx-hook'
      """
    And the executable file "statusline.d/20-usagehook.sh":
      """
      #!/bin/sh
      # statusline-slot: usage
      printf '\033[32musage-hook\033[0m'
      """
    And the executable file "statusline.d/30-endhook.sh":
      """
      #!/bin/sh
      # statusline-slot: end
      printf 'end-hook'
      """
    When the status line renders at 100 columns
    Then the output is:
      """
      /srv/app · Opus 4.7 [high] · ctx: ██▌░░░ 42% · ctx-hook
      11111111 d                 d      gggddd     d
      ↓ 1.5k · ↑ 23.4k · ⧉ 85% 5h: ████▌┃░░░░ 45% · 7d: ████░┃░░░░ 40%
             d         d           gggggBdddd     d     ggggdBdddd
      usage-hook · end-hook
      gggggggggg d

      1 = bold 256:33
      """

  Scenario: Custom hooks in the ctx, usage and end slots at 200 columns
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
        }
      }
      """
    And the executable file "statusline.d/10-ctxhook.sh":
      """
      #!/bin/sh
      printf 'ctx-hook'
      """
    And the executable file "statusline.d/20-usagehook.sh":
      """
      #!/bin/sh
      # statusline-slot: usage
      printf '\033[32musage-hook\033[0m'
      """
    And the executable file "statusline.d/30-endhook.sh":
      """
      #!/bin/sh
      # statusline-slot: end
      printf 'end-hook'
      """
    When the status line renders at 200 columns
    Then the output is:
      """
      /srv/app · Opus 4.7 [high] · ctx: ████▎░░░░░ 42% · ctx-hook · ↓ 1.5k · ↑ 23.4k · ⧉ 85% · usage-hook · 5h: █████████░┃░░░░░░░░░ 45% · 7d: ████████░░┃░░░░░░░░░ 40% · end-hook
      11111111 d                 d      gggggddddd     d          d        d         d       d gggggggggg d     gggggggggdBddddddddd     d     ggggggggddBddddddddd     d

      1 = bold 256:33
      """

  Scenario: A hook named like a built-in segment in the segments list replaces it
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
        "cwd": "/srv/app"
      }
      """
    And the config:
      """
      segments = path model
      """
    And the executable file "statusline.d/model.sh":
      """
      #!/bin/sh
      printf 'my-model'
      """
    When the status line renders
    Then the output is:
      """
      /srv/app · my-model
      11111111 d

      1 = bold 256:33
      """
