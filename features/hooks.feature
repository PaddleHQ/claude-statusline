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

  Scenario: Custom hooks in the ctx, usage and end slots at 60 columns
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
    When the status line renders at 60 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7 [high] \e[2m·\e[0m ctx: \e[32m██▌\e[0m\e[2m░░░\e[0m 42% \e[2m·\e[0m ctx-hook
      ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85%
      \e[2m↳\e[0m          5h: \e[32m████▌\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 40%
      \e[32musage-hook\e[0m \e[2m·\e[0m end-hook
      """

  Scenario: A hook whose cached output is younger than the default time to live is not run again
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}}}
      """
    And the executable file "statusline.d/10-c.sh":
      """
      #!/bin/sh
      printf fresh
      """
    And the file "claude-statusline/cache/hooks/abc123/10-c.sh":
      """
      1759999990
      stale
      """
    When the status line renders at 200 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7 [high] \e[2m·\e[0m ctx: \e[32m████▎\e[0m\e[2m░░░░░\e[0m 42% \e[2m·\e[0m stale \e[2m·\e[0m ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%
      """

  Scenario: A hook whose cached output is younger than its own statusline-ttl is not run again
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}}}
      """
    And the executable file "statusline.d/10-c.sh":
      """
      #!/bin/sh
      # statusline-ttl: 300
      printf fresh
      """
    And the file "claude-statusline/cache/hooks/abc123/10-c.sh":
      """
      1759999800
      stale
      """
    When the status line renders at 200 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7 [high] \e[2m·\e[0m ctx: \e[32m████▎\e[0m\e[2m░░░░░\e[0m 42% \e[2m·\e[0m stale \e[2m·\e[0m ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%
      """

  Scenario: A custom hook in the default ctx slot at 100 columns
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}}}
      """
    And the executable file "statusline.d/10-ctxhook.sh":
      """
      #!/bin/sh
      printf 'ctx-hook'
      """
    When the status line renders at 100 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7 [high] \e[2m·\e[0m ctx: \e[32m██▌\e[0m\e[2m░░░\e[0m 42% \e[2m·\e[0m ctx-hook
      ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% 5h: \e[32m████▌\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 40%
      """

  Scenario: A custom hook in the default ctx slot at 200 columns
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}}}
      """
    And the executable file "statusline.d/10-ctxhook.sh":
      """
      #!/bin/sh
      printf 'ctx-hook'
      """
    When the status line renders at 200 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7 [high] \e[2m·\e[0m ctx: \e[32m████▎\e[0m\e[2m░░░░░\e[0m 42% \e[2m·\e[0m ctx-hook \e[2m·\e[0m ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%
      """

  Scenario: A custom hook in the default ctx slot at 60 columns
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}}}
      """
    And the executable file "statusline.d/10-ctxhook.sh":
      """
      #!/bin/sh
      printf 'ctx-hook'
      """
    When the status line renders at 60 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7 [high] \e[2m·\e[0m ctx: \e[32m██▌\e[0m\e[2m░░░\e[0m 42% \e[2m·\e[0m ctx-hook
      ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85%
      \e[2m↳\e[0m          5h: \e[32m████▌\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 40%
      """

  Scenario: Two custom hooks in the ctx slot at 80 columns, shown in file name order
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}}}
      """
    And the executable file "statusline.d/10-ctxhook.sh":
      """
      #!/bin/sh
      printf 'ctx-hook'
      """
    And the executable file "statusline.d/11-second.sh":
      """
      #!/bin/sh
      printf 'second-hook'
      """
    When the status line renders at 80 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7 [high] \e[2m·\e[0m ctx: \e[32m██▌\e[0m\e[2m░░░\e[0m 42% \e[2m·\e[0m ctx-hook \e[2m·\e[0m second-hook
      ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% 5h: \e[32m████▌\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 40%
      """

  Scenario: A hook that now prints nothing replaces its cached output, so the segment disappears
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}}}
      """
    And the executable file "statusline.d/10-c.sh":
      """
      #!/bin/sh
      exit 0
      """
    And the file "claude-statusline/cache/hooks/abc123/10-c.sh":
      """
      1759999000
      stale
      """
    When the status line renders at 200 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7 [high] \e[2m·\e[0m ctx: \e[32m████▎\e[0m\e[2m░░░░░\e[0m 42% \e[2m·\e[0m ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%
      """

  Scenario: A custom hook in the end slot at 100 columns
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}}}
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
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7 [high] \e[2m·\e[0m ctx: \e[32m██▌\e[0m\e[2m░░░\e[0m 42%
      ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% 5h: \e[32m████▌\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 40%
      end-hook
      """

  Scenario: A custom hook in the end slot at 200 columns
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}}}
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
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7 [high] \e[2m·\e[0m ctx: \e[32m████▎\e[0m\e[2m░░░░░\e[0m 42% \e[2m·\e[0m ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40% \e[2m·\e[0m end-hook
      """

  Scenario: A custom hook in the end slot at 60 columns
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}}}
      """
    And the executable file "statusline.d/30-endhook.sh":
      """
      #!/bin/sh
      # statusline-slot: end
      printf 'end-hook'
      """
    When the status line renders at 60 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7 [high] \e[2m·\e[0m ctx: \e[32m██▌\e[0m\e[2m░░░\e[0m 42%
      ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85%
      \e[2m↳\e[0m          5h: \e[32m████▌\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 40%
      end-hook
      """

  Scenario: A hook whose cached output is older than the default time to live runs again
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}}}
      """
    And the executable file "statusline.d/10-c.sh":
      """
      #!/bin/sh
      printf fresh
      """
    And the file "claude-statusline/cache/hooks/abc123/10-c.sh":
      """
      1759999900
      stale
      """
    When the status line renders at 200 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7 [high] \e[2m·\e[0m ctx: \e[32m████▎\e[0m\e[2m░░░░░\e[0m 42% \e[2m·\e[0m fresh \e[2m·\e[0m ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%
      """

  Scenario: Failing and silent hooks add nothing, and a hook naming an unknown slot is placed in the default ctx slot
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}}}
      """
    And the executable file "statusline.d/10-fail.sh":
      """
      #!/bin/sh
      exit 1
      """
    And the executable file "statusline.d/20-empty.sh":
      """
      #!/bin/sh
      exit 0
      """
    And the executable file "statusline.d/30-typo.sh":
      """
      #!/bin/sh
      # statusline-slot: bogus
      printf 'typo-slot'
      """
    When the status line renders at 200 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7 [high] \e[2m·\e[0m ctx: \e[32m████▎\e[0m\e[2m░░░░░\e[0m 42% \e[2m·\e[0m typo-slot \e[2m·\e[0m ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%
      """

  Scenario: A hook that fails keeps showing its cached output
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}}}
      """
    And the executable file "statusline.d/10-c.sh":
      """
      #!/bin/sh
      exit 1
      """
    And the file "claude-statusline/cache/hooks/abc123/10-c.sh":
      """
      1759999000
      stale
      """
    When the status line renders at 200 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7 [high] \e[2m·\e[0m ctx: \e[32m████▎\e[0m\e[2m░░░░░\e[0m 42% \e[2m·\e[0m stale \e[2m·\e[0m ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%
      """

  Scenario: Only the first line of a hook's output is shown
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}}}
      """
    And the executable file "statusline.d/10-c.sh":
      """
      #!/bin/sh
      printf 'one
      two
      '
      """
    When the status line renders at 200 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7 [high] \e[2m·\e[0m ctx: \e[32m████▎\e[0m\e[2m░░░░░\e[0m 42% \e[2m·\e[0m one \e[2m·\e[0m ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%
      """

  Scenario: A custom hook in the usage slot at 100 columns
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}}}
      """
    And the executable file "statusline.d/20-usagehook.sh":
      """
      #!/bin/sh
      # statusline-slot: usage
      printf '\033[32musage-hook\033[0m'
      """
    When the status line renders at 100 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7 [high] \e[2m·\e[0m ctx: \e[32m██▌\e[0m\e[2m░░░\e[0m 42%
      ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% 5h: \e[32m████▌\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 40%
      \e[32musage-hook\e[0m
      """

  Scenario: A custom hook in the usage slot at 200 columns
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}}}
      """
    And the executable file "statusline.d/20-usagehook.sh":
      """
      #!/bin/sh
      # statusline-slot: usage
      printf '\033[32musage-hook\033[0m'
      """
    When the status line renders at 200 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7 [high] \e[2m·\e[0m ctx: \e[32m████▎\e[0m\e[2m░░░░░\e[0m 42% \e[2m·\e[0m ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% \e[2m·\e[0m \e[32musage-hook\e[0m \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%
      """

  Scenario: A custom hook in the usage slot at 60 columns
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}}}
      """
    And the executable file "statusline.d/20-usagehook.sh":
      """
      #!/bin/sh
      # statusline-slot: usage
      printf '\033[32musage-hook\033[0m'
      """
    When the status line renders at 60 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7 [high] \e[2m·\e[0m ctx: \e[32m██▌\e[0m\e[2m░░░\e[0m 42%
      ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85%
      \e[2m↳\e[0m          5h: \e[32m████▌\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 40%
      \e[32musage-hook\e[0m
      """
