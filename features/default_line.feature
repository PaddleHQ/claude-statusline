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
      {"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"rate_limits":{"five_hour":{"used_percentage":45.4,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"session_id":"abc123"}
      """
    When the status line renders
    Then the output is:
      """
      Opus 4.7 [high] \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%
      """

  Scenario: Every default segment with a full payload, a git repository and a path under the home directory
    Given the session:
      """json
      {"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"workspace":{"current_dir":"@TMP@/code/app/internal"},"context_window":{"used_percentage":42.4,"total_input_tokens":1500,"total_output_tokens":23400,"current_usage":{"input_tokens":2000,"cache_creation_input_tokens":1000,"cache_read_input_tokens":17000}},"rate_limits":{"five_hour":{"used_percentage":45.4,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}}}
      """
    And the file "code/app/.git/HEAD":
      """
      ref: refs/heads/feature/core-segments
      """
    When the status line renders
    Then the output is:
      """
      \e[1m\e[38;5;33m~/code/app/internal (feature/core-segments)\e[0m \e[2m·\e[0m Opus 4.7 [high] \e[2m·\e[0m ctx: \e[32m████▎\e[0m\e[2m░░░░░\e[0m 42% \e[2m·\e[0m ↓ 1.5k \e[2m·\e[0m ↑ 23.4k \e[2m·\e[0m ⧉ 85% \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%
      """
