@rate_limit_cache
Feature: Shared rate-limit cache
  Sessions share rate-limit readings through a cache file, so an idle session shows what a busy
  one has already seen, and a stale reading never replaces a newer one.

  Background:
    Given the environment variable "STATUSLINE_NOW" is "1760000000"
    And the environment variable "COLUMNS" is "200"

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
