@rate_bars
Feature: Rate-limit bars
  The five hour and seven day usage bars: the colour follows the pace marker.

  Background:
    Given the environment variable "STATUSLINE_NOW" is "1760000000"
    And the environment variable "COLUMNS" is "200"

  Scenario: Usage of 75% with half the window elapsed is drawn in yellow
    Given the session:
      """json
      {"model":{"display_name":"Opus 4.7"},"rate_limits":{"five_hour":{"used_percentage":75,"resets_at":1760009000}},"session_id":"abc123"}
      """
    When the status line renders
    Then the output is:
      """
      Opus 4.7 \e[2m·\e[0m 5h: \e[33m██████████\e[0m\e[1m┃\e[0m\e[33m████\e[0m\e[2m░░░░░\e[0m 75%
      """
