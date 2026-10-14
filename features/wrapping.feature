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
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}}}
      """
    When the status line renders at 200 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7 [high] \e[2m·\e[0m ctx: \e[32m████▎\e[0m\e[2m░░░░░\e[0m 42% \e[2m·\e[0m ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%
      """

  Scenario: Segments wrap into two rows at 120 columns
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}}}
      """
    When the status line renders at 120 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7 [high] \e[2m·\e[0m ctx: \e[32m███▍\e[0m\e[2m░░░░\e[0m 42%
      ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% 5h: \e[32m██████▎\e[0m\e[1m┃\e[0m\e[2m░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m█████▋\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░\e[0m 40%
      """

  Scenario: Segments wrap into two rows at 100 columns
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}}}
      """
    When the status line renders at 100 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7 [high] \e[2m·\e[0m ctx: \e[32m██▌\e[0m\e[2m░░░\e[0m 42%
      ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% 5h: \e[32m████▌\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 40%
      """

  Scenario: Segments wrap into two rows at 80 columns
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}}}
      """
    When the status line renders at 80 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7 [high] \e[2m·\e[0m ctx: \e[32m██▌\e[0m\e[2m░░░\e[0m 42%
      ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% 5h: \e[32m████▌\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 40%
      """

  Scenario: Segments wrap into several rows at 60 columns
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}}}
      """
    When the status line renders at 60 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7 [high] \e[2m·\e[0m ctx: \e[32m██▌\e[0m\e[2m░░░\e[0m 42%
      ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85%
      \e[2m↳\e[0m          5h: \e[32m████▌\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 40%
      """

  Scenario: Segments wrap into several rows at 40 columns, with continuation rows marked by an arrow
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}}}
      """
    When the status line renders at 40 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7 [high]
      \e[2m↳\e[0m          ctx: \e[32m██▌\e[0m\e[2m░░░\e[0m 42%
      ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85%
      \e[2m↳\e[0m          5h: \e[32m████▌\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 40%
      """

  Scenario: At 10 columns every group of segments gets its own row
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}}}
      """
    When the status line renders at 10 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m
      Opus 4.7 [high]
      ctx: \e[32m██▌\e[0m\e[2m░░░\e[0m 42%
      ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85%
      5h: \e[32m████▌\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 40%
      """

  Scenario: Wrapping at 60 columns when the payload has no rate limits
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":1000,"cache_creation_input_tokens":500,"cache_read_input_tokens":8500}}}
      """
    When the status line renders at 60 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7 [high] \e[2m·\e[0m ctx: \e[32m██▌\e[0m\e[2m░░░\e[0m 42%
      ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85%
      """

  Scenario: Wrapping at 30 columns when the payload has neither rate limits nor token counts
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"cwd":"/srv/app","context_window":{"used_percentage":42}}
      """
    When the status line renders at 30 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7
      ctx: \e[32m██▌\e[0m\e[2m░░░\e[0m 42%
      """

  Scenario: Wrapping at 80 columns when the payload has no token counts
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","context_window":{"used_percentage":42,"current_usage":{}},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}}}
      """
    When the status line renders at 80 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7 [high] \e[2m·\e[0m ctx: \e[32m██▌\e[0m\e[2m░░░\e[0m 42%
      \e[2m↳\e[0m          5h: \e[32m████▌\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 40%
      """

  Scenario: Wrapping at 50 columns when the payload has rate limits but no token counts
    Given the session:
      """json
      {"session_id":"abc123","model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"cwd":"/srv/app","rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}}}
      """
    When the status line renders at 50 columns
    Then the output is:
      """
      \e[1m\e[38;5;33m/srv/app\e[0m \e[2m·\e[0m Opus 4.7 [high]
      \e[2m↳\e[0m          5h: \e[32m████▌\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 40%
      """
