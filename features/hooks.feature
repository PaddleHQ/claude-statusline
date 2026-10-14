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
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}}}
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
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7 [high] \e[2m·\e[0m ctx: \e[32m██▌\e[0m\e[2m░░░\e[0m 42% \e[2m·\e[0m ctx-hook
      ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% 5h: \e[32m████▌\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 40%
      \e[32musage-hook\e[0m \e[2m·\e[0m end-hook
      """

  Scenario: Custom hooks in the ctx, usage and end slots at 200 columns
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}}}
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
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7 [high] \e[2m·\e[0m ctx: \e[32m████▎\e[0m\e[2m░░░░░\e[0m 42% \e[2m·\e[0m ctx-hook \e[2m·\e[0m ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% \e[2m·\e[0m \e[32musage-hook\e[0m \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40% \e[2m·\e[0m end-hook
      """

  Scenario: A hook named like a built-in segment in the segments list replaces it
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app"}
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
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m my-model
      """
