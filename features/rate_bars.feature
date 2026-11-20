@rate_bars
Feature: Rate-limit bars
  The five hour and seven day usage bars: the colour follows the pace marker.

  Background:
    Given the environment variable "STATUSLINE_NOW" is "1760000000"
    And the environment variable "COLUMNS" is "200"

  Scenario: Usage of 75% with half the window elapsed is drawn in yellow
    Given the session:
      """json
      {
        "model": {
          "display_name": "Opus 4.7"
        },
        "rate_limits": {
          "five_hour": {
            "used_percentage": 75,
            "resets_at": 1760009000
          }
        }
      }
      """
    When the status line renders
    Then the output is:
      """
      Opus 4.7 · 5h: ██████████┃████░░░░░ 75%
               d     yyyyyyyyyyByyyyddddd
      """

  Scenario: With reset turned on, each bar is followed by when its window resets and how long is left
    Given the session:
      """json
      {
        "model": {
          "display_name": "Opus 4.7"
        },
        "rate_limits": {
          "five_hour": {
            "used_percentage": 75,
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
      rate5h: {reset: true}
      rate7d: {reset: true}
      """
    When the status line renders
    Then the output is:
      """
      Opus 4.7 · 5h: ██████████┃████░░░░░ 75% resets 11:23 in 2h30m · 7d: ████████░░┃░░░░░░░░░ 40% resets Sun 20:53 in 3d12h
               d     yyyyyyyyyyByyyyddddd     ddddddddddddddddddddd d     ggggggggddBddddddddd     ddddddddddddddddddddddddd
      """

  Scenario: With reset turned on at 80 columns, each bar takes its own row because one row cannot hold both
    Given the session:
      """json
      {
        "model": {
          "display_name": "Opus 4.7"
        },
        "rate_limits": {
          "five_hour": {
            "used_percentage": 75,
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
      rate5h: {reset: true}
      rate7d: {reset: true}
      """
    And the environment variable "NO_COLOR" is "1"
    When the status line renders at 80 columns
    Then the text is:
      """
      Opus 4.7
      ↳          5h: █████┃█▌░░ 75% resets 11:23 in 2h30m
      ↳          7d: ████░┃░░░░ 40% resets Sun 20:53 in 3d12h
      """

  Scenario: The rate-bars command prints both windows as JSON for other tools to draw
    Given the rate-limit cache:
      """
      {"five_hour":{"used":75,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400}}
      """
    When the rate bars render for 4 to 5 cells
    Then the raw output is:
      """
      {"rate5h":{"ok":true,"percent":"75%","bars":{"4":"\u001b[33m██\u001b[0m\u001b[1m┃\u001b[0m\u001b[2m░\u001b[0m","5":"\u001b[33m██\u001b[0m\u001b[1m┃\u001b[0m\u001b[33m▊\u001b[0m\u001b[2m░\u001b[0m"}},"rate7d":{"ok":true,"percent":"40%","bars":{"4":"\u001b[32m█▋\u001b[0m\u001b[1m┃\u001b[0m\u001b[2m░\u001b[0m","5":"\u001b[32m██\u001b[0m\u001b[1m┃\u001b[0m\u001b[2m░░\u001b[0m"}}}
      """
