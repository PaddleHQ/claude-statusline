@rate_limit_cache
Feature: Shared rate-limit cache
  Sessions share rate-limit readings through a cache file, so an idle session shows what a busy
  one has already seen, and a stale reading never replaces a newer one.

  Background:
    Given the environment variable "STATUSLINE_NOW" is "1760000000"
    And the environment variable "COLUMNS" is "200"

  Scenario: The first rate-limit reading is written to the shared cache file
    Given the session:
      """json
      {"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"session_id":"abc123"}
      """
    When the status line renders
    Then the output is:
      """
      Opus 4.7 [high] \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%
      """
    And the rate-limit cache is:
      """
      {"five_hour":{"used":45,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400}}
      """

  Scenario: Rate limits missing from the payload are filled in from the shared cache file
    Given the session:
      """json
      {"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"session_id":"abc123"}
      """
    And the rate-limit cache:
      """
      {"five_hour":{"used":45,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400}}
      """
    When the status line renders
    Then the output is:
      """
      Opus 4.7 [high] \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%
      """
    And the rate-limit cache is:
      """
      {"five_hour":{"used":45,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400}}
      """

  Scenario: A higher usage reading replaces the cached one and is written back to the cache
    Given the session:
      """json
      {"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"rate_limits":{"five_hour":{"used_percentage":55,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"session_id":"abc123"}
      """
    And the rate-limit cache:
      """
      {"five_hour":{"used":45,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400}}
      """
    When the status line renders
    Then the output is:
      """
      Opus 4.7 [high] \e[2m·\e[0m 5h: \e[32m██████████\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 55% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%
      """
    And the rate-limit cache is:
      """
      {"five_hour":{"used":55,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400}}
      """

  Scenario: A lower reading from an idle session is replaced by the higher cached one from the busy session
    Given the session:
      """json
      {"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"rate_limits":{"five_hour":{"used_percentage":40,"resets_at":1760009000},"seven_day":{"used_percentage":38,"resets_at":1760302400}},"session_id":"abc123"}
      """
    And the rate-limit cache:
      """
      {"five_hour":{"used":45,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400}}
      """
    When the status line renders
    Then the output is:
      """
      Opus 4.7 [high] \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%
      """
    And the rate-limit cache is:
      """
      {"five_hour":{"used":45,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400}}
      """

  Scenario: A usage reading 10 points below the cached one is treated as stale and the cached value is kept
    Given the session:
      """json
      {"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"rate_limits":{"five_hour":{"used_percentage":35,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"session_id":"abc123"}
      """
    And the rate-limit cache:
      """
      {"five_hour":{"used":45,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400}}
      """
    When the status line renders
    Then the output is:
      """
      Opus 4.7 [high] \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%
      """
    And the rate-limit cache is:
      """
      {"five_hour":{"used":45,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400}}
      """

  Scenario: A usage reading more than 10 points below the cached one is accepted as a real reset
    Given the session:
      """json
      {"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"rate_limits":{"five_hour":{"used_percentage":20,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"session_id":"abc123"}
      """
    And the rate-limit cache:
      """
      {"five_hour":{"used":45,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400}}
      """
    When the status line renders
    Then the output is:
      """
      Opus 4.7 [high] \e[2m·\e[0m 5h: \e[33m████\e[0m\e[2m░░░░░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 20% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%
      """
    And the rate-limit cache is:
      """
      {"five_hour":{"used":20,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400}}
      """

  Scenario: A reading from an earlier rate-limit window than the cached one is ignored
    Given the session:
      """json
      {"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"rate_limits":{"five_hour":{"used_percentage":5,"resets_at":1759991000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"session_id":"abc123"}
      """
    And the rate-limit cache:
      """
      {"five_hour":{"used":45,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400}}
      """
    When the status line renders
    Then the output is:
      """
      Opus 4.7 [high] \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%
      """
    And the rate-limit cache is:
      """
      {"five_hour":{"used":45,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400}}
      """

  Scenario: A reading from a later rate-limit window replaces the cached one even when it is lower
    Given the session:
      """json
      {"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"rate_limits":{"five_hour":{"used_percentage":5,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"session_id":"abc123"}
      """
    And the rate-limit cache:
      """
      {"five_hour":{"used":90,"resets_at":1759991000},"seven_day":{"used":40,"resets_at":1760302400}}
      """
    When the status line renders
    Then the output is:
      """
      Opus 4.7 [high] \e[2m·\e[0m 5h: \e[31m█\e[0m\e[2m░░░░░░░░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 5% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%
      """
    And the rate-limit cache is:
      """
      {"five_hour":{"used":5,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400}}
      """
